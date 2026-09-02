package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const (
	IdentitySchemaVersion = 1
	CorePersonaName       = "boi"
	IdentityFilename      = "agent.yaml"
	DefaultAgentName      = "BOI Agent"
)

// Identity is the persisted identity of one user-named Agent instance.
// Persona is fixed by Core and is not a user-selectable runtime mode.
type Identity struct {
	SchemaVersion int       `yaml:"schema_version"`
	ID            string    `yaml:"id"`
	Name          string    `yaml:"name"`
	CorePersona   string    `yaml:"core_persona"`
	CreatedAt     time.Time `yaml:"created_at"`
}

func NewIdentity(name string, now time.Time) (*Identity, error) {
	name = strings.TrimSpace(name)
	if now.IsZero() {
		now = time.Now()
	}
	identity := &Identity{
		SchemaVersion: IdentitySchemaVersion,
		ID:            fmt.Sprintf("agent_%d", now.UnixNano()),
		Name:          name,
		CorePersona:   CorePersonaName,
		CreatedAt:     now.UTC(),
	}
	if err := identity.Validate(); err != nil {
		return nil, err
	}
	return identity, nil
}

func (i Identity) Validate() error {
	if i.SchemaVersion != IdentitySchemaVersion {
		return fmt.Errorf("unsupported Agent identity schema version %d", i.SchemaVersion)
	}
	if strings.TrimSpace(i.ID) == "" {
		return fmt.Errorf("Agent identity ID is required")
	}
	name := strings.TrimSpace(i.Name)
	if name == "" {
		return fmt.Errorf("Agent name is required")
	}
	if utf8.RuneCountInString(name) > 64 {
		return fmt.Errorf("Agent name must not exceed 64 characters")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("Agent name contains control characters")
		}
	}
	if i.CorePersona != CorePersonaName {
		return fmt.Errorf("Core Persona must be %q", CorePersonaName)
	}
	if i.CreatedAt.IsZero() {
		return fmt.Errorf("Agent identity creation time is required")
	}
	return nil
}

func LoadIdentity(path string) (*Identity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Agent identity: %w", err)
	}
	var identity Identity
	if err := yaml.Unmarshal(data, &identity); err != nil {
		return nil, fmt.Errorf("parse Agent identity: %w", err)
	}
	if err := identity.Validate(); err != nil {
		return nil, fmt.Errorf("validate Agent identity: %w", err)
	}
	identity.Name = strings.TrimSpace(identity.Name)
	return &identity, nil
}

func SaveIdentity(path string, identity *Identity) error {
	if identity == nil {
		return fmt.Errorf("Agent identity is required")
	}
	if err := identity.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create Agent identity directory: %w", err)
	}
	data, err := yaml.Marshal(identity)
	if err != nil {
		return fmt.Errorf("marshal Agent identity: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write Agent identity: %w", err)
	}
	return nil
}
