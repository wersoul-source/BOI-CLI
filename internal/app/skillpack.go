package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/boi-family/boi-cli/internal/equipment/skill"
)

// maxSkillPackFiles bounds one Skill folder; each file is bounded by
// skill.MaxFileBytes.
const maxSkillPackFiles = 15

// loadSkillFolder reads the *.skill.md files of one folder inside boiDir. It
// returns the Skills, or the reason the folder cannot be used. The folder must
// stay inside boiDir after symlinks are resolved, and every entry must be a
// plain file within the size limit; nothing outside the folder is read.
func loadSkillFolder(boiDir, dir string) ([]*skill.Skill, string) {
	if boiDir == "" {
		return nil, "there is no .boi directory to read Skills from"
	}
	clean := filepath.Clean(dir)
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, fmt.Sprintf("Skill folder %q must be a folder inside .boi", dir)
	}
	root, err := filepath.EvalSymlinks(boiDir)
	if err != nil {
		return nil, "the .boi directory cannot be read: " + err.Error()
	}
	folder, err := filepath.EvalSymlinks(filepath.Join(root, clean))
	if err != nil {
		return nil, fmt.Sprintf("Skill folder %q does not exist", dir)
	}
	if rel, err := filepath.Rel(root, folder); err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Sprintf("Skill folder %q leaves .boi", dir)
	}
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, fmt.Sprintf("Skill folder %q cannot be read: %v", dir, err)
	}
	var skills []*skill.Skill
	seen := map[string]string{}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".skill.md") {
			continue
		}
		if len(skills) >= maxSkillPackFiles {
			return nil, fmt.Sprintf("Skill folder %q holds more than %d Skills", dir, maxSkillPackFiles)
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Sprintf("%s is not a regular file", entry.Name())
		}
		if info.Size() > skill.MaxFileBytes {
			return nil, fmt.Sprintf("%s is larger than %d bytes", entry.Name(), skill.MaxFileBytes)
		}
		loaded, err := skill.LoadConfined(folder, entry.Name())
		if err != nil {
			return nil, fmt.Sprintf("%s: %v", entry.Name(), err)
		}
		if other, dup := seen[loaded.Name]; dup {
			return nil, fmt.Sprintf("Skill %s is defined in both %s and %s", loaded.Name, other, entry.Name())
		}
		seen[loaded.Name] = entry.Name()
		skills = append(skills, loaded)
	}
	if len(skills) == 0 {
		return nil, fmt.Sprintf("Skill folder %q has no *.skill.md files", dir)
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills, ""
}

func skillNames(skills []*skill.Skill) []string {
	names := make([]string, 0, len(skills))
	for _, item := range skills {
		names = append(names, item.Name)
	}
	return names
}
