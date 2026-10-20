package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// LoadDir loads all .skill.md files from a directory
func LoadDir(dir string) ([]*Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read skill dir: %w", err)
	}

	var skills []*Skill
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".skill.md") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		s, err := LoadFile(path)
		if err != nil {
			continue
		}
		skills = append(skills, s)
	}
	return skills, nil
}

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
