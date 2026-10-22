package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	coreblock "github.com/boi-family/boi-cli/internal/core"
	"github.com/boi-family/boi-cli/internal/equipment/capability"
)

// A cloned workspace can list a Skill whose file is a symlink to any file the
// user can read. The Skill's text goes to the model, so it must not load.
func TestRegistrySkillCannotLeaveTheSkillsFolder(t *testing.T) {
	boiDir := t.TempDir()
	if err := EnsureCapabilityIndexes(boiDir); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "private.skill.md")
	if err := os.WriteFile(outside, []byte("---\nname: leak\ndescription: leak\n---\n\nTOP-SECRET-CONTENT"), 0o600); err != nil {
		t.Fatal(err)
	}
	skills := filepath.Join(boiDir, "skills")
	if err := os.MkdirAll(skills, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(skills, "leak.skill.md")); err != nil {
		t.Skip("symlinks are not available:", err)
	}
	entry := capability.Entry{Name: "leak", Source: "leak.skill.md", Summary: "leak", Enabled: true, Tags: []string{"leak"}, ContextCost: 100}
	if err := capability.AddEntry(capability.IndexPath(boiDir, capability.KindSkill), capability.KindSkill, entry); err != nil {
		t.Fatal(err)
	}
	environment := coreblock.AgentEnvironment{ToolCalling: true, SkillCalling: true, ContextBytes: 4096}
	set, err := SelectCapabilities(boiDir, "leak", environment, builtinToolNames)
	if err != nil {
		t.Fatal(err)
	}
	for _, doc := range set.SkillDocs() {
		if strings.Contains(doc.Instructions, "TOP-SECRET") {
			t.Fatalf("a symlinked Skill outside .boi/skills was loaded: %+v", doc)
		}
	}
	if len(set.Skills.Active) != 0 {
		t.Fatalf("the escaping Skill is active: %v", set.Skills.Active)
	}
	if _, err := LoadRegisteredSkill(boiDir, "leak"); err == nil {
		t.Fatal("LoadRegisteredSkill read a file outside .boi/skills")
	}
}
