package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boi-family/boi-cli/internal/tool/filesystem"
	"github.com/boi-family/boi-cli/internal/workspace"
)

func TestWorkspaceReadCommandUsesSandbox(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	root := filepath.Join(parent, "workspace")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "note.txt"), []byte("sandboxed"), 0o600); err != nil {
		t.Fatalf("create workspace file: %v", err)
	}
	outside := filepath.Join(parent, "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatalf("create outside file: %v", err)
	}
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	model := &Model{
		root:            root,
		workspaceReader: filesystem.NewReader(sandbox),
	}

	inside := model.callWorkspaceCmd("/read note.txt")().(workspaceResponseMsg)
	if inside.err != nil || !strings.Contains(inside.content, "sandboxed") {
		t.Fatalf("inside read = %#v", inside)
	}

	escape := model.callWorkspaceCmd("/read ../outside.txt")().(workspaceResponseMsg)
	if !errors.Is(escape.err, workspace.ErrOutsideWorkspace) {
		t.Fatalf("escape error = %v, want ErrOutsideWorkspace", escape.err)
	}
}

func TestWorkspaceCommandRecognition(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"/workspace", "/ls", "/ls src", "/read README.md"} {
		if !isWorkspaceCommand(input) {
			t.Errorf("%q was not recognized", input)
		}
	}
	if isWorkspaceCommand("/provider") {
		t.Fatal("provider command was classified as workspace command")
	}
}
