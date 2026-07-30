package persona

import (
	"fmt"
	"os"

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
