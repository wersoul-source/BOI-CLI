package catalog

import (
	_ "embed"
	"log/slog"
)

//go:embed providers.json
var embeddedProviders []byte

// LoadEmbedded loads the embedded registry (always available).
func LoadEmbedded() *Registry {
	r, err := Load(embeddedProviders)
	if err != nil {
		slog.Error("failed to load embedded registry", "error", err)
		// Return empty registry as fallback
		return &Registry{}
	}
	return r
}
