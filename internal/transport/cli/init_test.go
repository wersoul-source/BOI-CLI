package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInitAtUsesWorkspaceRootAndProtectsLocalState(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git", "info"), 0o755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RunInitAt(root, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".boi", "config.yaml")); err != nil {
		t.Fatalf("root config missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(nested, ".boi")); !os.IsNotExist(err) {
		t.Fatalf("init leaked into nested directory: %v", err)
	}
	exclude, err := os.ReadFile(filepath.Join(root, ".git", "info", "exclude"))
	if err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []string{".boi/", ".env", ".env.boi-backup-*"} {
		if !strings.Contains(string(exclude), pattern) {
			t.Fatalf("Git exclude missing %q: %s", pattern, exclude)
		}
	}
	localIgnore, err := os.ReadFile(filepath.Join(root, ".boi", ".gitignore"))
	if err != nil || string(localIgnore) != "*\n!.gitignore\n" {
		t.Fatalf("BOI local ignore=%q err=%v", localIgnore, err)
	}
}
