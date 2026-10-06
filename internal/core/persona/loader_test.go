package persona

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCorePersonaIsBoi(t *testing.T) {
	t.Parallel()

	p := CorePersona()
	if p == nil {
		t.Fatal("CorePersona returned nil")
	}
	if p.Name != "boi" {
		t.Fatalf("Core Persona name = %q, want boi", p.Name)
	}
	if p.SystemPrompt == "" {
		t.Fatal("Core Persona system prompt is empty")
	}
}

func writePersona(t *testing.T, dir, file, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, file), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadPersonaValidation(t *testing.T) {
	dir := t.TempDir()
	writePersona(t, dir, "ok.yaml", "name: Dang\ntemperature: 0.2\nmax_tokens: 100\n")
	writePersona(t, dir, "noname.yaml", "description: x\n")
	writePersona(t, dir, "bad.yaml", "name: [unclosed\n")

	p, err := LoadPersona(filepath.Join(dir, "ok.yaml"))
	if err != nil || p.Name != "Dang" || p.MaxTokens != 100 {
		t.Fatalf("ok: %+v %v", p, err)
	}
	for _, f := range []string{"noname.yaml", "bad.yaml", "missing.yaml"} {
		if _, err := LoadPersona(filepath.Join(dir, f)); err == nil {
			t.Errorf("%s must fail", f)
		}
	}
}

func TestLoadDirStopsOnInvalidPersona(t *testing.T) {
	dir := t.TempDir()
	writePersona(t, dir, "a.yaml", "name: a\n")
	writePersona(t, dir, "b.yml", "name: b\n")
	writePersona(t, dir, "note.txt", "ignored")
	ps, err := LoadDir(dir)
	if err != nil || len(ps) != 2 {
		t.Fatalf("LoadDir: %d %v", len(ps), err)
	}
	writePersona(t, dir, "c.yaml", "description: no name\n")
	if _, err := LoadDir(dir); err == nil {
		t.Fatal("invalid persona must fail the directory load")
	}
}

func TestRegistryLoadFindsMixedCaseNames(t *testing.T) {
	dir := t.TempDir()
	writePersona(t, dir, "kamkaew.yaml", "name: kamkaew\n")
	writePersona(t, dir, "dang.yaml", "name: Dang\n")
	r, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if r.Find("dang") == nil || r.Find("  DANG ") == nil {
		t.Fatal("persona with capitalised name must be findable case-insensitively")
	}
	if _, err := r.Get("nope"); err == nil || !strings.Contains(err.Error(), "Available") {
		t.Fatalf("Get must list available personas: %v", err)
	}
	if r.Default().Name != "kamkaew" || r.Count() != 2 {
		t.Fatal("default or count wrong")
	}
	if !reflect.DeepEqual(r.List(), []string{"dang", "kamkaew"}) {
		t.Fatalf("list %v", r.List())
	}
}

func TestRegistryLoadRejectsEmptyAndMissingDir(t *testing.T) {
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatal("empty dir must error")
	}
	if _, err := Load(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("missing dir must error")
	}
}

func TestEmbeddedDefaultsAllParse(t *testing.T) {
	entries, err := DefaultPersonas.ReadDir("defaults")
	if err != nil || len(entries) == 0 {
		t.Fatalf("embedded defaults: %v", err)
	}
	for _, e := range entries {
		data, _ := DefaultPersonas.ReadFile("defaults/" + e.Name())
		var p Persona
		if err := yaml.Unmarshal(data, &p); err != nil || p.Name == "" {
			t.Errorf("%s does not parse to a named persona: %v", e.Name(), err)
		}
	}
}
