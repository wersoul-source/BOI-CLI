package filesystem

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/boi-family/boi-cli/internal/workspace"
)

const (
	defaultMaxReadBytes = 256 * 1024
	defaultMaxEntries   = 200
)

// Reader exposes bounded, read-only workspace capabilities.
type Reader struct {
	sandbox      *workspace.Sandbox
	maxReadBytes int64
	maxEntries   int
}

type ReadResult struct {
	Path      string
	Content   string
	Size      int64
	Truncated bool
}

type Entry struct {
	Name  string
	IsDir bool
	Size  int64
}

type ListResult struct {
	Path      string
	Entries   []Entry
	Truncated bool
}

func NewReader(sandbox *workspace.Sandbox) *Reader {
	return &Reader{
		sandbox:      sandbox,
		maxReadBytes: defaultMaxReadBytes,
		maxEntries:   defaultMaxEntries,
	}
}

func (r *Reader) Read(path string) (*ReadResult, error) {
	if r == nil || r.sandbox == nil {
		return nil, fmt.Errorf("workspace reader is not configured")
	}
	resolved, err := r.sandbox.ResolveExisting(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("stat workspace file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("workspace path is not a regular file: %s", path)
	}

	file, err := os.Open(resolved)
	if err != nil {
		return nil, fmt.Errorf("open workspace file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, r.maxReadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read workspace file: %w", err)
	}
	if bytes.IndexByte(content, 0) >= 0 {
		return nil, fmt.Errorf("workspace file appears to be binary: %s", path)
	}

	truncated := int64(len(content)) > r.maxReadBytes
	if truncated {
		content = content[:r.maxReadBytes]
	}

	displayPath, err := r.sandbox.RelativePath(resolved)
	if err != nil {
		return nil, err
	}
	return &ReadResult{
		Path:      displayPath,
		Content:   string(content),
		Size:      info.Size(),
		Truncated: truncated,
	}, nil
}

func (r *Reader) List(path string) (*ListResult, error) {
	if r == nil || r.sandbox == nil {
		return nil, fmt.Errorf("workspace reader is not configured")
	}
	if path == "" {
		path = "."
	}
	resolved, err := r.sandbox.ResolveExisting(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("stat workspace directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("workspace path is not a directory: %s", path)
	}

	dirEntries, err := os.ReadDir(resolved)
	if err != nil {
		return nil, fmt.Errorf("list workspace directory: %w", err)
	}
	truncated := len(dirEntries) > r.maxEntries
	if truncated {
		dirEntries = dirEntries[:r.maxEntries]
	}

	entries := make([]Entry, 0, len(dirEntries))
	for _, entry := range dirEntries {
		item := Entry{Name: entry.Name(), IsDir: entry.IsDir()}
		if !entry.IsDir() {
			entryInfo, infoErr := entry.Info()
			if infoErr == nil {
				item.Size = entryInfo.Size()
			}
		}
		entries = append(entries, item)
	}

	displayPath, err := r.sandbox.RelativePath(resolved)
	if err != nil {
		return nil, err
	}
	return &ListResult{
		Path:      displayPath,
		Entries:   entries,
		Truncated: truncated,
	}, nil
}
