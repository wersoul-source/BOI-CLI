package skill

import (
	"reflect"
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
