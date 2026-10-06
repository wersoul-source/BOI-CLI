package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureLocalGitExcludesIsAdditiveAndIdempotent(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(filepath.Join(gitDir, "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(gitDir, "info", "exclude")
	if err := os.WriteFile(path, []byte("existing.tmp\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureLocalGitExcludes(root, ".env", ".boi/"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureLocalGitExcludes(root, ".env", ".boi/"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"existing.tmp", ".env", ".boi/"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Git exclude missing %q: %s", want, text)
		}
	}
	if strings.Count(text, ".env") != 1 || strings.Count(text, ".boi/") != 1 {
		t.Fatalf("Git excludes duplicated: %s", text)
	}
}
