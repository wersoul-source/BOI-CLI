package acceptance_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/acceptance/modelzoo"
	"github.com/boi-family/boi-cli/internal/agentfolder"
	"github.com/boi-family/boi-cli/internal/app"
	coreblock "github.com/boi-family/boi-cli/internal/core"
	"github.com/boi-family/boi-cli/internal/runtime/agent"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
	"github.com/boi-family/boi-cli/internal/service/provider/adapters"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
)

// qualifiedZooAgent takes one simulated model down the path a real Provider
// takes: the probe suite measures it, the profile is saved, and the
// application composes the Agent from that profile alone. Nothing in the
// test sets a Runtime switch by hand.
func qualifiedZooAgent(t *testing.T, trait modelzoo.Trait) (*app.Agent, *modelzoo.Endpoint, string) {
	t.Helper()
	t.Setenv("BOI_CONFIG_DIR", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(modelzoo.Marker+"\nsecond line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	boiDir := filepath.Join(root, ".boi")
	if err := app.EnsureCapabilityIndexes(boiDir); err != nil {
		t.Fatal(err)
	}
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	store, err := agentfolder.NewStore(filepath.Join(root, "agent-folder"))
	if err != nil {
		t.Fatal(err)
	}
	runtime := &app.Runtime{Version: "test", WorkspaceRoot: root, BoiDir: boiDir, Sandbox: sandbox, AgentFolder: store}

	endpoint := modelzoo.Start(t, trait)
	name := "zoo-" + string(trait)
	item := llmfactory.ConfiguredProvider{Provider: adapters.NewOpenAIProvider(name, "fixture-key", endpoint.URL, "zoo-model"), Name: name, Model: "zoo-model", EndpointClass: "local"}
	target := coreblock.ProviderTarget{Provider: item.Name, Model: item.Model, EndpointClass: item.EndpointClass}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	profile, err := coreblock.Qualifier{Timeout: 5 * time.Second}.Run(ctx, target, item.Provider)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.SaveProfile(boiDir, target, profile); err != nil {
		t.Fatal(err)
	}
	composed, err := runtime.BuildAgent([]llmfactory.ConfiguredProvider{item})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := composed.PrepareTask(zooTask); err != nil {
		t.Fatal(err)
	}
	return composed, endpoint, root
}

// The owner's case, end to end (ADR-006 worked example, P2.5). A model whose
// tool_observation probe returns a corrupted token used to get no Tools at
// all. Composed from its measured profile it now keeps its Tools, and every
// call is asked for.
func TestAModelThatFailedASafetyProbeKeepsItsToolsUnderApproval(t *testing.T) {
	composed, endpoint, root := qualifiedZooAgent(t, modelzoo.CorruptToken)
	if !composed.Environment.ToolCalling || !composed.Environment.ApprovalFloor || !composed.Service.ApprovalFloor() {
		t.Fatalf("the Agent was not composed under the floor: %+v", composed.Environment)
	}
	probes := endpoint.Requests()
	result, asked, _ := runCounting(t, composed.Service, agent.ApprovalApproved)
	if result == nil || result.Usage.ToolCalls == 0 {
		t.Fatalf("the model was offered no Tool, or called none: %s", describeResult(result))
	}
	if asked != result.Usage.ToolCalls {
		t.Fatalf("%d Tool call(s) ran with %d approval(s)", result.Usage.ToolCalls, asked)
	}
	if endpoint.Requests()-probes < 2 {
		t.Fatal("the model never received the Tool result")
	}
	log, _ := readTaskEvents(t, root)
	want := app.RecipeVersion(filepath.Join(root, ".boi"), composed.Qualified)
	if want == "" {
		t.Fatal("no Recipe version")
	}
	for _, event := range log.Events {
		if event.RecipeVersion != want {
			t.Fatalf("event %d carries recipe_version %q, want %q", event.Seq, event.RecipeVersion, want)
		}
	}
}

// D2 on the same path: the unattended run is refused before any Tool runs.
func TestAnUnattendedRunOfAFlooredModelIsRefused(t *testing.T) {
	composed, endpoint, _ := qualifiedZooAgent(t, modelzoo.CorruptToken)
	probes := endpoint.Requests()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, _ := composed.Service.Run(ctx, zooTask)
	if result == nil || result.StopReason != agent.StopNeedsApproval || endpoint.Requests()-probes != 1 {
		t.Fatalf("requests=%d %s", endpoint.Requests()-probes, describeResult(result))
	}
	if !strings.Contains(result.Notice, "safety check did not pass") {
		t.Fatalf("the refusal does not say why: %q", result.Notice)
	}
}

// A model that passed every probe is composed exactly as before: Tools on,
// no floor, nothing asked for a read.
func TestAModelThatPassedEverythingIsUnchanged(t *testing.T) {
	composed, _, _ := qualifiedZooAgent(t, modelzoo.Native)
	if !composed.Environment.ToolCalling || composed.Environment.ApprovalFloor || composed.Service.ApprovalFloor() {
		t.Fatalf("environment = %+v", composed.Environment)
	}
	result, asked, err := runCounting(t, composed.Service, agent.ApprovalRejected)
	if err != nil || result == nil || asked != 0 || !strings.Contains(result.Response, modelzoo.Marker) {
		t.Fatalf("err=%v asked=%d %s", err, asked, describeResult(result))
	}
}

// A dialect model is composed with Tools, assisted, and no floor.
func TestADialectModelIsComposedWithAssistedTools(t *testing.T) {
	composed, _, root := qualifiedZooAgent(t, modelzoo.DSML)
	if !composed.Environment.ToolCalling || composed.Environment.ApprovalFloor {
		t.Fatalf("environment = %+v", composed.Environment)
	}
	recipe := app.ProviderRecipe(filepath.Join(root, ".boi"), composed.Qualified[0])
	if recipe.State(coreblock.RecipeTools) != coreblock.StateAssisted || recipe.ToolProtocol != "dialect:dsml" {
		t.Fatalf("recipe tools=%s protocol=%s", recipe.State(coreblock.RecipeTools), recipe.ToolProtocol)
	}
	result, asked, err := runCounting(t, composed.Service, agent.ApprovalRejected)
	if err != nil || result == nil || asked != 0 || !strings.Contains(result.Response, modelzoo.Marker) {
		t.Fatalf("err=%v asked=%d %s", err, asked, describeResult(result))
	}
}
