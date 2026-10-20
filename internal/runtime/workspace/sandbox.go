package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrOutsideWorkspace = errors.New("path is outside workspace sandbox")

// Sandbox resolves filesystem paths against one immutable workspace root.
// It is a path boundary, not an operating-system process sandbox.
type Sandbox struct {
	root         string
	resolvedRoot string
}

// NewSandbox creates a workspace path boundary and resolves root symlinks.
func NewSandbox(root string) (*Sandbox, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root: %w", err)
	}
	absRoot = filepath.Clean(absRoot)

	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("stat workspace root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("workspace root is not a directory: %s", absRoot)
	}

	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root symlinks: %w", err)
	}
	resolvedRoot, err = filepath.Abs(resolvedRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve canonical workspace root: %w", err)
	}

	return &Sandbox{
		root:         absRoot,
		resolvedRoot: filepath.Clean(resolvedRoot),
	}, nil
}

// Root returns the lexical workspace root shown to the user.
func (s *Sandbox) Root() string {
	return s.root
}

// RelativePath converts a previously resolved sandbox path to a stable,
// slash-separated path for display and logs.
func (s *Sandbox) RelativePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve relative sandbox path: %w", err)
	}
	if !within(s.resolvedRoot, absPath) {
		return "", fmt.Errorf("%w: %s", ErrOutsideWorkspace, path)
	}
	rel, err := filepath.Rel(s.resolvedRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("make sandbox path relative: %w", err)
	}
	return filepath.ToSlash(rel), nil
}

// ResolveExisting resolves an existing path and rejects workspace or symlink
// escapes.
func (s *Sandbox) ResolveExisting(path string) (string, error) {
	resolved, err := s.resolve(path)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(resolved); err != nil {
		return "", fmt.Errorf("stat sandbox path: %w", err)
	}
	return resolved, nil
}

// ResolveForWrite resolves a path that may not exist yet. Its nearest existing
// parent is evaluated for symlink escapes before the candidate is accepted.
func (s *Sandbox) ResolveForWrite(path string) (string, error) {
	return s.resolve(path)
}

func (s *Sandbox) resolve(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("sandbox path is empty")
	}

	candidate := path
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(s.root, candidate)
	}
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve sandbox path: %w", err)
	}
	absCandidate = filepath.Clean(absCandidate)
	if !within(s.root, absCandidate) {
		return "", fmt.Errorf("%w: %s", ErrOutsideWorkspace, path)
	}

	resolvedCandidate, err := resolveWithMissingTail(absCandidate)
	if err != nil {
		return "", err
	}
	if !within(s.resolvedRoot, resolvedCandidate) {
		return "", fmt.Errorf("%w: %s", ErrOutsideWorkspace, path)
	}
	return resolvedCandidate, nil
}

func resolveWithMissingTail(path string) (string, error) {
	current := path
	var missing []string

	for {
		_, err := os.Lstat(current)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect sandbox path: %w", err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("find existing sandbox parent: %s", path)
		}
		missing = append([]string{filepath.Base(current)}, missing...)
		current = parent
	}

	resolved, err := filepath.EvalSymlinks(current)
	if err != nil {
		return "", fmt.Errorf("resolve sandbox path symlinks: %w", err)
	}
	for _, part := range missing {
		resolved = filepath.Join(resolved, part)
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve canonical sandbox path: %w", err)
	}
	return filepath.Clean(resolved), nil
}

func within(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil || filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
