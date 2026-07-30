package memory

import (
	"os"
	"path/filepath"
	"strings"
)

// RepoMapFile represents a file in the project
type RepoMapFile struct {
	Path string
	Size int64
	Ext  string
}

// RepoMap holds the project file structure
type RepoMap struct {
	Root      string
	Files     []RepoMapFile
	FileCount int
	TotalSize int64
}

// ScanRepo scans the project directory for code files
func ScanRepo(root string) (*RepoMap, error) {
	m := &RepoMap{Root: root}

	skipDirs := map[string]bool{
		".boi": true, ".git": true, "node_modules": true,
		"vendor": true, ".evolution": true, "__pycache__": true,
		"bin": true, "obj": true, "dist": true, "build": true,
	}

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		ext := filepath.Ext(path)

		m.Files = append(m.Files, RepoMapFile{
			Path: rel,
			Size: info.Size(),
			Ext:  ext,
		})
		m.FileCount++
		m.TotalSize += info.Size()
		return nil
	})

	return m, nil
}

// Summary returns a text summary of the repo structure
func (m *RepoMap) Summary() string {
	extCount := make(map[string]int)
	for _, f := range m.Files {
		e := f.Ext
		if e == "" {
			e = "(no ext)"
		}
		extCount[e]++
	}

	var sb strings.Builder
	sb.WriteString("Project: ")
	sb.WriteString(filepath.Base(m.Root))
	sb.WriteString("\n")
	sb.WriteString("Files: ")
	sb.WriteString(itoa(m.FileCount))

	if len(extCount) > 0 {
		sb.WriteString(" | ")
		for ext, count := range extCount {
			sb.WriteString(ext)
			sb.WriteString(":")
			sb.WriteString(itoa(count))
			sb.WriteString(" ")
		}
	}
	return sb.String()
}
