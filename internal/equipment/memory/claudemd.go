package memory

import (
	"os"
	"path/filepath"
)

// LoadMemoryFiles loads PROJECT_MEMORY.md from the directory tree
// Search order: .boi/memory.md → BOI_MEMORY.md → .git/BOI_MEMORY.md
func LoadMemoryFiles(root string) (string, error) {
	searchPaths := []string{
		filepath.Join(root, ".boi", "memory.md"),
		filepath.Join(root, "BOI_MEMORY.md"),
		filepath.Join(root, "docs", "MEMORY.md"),
	}

	for _, path := range searchPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		return string(data), nil
	}

	return "", nil
}

// SaveMemoryFile saves a memory file
func SaveMemoryFile(root, content string) error {
	path := filepath.Join(root, ".boi", "memory.md")
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
