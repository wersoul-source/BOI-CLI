package acceptance_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/acceptance/modelzoo"
	"github.com/boi-family/boi-cli/internal/core/persona"
	"github.com/boi-family/boi-cli/internal/equipment/tools/filesystem"
	"github.com/boi-family/boi-cli/internal/runtime/agent"
	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
	"github.com/boi-family/boi-cli/internal/service/provider/adapters"
)

// flooredZooService is an Agent under the approval floor, with the workspace
// Tools and the README the zoo's scripted task reads.
func flooredZooService(t *testing.T, providers []llm.Provider) *agent.Service {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(modelzoo.Marker+"\nsecond line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	service := agent.NewService(persona.CorePersona(), llm.NewRouter(providers), nil, sandbox)
	var names []string
	for _, tool := range filesystem.Tools(sandbox) {
		if err := service.RegisterTool(tool); err != nil {
			t.Fatal(err)
		}
		names = append(names, tool.Spec().Name)
	}
	if err := service.SetActiveTools(names); err != nil {
		t.Fatal(err)
	}
	service.SetToolCallingAllowed(true)
	service.SetNativeToolCalling(true)
	service.SetApprovalFloor(true)
	return service
}

// runCounting answers every approval request with decision and counts them.
func runCounting(t *testing.T, service *agent.Service, decision agent.ApprovalState) (*agent.AgentResult, int, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var result *agent.AgentResult
	var runErr error
	asked := 0
	for event := range service.Start(ctx, zooTask) {
		if event.Approval != nil {
			asked++
			event.Approval.Decisions <- agent.ApprovalDecision{RequestID: event.Approval.Request.ID, State: decision, DecidedAt: time.Now()}
		}
		if event.Result != nil || event.Err != nil {
			result, runErr = event.Result, event.Err
		}
	}
	return result, asked, runErr
}

// TestApprovalFloorOnTheModelZoo is the end-to-end evidence for the approval
// floor (ADR-006 D1-D3, P2.3). Under the floor, for every kind of model that
// can call Tools: each Tool call is asked for, exactly once, whatever
// protocol carried it; a batch of parallel reads is asked for one by one;
// and a fallback from native calls to text envelopes stays under the floor.
func TestApprovalFloorOnTheModelZoo(t *testing.T) {
	for _, trait := range []modelzoo.Trait{modelzoo.Native, modelzoo.NoStream, modelzoo.DSML, modelzoo.Hermes, modelzoo.QwenXML, modelzoo.ReasoningReplay, modelzoo.RejectsReplay, modelzoo.MissingIndex, modelzoo.CorruptToken} {
		trait := trait
		t.Run(string(trait), func(t *testing.T) {
			t.Parallel()
			endpoint := modelzoo.Start(t, trait)
			service := flooredZooService(t, []llm.Provider{adapters.NewOpenAIProvider("zoo-"+string(trait), "fixture-key", endpoint.URL, "zoo-model")})
			result, asked, err := runCounting(t, service, agent.ApprovalApproved)
			if err != nil || result == nil {
				t.Fatalf("run: err=%v", err)
			}
			if result.Usage.ToolCalls == 0 {
				t.Fatalf("no Tool call was made, so the floor was not exercised: %s", describeResult(result))
			}
			if asked != result.Usage.ToolCalls {
				t.Fatalf("%d Tool call(s) ran but approval was asked %d time(s)", result.Usage.ToolCalls, asked)
			}
			if trait == modelzoo.MissingIndex && asked != 2 {
				t.Fatalf("the two parallel reads were asked for %d time(s), want 2", asked)
			}
			// The floor adds approval; it does not stop the work.
			if trait != modelzoo.CorruptToken && !strings.Contains(result.Response, modelzoo.Marker) {
				t.Fatalf("the approved task did not complete: %s", describeResult(result))
			}
		})
	}
}

// A refused call does not run: the model never sees the file.
func TestRefusedFlooredCallDoesNotRun(t *testing.T) {
	endpoint := modelzoo.Start(t, modelzoo.CorruptToken)
	service := flooredZooService(t, []llm.Provider{adapters.NewOpenAIProvider("zoo-corrupt", "fixture-key", endpoint.URL, "zoo-model")})
	result, asked, err := runCounting(t, service, agent.ApprovalRejected)
	// A stopped task comes back as a result together with an error.
	if result == nil || asked != 1 {
		t.Fatalf("err=%v asked=%d result=%s", err, asked, describeResult(result))
	}
	if result.StopReason != agent.StopRejected || strings.Contains(result.Response, "ZOO-MARKER") || endpoint.Requests() != 1 {
		t.Fatalf("a refused read still reached the model: requests=%d %s", endpoint.Requests(), describeResult(result))
	}
}

// D2: an unattended run under the floor is refused before any Tool runs, and
// says why and what to do.
func TestUnattendedRunUnderTheFloorIsRefused(t *testing.T) {
	endpoint := modelzoo.Start(t, modelzoo.CorruptToken)
	service := flooredZooService(t, []llm.Provider{adapters.NewOpenAIProvider("zoo-corrupt", "fixture-key", endpoint.URL, "zoo-model")})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := service.Run(ctx, zooTask)
	if result == nil {
		t.Fatalf("run: %v", err)
	}
	if result.StopReason != agent.StopNeedsApproval || endpoint.Requests() != 1 || strings.Contains(result.Response, "ZOO-MARKER") {
		t.Fatalf("the unattended run was not refused: requests=%d %s", endpoint.Requests(), describeResult(result))
	}
	for _, want := range []string{"needs approval", "safety check did not pass", "interactively"} {
		if !strings.Contains(result.Notice, want) {
			t.Errorf("the notice lacks %q: %q", want, result.Notice)
		}
	}
}

// D3: the floor belongs to the Agent, so a failover to another Provider is
// still under it. No right grows from a Provider switch.
func TestFailoverStaysUnderTheFloor(t *testing.T) {
	down := modelzoo.Start(t, modelzoo.Down)
	live := modelzoo.Start(t, modelzoo.Native)
	service := flooredZooService(t, []llm.Provider{
		adapters.NewOpenAIProvider("zoo-replica-a", "fixture-key", down.URL, "zoo-model"),
		adapters.NewOpenAIProvider("zoo-replica-b", "fixture-key", live.URL, "zoo-model"),
	})
	result, asked, err := runCounting(t, service, agent.ApprovalApproved)
	if err != nil || result == nil || down.Requests() == 0 || live.Requests() == 0 {
		t.Fatalf("failover was not exercised: err=%v down=%d live=%d", err, down.Requests(), live.Requests())
	}
	if result.Usage.ToolCalls == 0 || asked != result.Usage.ToolCalls || !strings.Contains(result.Response, modelzoo.Marker) {
		t.Fatalf("after failover %d call(s) ran with %d approval(s): %s", result.Usage.ToolCalls, asked, describeResult(result))
	}
}

// Without the floor the same model's reads are automatic: the floor is what
// makes the difference, not the test setup.
func TestWithoutTheFloorReadsAreAutomatic(t *testing.T) {
	endpoint := modelzoo.Start(t, modelzoo.Native)
	service := flooredZooService(t, []llm.Provider{adapters.NewOpenAIProvider("zoo-native", "fixture-key", endpoint.URL, "zoo-model")})
	service.SetApprovalFloor(false)
	result, asked, err := runCounting(t, service, agent.ApprovalRejected)
	if err != nil || result == nil || asked != 0 || !strings.Contains(result.Response, modelzoo.Marker) {
		t.Fatalf("err=%v asked=%d %s", err, asked, describeResult(result))
	}
}
