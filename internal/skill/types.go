package skill

import "time"

// Skill represents a loaded SKILL.md
type Skill struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Version     string   `yaml:"version"`
	Requires    []string `yaml:"requires"`
	Prompt      string   // markdown body content
	Path        string   // source file path
	BuiltIn     bool     // embedded in binary
	LoadedAt    time.Time
}

// SkillConfig is the YAML frontmatter in SKILL.md
type SkillConfig struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Version     string   `yaml:"version"`
	Requires    []string `yaml:"requires"`
}
