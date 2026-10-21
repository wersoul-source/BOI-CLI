package persona

import (
	"embed"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed defaults/*.yaml
var DefaultPersonas embed.FS

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
