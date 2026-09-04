package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnsureLocalGitExcludes adds machine-local paths to .git/info/exclude without
// changing the repository's tracked .gitignore. Non-Git workspaces are a no-op.
func EnsureLocalGitExcludes(root string, patterns ...string) error {
	gitDir, ok, err := resolveGitDir(root)
	if err != nil || !ok {
		return err
	}
	infoDir := filepath.Join(gitDir, "info")
	if err := os.MkdirAll(infoDir, 0o755); err != nil {
		return fmt.Errorf("create Git info directory: %w", err)
	}
	path := filepath.Join(infoDir, "exclude")
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read Git exclude: %w", err)
	}
	present := make(map[string]bool)
	for _, line := range strings.Split(string(existing), "\n") {
		present[strings.TrimSpace(line)] = true
	}
	var additions []string
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" && !present[pattern] {
			additions = append(additions, pattern)
			present[pattern] = true
		}
	}
	if len(additions) == 0 {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open Git exclude: %w", err)
	}
	defer f.Close()
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		if _, err := f.WriteString("\n"); err != nil {
			return fmt.Errorf("separate Git exclude entries: %w", err)
		}
	}
	if _, err := f.WriteString("# BOI CLI local state\n" + strings.Join(additions, "\n") + "\n"); err != nil {
		return fmt.Errorf("write Git exclude: %w", err)
	}
	return f.Sync()
}

func resolveGitDir(root string) (string, bool, error) {
	marker := filepath.Join(root, ".git")
	info, err := os.Stat(marker)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("inspect Git marker: %w", err)
	}
	if info.IsDir() {
		return marker, true, nil
	}
	data, err := os.ReadFile(marker)
	if err != nil {
		return "", false, fmt.Errorf("read Git worktree marker: %w", err)
	}
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(strings.ToLower(line), "gitdir:") {
		return "", false, fmt.Errorf("invalid Git worktree marker")
	}
	gitDir := strings.TrimSpace(line[len("gitdir:"):])
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(root, gitDir)
	}
	return filepath.Clean(gitDir), true, nil
}
