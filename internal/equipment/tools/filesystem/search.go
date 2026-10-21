package filesystem

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxSearchEntries = 20000
	maxSearchFiles   = 2000
	maxSearchMatches = 50
	maxMatchLine     = 300
)

// skippedDirs hold host state or generated output, never project sources.
var skippedDirs = map[string]bool{".git": true, ".boi": true, "agent-folder": true, "node_modules": true}

type SearchMatch struct {
	Path string
	Line int
	Text string
}

type SearchResult struct {
	Pattern      string
	Path         string
	Matches      []SearchMatch
	FilesScanned int
	Truncated    bool
}

// Search finds lines containing pattern, case-insensitively, in one file or
// every text file under a directory. Symlinks are skipped so the walk never
// leaves the workspace; reads use the same size cap and binary check as Read.
func (r *Reader) Search(ctx context.Context, pattern, path string) (*SearchResult, error) {
	if r == nil || r.sandbox == nil {
		return nil, fmt.Errorf("workspace reader is not configured")
	}
	if strings.TrimSpace(pattern) == "" {
		return nil, fmt.Errorf("search pattern is empty")
	}
	if path == "" {
		path = "."
	}
	root, err := r.sandbox.ResolveExisting(path)
	if err != nil {
		return nil, err
	}
	display, err := r.sandbox.RelativePath(root)
	if err != nil {
		return nil, err
	}
	result := &SearchResult{Pattern: pattern, Path: display}
	needle := strings.ToLower(pattern)
	visited := 0
	walkErr := filepath.WalkDir(root, func(current string, entry fs.DirEntry, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil {
			return nil
		}
		// Count every entry, not only files, so a tree of empty folders
		// cannot keep the walk going past its timeout.
		if visited++; visited > maxSearchEntries {
			result.Truncated = true
			return filepath.SkipAll
		}
		if entry.IsDir() {
			if current != root && skippedDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() || IsSecretFile(current) {
			return nil
		}
		if result.FilesScanned >= maxSearchFiles || len(result.Matches) >= maxSearchMatches {
			result.Truncated = true
			return filepath.SkipAll
		}
		result.FilesScanned++
		return r.searchFile(current, needle, result)
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return result, nil
}

func (r *Reader) searchFile(path, needle string, result *SearchResult) error {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, r.maxReadBytes))
	if err != nil || bytes.IndexByte(content, 0) >= 0 {
		return nil
	}
	relative, err := r.sandbox.RelativePath(path)
	if err != nil {
		return nil
	}
	for index, line := range strings.Split(string(content), "\n") {
		if !strings.Contains(strings.ToLower(line), needle) {
			continue
		}
		if len(result.Matches) >= maxSearchMatches {
			result.Truncated = true
			return filepath.SkipAll
		}
		if len(line) > maxMatchLine {
			line = strings.ToValidUTF8(line[:maxMatchLine], "") + "..."
		}
		result.Matches = append(result.Matches, SearchMatch{Path: relative, Line: index + 1, Text: line})
	}
	return nil
}
