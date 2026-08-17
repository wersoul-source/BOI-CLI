package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSandboxResolvesWorkspacePaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "src", "main.go")
	if err := os.MkdirAll(filepath.Dir(nested), 0o755); err != nil {
		t.Fatalf("create nested directory: %v", err)
	}
	if err := os.WriteFile(nested, []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("create nested file: %v", err)
	}

	sandbox, err := NewSandbox(root)
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}

	got, err := sandbox.ResolveExisting(filepath.Join("src", "main.go"))
	if err != nil {
		t.Fatalf("resolve existing path: %v", err)
	}
	gotInfo, err := os.Stat(got)
	if err != nil {
		t.Fatalf("stat resolved path: %v", err)
	}
	wantInfo, err := os.Stat(nested)
	if err != nil {
		t.Fatalf("stat expected path: %v", err)
	}
	if !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("resolved path = %q, want same file as %q", got, nested)
	}

	writable, err := sandbox.ResolveForWrite(filepath.Join("output", "result.txt"))
	if err != nil {
		t.Fatalf("resolve write path: %v", err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("resolve expected root: %v", err)
	}
	if writable != filepath.Join(canonicalRoot, "output", "result.txt") {
		t.Fatalf("write path = %q", writable)
	}
}

func TestSandboxRejectsWorkspaceEscape(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	root := filepath.Join(parent, "workspace")
	outside := filepath.Join(parent, "outside.txt")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatalf("create outside file: %v", err)
	}

	sandbox, err := NewSandbox(root)
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}

	paths := []string{
		filepath.Join("..", "outside.txt"),
		outside,
		filepath.Join(parent, "workspace-sibling", "file.txt"),
	}
	for _, path := range paths {
		if _, err := sandbox.ResolveForWrite(path); !errors.Is(err, ErrOutsideWorkspace) {
			t.Errorf("ResolveForWrite(%q) error = %v, want ErrOutsideWorkspace", path, err)
		}
	}
}

func TestSandboxRejectsSymlinkEscape(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	root := filepath.Join(parent, "workspace")
	outside := filepath.Join(parent, "outside")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatalf("create outside directory: %v", err)
	}

	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}

	sandbox, err := NewSandbox(root)
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	if _, err := sandbox.ResolveForWrite(filepath.Join("escape", "file.txt")); !errors.Is(err, ErrOutsideWorkspace) {
		t.Fatalf("symlink escape error = %v, want ErrOutsideWorkspace", err)
	}
}
