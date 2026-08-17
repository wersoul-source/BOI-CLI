package process

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/boi-family/boi-cli/internal/workspace"
)

func TestRunWithDirRejectsOutsideWorkspaceBeforeExecution(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	root := filepath.Join(parent, "workspace")
	outside := filepath.Join(parent, "outside")
	if err := makeDirs(root, outside); err != nil {
		t.Fatalf("create test directories: %v", err)
	}

	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatalf("create workspace sandbox: %v", err)
	}
	executor := NewExecutor(WithWorkspace(sandbox))

	_, err = executor.RunWithDir("this-command-must-never-run", outside)
	if !errors.Is(err, workspace.ErrOutsideWorkspace) {
		t.Fatalf("RunWithDir() error = %v, want ErrOutsideWorkspace", err)
	}
}

func makeDirs(paths ...string) error {
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}
