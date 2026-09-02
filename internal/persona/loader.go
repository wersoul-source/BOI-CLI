package persona

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadPersona(path string) (*Persona, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read persona file: %w", err)
	}

	var p Persona
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse persona: %w", err)
	}

	if p.Name == "" {
		return nil, fmt.Errorf("persona name is required in %s", path)
	}

	return &p, nil
}

func LoadDir(dir string) ([]*Persona, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read persona directory: %w", err)
	}

	var personas []*Persona
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		p, err := LoadPersona(fullPath)
		if err != nil {
			return nil, fmt.Errorf("load persona %s: %w", entry.Name(), err)
		}
		personas = append(personas, p)
	}

	return personas, nil
}

func DefaultPersona() *Persona {
	return CorePersona()
}

// CorePersona returns the single Persona owned by BOI Core. It is embedded in
// the binary so workspace Persona files cannot silently replace Core identity.
func CorePersona() *Persona {
	data, err := DefaultPersonas.ReadFile("defaults/boi.yaml")
	if err == nil {
		var p Persona
		if yaml.Unmarshal(data, &p) == nil && strings.EqualFold(p.Name, "boi") {
			p.Name = "boi"
			return &p
		}
	}
	return &Persona{
		Name:        "boi",
		Description: "BOI Core Persona",
		Model:       "gpt-4.1-mini",
		Temperature: 0.4,
		MaxTokens:   4096,
	}
}
