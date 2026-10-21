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

func TestSearchFindsLinesAcrossFiles(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var log strings.Builder
	for line := 1; line <= 3000; line++ {
		if line == 1700 {
			log.WriteString("line 1700: the Secret word is quartz\n")
		} else {
			log.WriteString("line: routine\n")
		}
	}
	write("big.log", log.String())
	write("docs/guide.md", "carry a SECRET lantern\n")
	write(".git/config", "secret in git\n")
	write("agent-folder/out.txt", "secret in output\n")
	write("blob.bin", "secret\x00binary")
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	tool := searchTool{NewReader(sandbox)}

	output, err := tool.Execute(context.Background(), map[string]any{"pattern": "secret"})
	if err != nil {
		t.Fatal(err)
	}
	var result SearchResult
	if err := json.Unmarshal([]byte(output.Output), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Matches) != 2 {
		t.Fatalf("want matches in big.log and docs/guide.md only, got %+v", result.Matches)
	}
	if result.Matches[0].Path != "big.log" || result.Matches[0].Line != 1700 || !strings.Contains(result.Matches[0].Text, "quartz") {
		t.Fatalf("wrong first match: %+v", result.Matches[0])
	}

	single, err := tool.Execute(context.Background(), map[string]any{"pattern": "LANTERN", "path": "docs/guide.md"})
	if err != nil || !strings.Contains(single.Output, `"Line":1`) {
		t.Fatalf("search in one file failed: %v %s", err, single.Output)
	}
	if _, err := tool.Execute(context.Background(), map[string]any{"pattern": "x", "path": "../"}); err == nil {
		t.Fatal("search must not leave the workspace")
	}
	if _, err := tool.Execute(context.Background(), map[string]any{"pattern": ""}); err == nil {
		t.Fatal("an empty pattern must be rejected")
	}
}

func TestSearchStopsAtMatchLimit(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "many.txt"), []byte(strings.Repeat("hit\n", 500)), 0o600); err != nil {
		t.Fatal(err)
	}
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewReader(sandbox).Search(context.Background(), "hit", ".")
	if err != nil || len(result.Matches) != maxSearchMatches || !result.Truncated {
		t.Fatalf("want %d matches and Truncated, got %d %v %v", maxSearchMatches, len(result.Matches), result.Truncated, err)
	}
}

func TestCredentialFilesAreNeverRead(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{".env": "PSC_1_API_KEY=secret", ".env.local": "TOKEN=secret", ".env.example": "PSC_1_API_KEY=", "notes.txt": "secret word"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	reader := NewReader(sandbox)
	for _, name := range []string{".env", ".env.local", "./.ENV"} {
		if _, err := reader.Read(name); err == nil {
			t.Errorf("%s must not be readable", name)
		}
	}
	if _, err := reader.Read(".env.example"); err != nil {
		t.Errorf("a template without values may be read: %v", err)
	}
	result, err := reader.Search(context.Background(), "secret", ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, match := range result.Matches {
		if strings.HasPrefix(match.Path, ".env") {
			t.Fatalf("search must skip credential files, found %+v", match)
		}
	}
}

func TestSearchStopsWhenItsTimeoutEnds(t *testing.T) {
	sandbox, err := workspace.NewSandbox(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewReader(sandbox).Search(ctx, "x", "."); err == nil {
		t.Fatal("a cancelled search must stop")
	}
}

func TestEmptyPathWithPathsReadsEveryFile(t *testing.T) {
	tool := readFixture(t)
	output, err := tool.Execute(context.Background(), map[string]any{"path": "", "paths": []any{"a.txt", "b.txt"}})
	if err != nil || !strings.Contains(output.Output, "amber") || !strings.Contains(output.Output, "birch") {
		t.Fatalf("both files must be read: %v %s", err, output.Output)
	}
}
