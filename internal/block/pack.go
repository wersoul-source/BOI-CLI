package block

import (
	"fmt"
	"strings"
)

// PackSchemaVersion is the version of the Block pack and environment files.
const PackSchemaVersion = 1

// PackKind is the kind of capability a pack plugs in.
type PackKind string

const (
	ToolPack PackKind = "tool-pack"
	// SkillPack is a folder of Skill files named by an environment entry.
	SkillPack PackKind = "skill-pack"
)

// PackManifest declares one pluggable Block pack: what it provides and what it
// needs. It is data only. It can narrow what a pack may do (MaxRisk) but never
// widen it: each Tool's own spec stays the source of its risk and approval.
type PackManifest struct {
	SchemaVersion int          `json:"schema_version"`
	ID            string       `json:"id"`
	Kind          PackKind     `json:"kind"`
	Version       string       `json:"version"`
	Provides      []string     `json:"provides"`
	Requires      PackRequires `json:"requires"`
	MaxRisk       string       `json:"max_risk"`
}

// PackRequires lists what must hold before a pack is activated.
type PackRequires struct {
	// Runtime is a minimum version written as ">=1.8.0". Empty means any.
	Runtime string `json:"runtime,omitempty"`
	// Capabilities are qualified Provider capabilities, such as "tool_calling".
	Capabilities []string `json:"capabilities,omitempty"`
}

func (m PackManifest) Validate() error {
	if m.SchemaVersion != PackSchemaVersion {
		return fmt.Errorf("pack %q: unsupported schema version %d", m.ID, m.SchemaVersion)
	}
	if strings.TrimSpace(m.ID) == "" {
		return fmt.Errorf("pack has no id")
	}
	if m.Kind != ToolPack {
		return fmt.Errorf("pack %q: unsupported kind %q", m.ID, m.Kind)
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("pack %q has no version", m.ID)
	}
	if len(m.Provides) == 0 {
		return fmt.Errorf("pack %q provides nothing", m.ID)
	}
	if strings.TrimSpace(m.MaxRisk) == "" {
		return fmt.Errorf("pack %q has no max_risk", m.ID)
	}
	if _, err := m.Requires.MinimumRuntime(); err != nil {
		return fmt.Errorf("pack %q: %w", m.ID, err)
	}
	return nil
}

// MinimumRuntime returns the version after ">=", or "" when unset.
func (r PackRequires) MinimumRuntime() (string, error) {
	value := strings.TrimSpace(r.Runtime)
	if value == "" {
		return "", nil
	}
	if !strings.HasPrefix(value, ">=") || strings.TrimSpace(value[2:]) == "" {
		return "", fmt.Errorf("runtime requirement %q must be written as >=X.Y.Z", value)
	}
	return strings.TrimSpace(value[2:]), nil
}

// Environment names the packs one Agent is composed from.
type Environment struct {
	SchemaVersion int                `json:"schema_version"`
	Name          string             `json:"name"`
	Blocks        []EnvironmentBlock `json:"blocks"`
	// Trusted is set by the host, never read from the file: only a trusted
	// environment may start MCP server processes.
	Trusted bool `json:"-"`
	// Dir is the .boi directory the file was read from, set by the host. A
	// Skill folder is resolved inside it.
	Dir string `json:"-"`
}

// EnvironmentBlock selects one pack. A missing required pack stops start-up;
// a missing optional pack is skipped and reported.
type EnvironmentBlock struct {
	ID       string     `json:"id"`
	Required bool       `json:"required"`
	MCP      *MCPServer `json:"mcp,omitempty"`
	// Skills makes the entry a Skill pack: a folder of Skill files.
	Skills *SkillFolder `json:"skills,omitempty"`
}

// SkillFolder is a folder of *.skill.md files, relative to the workspace's
// .boi directory. A Skill is untrusted text; it adds no Tool and no authority.
type SkillFolder struct {
	Dir string `json:"dir"`
}

// MCPServer makes an environment entry an MCP Block: a local server process
// whose tools become approval-gated Tools named "mcp.<id>.<tool>". It adds
// capabilities without building BOI.
type MCPServer struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	// Tools limits which of the server's tools are plugged in; empty means all.
	Tools []string `json:"tools,omitempty"`
}

func (e Environment) Validate() error {
	if e.SchemaVersion != PackSchemaVersion {
		return fmt.Errorf("environment: unsupported schema version %d", e.SchemaVersion)
	}
	if strings.TrimSpace(e.Name) == "" {
		return fmt.Errorf("environment has no name")
	}
	seen := map[string]bool{}
	for _, item := range e.Blocks {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("environment %q lists a Block without id", e.Name)
		}
		if item.MCP != nil && strings.TrimSpace(item.MCP.Command) == "" {
			return fmt.Errorf("environment %q: MCP Block %q has no command", e.Name, item.ID)
		}
		if item.Skills != nil && item.MCP != nil {
			return fmt.Errorf("environment %q: Block %q cannot be both an MCP server and a Skill folder", e.Name, item.ID)
		}
		if item.Skills != nil && strings.TrimSpace(item.Skills.Dir) == "" {
			return fmt.Errorf("environment %q: Skill pack %q has no dir", e.Name, item.ID)
		}
		if strings.ContainsAny(item.ID, ". /\\") {
			return fmt.Errorf("environment %q: Block id %q may not contain dots, spaces or slashes", e.Name, item.ID)
		}
		if seen[item.ID] {
			return fmt.Errorf("environment %q lists Block %q twice", e.Name, item.ID)
		}
		seen[item.ID] = true
	}
	return nil
}
