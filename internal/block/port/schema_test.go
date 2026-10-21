package port

import (
	"strings"
	"testing"
)

func TestProviderSchemaDropsUnsupportedKeywords(t *testing.T) {
	spec := ToolSpec{Name: "x", Parameters: map[string]any{
		"type": "object", "$schema": "https://json-schema.org/draft/2020-12/schema",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "format": "uri", "pattern": "^a"},
			"mode": map[string]any{"oneOf": []any{map[string]any{"type": "string"}}},
		},
		"required":             []any{"path"},
		"additionalProperties": map[string]any{"type": "string"},
	}}
	schema := spec.ProviderSchema()
	for _, keyword := range []string{"$schema", "additionalProperties"} {
		if _, present := schema[keyword]; present {
			t.Errorf("%s must not reach a Provider", keyword)
		}
	}
	path := schema["properties"].(map[string]any)["path"].(map[string]any)
	if _, present := path["format"]; present || path["type"] != "string" {
		t.Errorf("nested schema not normalized: %v", path)
	}
	if mode := schema["properties"].(map[string]any)["mode"].(map[string]any); len(mode) != 0 {
		t.Errorf("unsupported combinator must be dropped, got %v", mode)
	}
	if (ToolSpec{Parameters: map[string]any{"type": "array"}}).ProviderSchema()["type"] != "object" {
		t.Error("a non-object root must fall back to an object")
	}
}

func TestValidateArguments(t *testing.T) {
	write := ToolSpec{Name: "workspace.write", Usage: "workspace.write(path, content)"}
	numbers := ToolSpec{Name: "n", Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"count": map[string]any{"type": "integer"},
			"mode":  map[string]any{"type": "string", "enum": []any{"fast", "safe"}},
			"tags":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
	}}
	cases := []struct {
		name string
		spec ToolSpec
		args map[string]any
		want string
	}{
		{"valid", write, map[string]any{"path": "a", "content": "b"}, ""},
		{"missing required", write, map[string]any{"path": "a"}, "arguments.content is required"},
		{"wrong type", write, map[string]any{"path": 7.0, "content": "b"}, "arguments.path must be string"},
		{"unknown argument", write, map[string]any{"path": "a", "content": "b", "mode": "x"}, "arguments.mode is not a known argument"},
		{"integer", numbers, map[string]any{"count": 2.5}, "arguments.count must be integer"},
		{"enum", numbers, map[string]any{"mode": "unsafe"}, "arguments.mode must be one of"},
		{"array items", numbers, map[string]any{"tags": []any{"a", 3.0}}, "arguments.tags[1] must be string"},
		{"open schema accepts extras", numbers, map[string]any{"other": true}, ""},
		{"free-form usage", ToolSpec{Name: "mcp.s.t", Usage: "mcp.s.t(server-defined arguments)"}, map[string]any{"anything": 1.0}, ""},
		{"nil arguments with required", write, nil, "is required"},
	}
	for _, tc := range cases {
		err := tc.spec.ValidateArguments(tc.args)
		switch {
		case tc.want == "" && err != nil:
			t.Errorf("%s: unexpected error %v", tc.name, err)
		case tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)):
			t.Errorf("%s: error %v, want %q", tc.name, err, tc.want)
		}
	}
}

func TestEnumOfArraysDoesNotPanic(t *testing.T) {
	spec := ToolSpec{Name: "x", Parameters: map[string]any{"type": "object", "properties": map[string]any{
		"pair": map[string]any{"enum": []any{[]any{1.0, 2.0}}},
	}}}
	if err := spec.ValidateArguments(map[string]any{"pair": []any{1.0, 2.0}}); err != nil {
		t.Fatalf("an equal array must match: %v", err)
	}
	if err := spec.ValidateArguments(map[string]any{"pair": []any{3.0}}); err == nil {
		t.Fatal("a different array must not match")
	}
}
