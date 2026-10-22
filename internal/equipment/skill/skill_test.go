package skill

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const sample = "---\nname: demo\ndescription: A demo\nversion: \"2.0\"\nrequires: [read_file, run]\n---\n\nHello {{who}}.\n"

func TestParseFrontmatterAndBody(t *testing.T) {
	s, err := Parse([]byte(sample), "x/demo.skill.md", false)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "demo" || s.Version != "2.0" || s.Description != "A demo" || s.Prompt != "Hello {{who}}." {
		t.Fatalf("parsed %+v", s)
	}
	if !reflect.DeepEqual(s.Requires, []string{"read_file", "run"}) {
		t.Fatalf("requires %v", s.Requires)
	}
}

func TestParseDefaultsAndInvalidYAML(t *testing.T) {
	s, err := Parse([]byte("just a body"), "dir/foo.skill.md", false)
	if err != nil || s.Name != "foo" || s.Version != "1.0" {
		t.Fatalf("defaults: %+v %v", s, err)
	}
	if _, err := Parse([]byte("---\nname: [unclosed\n---\nbody"), "bad.skill.md", false); err == nil {
		t.Fatal("invalid frontmatter must error")
	}
}

func TestLoadConfinedStaysInsideItsRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "out.skill.md")
	body := "---\nname: demo\ndescription: d\n---\n\nbody"
	for path, content := range map[string]string{filepath.Join(root, "demo.skill.md"): body, outside: body} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.Symlink(filepath.Join(root, "demo.skill.md"), filepath.Join(root, "inside.skill.md"))
	_ = os.Symlink(outside, filepath.Join(root, "escape.skill.md"))
	if err := os.WriteFile(filepath.Join(root, "big.skill.md"), []byte(strings.Repeat("x", MaxFileBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if loaded, err := LoadConfined(root, "demo.skill.md"); err != nil || loaded.Prompt != "body" {
		t.Fatalf("plain file: %v %+v", err, loaded)
	}
	if _, err := LoadConfined(root, "inside.skill.md"); err != nil {
		t.Fatalf("a symlink to another file inside the root must load: %v", err)
	}
	for name, source := range map[string]string{
		"symlink out":   "escape.skill.md",
		"parent":        "../out.skill.md",
		"absolute":      outside,
		"directory":     "sub",
		"too large":     "big.skill.md",
		"missing":       "none.skill.md",
		"the root":      ".",
		"nested escape": "sub/../../out.skill.md",
	} {
		if _, err := LoadConfined(root, source); err == nil {
			t.Errorf("%s: %q loaded", name, source)
		}
	}
	if _, err := LoadConfined(root, "escape.skill.md"); !errors.Is(err, ErrOutsideRoot) {
		t.Errorf("a symlink out of the root must report ErrOutsideRoot, got %v", err)
	}
}
