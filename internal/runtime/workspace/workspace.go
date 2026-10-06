package workspace

import (
	"os"
	"path/filepath"
)

const BoiDir = ".boi"

// DetectRoot finds the project root (containing .boi/ or .git/)
// Walks up the directory tree from current working directory.
func DetectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := cwd
	for {
		// Check for .boi/
		boiPath := filepath.Join(dir, BoiDir)
		if info, err := os.Stat(boiPath); err == nil && info.IsDir() {
			return dir, nil
		}

		// Check for .git/ (fallback)
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return dir, nil
		}

		// Go up one level
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Not found — use cwd
	return cwd, nil
}

// GetBoiDir returns the .boi/ directory path for a project root
func GetBoiDir(root string) string {
	return filepath.Join(root, BoiDir)
}
