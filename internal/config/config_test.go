package config

import (
	"path/filepath"
	"testing"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".boi", "config.yaml")
	want := &Config{
		Provider:  "anthropic",
		Model:     "claude-test",
		Persona:   "boi",
		LogLevel:  "debug",
		Workspace: "workspace",
		APIKeys:   map[string]string{},
		Sandbox: SandboxConfig{
			Enabled:      true,
			DenyCommands: []string{"sudo "},
		},
	}

	if err := want.SaveTo(path); err != nil {
		t.Fatalf("save config: %v", err)
	}

	got, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got.Provider != want.Provider || got.Model != want.Model || got.Persona != want.Persona {
		t.Fatalf("loaded identity fields = %#v, want %#v", got, want)
	}
	if got.LogLevel != want.LogLevel || got.Workspace != want.Workspace {
		t.Fatalf("loaded runtime fields = %#v, want %#v", got, want)
	}
	if got.Sandbox.Enabled != want.Sandbox.Enabled || len(got.Sandbox.DenyCommands) != 1 {
		t.Fatalf("loaded sandbox = %#v, want %#v", got.Sandbox, want.Sandbox)
	}
}

func TestEnvironmentOverridesFileValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := Default()
	cfg.Provider = "openai"
	cfg.Model = "file-model"
	cfg.LogLevel = "info"
	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("save config: %v", err)
	}

	t.Setenv("BOI_PROVIDER", "google")
	t.Setenv("BOI_MODEL", "env-model")
	t.Setenv("BOI_LOG_LEVEL", "warn")

	got, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got.Provider != "google" || got.Model != "env-model" || got.LogLevel != "warn" {
		t.Fatalf("environment overrides not applied: %#v", got)
	}
}
