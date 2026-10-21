package memory

import (
	"os"
	"path/filepath"
)

// SaveMemoryFile saves a memory file
func SaveMemoryFile(root, content string) error {
	path := filepath.Join(root, ".boi", "memory.md")
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}
