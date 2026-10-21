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
}

// EnvironmentBlock selects one pack. A missing required pack stops start-up;
// a missing optional pack is skipped and reported.
type EnvironmentBlock struct {
	ID       string `json:"id"`
	Required bool   `json:"required"`
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
		if seen[item.ID] {
			return fmt.Errorf("environment %q lists Block %q twice", e.Name, item.ID)
		}
		seen[item.ID] = true
	}
	return nil
}
