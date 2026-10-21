package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boi-family/boi-cli/internal/runtime/workspace"
)

func editFixture(t *testing.T, files map[string]string) (editTool, string) {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	return editTool{sandbox}, root
}

func TestEditChangesOnlyTheMatchedText(t *testing.T) {
	config := "{\n  \"port\": 3000,\n  \"debug\": true,\n  \"name\": \"bench\"\n}\n"
	tool, root := editFixture(t, map[string]string{"config.json": config})
	arguments := map[string]any{"path": "config.json", "old": `"port": 3000`, "new": `"port": 8080`}
	output, err := tool.Execute(context.Background(), arguments)
	if err != nil {
		t.Fatal(err)
	}
	if output.Output != "edited config.json at line 2" || len(output.ChangedPaths) != 1 {
		t.Fatalf("output = %+v", output)
	}
	got, _ := os.ReadFile(filepath.Join(root, "config.json"))
	if string(got) != strings.Replace(config, "3000", "8080", 1) {
		t.Fatalf("file = %q", got)
	}
	if info, _ := os.Stat(filepath.Join(root, "config.json")); info.Mode().Perm() != 0o640 {
		t.Fatalf("mode changed to %v", info.Mode().Perm())
	}
	if _, err := tool.Verify(context.Background(), arguments, output); err != nil {
		t.Fatal(err)
	}
}

func TestEditRefusesAmbiguousOrMissingText(t *testing.T) {
	tool, root := editFixture(t, map[string]string{"a.go": "x := 1\nx := 1\n", ".env": "KEY=secret\n"})
	cases := map[string]struct {
		arguments map[string]any
		want      string
	}{
		"twice":      {map[string]any{"path": "a.go", "old": "x := 1", "new": "x := 2"}, "appears 2 times"},
		"absent":     {map[string]any{"path": "a.go", "old": "y := 1", "new": "y := 2"}, "not found"},
		"no file":    {map[string]any{"path": "b.go", "old": "x", "new": "y"}, ""},
		"credential": {map[string]any{"path": ".env", "old": "secret", "new": "x"}, "credential file"},
		"empty old":  {map[string]any{"path": "a.go", "old": "", "new": "y"}, ""},
	}
	for name, tc := range cases {
		if _, err := tool.Execute(context.Background(), tc.arguments); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error = %v, want %q", name, err, tc.want)
		}
	}
	if got, _ := os.ReadFile(filepath.Join(root, "a.go")); string(got) != "x := 1\nx := 1\n" {
		t.Fatalf("a refused edit changed the file: %q", got)
	}
}

func TestEditMatchesWindowsLineEndings(t *testing.T) {
	tool, root := editFixture(t, map[string]string{"a.txt": "one\r\ntwo\r\nthree\r\n"})
	if _, err := tool.Execute(context.Background(), map[string]any{"path": "a.txt", "old": "one\ntwo", "new": "one\n2"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "a.txt")); string(got) != "one\r\n2\r\nthree\r\n" {
		t.Fatalf("file = %q", got)
	}
}
