package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectRootFromNestedDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("create root marker: %v", err)
	}
	nested := filepath.Join(root, "one", "two")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("create nested directory: %v", err)
	}

	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	got, err := DetectRoot()
	if err != nil {
		t.Fatalf("detect root: %v", err)
	}
	if got != root {
		t.Fatalf("DetectRoot() = %q, want %q", got, root)
	}
}

func TestGetBoiDir(t *testing.T) {
	t.Parallel()

	root := filepath.Join("workspace", "project")
	want := filepath.Join(root, ".boi")
	if got := GetBoiDir(root); got != want {
		t.Fatalf("GetBoiDir(%q) = %q, want %q", root, got, want)
	}
}
