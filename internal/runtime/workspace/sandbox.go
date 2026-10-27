package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var ErrOutsideWorkspace = errors.New("path is outside workspace sandbox")

// root is one allowed folder, kept twice: the path the user sees and the path
// with every symlink resolved, which is what an escape is measured against.
type root struct {
	lexical  string
	resolved string
}

// Sandbox resolves filesystem paths against a workspace access policy: one
// primary root, any folders the policy adds, and any folder granted for the
// current task. It is a path boundary, not an operating-system process
// sandbox.
//
// Widening the policy never turns the boundary off. A path is accepted only
// when it stays inside one named folder, before and after symlinks resolve.
type Sandbox struct {
	primary root
	mu      sync.RWMutex
	// extra holds folders the access policy allows for the whole session.
	extra []root
	// granted holds folders allowed for the current task only. They are
	// dropped by ReleaseTaskGrants when the task ends, so permission given
	// once does not silently outlive the work it was given for.
	granted []root
}

// NewSandbox creates a workspace path boundary around one root.
func NewSandbox(path string) (*Sandbox, error) {
	primary, err := newRoot(path)
	if err != nil {
		return nil, err
	}
	return &Sandbox{primary: primary}, nil
}

func newRoot(path string) (root, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return root{}, fmt.Errorf("resolve workspace root: %w", err)
	}
	absolute = filepath.Clean(absolute)

	info, err := os.Stat(absolute)
	if err != nil {
		return root{}, fmt.Errorf("stat workspace root: %w", err)
	}
	if !info.IsDir() {
		return root{}, fmt.Errorf("workspace root is not a directory: %s", absolute)
	}

	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return root{}, fmt.Errorf("resolve workspace root symlinks: %w", err)
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return root{}, fmt.Errorf("resolve canonical workspace root: %w", err)
	}
	return root{lexical: absolute, resolved: filepath.Clean(resolved)}, nil
}

// AddRoot allows a folder for the whole session. It is how an access policy
// names a second project, a shared data folder or a scratch area without
// moving the primary root or turning the boundary off.
func (s *Sandbox) AddRoot(path string) error {
	added, err := newRoot(path)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.known(added) {
		return nil
	}
	s.extra = append(s.extra, added)
	return nil
}

// GrantForTask allows a folder until ReleaseTaskGrants is called, which the
// Runtime does when the task ends. Permission given for one piece of work
// therefore covers that work and nothing after it.
func (s *Sandbox) GrantForTask(path string) error {
	granted, err := newRoot(path)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.known(granted) {
		return nil
	}
	s.granted = append(s.granted, granted)
	return nil
}

// ReleaseTaskGrants withdraws every folder granted for the current task.
func (s *Sandbox) ReleaseTaskGrants() {
	s.mu.Lock()
	s.granted = nil
	s.mu.Unlock()
}

// known reports whether a folder is already allowed. The caller holds the lock.
func (s *Sandbox) known(candidate root) bool {
	for _, allowed := range s.allowedLocked() {
		if allowed.resolved == candidate.resolved {
			return true
		}
	}
	return false
}

func (s *Sandbox) allowedLocked() []root {
	allowed := make([]root, 0, 1+len(s.extra)+len(s.granted))
	allowed = append(allowed, s.primary)
	allowed = append(allowed, s.extra...)
	return append(allowed, s.granted...)
}

func (s *Sandbox) allowed() []root {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.allowedLocked()
}

// Root returns the primary workspace root shown to the user.
func (s *Sandbox) Root() string {
	return s.primary.lexical
}

// Roots returns every folder the policy currently allows, primary first, for
// display and for explaining a refusal.
func (s *Sandbox) Roots() []string {
	allowed := s.allowed()
	paths := make([]string, 0, len(allowed))
	for _, item := range allowed {
		paths = append(paths, item.lexical)
	}
	return paths
}

// RelativePath converts a resolved sandbox path to a stable, slash-separated
// path for display and logs. A path inside the primary root is relative to it;
// a path in another allowed folder is shown in full, because a relative path
// climbing out of the workspace would read as an escape.
func (s *Sandbox) RelativePath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve relative sandbox path: %w", err)
	}
	if within(s.primary.resolved, absolute) {
		relative, err := filepath.Rel(s.primary.resolved, absolute)
		if err != nil {
			return "", fmt.Errorf("make sandbox path relative: %w", err)
		}
		return filepath.ToSlash(relative), nil
	}
	for _, allowed := range s.allowed() {
		if within(allowed.resolved, absolute) {
			return filepath.ToSlash(filepath.Clean(absolute)), nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrOutsideWorkspace, path)
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
		// A relative path always means the primary root: the workspace the
		// user is working in, never a folder the policy happens to allow.
		candidate = filepath.Join(s.primary.lexical, candidate)
	}
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve sandbox path: %w", err)
	}
	absCandidate = filepath.Clean(absCandidate)

	for _, allowed := range s.allowed() {
		if !within(allowed.lexical, absCandidate) {
			continue
		}
		resolvedCandidate, err := resolveWithMissingTail(absCandidate)
		if err != nil {
			return "", err
		}
		// Checked again after symlinks resolve, so a link inside an allowed
		// folder cannot point out of it.
		if within(allowed.resolved, resolvedCandidate) {
			return resolvedCandidate, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrOutsideWorkspace, path)
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
