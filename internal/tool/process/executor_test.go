package process

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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

func TestRunContextHonorsCancellation(t *testing.T) {
	command := "sleep 5"
	if runtime.GOOS == "windows" {
		command = "Start-Sleep -Seconds 5"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := NewExecutor().RunContext(ctx, command)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline, got %v", err)
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
