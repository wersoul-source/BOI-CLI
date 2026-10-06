package skill

import (
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

func TestLoadDir(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.skill.md"), []byte(sample), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "broken.skill.md"), []byte("---\nname: [x\n---\n"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "ignored.md"), []byte(sample), 0o600)
	_ = os.Mkdir(filepath.Join(dir, "sub.skill.md"), 0o700)

	skills, err := LoadDir(dir)
	if err != nil || len(skills) != 1 || skills[0].Name != "demo" {
		t.Fatalf("LoadDir: %+v %v", skills, err)
	}
	if got, err := LoadDir(filepath.Join(dir, "missing")); err != nil || got != nil {
		t.Fatalf("missing dir must be empty, not error: %v %v", got, err)
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	r.Register(&Skill{Name: "b"})
	r.Register(&Skill{Name: "a"})
	r.Register(&Skill{Name: "a", Description: "replaced"})
	if r.Count() != 2 || !reflect.DeepEqual(r.List(), []string{"a", "b"}) {
		t.Fatalf("registry %v", r.List())
	}
	if s, _ := r.Get("a"); s.Description != "replaced" {
		t.Fatal("register must replace")
	}
	if _, err := r.Get("nope"); err == nil {
		t.Fatal("missing skill must error")
	}
	if all := r.All(); all[0].Name != "a" {
		t.Fatal("All must be sorted")
	}
}

func TestApplyContextAndRequirements(t *testing.T) {
	s, _ := Parse([]byte(sample), "demo.skill.md", false)
	out := ApplyContext(s, map[string]string{"who": "world"})
	if !strings.Contains(out, "Hello world.") || !strings.Contains(out, "read_file, run") {
		t.Fatalf("ApplyContext: %q", out)
	}
	if miss := MatchRequirements(s, []string{"read_file"}); !reflect.DeepEqual(miss, []string{"run"}) {
		t.Fatalf("missing %v", miss)
	}
	if InjectSkills(nil, nil) != "" {
		t.Fatal("no skills must inject nothing")
	}
	if !strings.Contains(InjectSkills([]*Skill{s}, nil), "### demo") {
		t.Fatal("InjectSkills must list skills")
	}
}
