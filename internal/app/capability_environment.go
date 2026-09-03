package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	coreblock "github.com/boi-family/boi-cli/internal/block/core"
	"github.com/boi-family/boi-cli/internal/capability"
	"github.com/boi-family/boi-cli/internal/skill"
)

type CapabilitySet struct {
	Tools        capability.Selection
	Skills       capability.Selection
	LoadedSkills []*skill.Skill
}

func (set *CapabilitySet) SkillSummaryPrompt() string {
	if set == nil || len(set.LoadedSkills) == 0 {
		return ""
	}
	var lines []string
	for _, item := range set.LoadedSkills {
		lines = append(lines, "- "+item.Name+": "+item.Description)
	}
	return strings.Join(lines, "\n")
}

// Skill returns full instructions only for a Skill in the task's active set.
// Skill content remains untrusted context and cannot expand Tool authority.
func (set *CapabilitySet) Skill(name string) (*skill.Skill, error) {
	if set == nil {
		return nil, fmt.Errorf("capability set is unavailable")
	}
	active := false
	for _, candidate := range set.Skills.Active {
		if candidate == name {
			active = true
			break
		}
	}
	if !active {
		return nil, fmt.Errorf("Skill is not active for this task: %s", name)
	}
	for _, item := range set.LoadedSkills {
		if item.Name == name {
			return item, nil
		}
	}
	return nil, fmt.Errorf("active Skill instructions are unavailable: %s", name)
}

func DefaultCapabilityIndexes() (capability.Index, capability.Index) {
	tools := capability.Index{SchemaVersion: capability.IndexSchemaVersion, Kind: capability.KindTool, Entries: []capability.Entry{
		{Name: "workspace.list", Source: "builtin", Summary: "List files inside the workspace", Enabled: true, Priority: 100, Tags: []string{"list", "files"}},
		{Name: "workspace.read", Source: "builtin", Summary: "Read a file inside the workspace", Enabled: true, Priority: 100, Tags: []string{"read", "file"}},
		{Name: "workspace.write", Source: "builtin", Summary: "Write a file with approval", Enabled: true, Priority: 90, Tags: []string{"write", "file"}},
		{Name: "process.run", Source: "builtin", Summary: "Run a bounded workspace process with approval", Enabled: true, Priority: 80, Tags: []string{"run", "command", "test"}},
	}}
	skills := capability.Index{SchemaVersion: capability.IndexSchemaVersion, Kind: capability.KindSkill, Entries: []capability.Entry{}}
	return tools, skills
}

func EnsureCapabilityIndexes(boiDir string) error {
	tools, skills := DefaultCapabilityIndexes()
	for _, item := range []struct {
		kind  capability.Kind
		index capability.Index
	}{{capability.KindTool, tools}, {capability.KindSkill, skills}} {
		path := capability.IndexPath(boiDir, item.kind)
		if _, err := os.Stat(path); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := capability.SaveIndex(path, &item.index); err != nil {
			return err
		}
	}
	return nil
}

func SelectCapabilities(boiDir, task string, environment coreblock.AgentEnvironment) (*CapabilitySet, error) {
	toolIndex, err := capability.LoadIndex(capability.IndexPath(boiDir, capability.KindTool), capability.KindTool)
	if err != nil {
		return nil, err
	}
	installedTools := map[string]bool{"workspace.list": true, "workspace.read": true, "workspace.write": true, "process.run": true}
	tools := capability.Select(*toolIndex, capability.SelectionInput{Task: task, Installed: installedTools, ProviderAllows: environment.ToolCalling})
	available := map[string]bool{}
	for _, name := range tools.Active {
		available[name] = true
	}

	skillIndex, err := capability.LoadIndex(capability.IndexPath(boiDir, capability.KindSkill), capability.KindSkill)
	if err != nil {
		return nil, err
	}
	skillDir := filepath.Join(boiDir, "skills")
	installedSkills := map[string]bool{}
	loadedByName := map[string]*skill.Skill{}
	for _, entry := range skillIndex.Entries {
		path, resolveErr := resolveIndexedSkill(skillDir, entry.Source)
		if resolveErr != nil {
			continue
		}
		loaded, loadErr := skill.LoadFile(path)
		if loadErr == nil && loaded.Name == entry.Name {
			installedSkills[entry.Name], loadedByName[entry.Name] = true, loaded
		}
	}
	skills := capability.Select(*skillIndex, capability.SelectionInput{Task: task, Installed: installedSkills, AvailableRequirements: available, ProviderAllows: environment.SkillCalling, ContextBudget: environment.ContextBytes})
	loaded := make([]*skill.Skill, 0, len(skills.Active))
	for _, name := range skills.Active {
		if item := loadedByName[name]; item != nil {
			loaded = append(loaded, item)
		}
	}
	return &CapabilitySet{Tools: tools, Skills: skills, LoadedSkills: loaded}, nil
}

func resolveIndexedSkill(root, source string) (string, error) {
	if filepath.IsAbs(source) {
		return "", fmt.Errorf("Skill source must be relative")
	}
	clean := filepath.Clean(source)
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("Skill source escapes registry")
	}
	resolved := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, resolved)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("Skill source escapes registry")
	}
	return resolved, nil
}

func LoadRegisteredSkill(boiDir, name string) (*skill.Skill, error) {
	index, err := capability.LoadIndex(capability.IndexPath(boiDir, capability.KindSkill), capability.KindSkill)
	if err != nil {
		return nil, err
	}
	for _, entry := range index.Entries {
		if entry.Name != name {
			continue
		}
		path, err := resolveIndexedSkill(filepath.Join(boiDir, "skills"), entry.Source)
		if err != nil {
			return nil, err
		}
		loaded, err := skill.LoadFile(path)
		if err != nil {
			return nil, err
		}
		if loaded.Name != entry.Name {
			return nil, fmt.Errorf("indexed Skill name %q does not match file name %q", entry.Name, loaded.Name)
		}
		return loaded, nil
	}
	return nil, fmt.Errorf("Skill is not registered: %s", name)
}
