package app

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	coreblock "github.com/boi-family/boi-cli/internal/core"
)

func TestPresetsComposeFromRealBlocks(t *testing.T) {
	want := map[string][]string{
		"minimal": {"workspace.list", "workspace.read", "workspace.search"},
		"coding":  {"workspace.list", "workspace.read", "workspace.search", "workspace.edit", "workspace.write", "process.run"},
	}
	for name, preset := range Presets() {
		composition, err := Compose(preset.Environment, BuiltinPacks(), testSandbox(t), "1.11.0", coreblock.AgentEnvironment{})
		if err != nil {
			t.Fatalf("%s: preset must compose: %v", name, err)
		}
		if got := composition.ToolNames(); !slices.Equal(got, want[name]) {
			t.Errorf("%s: Tools = %v, want %v", name, got, want[name])
		}
	}
	if len(Presets()) != len(want) {
		t.Fatalf("every preset needs an expected Tool set here: %v", PresetNames())
	}
}

func TestMinimalPresetCannotChangeAnything(t *testing.T) {
	composition, err := Compose(Presets()["minimal"].Environment, BuiltinPacks(), testSandbox(t), "1.11.0", coreblock.AgentEnvironment{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range composition.Tools {
		if spec := tool.Spec(); spec.Risk != "read" || spec.Approval != "auto" {
			t.Fatalf("minimal must be read-only, found %s (%s/%s)", spec.Name, spec.Risk, spec.Approval)
		}
	}
}

func TestUsePresetKeepsAnExistingEnvironment(t *testing.T) {
	boiDir := t.TempDir()
	path, err := UsePreset(boiDir, "coding", false)
	if err != nil {
		t.Fatal(err)
	}
	environment, err := LoadEnvironment(boiDir)
	if err != nil || environment.Name != "coding" || path != filepath.Join(boiDir, EnvironmentFile) {
		t.Fatalf("written preset must load back: %+v %v", environment, err)
	}

	custom := []byte(`{"schema_version":1,"name":"mine","blocks":[{"id":"workspace-read","required":true}]}`)
	if err := os.WriteFile(path, custom, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := UsePreset(boiDir, "minimal", false); !errors.Is(err, ErrEnvironmentExists) {
		t.Fatalf("an existing environment must not be overwritten silently: %v", err)
	}
	if _, err := UsePreset(boiDir, "minimal", true); err != nil {
		t.Fatal(err)
	}
	backups, _ := filepath.Glob(path + ".*.bak")
	if len(backups) != 1 {
		t.Fatalf("replace must keep a backup, found %v", backups)
	}
	if saved, _ := os.ReadFile(backups[0]); string(saved) != string(custom) {
		t.Fatal("the backup must hold the replaced file")
	}
	if _, err := UsePreset(boiDir, "research", false); err == nil || !strings.Contains(err.Error(), "unknown preset") {
		t.Fatalf("a preset that does not exist must be rejected: %v", err)
	}
}
