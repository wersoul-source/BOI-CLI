package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/boi-family/boi-cli/internal/block"
)

// Preset is a ready-made environment for one kind of work.
type Preset struct {
	Name        string
	Description string
	Environment block.Environment
}

// Presets are built from Blocks that exist. A preset whose work needs a
// Block BOI does not ship yet (Research needs web search) is not offered;
// an MCP entry added to a written environment.json fills that gap.
func Presets() map[string]Preset {
	environment := func(name string, blocks ...block.EnvironmentBlock) block.Environment {
		return block.Environment{SchemaVersion: block.PackSchemaVersion, Name: name, Blocks: blocks}
	}
	return map[string]Preset{
		"minimal": {
			Name:        "minimal",
			Description: "Read-only: list, read and search the workspace. Nothing is written or run.",
			Environment: environment("minimal", block.EnvironmentBlock{ID: "workspace-read", Required: true}),
		},
		"coding": {
			Name:        "coding",
			Description: "Read, search and write files, and run commands such as tests; writes and commands need approval.",
			Environment: environment("coding",
				block.EnvironmentBlock{ID: "workspace-tools", Required: true},
				block.EnvironmentBlock{ID: "process-tools", Required: true},
			),
		},
	}
}

// PresetNames lists the presets in a stable order.
func PresetNames() []string {
	names := make([]string, 0, len(Presets()))
	for name := range Presets() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ErrEnvironmentExists means a workspace already chose its Blocks.
var ErrEnvironmentExists = errors.New("environment.json already exists")

// UsePreset writes the preset as the workspace's environment. An existing
// file is kept unless replace is set; then it is backed up first, because it
// may hold hand-written MCP entries.
func UsePreset(boiDir, name string, replace bool) (string, error) {
	preset, ok := Presets()[name]
	if !ok {
		return "", fmt.Errorf("unknown preset %q; choose one of %v", name, PresetNames())
	}
	path := filepath.Join(boiDir, EnvironmentFile)
	if existing, err := os.ReadFile(path); err == nil {
		if !replace {
			return "", fmt.Errorf("%w at %s; pass --replace to back it up and replace it", ErrEnvironmentExists, path)
		}
		backup := fmt.Sprintf("%s.%s.bak", path, time.Now().UTC().Format("20060102T150405Z"))
		if err := os.WriteFile(backup, existing, 0o600); err != nil {
			return "", fmt.Errorf("back up %s: %w", path, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	data, err := json.MarshalIndent(preset.Environment, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(boiDir, 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return "", err
	}
	return path, nil
}
