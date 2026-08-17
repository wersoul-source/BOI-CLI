package filesystem

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/boi-family/boi-cli/internal/workspace"
)

func TestReaderReadsAndListsInsideWorkspace(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	reader := NewReader(sandbox)

	readResult, err := reader.Read("hello.txt")
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if readResult.Path != "hello.txt" || readResult.Content != "hello" || readResult.Truncated {
		t.Fatalf("unexpected read result: %#v", readResult)
	}

	listResult, err := reader.List(".")
	if err != nil {
		t.Fatalf("list directory: %v", err)
	}
	if len(listResult.Entries) != 1 || listResult.Entries[0].Name != "hello.txt" {
		t.Fatalf("unexpected list result: %#v", listResult)
	}
}

func TestReaderRejectsOutsideWorkspace(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	root := filepath.Join(parent, "workspace")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	outside := filepath.Join(parent, "outside.txt")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatalf("create outside file: %v", err)
	}
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	reader := NewReader(sandbox)

	_, err = reader.Read(outside)
	if !errors.Is(err, workspace.ErrOutsideWorkspace) {
		t.Fatalf("Read() error = %v, want ErrOutsideWorkspace", err)
	}
}

func TestReaderRejectsBinaryAndTruncatesLargeText(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "binary.bin"), []byte{'a', 0, 'b'}, 0o600); err != nil {
		t.Fatalf("create binary fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "large.txt"), bytes.Repeat([]byte{'a'}, defaultMaxReadBytes+10), 0o600); err != nil {
		t.Fatalf("create large fixture: %v", err)
	}
	sandbox, err := workspace.NewSandbox(root)
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	reader := NewReader(sandbox)

	if _, err := reader.Read("binary.bin"); err == nil {
		t.Fatal("binary file was accepted")
	}
	result, err := reader.Read("large.txt")
	if err != nil {
		t.Fatalf("read large file: %v", err)
	}
	if !result.Truncated || len(result.Content) != defaultMaxReadBytes {
		t.Fatalf("large file was not truncated: %#v", result)
	}
}
