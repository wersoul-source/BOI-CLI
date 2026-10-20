package filesystem

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boi-family/boi-cli/internal/runtime/workspace"
)

func readFixture(t *testing.T) readTool {
	t.Helper()
	root := t.TempDir()
	for name, body := range map[string]string{"a.txt": "amber", "b.txt": "birch"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	return readTool{NewReader(sandbox)}
}

func TestReadSeveralPathsInOneCall(t *testing.T) {
	tool := readFixture(t)
	output, err := tool.Execute(context.Background(), map[string]any{"paths": []any{"a.txt", "missing.txt", "b.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	var results []map[string]any
	if err := json.Unmarshal([]byte(output.Output), &results); err != nil || len(results) != 3 {
		t.Fatalf("want one result per path: %v %s", err, output.Output)
	}
	if results[0]["Content"] != "amber" || results[2]["Content"] != "birch" {
		t.Fatalf("contents out of order: %s", output.Output)
	}
	if results[1]["Path"] != "missing.txt" || results[1]["Error"] == nil {
		t.Fatalf("a missing file must be reported on its own: %s", output.Output)
	}
}

func TestReadArgumentRules(t *testing.T) {
	tool := readFixture(t)
	for name, arguments := range map[string]map[string]any{
		"neither":        {},
		"both":           {"path": "a.txt", "paths": []any{"b.txt"}},
		"too many":       {"paths": []any{"a", "b", "c", "d", "e", "f"}},
		"empty entry":    {"paths": []any{"a.txt", ""}},
		"all missing":    {"paths": []any{"x.txt", "y.txt"}},
		"single missing": {"path": "x.txt"},
	} {
		if _, err := tool.Execute(context.Background(), arguments); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	output, err := tool.Execute(context.Background(), map[string]any{"path": "a.txt"})
	if err != nil || !strings.Contains(output.Output, `"Content":"amber"`) {
		t.Fatalf("single path must keep the v1.6 output shape: %v %s", err, output.Output)
	}
	spec := tool.Spec()
	if err := spec.ValidateArguments(map[string]any{"paths": []any{"a.txt", 3.0}}); err == nil {
		t.Error("schema must reject a non-string path")
	}
	if err := spec.ValidateArguments(map[string]any{"file": "a.txt"}); err == nil {
		t.Error("schema must reject an unknown argument")
	}
}
