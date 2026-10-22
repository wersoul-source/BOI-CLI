// Command codingbench measures whether BOI finishes real coding tasks with a
// real Provider: it edits files, runs the tests, and a host-side check (not
// the model's own claim) decides pass or fail.
//
//	go run ./scripts/acceptance/codingbench -runs 3
//
// The Provider comes from PSC_1_* variables, as for boi itself. Every write
// and command the Agent asks for is approved automatically, which is why this
// harness lives under scripts/ and is never part of the boi binary. It needs
// the go toolchain on PATH because the tasks are small Go modules.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/app"
	coreblock "github.com/boi-family/boi-cli/internal/core"
	"github.com/boi-family/boi-cli/internal/runtime/agent"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
)

type task struct {
	id     string
	prompt string
	files  map[string]string
	// protected files must not change; the Agent may not edit a test to pass.
	protected []string
	check     func(dir string) error
}

const goMod = "module bench\n\ngo 1.22\n"

var tasks = []task{
	{
		id:     "fix_bug",
		prompt: "The tests in this Go module fail. Find and fix the bug in calc.go without changing calc_test.go, then run `go test ./...` to confirm they pass.",
		files: map[string]string{
			"go.mod":       goMod,
			"calc.go":      "package bench\n\n// Add returns the sum of a and b.\nfunc Add(a, b int) int {\n\treturn a - b\n}\n",
			"calc_test.go": "package bench\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif got := Add(2, 3); got != 5 {\n\t\tt.Fatalf(\"Add(2, 3) = %d, want 5\", got)\n\t}\n}\n",
		},
		protected: []string{"calc_test.go"},
		check:     goTestPasses,
	},
	{
		id:     "implement",
		prompt: "Implement Reverse in strutil.go so it reverses a string by runes; Thai text must stay valid. Do not change strutil_test.go. Run `go test ./...` to confirm.",
		files: map[string]string{
			"go.mod":          goMod,
			"strutil.go":      "package bench\n\n// Reverse returns s with its runes in reverse order.\nfunc Reverse(s string) string {\n\t// TODO: implement\n\treturn s\n}\n",
			"strutil_test.go": "package bench\n\nimport \"testing\"\n\nfunc TestReverse(t *testing.T) {\n\tfor in, want := range map[string]string{\"abc\": \"cba\", \"\": \"\", \"สวัสดี\": \"ีดสัวส\"} {\n\t\tif got := Reverse(in); got != want {\n\t\t\tt.Errorf(\"Reverse(%q) = %q, want %q\", in, got, want)\n\t\t}\n\t}\n}\n",
		},
		protected: []string{"strutil_test.go"},
		check:     goTestPasses,
	},
	{
		id:     "edit_config",
		prompt: "Change the port in config.json to 8080 and keep every other setting unchanged.",
		files:  map[string]string{"config.json": "{\n  \"port\": 3000,\n  \"debug\": true,\n  \"name\": \"bench\"\n}\n"},
		check: func(dir string) error {
			var config map[string]any
			data, err := os.ReadFile(filepath.Join(dir, "config.json"))
			if err != nil {
				return err
			}
			if err := json.Unmarshal(data, &config); err != nil {
				return fmt.Errorf("config.json is no longer valid JSON: %w", err)
			}
			if config["port"] != 8080.0 || config["debug"] != true || config["name"] != "bench" || len(config) != 3 {
				return fmt.Errorf("config.json = %v", config)
			}
			return nil
		},
	},
	{
		id:     "rename",
		prompt: "Rename the function Greet to Hello in greet.go and update every caller, then run `go build ./...` to confirm the module still builds.",
		files: map[string]string{
			"go.mod":   goMod,
			"greet.go": "package bench\n\n// Greet returns a greeting for name.\nfunc Greet(name string) string {\n\treturn \"hello \" + name\n}\n",
			"use.go":   "package bench\n\n// Welcome greets the team.\nfunc Welcome() string {\n\treturn Greet(\"team\")\n}\n",
		},
		check: func(dir string) error {
			for _, name := range []string{"greet.go", "use.go"} {
				data, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					return err
				}
				if strings.Contains(string(data), "Greet(") || !strings.Contains(string(data), "Hello(") {
					return fmt.Errorf("%s was not renamed", name)
				}
			}
			return run(dir, "go", "build", "./...")
		},
	},
}

func goTestPasses(dir string) error { return run(dir, "go", "test", "./...") }

func run(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %s: %v\n%s", name, strings.Join(args, " "), err, output)
	}
	return nil
}

func digest(dir string, names []string) string {
	hash := sha256.New()
	for _, name := range names {
		data, _ := os.ReadFile(filepath.Join(dir, name))
		hash.Write(data)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

type row struct {
	task, result, stop, outcome, detail, trace      string
	providerCalls, toolCalls, approvals, tokens, ms int
}

func main() {
	runs := flag.Int("runs", 1, "runs per task")
	timeout := flag.Duration("timeout", 0, "limit per task (default: BOI_TASK_TIMEOUT or 2m, plus one minute)")
	flag.Parse()
	limits, err := app.LimitsFromEnv()
	if err != nil {
		fail(err.Error())
	}
	if *timeout <= 0 {
		*timeout = limits.Timeout + time.Minute
	}
	if _, err := exec.LookPath("go"); err != nil {
		fail("the tasks need the go toolchain on PATH")
	}
	configured, err := llmfactory.LoadConfiguredProvidersFromEnv()
	if err != nil || len(configured) == 0 {
		fail("set PSC_1_NAME, PSC_1_API_KEY and PSC_1_MODEL (and PSC_1_BASE_URL for OpenAI-compatible endpoints)")
	}
	configDir, _ := os.MkdirTemp("", "boi-codingbench-config-")
	defer os.RemoveAll(configDir)
	os.Setenv("BOI_CONFIG_DIR", configDir)
	qualify(configured)

	var rows []row
	for _, item := range tasks {
		for index := 0; index < *runs; index++ {
			result := runTask(item, configured, *timeout)
			rows = append(rows, result)
			fmt.Printf("%s\t%s\tcalls=%d\ttools=%d\tapprovals=%d\ttokens=%d\t%dms\t%s/%s\t%s\t%s\n", result.task, result.result, result.providerCalls, result.toolCalls, result.approvals, result.tokens, result.ms, result.stop, result.outcome, result.trace, result.detail)
		}
	}
	passed, falseDone, calls, toolCalls, tokens, ms := 0, 0, 0, 0, 0, 0
	for _, item := range rows {
		if item.result == "pass" {
			passed++
		} else if item.stop == "completed" {
			// The run ended as completed but the host check failed: the Agent
			// said it was done and was not.
			falseDone++
		}
		calls, toolCalls, tokens, ms = calls+item.providerCalls, toolCalls+item.toolCalls, tokens+item.tokens, ms+item.ms
	}
	fmt.Printf("\n== summary\nsuccess %d/%d, false completions %d, provider calls %d, tool calls %d, tokens %d, time %.1fs\n", passed, len(rows), falseDone, calls, toolCalls, tokens, float64(ms)/1000)
}

// qualify runs the real probes once; profiles land in the per-user directory
// every task workspace reads.
func qualify(configured []llmfactory.ConfiguredProvider) {
	scratch, _ := os.MkdirTemp("", "boi-codingbench-qualify-")
	defer os.RemoveAll(scratch)
	for _, item := range configured {
		target := coreblock.ProviderTarget{Provider: item.Name, Model: item.Model, EndpointClass: item.EndpointClass}
		profile, err := (coreblock.Qualifier{}).Run(context.Background(), target, item.Provider)
		if err != nil {
			fail("qualify " + item.Name + ": " + err.Error())
		}
		if _, err := app.SaveProfile(filepath.Join(scratch, ".boi"), target, profile); err != nil {
			fail(err.Error())
		}
		completed := false
		for _, result := range profile.Results {
			fmt.Printf("  %-20s %s\t%s\n", result.Capability, result.Status, oneLine(result.Detail))
			if result.Capability == "completion" && result.Status == coreblock.CapabilityPassed {
				completed = true
			}
		}
		// Without one completed request no task can run; stop here so the
		// Provider's reason is the last thing printed, not twelve failed tasks.
		if !completed {
			fail("provider " + item.Name + " did not complete a request; check PSC_1_BASE_URL, PSC_1_MODEL and the API key")
		}
	}
}

func runTask(item task, configured []llmfactory.ConfiguredProvider, timeout time.Duration) row {
	out := row{task: item.id, result: "fail"}
	dir, _ := os.MkdirTemp("", "boi-codingbench-"+item.id+"-")
	defer os.RemoveAll(dir)
	_ = os.Mkdir(filepath.Join(dir, ".git"), 0o755)
	for name, body := range item.files {
		_ = os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644)
	}
	before := digest(dir, item.protected)

	original, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() { _ = os.Chdir(original) }()
	runtime, err := app.NewRuntime("bench")
	if err == nil {
		err = runtime.EnsureWorkspaceState()
	}
	if err == nil {
		_, err = app.UsePreset(runtime.BoiDir, "coding", false)
	}
	var composed *app.Agent
	if err == nil {
		composed, err = runtime.BuildAgent(configured)
	}
	if err == nil {
		_, err = composed.PrepareTask(item.prompt)
	}
	if err != nil {
		out.detail = err.Error()
		return out
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	started := time.Now()
	var result *agent.AgentResult
	for event := range composed.Service.Start(ctx, item.prompt) {
		if event.Approval != nil {
			out.approvals++
			event.Approval.Decisions <- agent.ApprovalDecision{RequestID: event.Approval.Request.ID, State: agent.ApprovalApproved, DecidedAt: time.Now()}
		}
		if event.Result != nil {
			result = event.Result
		}
		if event.Engine != nil && event.Engine.Kind == "tool_result" && event.Engine.ToolCall != nil {
			// The order of Tool calls, "!" after one that failed, shows how
			// the model worked, not only whether it got there.
			step := strings.TrimPrefix(event.Engine.ToolCall.Tool, "workspace.")
			if event.Engine.ToolResult != nil && event.Engine.ToolResult.Status != agent.ToolSucceeded {
				step += "!"
			}
			if out.trace != "" {
				step = ">" + step
			}
			out.trace += step
		}
		if event.Engine != nil && event.Engine.Kind == "verification" && event.Engine.Verification != nil && !event.Engine.Verification.Passed {
			// The host sent an answer back (no Tool was used).
			if out.trace != "" {
				out.trace += ">"
			}
			out.trace += "answer?"
		}
		if event.Err != nil && out.detail == "" {
			out.detail = event.Err.Error()
		}
	}
	out.ms = int(time.Since(started).Milliseconds())
	if result != nil {
		out.stop, out.outcome = string(result.StopReason), string(result.Outcome)
		out.providerCalls, out.toolCalls, out.tokens = result.Usage.ProviderCalls, result.Usage.ToolCalls, result.Usage.TotalTokens()
	}
	// The Agent could not run at all (no qualified Provider, provider error):
	// its reason matters more than the unchanged workspace.
	if result == nil && out.detail != "" {
		out.detail = oneLine(out.detail)
		return out
	}
	if len(item.protected) > 0 && digest(dir, item.protected) != before {
		out.detail = "the Agent changed a file it was told not to change"
		return out
	}
	if err := item.check(dir); err != nil {
		out.detail = oneLine(err.Error())
		return out
	}
	out.result, out.detail = "pass", ""
	return out
}

func oneLine(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 200 {
		text = text[:200]
	}
	return text
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, "FAIL: "+message)
	os.Exit(1)
}
