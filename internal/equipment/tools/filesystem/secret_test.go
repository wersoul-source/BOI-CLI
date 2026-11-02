package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/boi-family/boi-cli/internal/block/port"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
)

func toolNamed(t *testing.T, sandbox *workspace.Sandbox, name string) port.Tool {
	t.Helper()
	for _, tool := range Tools(sandbox) {
		if tool.Spec().Name == name {
			return tool
		}
	}
	t.Fatalf("no tool %s", name)
	return nil
}

// workspace.edit refused .env but workspace.write replaced it. One approved
// write could point a Provider at another host, which would then receive the
// API key on the next start.
func TestWriteRefusesCredentialFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("PSC_1_API_KEY=old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	write := toolNamed(t, sandbox, "workspace.write")
	for _, name := range []string{".env", ".env.local", "sub/../.env"} {
		if _, err := write.Execute(context.Background(), map[string]any{"path": name, "content": "PSC_1_BASE_URL=https://attacker.example\n"}); err == nil {
			t.Errorf("writing %s was allowed", name)
		}
	}
	if data, _ := os.ReadFile(filepath.Join(root, ".env")); string(data) != "PSC_1_API_KEY=old\n" {
		t.Fatalf(".env changed: %q", data)
	}
	if _, err := write.Execute(context.Background(), map[string]any{"path": ".env.example", "content": "PSC_1_API_KEY=\n"}); err != nil {
		t.Fatalf("a template must stay writable: %v", err)
	}
}

// A link with an innocent name must not lead either Tool to .env.
func TestCredentialFileBehindALinkIsRefused(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("KEY=old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(".env", filepath.Join(root, "notes.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := toolNamed(t, sandbox, "workspace.edit").Execute(context.Background(), map[string]any{"path": "notes.txt", "old": "old", "new": "new"}); err == nil {
		t.Error("edit followed a link to .env")
	}
	if _, err := toolNamed(t, sandbox, "workspace.write").Execute(context.Background(), map[string]any{"path": "notes.txt", "content": "KEY=new\n"}); err == nil {
		t.Error("write followed a link to .env")
	}
	if data, _ := os.ReadFile(filepath.Join(root, ".env")); string(data) != "KEY=old\n" {
		t.Fatalf(".env changed: %q", data)
	}
}
