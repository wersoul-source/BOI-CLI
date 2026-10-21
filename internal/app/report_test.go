package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/block"
	coreblock "github.com/boi-family/boi-cli/internal/core"
	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
)

type idleProvider struct{}

func (idleProvider) Name() string { return "fake" }
func (idleProvider) Complete(context.Context, llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return nil, errors.New("unused")
}
func (idleProvider) Stream(context.Context, llm.CompletionRequest) (<-chan llm.Token, error) {
	return nil, errors.New("unused")
}

// qualifiedFake writes a passing profile so the fake Provider allows Tool calling.
func qualifiedFake(t *testing.T, boiDir string) []llmfactory.ConfiguredProvider {
	t.Helper()
	item := llmfactory.ConfiguredProvider{Name: "fake", Model: "m", EndpointClass: "custom", Provider: idleProvider{}}
	target := coreblock.ProviderTarget{Provider: item.Name, Model: item.Model, EndpointClass: item.EndpointClass}
	profile := &coreblock.CapabilityProfile{SchemaVersion: 1, ProbeVersion: coreblock.ProbeSuiteVersion, Fingerprint: coreblock.TargetFingerprint(target, coreblock.ProbeSuiteVersion), Target: target, GeneratedAt: time.Now(), Results: []coreblock.CapabilityResult{
		{Capability: "completion", Status: coreblock.CapabilityPassed},
		{Capability: "tool_calling", Status: coreblock.CapabilityPassed},
		{Capability: "tool_observation", Status: coreblock.CapabilityPassed},
		{Capability: "authority", Status: coreblock.CapabilityPassed},
	}}
	if err := coreblock.SaveCapabilityProfile(coreblock.CapabilityProfilePath(boiDir, target), profile); err != nil {
		t.Fatal(err)
	}
	return []llmfactory.ConfiguredProvider{item}
}

func reportRuntime(t *testing.T, environment *block.Environment) *Runtime {
	t.Helper()
	root := t.TempDir()
	boiDir := filepath.Join(root, ".boi")
	if err := EnsureCapabilityIndexes(boiDir); err != nil {
		t.Fatal(err)
	}
	if environment != nil {
		data, _ := json.Marshal(environment)
		if err := os.WriteFile(filepath.Join(boiDir, EnvironmentFile), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return &Runtime{Version: "1.10.0", WorkspaceRoot: root, BoiDir: boiDir, Sandbox: testSandbox(t)}
}

func blockByID(t *testing.T, report *AgentReport, id string) BlockReport {
	t.Helper()
	for _, item := range report.Blocks {
		if item.ID == id {
			return item
		}
	}
	t.Fatalf("Block %s missing from report: %+v", id, report.Blocks)
	return BlockReport{}
}

func TestReportMatchesWhatTheBrokerHolds(t *testing.T) {
	runtime := reportRuntime(t, nil)
	configured := qualifiedFake(t, runtime.BoiDir)
	report, err := runtime.Report(configured)
	if err != nil || !report.Ready {
		t.Fatalf("report=%+v err=%v", report, err)
	}

	built, err := runtime.BuildAgent(configured)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := built.PrepareTask(""); err != nil {
		t.Fatal(err)
	}
	registered, offered := setOf(built.Service.RegisteredTools()), setOf(built.Service.OfferedTools())
	reported := map[string]ToolReport{}
	for _, item := range report.Blocks {
		if item.State != StateActive {
			t.Errorf("built-in Block %s should be active with a qualified Provider: %+v", item.ID, item)
		}
		for _, tool := range item.Tools {
			reported[tool.Name] = tool
		}
	}
	for _, tool := range report.RuntimeTools {
		reported[tool.Name] = tool
	}
	for name := range registered {
		tool, ok := reported[name]
		if !ok || !tool.Registered || tool.Offered != offered[name] {
			t.Errorf("report disagrees with the Broker for %s: %+v (offered=%v)", name, tool, offered[name])
		}
	}
	if len(reported) != len(registered) {
		t.Errorf("report lists %d Tools, Broker holds %d", len(reported), len(registered))
	}
}

func TestReportSeparatesDeclaredLoadedAndActive(t *testing.T) {
	original := MCPDiscoveryTimeout
	MCPDiscoveryTimeout = 500 * time.Millisecond
	t.Cleanup(func() { MCPDiscoveryTimeout = original })
	environment := environmentOf(
		block.EnvironmentBlock{ID: "workspace-tools", Required: true},
		block.EnvironmentBlock{ID: "dead", MCP: &block.MCPServer{Command: filepath.Join(t.TempDir(), "missing")}},
	)

	runtime := reportRuntime(t, &environment)
	report, err := runtime.Report(nil)
	if err != nil {
		t.Fatal(err)
	}
	workspace := blockByID(t, report, "workspace-tools")
	if workspace.State != StateLoaded || workspace.Reason != "no qualified Provider allows Tool calling" {
		t.Fatalf("without a qualified Provider the Tools are loaded, not active: %+v", workspace)
	}
	if dead := blockByID(t, report, "dead"); dead.State != StateDeclared || dead.Reason == "" || len(dead.Tools) != 0 {
		t.Fatalf("an MCP server that does not answer is only declared: %+v", dead)
	}

	runtime = reportRuntime(t, &environment)
	report, err = runtime.Report(qualifiedFake(t, runtime.BoiDir))
	if err != nil {
		t.Fatal(err)
	}
	if blockByID(t, report, "workspace-tools").State != StateActive || blockByID(t, report, "dead").State != StateDeclared {
		t.Fatalf("with a qualified Provider: %+v", report.Blocks)
	}
}

func TestReportShowsEveryProblemInsteadOfStopping(t *testing.T) {
	environment := environmentOf(
		block.EnvironmentBlock{ID: "missing-one", Required: true},
		block.EnvironmentBlock{ID: "workspace-tools", Required: true},
	)
	runtime := reportRuntime(t, &environment)
	report, err := runtime.Report(nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Ready || report.Problem == "" {
		t.Fatalf("a missing required Block must make the Agent not ready: %+v", report)
	}
	if missing := blockByID(t, report, "missing-one"); missing.State != StateDeclared || missing.Reason != "no such Block is installed" {
		t.Fatalf("missing Block: %+v", missing)
	}
	if len(blockByID(t, report, "workspace-tools").Tools) != 4 {
		t.Fatal("the report must still list the Blocks after the failing one")
	}

	if err := os.WriteFile(filepath.Join(runtime.BoiDir, EnvironmentFile), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if report, err = runtime.Report(nil); err != nil || report.Ready || report.Problem == "" {
		t.Fatalf("an unreadable environment must be reported, not crash: %+v %v", report, err)
	}
}
