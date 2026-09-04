package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewRuntimeResolvesWorkspacePaths(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("create workspace marker: %v", err)
	}

	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	runtime, err := NewRuntime("test-version")
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}
	if runtime.WorkspaceRoot != root {
		t.Fatalf("WorkspaceRoot = %q, want %q", runtime.WorkspaceRoot, root)
	}
	if runtime.BoiDir != filepath.Join(root, ".boi") {
		t.Fatalf("BoiDir = %q", runtime.BoiDir)
	}
	if runtime.EnvPath != filepath.Join(root, ".env") {
		t.Fatalf("EnvPath = %q", runtime.EnvPath)
	}
	if runtime.AgentFolderRoot != filepath.Join(root, "agent-folder") || runtime.AgentFolder == nil {
		t.Fatalf("Agent Folder = (%q, %#v)", runtime.AgentFolderRoot, runtime.AgentFolder)
	}
	if _, statErr := os.Stat(runtime.AgentFolderRoot); !os.IsNotExist(statErr) {
		t.Fatalf("read-only runtime created Agent Folder: %v", statErr)
	}
	if _, statErr := os.Stat(runtime.BoiDir); !os.IsNotExist(statErr) {
		t.Fatalf("read-only runtime created BOI directory: %v", statErr)
	}
	if err := runtime.EnsureWorkspaceState(); err != nil {
		t.Fatalf("ensure workspace state: %v", err)
	}
	for _, tray := range []string{"bin", "output"} {
		if info, statErr := os.Stat(filepath.Join(runtime.AgentFolderRoot, tray)); statErr != nil || !info.IsDir() {
			t.Fatalf("Agent Folder tray %s missing: %v", tray, statErr)
		}
	}
	if runtime.Version != "test-version" {
		t.Fatalf("Version = %q", runtime.Version)
	}
	if runtime.Sandbox == nil || runtime.Sandbox.Root() != root {
		t.Fatalf("Sandbox root = %v, want %q", runtime.Sandbox, root)
	}
}

func TestRuntimeContextRoundTrip(t *testing.T) {
	t.Parallel()

	want := &Runtime{Version: "test"}
	ctx := WithRuntime(context.Background(), want)
	got, ok := RuntimeFromContext(ctx)
	if !ok || got != want {
		t.Fatalf("RuntimeFromContext() = (%#v, %v), want (%#v, true)", got, ok, want)
	}
}
