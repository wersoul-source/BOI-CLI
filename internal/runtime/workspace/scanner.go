package workspace

import (
	"os"
	"path/filepath"
	"sort"
)

// ScanResult holds project structure scan results
type ScanResult struct {
	FileCount int
	DirCount  int
	Languages map[string]int // extension -> count
	Files     []string       // relative paths
}

// ScanProject scans the project directory structure
// Skips .boi/, .git/, node_modules/, vendor/
func ScanProject(root string) (*ScanResult, error) {
	result := &ScanResult{
		Languages: make(map[string]int),
	}

	skipDirs := map[string]bool{
		".boi":         true,
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		"__pycache__":  true,
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}

		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			result.DirCount++
			return nil
		}

		result.FileCount++

		relPath, _ := filepath.Rel(root, path)
		result.Files = append(result.Files, relPath)

		ext := filepath.Ext(path)
		if ext == "" {
			ext = "no-ext"
		}
		result.Languages[ext]++

		return nil
	})

	sort.Strings(result.Files)
	return result, err
}
