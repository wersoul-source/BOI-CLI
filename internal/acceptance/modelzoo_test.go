package acceptance_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/acceptance/modelzoo"
	"github.com/boi-family/boi-cli/internal/agentfolder"
	coreblock "github.com/boi-family/boi-cli/internal/core"
	"github.com/boi-family/boi-cli/internal/core/persona"
	"github.com/boi-family/boi-cli/internal/equipment/tools/filesystem"
	"github.com/boi-family/boi-cli/internal/runtime/agent"
	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
	"github.com/boi-family/boi-cli/internal/service/provider/adapters"
)

// zooTask is the one scripted task every simulated endpoint runs.
const zooTask = "Read README.md and reply with its first line exactly."

// zooBaseline records what BOI does today with each simulated model. It is a
// characterisation, not a target: a row that reads "gap" is a limitation
// Architecture IV must close (ADR-006). Changing a row changes what a model
// can do, so the PR doing it answers Gap / Composition / Evidence.
type zooBaseline struct {
	// marker: the scripted task ends with the README marker in the reply.
	marker bool
	// rejected: the endpoint refused at least one request (HTTP 400).
	rejected bool
	// tools: ComposeEnvironment grants Tool Calling from the probe run.
	tools bool
	// completion: the completion probe passes.
	completion bool
	// assisted: the compensation native_tool_schema passed with, or "" when
	// the model did it alone. It must never be empty for a compensated pass.
	assisted string
	// fallback: the route change the run must report ("" when none).
	fallback string
	// misquote: the run must end with a notice that the answer altered a
	// value a Tool returned (audit H1, S2.3). Every other trait must not.
	misquote bool
	gap      string
}

var zooExpectations = map[modelzoo.Trait]zooBaseline{
	modelzoo.Native:          {marker: true, tools: true, completion: true},
	modelzoo.NoStream:        {marker: true, tools: true, completion: true, fallback: "stream→whole-response"},
	modelzoo.SlowFirstLoad:   {marker: true, tools: true, completion: true},
	modelzoo.MissingIndex:    {marker: true, tools: true, completion: true},
	modelzoo.ReasoningReplay: {marker: true, tools: true, completion: true},
	modelzoo.RejectsReplay:   {marker: true, rejected: true, tools: true, completion: true, fallback: "native→envelope"},
	modelzoo.DSML:            {marker: true, tools: true, completion: true, assisted: "dialect:dsml"},
	modelzoo.Hermes:          {marker: true, tools: true, completion: true, assisted: "dialect:hermes"},
	modelzoo.QwenXML:         {marker: true, tools: true, completion: true, assisted: "dialect:qwen-xml"},
	modelzoo.CorruptToken:    {completion: true, misquote: true, gap: "the answer is still wrong; it is now flagged. A wrong answer that is not an altered identifier would not be"},
	modelzoo.Down:            {gap: "none: completion fails, exclusion is correct (ADR-006)"},
}

func TestModelZooEveryTraitHasAnExpectation(t *testing.T) {
	for _, trait := range modelzoo.Traits {
		if _, ok := zooExpectations[trait]; !ok {
			t.Errorf("trait %s has no recorded baseline", trait)
		}
	}
	if len(zooExpectations) != len(modelzoo.Traits) {
		t.Errorf("baseline has %d rows for %d traits", len(zooExpectations), len(modelzoo.Traits))
	}
}

// TestModelZooScriptedTask runs the same task through the real Service and
// the real OpenAI adapter against every simulated endpoint.
func TestModelZooScriptedTask(t *testing.T) {
	for _, trait := range modelzoo.Traits {
		if trait == modelzoo.Down {
			continue // a lone dead endpoint has nothing to run; see the cluster test
		}
		trait := trait
		t.Run(string(trait), func(t *testing.T) {
			t.Parallel()
			want := zooExpectations[trait]
			endpoint := modelzoo.Start(t, trait)
			provider := adapters.NewOpenAIProvider("zoo-"+string(trait), "fixture-key", endpoint.URL, "zoo-model")
			result, err := runZooTask(t, []llm.Provider{provider})
			got := result != nil && err == nil && strings.Contains(result.Response, modelzoo.Marker)
			if got != want.marker {
				t.Errorf("marker in reply = %v, baseline %v (err=%v, result=%s)", got, want.marker, err, describeResult(result))
			}
			if rejected := endpoint.Rejected() > 0; rejected != want.rejected {
				t.Errorf("endpoint rejected a request = %v, baseline %v", rejected, want.rejected)
			}
			var kinds []string
			if result != nil {
				for _, fallback := range result.Fallbacks {
					kinds = append(kinds, fallback.Kind)
				}
			}
			if want.fallback == "" && len(kinds) != 0 || want.fallback != "" && (len(kinds) != 1 || kinds[0] != want.fallback) {
				t.Errorf("fallbacks = %v, baseline %q", kinds, want.fallback)
			}
			flagged := result != nil && len(result.Misquotes) > 0
			if flagged != want.misquote {
				t.Errorf("misquote flagged = %v, baseline %v (notice %q)", flagged, want.misquote, describeResult(result))
			}
			if want.misquote && (!strings.Contains(result.Notice, "ZOO-MARKER-public") || !strings.Contains(result.Notice, modelzoo.Marker)) {
				t.Errorf("the notice must name both values: %q", result.Notice)
			}
			if want.fallback != "" && (result == nil || !strings.Contains(result.Notice, want.fallback)) {
				t.Errorf("a fallback must reach the user's notice; notice = %q", describeResult(result))
			}
			t.Logf("%-16s marker=%-5v requests=%d rejected=%d streamed=%d gap=%q", trait, got, endpoint.Requests(), endpoint.Rejected(), endpoint.Streamed(), want.gap)
		})
	}
}

// TestModelZooClusterFailsOver: a dead replica ahead of a working one does
// not fail the task.
func TestModelZooClusterFailsOver(t *testing.T) {
	down := modelzoo.Start(t, modelzoo.Down)
	live := modelzoo.Start(t, modelzoo.Native)
	providers := []llm.Provider{
		adapters.NewOpenAIProvider("zoo-replica-a", "fixture-key", down.URL, "zoo-model"),
		adapters.NewOpenAIProvider("zoo-replica-b", "fixture-key", live.URL, "zoo-model"),
	}
	result, err := runZooTask(t, providers)
	if err != nil || result == nil || !strings.Contains(result.Response, modelzoo.Marker) {
		t.Fatalf("cluster task did not complete through the live replica: err=%v result=%s", err, describeResult(result))
	}
	if down.Requests() == 0 {
		t.Fatal("the dead replica was never tried; the test does not exercise failover")
	}
}

// TestModelZooQualification runs the real probe suite against every
// simulated endpoint and records what ComposeEnvironment grants.
func TestModelZooQualification(t *testing.T) {
	for _, trait := range modelzoo.Traits {
		trait := trait
		t.Run(string(trait), func(t *testing.T) {
			t.Parallel()
			want := zooExpectations[trait]
			endpoint := modelzoo.Start(t, trait)
			name := "zoo-" + string(trait)
			provider := adapters.NewOpenAIProvider(name, "fixture-key", endpoint.URL, "zoo-model")
			target := coreblock.ProviderTarget{Provider: name, Model: "zoo-model", EndpointClass: "local"}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			profile, err := coreblock.Qualifier{Timeout: 5 * time.Second}.Run(ctx, target, provider)
			if err != nil {
				t.Fatalf("qualification run: %v", err)
			}
			env := coreblock.ComposeEnvironment(profile)
			if env.Completion != want.completion {
				t.Errorf("completion = %v, baseline %v\n%s", env.Completion, want.completion, describeProfile(profile))
			}
			if env.ToolCalling != want.tools {
				t.Errorf("tool calling = %v, baseline %v\n%s", env.ToolCalling, want.tools, describeProfile(profile))
			}
			var assisted string
			for _, result := range profile.Results {
				if result.Capability == "native_tool_schema" {
					assisted = result.Assisted
				}
			}
			if assisted != want.assisted {
				t.Errorf("native_tool_schema assisted = %q, baseline %q: an assisted pass must never read as native\n%s", assisted, want.assisted, describeProfile(profile))
			}
			t.Logf("%-16s completion=%-5v tools=%-5v native=%-5v assisted=%q gap=%q", trait, env.Completion, env.ToolCalling, env.NativeTools, assisted, want.gap)
		})
	}
}

func runZooTask(t *testing.T, providers []llm.Provider) (*agent.AgentResult, error) {
	t.Helper()
	result, _, err := runZooTaskIn(t, providers)
	return result, err
}

// runZooTaskIn also returns the workspace root, where the Agent Folder keeps
// the task's checkpoint and event log.
func runZooTaskIn(t *testing.T, providers []llm.Provider) (*agent.AgentResult, string, error) {
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
	store, err := agentfolder.NewStore(filepath.Join(root, "agent-folder"))
	if err != nil {
		t.Fatal(err)
	}
	service.SetTaskRecorder(store)
	service.SetToolCallingAllowed(true)
	service.SetNativeToolCalling(true)
	result, runErr := runInteractive(t, service, zooTask, agent.ApprovalRejected)
	return result, root, runErr
}

func describeResult(result *agent.AgentResult) string {
	if result == nil {
		return "<nil>"
	}
	return fmt.Sprintf("outcome=%s stop=%s response=%q notice=%q", result.Outcome, result.StopReason, result.Response, result.Notice)
}

func describeProfile(profile *coreblock.CapabilityProfile) string {
	var b strings.Builder
	for _, r := range profile.Results {
		fmt.Fprintf(&b, "  %s: %s [%s] %s\n", r.Capability, r.Status, r.Assisted, r.Detail)
	}
	return b.String()
}

// TestRecoveredToolCallStillNeedsApproval: a call recovered from markup goes
// through the same Broker as a native one. A write that the user refuses is
// not performed, whatever dialect asked for it (ADR-006: no compensation may
// bypass a safety rule).
func TestRecoveredToolCallStillNeedsApproval(t *testing.T) {
	markup := "<tool_call>\n{\"name\": \"workspace__write\", \"arguments\": {\"path\": \"owned.txt\", \"content\": \"pwned\"}}\n</tool_call>"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		content := markup
		if strings.Contains(string(body), `"role":"tool"`) {
			content = "The write was not approved."
		}
		w.Header().Set("Content-Type", "application/json")
		encoded, _ := json.Marshal(content)
		fmt.Fprintf(w, `{"choices":[{"message":{"role":"assistant","content":%s},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`, encoded)
	}))
	t.Cleanup(server.Close)
	root := t.TempDir()
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	provider := adapters.NewOpenAIProvider("zoo-hermes-write", "fixture-key", server.URL, "zoo-model")
	service := agent.NewService(persona.CorePersona(), llm.NewRouter([]llm.Provider{provider}), nil, sandbox)
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	approvals := 0
	for event := range service.Start(ctx, "Write owned.txt with the word pwned.") {
		if event.Approval != nil {
			approvals++
			event.Approval.Decisions <- agent.ApprovalDecision{RequestID: event.Approval.Request.ID, State: agent.ApprovalRejected, DecidedAt: time.Now()}
		}
	}
	if approvals == 0 {
		t.Fatal("a write recovered from Hermes markup ran without asking for approval")
	}
	if _, err := os.Stat(filepath.Join(root, "owned.txt")); !os.IsNotExist(err) {
		t.Fatalf("a refused write recovered from markup changed the workspace: %v", err)
	}
}

// TestReasoningDoesNotCrossProviders: reasoning is state of the endpoint that
// produced it. When that replica dies mid-task and the Router fails over, the
// next Provider must not receive it, and the drop is reported (audit M1, F4).
func TestReasoningDoesNotCrossProviders(t *testing.T) {
	thinker := modelzoo.Start(t, modelzoo.ReasoningReplay)
	thinker.FailAfter(1)
	other := modelzoo.Start(t, modelzoo.Native)
	providers := []llm.Provider{
		adapters.NewOpenAIProvider("zoo-thinker", "fixture-key", thinker.URL, "zoo-model"),
		adapters.NewOpenAIProvider("zoo-other", "fixture-key", other.URL, "zoo-model"),
	}
	result, err := runZooTask(t, providers)
	if err != nil || result == nil || !strings.Contains(result.Response, modelzoo.Marker) {
		t.Fatalf("the task did not complete through the second Provider: err=%v result=%s", err, describeResult(result))
	}
	if other.Requests() == 0 {
		t.Fatal("the second Provider was never used; the test does not exercise failover")
	}
	if got := other.ReasoningReceived(); got != 0 {
		t.Fatalf("the second Provider received another Provider's reasoning in %d request(s)", got)
	}
	reported := false
	for _, fallback := range result.Fallbacks {
		reported = reported || fallback.Kind == "reasoning→dropped"
	}
	if !reported || !strings.Contains(result.Notice, "reasoning→dropped") {
		t.Fatalf("the dropped reasoning was not reported: fallbacks=%+v notice=%q", result.Fallbacks, result.Notice)
	}
}

// TestModelZooRecipes: the Recipe Core composes for every simulated model,
// from the real probe run (P2.1, P2.2). "floored" is the Tool state once the
// approval floor is verified (P2.3); until then that route stays closed.
func TestModelZooRecipes(t *testing.T) {
	type want struct {
		tools, floored coreblock.CapabilityState
		protocol       string
		floor          string
		usable         bool
	}
	native := want{coreblock.StateNative, coreblock.StateNative, coreblock.ProtocolNative, coreblock.FloorNone, true}
	dialect := func(name string) want {
		return want{coreblock.StateAssisted, coreblock.StateAssisted, coreblock.ProtocolDialectPrefix + name, coreblock.FloorNone, true}
	}
	expectations := map[modelzoo.Trait]want{
		modelzoo.Native: native, modelzoo.NoStream: native, modelzoo.SlowFirstLoad: native, modelzoo.MissingIndex: native,
		modelzoo.ReasoningReplay: native, modelzoo.RejectsReplay: native,
		modelzoo.DSML: dialect("dsml"), modelzoo.Hermes: dialect("hermes"), modelzoo.QwenXML: dialect("qwen-xml"),
		// The model the owner's Provider resembles: Tools are not cut. They
		// wait for the approval floor, and then run as assisted.
		modelzoo.CorruptToken: {coreblock.StateUnverified, coreblock.StateAssisted, coreblock.ProtocolNative, coreblock.FloorConfirm, true},
		modelzoo.Down:         {coreblock.StateBlocked, coreblock.StateBlocked, coreblock.ProtocolNone, coreblock.FloorNone, false},
	}
	for _, trait := range modelzoo.Traits {
		trait := trait
		t.Run(string(trait), func(t *testing.T) {
			t.Parallel()
			expected, ok := expectations[trait]
			if !ok {
				t.Fatalf("trait %s has no expected Recipe", trait)
			}
			endpoint := modelzoo.Start(t, trait)
			name := "zoo-" + string(trait)
			provider := adapters.NewOpenAIProvider(name, "fixture-key", endpoint.URL, "zoo-model")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			profile, err := coreblock.Qualifier{Timeout: 5 * time.Second}.Run(ctx, coreblock.ProviderTarget{Provider: name, Model: "zoo-model", EndpointClass: "local"}, provider)
			if err != nil {
				t.Fatal(err)
			}
			recipe := coreblock.ComposeRecipe(profile, coreblock.VerifiedCompensations{})
			floored := coreblock.ComposeRecipe(profile, coreblock.VerifiedCompensations{ApprovalFloor: true})
			tools, _ := recipe.Capability(coreblock.RecipeTools)
			got := want{tools.State, floored.State(coreblock.RecipeTools), recipe.ToolProtocol, recipe.ApprovalFloor, recipe.Usable()}
			if got != expected {
				t.Errorf("recipe = %+v, want %+v\nevidence: %s", got, expected, tools.Evidence)
			}
			// Until a compensation is verified the Recipe allows exactly what
			// the Runtime is given today.
			if env := coreblock.ComposeEnvironment(profile); env.ToolCalling != tools.State.Usable() || env.Completion != recipe.Usable() {
				t.Errorf("recipe and today's environment disagree: %+v vs tools=%s usable=%v", env, tools.State, recipe.Usable())
			}
			t.Logf("%-16s tools=%-10s floored=%-10s protocol=%-16s floor=%-7s version=%s", trait, got.tools, got.floored, got.protocol, got.floor, recipe.Version)
		})
	}
}
