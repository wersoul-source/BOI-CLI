package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/block"
	"github.com/boi-family/boi-cli/internal/block/port"
	coreblock "github.com/boi-family/boi-cli/internal/core"
	"github.com/boi-family/boi-cli/internal/equipment/tools/filesystem"
	"github.com/boi-family/boi-cli/internal/equipment/tools/mcp"
	"github.com/boi-family/boi-cli/internal/equipment/tools/process"
	"github.com/boi-family/boi-cli/internal/platform/update"
	"github.com/boi-family/boi-cli/internal/runtime/agent"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
	"github.com/boi-family/boi-cli/internal/subagent"
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
				Provides: []string{"workspace.list", "workspace.read", "workspace.search", "workspace.edit", "workspace.write"},
				MaxRisk:  string(port.RiskChange),
			},
			Tools: filesystem.Tools,
		},
		"workspace-read": {
			Manifest: block.PackManifest{
				SchemaVersion: block.PackSchemaVersion, ID: "workspace-read", Kind: block.ToolPack, Version: "1.0.0",
				Provides: []string{"workspace.list", "workspace.read", "workspace.search"},
				MaxRisk:  string(port.RiskRead),
			},
			Tools: func(sandbox *workspace.Sandbox) []port.Tool {
				var readOnly []port.Tool
				for _, tool := range filesystem.Tools(sandbox) {
					if tool.Spec().Risk == port.RiskRead {
						readOnly = append(readOnly, tool)
					}
				}
				return readOnly
			},
		},
		"subagent": {
			Manifest: block.PackManifest{
				SchemaVersion: block.PackSchemaVersion, ID: "subagent", Kind: block.ToolPack, Version: "1.0.0",
				Provides: []string{agent.DelegateTool},
				Requires: block.PackRequires{Capabilities: []string{"tool_calling"}},
				MaxRisk:  string(port.RiskExternal),
			},
			Tools: func(*workspace.Sandbox) []port.Tool { return []port.Tool{subagent.NewTool()} },
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
	return block.Environment{SchemaVersion: block.PackSchemaVersion, Name: "default", Trusted: true, Blocks: []block.EnvironmentBlock{
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
	environment.Trusted = environmentTrusted(boiDir, data)
	return environment, nil
}

// Pack states in a composition report.
const (
	PackActive  = "active"
	PackSkipped = "skipped"
)

// CompositionEntry reports what happened to one pack named by the environment.
type CompositionEntry struct {
	ID string
	// Kind is "tool-pack" for a compiled-in pack or "mcp" for an MCP server.
	Kind     string
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
	return compose(environment, packs, sandbox, runtimeVersion, capabilities, true)
}

// compose resolves every pack. With strict, a refused required pack stops
// composition; without it (for reports) the pack is recorded as skipped and
// composition goes on, so a report can show every problem at once.
func compose(environment block.Environment, packs map[string]Pack, sandbox *workspace.Sandbox, runtimeVersion string, capabilities coreblock.AgentEnvironment, strict bool) (*Composition, error) {
	if err := environment.Validate(); err != nil {
		return nil, err
	}
	composition := &Composition{Environment: environment.Name}
	owner := map[string]string{}
	for _, item := range environment.Blocks {
		entry := CompositionEntry{ID: item.ID, Required: item.Required, Kind: string(block.ToolPack)}
		var tools []port.Tool
		var reason string
		if item.MCP != nil {
			entry.Kind = mcpKind
			if environment.Trusted {
				tools, reason = resolveMCP(item, packs, runtimeVersion)
			} else {
				// A cloned repository may ship this file; starting its
				// commands unasked would run untrusted code.
				reason = "MCP servers in this environment.json are not trusted yet; review the file, then run 'boi env trust'"
			}
		} else {
			tools, reason = resolvePack(item.ID, packs, sandbox, runtimeVersion, capabilities, &entry)
		}
		for _, tool := range tools {
			if other, taken := owner[tool.Spec().Name]; taken && reason == "" {
				reason = fmt.Sprintf("Tool %s is already provided by %s", tool.Spec().Name, other)
			}
		}
		if reason != "" {
			if item.Required && strict {
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

const mcpKind = "mcp"

// MCPDiscoveryTimeout bounds how long start-up waits for an MCP server to
// list its tools.
var MCPDiscoveryTimeout = 15 * time.Second

// resolveMCP starts an MCP server once to list its tools and adapts them.
// Every MCP Tool is external and needs approval; the server never receives
// the user's Provider keys (see mcp.serverEnvironment).
func resolveMCP(item block.EnvironmentBlock, packs map[string]Pack, runtimeVersion string) ([]port.Tool, string) {
	if _, builtin := packs[item.ID]; builtin {
		return nil, "the id is already used by a built-in pack"
	}
	client := mcp.NewClient()
	client.Version = runtimeVersion
	if err := client.Register(mcp.ServerConfig{Name: item.ID, Command: item.MCP.Command, Args: item.MCP.Args, Transport: "stdio"}); err != nil {
		return nil, err.Error()
	}
	ctx, cancel := context.WithTimeout(context.Background(), MCPDiscoveryTimeout)
	defer cancel()
	definitions, err := client.DiscoverToolsContext(ctx, item.ID)
	if err != nil {
		return nil, "MCP server did not list its tools: " + err.Error()
	}
	if len(item.MCP.Tools) > 0 {
		byName := make(map[string]mcp.ToolDef, len(definitions))
		for _, definition := range definitions {
			byName[definition.Name] = definition
		}
		selected := make([]mcp.ToolDef, 0, len(item.MCP.Tools))
		for _, name := range item.MCP.Tools {
			definition, ok := byName[name]
			if !ok {
				return nil, fmt.Sprintf("MCP server has no tool %q", name)
			}
			selected = append(selected, definition)
		}
		definitions = selected
	}
	if len(definitions) == 0 {
		return nil, "MCP server offers no tools"
	}
	tools, err := mcp.ToolsFromDefinitions(item.ID, definitions, client)
	if err != nil {
		return nil, err.Error()
	}
	for _, tool := range tools {
		if err := tool.Spec().Validate(); err != nil {
			return nil, err.Error()
		}
	}
	return tools, ""
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
