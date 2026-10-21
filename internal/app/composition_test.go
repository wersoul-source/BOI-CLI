package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/block"
	"github.com/boi-family/boi-cli/internal/block/port"
	coreblock "github.com/boi-family/boi-cli/internal/core"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
)

func testSandbox(t *testing.T) *workspace.Sandbox {
	t.Helper()
	sandbox, err := workspace.NewSandbox(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return sandbox
}

func environmentOf(blocks ...block.EnvironmentBlock) block.Environment {
	return block.Environment{SchemaVersion: block.PackSchemaVersion, Name: "test", Trusted: true, Blocks: blocks}
}

type riskyTool struct{ name string }

func (r riskyTool) Spec() port.ToolSpec {
	return port.ToolSpec{Name: r.name, Usage: r.name + "()", Risk: port.RiskExecute, Approval: port.ApprovalConfirm, Timeout: time.Second}
}
func (riskyTool) Describe(map[string]any) port.ToolPreview { return port.ToolPreview{} }
func (riskyTool) Execute(context.Context, map[string]any) (port.ToolOutput, error) {
	return port.ToolOutput{Output: "ran"}, nil
}

func pack(id, maxRisk string, requires block.PackRequires, tools ...string) Pack {
	return Pack{
		Manifest: block.PackManifest{SchemaVersion: block.PackSchemaVersion, ID: id, Kind: block.ToolPack, Version: "1.0.0", Provides: tools, Requires: requires, MaxRisk: maxRisk},
		Tools: func(*workspace.Sandbox) []port.Tool {
			built := make([]port.Tool, 0, len(tools))
			for _, name := range tools {
				built = append(built, riskyTool{name})
			}
			return built
		},
	}
}

func TestDefaultEnvironmentComposesTheV17ToolSet(t *testing.T) {
	composition, err := Compose(DefaultEnvironment(), BuiltinPacks(), testSandbox(t), "1.8.0", coreblock.AgentEnvironment{})
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(composition.ToolNames(), ",")
	want := "workspace.list,workspace.read,workspace.search,workspace.edit,workspace.write,process.run"
	if got != want {
		t.Fatalf("default Tools = %s, want %s", got, want)
	}
	for _, entry := range composition.Entries {
		if entry.Status != PackActive {
			t.Fatalf("built-in pack %s not active: %s", entry.ID, entry.Reason)
		}
	}
}

func TestRemovingAnOptionalPackKeepsTheRest(t *testing.T) {
	environment := environmentOf(block.EnvironmentBlock{ID: "workspace-tools", Required: true})
	composition, err := Compose(environment, BuiltinPacks(), testSandbox(t), "1.8.0", coreblock.AgentEnvironment{})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range composition.ToolNames() {
		if name == "process.run" {
			t.Fatal("a pack left out of the environment must not be registered")
		}
	}
	if len(composition.Tools) != 5 {
		t.Fatalf("workspace Tools missing: %v", composition.ToolNames())
	}
}

func TestCompositionRules(t *testing.T) {
	packs := map[string]Pack{
		"too-risky":   pack("too-risky", string(port.RiskRead), block.PackRequires{}, "x.run"),
		"future":      pack("future", string(port.RiskExecute), block.PackRequires{Runtime: ">=9.0.0"}, "y.run"),
		"needs-tools": pack("needs-tools", string(port.RiskExecute), block.PackRequires{Capabilities: []string{"tool_calling"}}, "z.run"),
		"duplicate":   pack("duplicate", string(port.RiskExecute), block.PackRequires{}, "workspace.read"),
		"liar": {
			Manifest: block.PackManifest{SchemaVersion: 1, ID: "liar", Kind: block.ToolPack, Version: "1", Provides: []string{"declared.only"}, MaxRisk: "execute"},
			Tools:    func(*workspace.Sandbox) []port.Tool { return []port.Tool{riskyTool{"hidden.tool"}} },
		},
	}
	for id, item := range BuiltinPacks() {
		packs[id] = item
	}
	cases := map[string]string{
		"missing":     "no such Block is installed",
		"too-risky":   "above the pack's max_risk",
		"future":      "needs runtime 9.0.0 or later",
		"needs-tools": "not qualified for tool_calling",
		"duplicate":   "already provided by workspace-tools",
		"liar":        "declares [declared.only] but builds [hidden.tool]",
	}
	for id, reason := range cases {
		optional := environmentOf(block.EnvironmentBlock{ID: "workspace-tools", Required: true}, block.EnvironmentBlock{ID: id})
		composition, err := Compose(optional, packs, testSandbox(t), "1.8.0", coreblock.AgentEnvironment{})
		if err != nil {
			t.Fatalf("%s: an optional pack must never stop composition: %v", id, err)
		}
		last := composition.Entries[len(composition.Entries)-1]
		if last.Status != PackSkipped || !strings.Contains(last.Reason, reason) {
			t.Errorf("%s: entry %+v, want skipped with %q", id, last, reason)
		}
		if len(composition.Tools) != 5 {
			t.Errorf("%s: a skipped pack must not add or remove other Tools: %v", id, composition.ToolNames())
		}

		required := environmentOf(block.EnvironmentBlock{ID: "workspace-tools", Required: true}, block.EnvironmentBlock{ID: id, Required: true})
		if _, err := Compose(required, packs, testSandbox(t), "1.8.0", coreblock.AgentEnvironment{}); err == nil || !strings.Contains(err.Error(), reason) {
			t.Errorf("%s: a required pack must stop composition with %q, got %v", id, reason, err)
		}
	}
}

func TestLoadEnvironment(t *testing.T) {
	boiDir := t.TempDir()
	environment, err := LoadEnvironment(boiDir)
	if err != nil || environment.Name != "default" {
		t.Fatalf("no file must give the default environment: %+v %v", environment, err)
	}
	write := func(body string) {
		if err := os.WriteFile(filepath.Join(boiDir, EnvironmentFile), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"schema_version":1,"name":"read-only","blocks":[{"id":"workspace-tools","required":true}]}`)
	if environment, err = LoadEnvironment(boiDir); err != nil || environment.Name != "read-only" || len(environment.Blocks) != 1 {
		t.Fatalf("valid file not loaded: %+v %v", environment, err)
	}
	for name, body := range map[string]string{
		"broken json":   `{`,
		"unknown field": `{"schema_version":1,"name":"x","blocks":[],"extra":true}`,
		"wrong schema":  `{"schema_version":2,"name":"x","blocks":[]}`,
		"duplicate":     `{"schema_version":1,"name":"x","blocks":[{"id":"a"},{"id":"a"}]}`,
		"no name":       `{"schema_version":1,"name":"","blocks":[]}`,
	} {
		write(body)
		if _, err := LoadEnvironment(boiDir); err == nil {
			t.Errorf("%s: an invalid environment file must be an error, never a silent default", name)
		}
	}
}
