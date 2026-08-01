package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// ProviderEntry represents a known AI provider with default endpoint and models.
type ProviderEntry struct {
	Name    string   `json:"name"`
	Label   string   `json:"label"`
	BaseURL string   `json:"base_url"`
	Models  []string `json:"models"`
}

// Registry holds the curated provider database.
type Registry struct {
	Providers []ProviderEntry `json:"providers"`
	mu        sync.RWMutex
}

// Load loads the registry from an embedded []byte (providers.json via go:embed).
func Load(embeddedJSON []byte) (*Registry, error) {
	var providers []ProviderEntry
	if err := json.Unmarshal(embeddedJSON, &providers); err != nil {
		return nil, err
	}
	return &Registry{Providers: providers}, nil
}

// LoadFromPath loads the registry from a cache file, falling back to embedded.
func LoadFromPath(cachePath string, embeddedJSON []byte) (*Registry, error) {
	if data, err := os.ReadFile(cachePath); err == nil {
		var providers []ProviderEntry
		if json.Unmarshal(data, &providers) == nil && len(providers) > 0 {
			return &Registry{Providers: providers}, nil
		}
	}
	return Load(embeddedJSON)
}

// SaveTo writes the registry to a cache file.
func (r *Registry) SaveTo(path string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(r.Providers, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Get returns a provider entry by name.
func (r *Registry) Get(name string) *ProviderEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := range r.Providers {
		if r.Providers[i].Name == name {
			return &r.Providers[i]
		}
	}
	return nil
}

// Names returns all provider names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, len(r.Providers))
	for i, p := range r.Providers {
		names[i] = p.Name
	}
	return names
}

// Labels returns provider labels for display.
func (r *Registry) Labels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	labels := make([]string, len(r.Providers))
	for i, p := range r.Providers {
		labels[i] = p.Label
	}
	return labels
}

// ModelsFor returns the model list for a given provider.
func (r *Registry) ModelsFor(name string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.Providers {
		if p.Name == name {
			models := make([]string, len(p.Models))
			copy(models, p.Models)
			return models
		}
	}
	return nil
}

// Count returns the number of providers.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Providers)
}

// UpdateFromRemote merges remote registry into current (for future refresh).
func (r *Registry) UpdateFromRemote(remoteProviders []ProviderEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing := make(map[string]int)
	for i, p := range r.Providers {
		existing[p.Name] = i
	}

	for _, rp := range remoteProviders {
		if idx, ok := existing[rp.Name]; ok {
			r.Providers[idx].Models = rp.Models
			if rp.BaseURL != "" {
				r.Providers[idx].BaseURL = rp.BaseURL
			}
		}
	}
}
