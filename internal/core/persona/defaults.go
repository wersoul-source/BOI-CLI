package persona

import "embed"

//go:embed defaults/*.yaml
var DefaultPersonas embed.FS

func DefaultPersonaFiles() []string {
	return []string{"boi.yaml"}
}
