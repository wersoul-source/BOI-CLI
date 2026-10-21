package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/boi-family/boi-cli/internal/block"
	"github.com/boi-family/boi-cli/internal/block/port"
	coreblock "github.com/boi-family/boi-cli/internal/core"
	"github.com/boi-family/boi-cli/internal/equipment/tools/filesystem"
	"github.com/boi-family/boi-cli/internal/equipment/tools/process"
	"github.com/boi-family/boi-cli/internal/platform/update"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
)

// EnvironmentFile selects the Blocks of a workspace's Agent, inside .boi.
const EnvironmentFile = "environment.json"

// Pack is one Block pack the composition root can plug in: its manifest and
// the Tools it builds for a workspace.
type Pack struct {
	Manifest block.PackManifest
	Tools    func(*workspace.Sandbox) []port.Tool
}

// BuiltinPacks are the packs compiled into the binary, by id.
func BuiltinPacks() map[string]Pack {
	return map[string]Pack{
		"workspace-tools": {
			Manifest: block.PackManifest{
				SchemaVersion: block.PackSchemaVersion, ID: "workspace-tools", Kind: block.ToolPack, Version: "1.0.0",
				Provides: []string{"workspace.list", "workspace.read", "workspace.search", "workspace.write"},
				MaxRisk:  string(port.RiskChange),
			},
			Tools: filesystem.Tools,
		},
		"process-tools": {
			Manifest: block.PackManifest{
				SchemaVersion: block.PackSchemaVersion, ID: "process-tools", Kind: block.ToolPack, Version: "1.0.0",
				Provides: []string{"process.run"},
				MaxRisk:  string(port.RiskExecute),
			},
			Tools: func(sandbox *workspace.Sandbox) []port.Tool { return []port.Tool{process.Tool(sandbox)} },
		},
	}
}

// DefaultEnvironment is used when a workspace has no environment file. It
// composes the same Tools as v1.7.
func DefaultEnvironment() block.Environment {
	return block.Environment{SchemaVersion: block.PackSchemaVersion, Name: "default", Blocks: []block.EnvironmentBlock{
		{ID: "workspace-tools", Required: true},
		{ID: "process-tools", Required: false},
	}}
}

// LoadEnvironment reads .boi/environment.json, or returns DefaultEnvironment
// when the file does not exist. A file that exists but is invalid is an error:
// the user asked for a specific composition and must not silently get another.
func LoadEnvironment(boiDir string) (block.Environment, error) {
	data, err := os.ReadFile(filepath.Join(boiDir, EnvironmentFile))
	if errors.Is(err, os.ErrNotExist) {
		return DefaultEnvironment(), nil
	}
	if err != nil {
		return block.Environment{}, fmt.Errorf("read %s: %w", EnvironmentFile, err)
	}
	var environment block.Environment
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&environment); err != nil {
		return block.Environment{}, fmt.Errorf("parse %s: %w", EnvironmentFile, err)
	}
	if err := environment.Validate(); err != nil {
		return block.Environment{}, fmt.Errorf("%s: %w", EnvironmentFile, err)
	}
	return environment, nil
}

// Pack states in a composition report.
const (
	PackActive  = "active"
	PackSkipped = "skipped"
)

// CompositionEntry reports what happened to one pack named by the environment.
type CompositionEntry struct {
	ID       string
	Version  string
	Required bool
	Status   string
	Reason   string
	Tools    []string
}

// Composition is the resolved set of Blocks for one Agent.
type Composition struct {
	Environment string
	Entries     []CompositionEntry
	Tools       []port.Tool
}

// ToolNames lists the Tools the composition registers.
func (c *Composition) ToolNames() []string {
	names := make([]string, 0, len(c.Tools))
	for _, tool := range c.Tools {
		names = append(names, tool.Spec().Name)
	}
	return names
}

// Compose resolves an environment against the available packs before any
// task runs. A required pack that is unknown, incompatible or broken stops
// composition with its reason; an optional one is skipped and reported.
func Compose(environment block.Environment, packs map[string]Pack, sandbox *workspace.Sandbox, runtimeVersion string, capabilities coreblock.AgentEnvironment) (*Composition, error) {
	if err := environment.Validate(); err != nil {
		return nil, err
	}
	composition := &Composition{Environment: environment.Name}
	owner := map[string]string{}
	for _, item := range environment.Blocks {
		entry := CompositionEntry{ID: item.ID, Required: item.Required}
		tools, reason := resolvePack(item.ID, packs, sandbox, runtimeVersion, capabilities, &entry)
		for _, tool := range tools {
			if other, taken := owner[tool.Spec().Name]; taken && reason == "" {
				reason = fmt.Sprintf("Tool %s is already provided by %s", tool.Spec().Name, other)
			}
		}
		if reason != "" {
			if item.Required {
				return nil, fmt.Errorf("required Block %s: %s", item.ID, reason)
			}
			entry.Status, entry.Reason = PackSkipped, reason
			composition.Entries = append(composition.Entries, entry)
			continue
		}
		for _, tool := range tools {
			owner[tool.Spec().Name] = item.ID
			entry.Tools = append(entry.Tools, tool.Spec().Name)
		}
		entry.Status = PackActive
		composition.Tools = append(composition.Tools, tools...)
		composition.Entries = append(composition.Entries, entry)
	}
	return composition, nil
}

// resolvePack returns the pack's Tools, or the reason it cannot be used.
func resolvePack(id string, packs map[string]Pack, sandbox *workspace.Sandbox, runtimeVersion string, capabilities coreblock.AgentEnvironment, entry *CompositionEntry) ([]port.Tool, string) {
	pack, ok := packs[id]
	if !ok {
		return nil, "no such Block is installed"
	}
	manifest := pack.Manifest
	entry.Version = manifest.Version
	if err := manifest.Validate(); err != nil {
		return nil, err.Error()
	}
	if manifest.ID != id {
		return nil, fmt.Sprintf("manifest id %q does not match %q", manifest.ID, id)
	}
	if minimum, _ := manifest.Requires.MinimumRuntime(); minimum != "" && runtimeVersion != "" && update.CompareVersions(runtimeVersion, minimum) < 0 {
		return nil, fmt.Sprintf("needs runtime %s or later, this is %s", minimum, runtimeVersion)
	}
	for _, capability := range manifest.Requires.Capabilities {
		if !providesCapability(capabilities, capability) {
			return nil, fmt.Sprintf("Provider is not qualified for %s", capability)
		}
	}
	if sandbox == nil || pack.Tools == nil {
		return nil, "no workspace to bind Tools to"
	}
	ceiling := port.Risk(manifest.MaxRisk).Rank()
	if ceiling == 0 {
		return nil, fmt.Sprintf("unknown max_risk %q", manifest.MaxRisk)
	}
	tools := pack.Tools(sandbox)
	var built []string
	for _, tool := range tools {
		spec := tool.Spec()
		if spec.Risk.Rank() > ceiling {
			return nil, fmt.Sprintf("Tool %s has risk %s above the pack's max_risk %s", spec.Name, spec.Risk, manifest.MaxRisk)
		}
		built = append(built, spec.Name)
	}
	declared := append([]string(nil), manifest.Provides...)
	sort.Strings(declared)
	sort.Strings(built)
	if strings.Join(declared, ",") != strings.Join(built, ",") {
		return nil, fmt.Sprintf("declares %v but builds %v", declared, built)
	}
	return tools, ""
}

func providesCapability(environment coreblock.AgentEnvironment, capability string) bool {
	switch capability {
	case "completion":
		return environment.Completion
	case "tool_calling":
		return environment.ToolCalling
	case "native_tool_schema":
		return environment.NativeTools
	case "skill_calling":
		return environment.SkillCalling
	default:
		return false
	}
}
