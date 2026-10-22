package skill

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// LoadFile loads a single SKILL.md file
func LoadFile(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return Parse(data, path, false)
}

// Parse parses SKILL.md content into a Skill
func Parse(content []byte, source string, builtIn bool) (*Skill, error) {
	text := string(content)
	lines := strings.Split(text, "\n")

	var cfg SkillConfig
	inFM := false
	fmLines := []string{}
	bodyStart := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			if !inFM {
				inFM = true
				continue
			}
			bodyStart = i + 1
			break
		}
		if inFM {
			fmLines = append(fmLines, line)
		}
	}

	if len(fmLines) > 0 {
		fm := strings.Join(fmLines, "\n")
		if err := yaml.Unmarshal([]byte(fm), &cfg); err != nil {
			return nil, fmt.Errorf("parse frontmatter in %s: %w", source, err)
		}
	}

	promptText := ""
	if bodyStart > 0 && bodyStart < len(lines) {
		promptText = strings.Join(lines[bodyStart:], "\n")
	}

	if cfg.Name == "" {
		cfg.Name = strings.TrimSuffix(filepath.Base(source), ".skill.md")
	}
	if cfg.Version == "" {
		cfg.Version = "1.0"
	}

	return &Skill{
		Name:        cfg.Name,
		Description: cfg.Description,
		Version:     cfg.Version,
		Requires:    cfg.Requires,
		Prompt:      strings.TrimSpace(promptText),
		Path:        source,
		BuiltIn:     builtIn,
		LoadedAt:    time.Now(),
	}, nil
}

// MaxFileBytes bounds one Skill file.
const MaxFileBytes = 64 << 10

// ErrOutsideRoot marks a Skill source that resolves outside its root.
var ErrOutsideRoot = errors.New("Skill source escapes its folder")

// LoadConfined loads the Skill file at source, a path relative to root. After
// symlinks are resolved the file must be a regular file inside root and within
// MaxFileBytes, so a registry entry cannot point the Agent at any file the
// user can read. The Skill keeps the name of the path it was listed under.
func LoadConfined(root, source string) (*Skill, error) {
	if filepath.IsAbs(source) {
		return nil, fmt.Errorf("Skill source must be relative")
	}
	clean := filepath.Clean(source)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, ErrOutsideRoot
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("Skill folder cannot be read: %w", err)
	}
	real, err := filepath.EvalSymlinks(filepath.Join(realRoot, clean))
	if err != nil {
		return nil, fmt.Errorf("Skill file cannot be read: %w", err)
	}
	if rel, err := filepath.Rel(realRoot, real); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, ErrOutsideRoot
	}
	info, err := os.Stat(real)
	if err != nil {
		return nil, fmt.Errorf("Skill file cannot be read: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("Skill source %s is not a regular file", clean)
	}
	if info.Size() > MaxFileBytes {
		return nil, fmt.Errorf("Skill source %s is larger than %d bytes", clean, MaxFileBytes)
	}
	data, err := os.ReadFile(real)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return Parse(data, filepath.Join(root, clean), false)
}
