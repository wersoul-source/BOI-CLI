package persona

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Registry struct {
	personas map[string]*Persona
	defaultName string
}

func Load(path string) (*Registry, error) {
	r := &Registry{
		personas:    make(map[string]*Persona),
		defaultName: "kamkaew",
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read persona directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		fullPath := filepath.Join(path, entry.Name())
		p, err := LoadPersona(fullPath)
		if err != nil {
			return nil, fmt.Errorf("load persona %s: %w", entry.Name(), err)
		}

		r.personas[p.Name] = p
	}

	if len(r.personas) == 0 {
		return nil, fmt.Errorf("no personas found in %s", path)
	}

	return r, nil
}

func (r *Registry) Get(name string) (*Persona, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	p, ok := r.personas[name]
	if !ok {
		return nil, fmt.Errorf("persona %q not found. Available: %s", name, strings.Join(r.List(), ", "))
	}
	return p, nil
}

func (r *Registry) List() []string {
	names := make([]string, 0, len(r.personas))
	for name := range r.personas {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *Registry) Default() *Persona {
	if p, ok := r.personas[r.defaultName]; ok {
		return p
	}
	return r.personas["kamkaew"]
}

func (r *Registry) Count() int {
	return len(r.personas)
}
