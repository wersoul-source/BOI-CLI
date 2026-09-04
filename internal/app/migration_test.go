package app

import (
	"os"
	"path/filepath"
	"testing"

	coreblock "github.com/boi-family/boi-cli/internal/block/core"
	"github.com/boi-family/boi-cli/internal/capability"
	"github.com/boi-family/boi-cli/internal/persona"
)

func TestLegacyWorkspaceMigratesWithoutOverwritingUserFiles(t *testing.T) {
	root := t.TempDir()
	for _, directory := range []string{".git", filepath.Join(".boi", "personas"), filepath.Join(".boi", "skills")} {
		if err := os.MkdirAll(filepath.Join(root, directory), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	legacyConfig := []byte("provider: legacy\nmodel: legacy-model\npersona: kamkaew\n")
	legacyPersona := []byte("name: kamkaew\ndescription: legacy user file\n")
	if err := os.WriteFile(filepath.Join(root, ".boi", "config.yaml"), legacyConfig, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".boi", "personas", "kamkaew.yaml"), legacyPersona, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".boi", "skills", "loose.skill.md"), []byte("unindexed"), 0o600); err != nil {
		t.Fatal(err)
	}

	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })

	runtime, err := NewRuntime("migration-test")
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.EnsureWorkspaceState(); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string][]byte{
		filepath.Join(root, ".boi", "config.yaml"):              legacyConfig,
		filepath.Join(root, ".boi", "personas", "kamkaew.yaml"): legacyPersona,
	} {
		got, readErr := os.ReadFile(path)
		if readErr != nil || string(got) != string(want) {
			t.Fatalf("legacy file changed: %s got=%q err=%v", path, got, readErr)
		}
	}
	for _, kind := range []capability.Kind{capability.KindSkill, capability.KindTool} {
		if _, err := capability.LoadIndex(capability.IndexPath(runtime.BoiDir, kind), kind); err != nil {
			t.Fatalf("migrated %s index: %v", kind, err)
		}
	}
	set, err := SelectCapabilities(runtime.BoiDir, "inspect files", coreblock.AgentEnvironment{ToolCalling: true, SkillCalling: true, ContextBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Tools.Active) != 4 || len(set.Skills.Active) != 0 {
		t.Fatalf("migrated capabilities=%#v", set)
	}
	if persona.CorePersona().Name != coreblock.CorePersonaName {
		t.Fatalf("legacy Persona overrode Core: %s", persona.CorePersona().Name)
	}
}
