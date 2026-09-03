package app

import (
	"os"
	"path/filepath"
	"testing"

	coreblock "github.com/boi-family/boi-cli/internal/block/core"
	"github.com/boi-family/boi-cli/internal/capability"
)

func TestLooseSkillFileIsNotExposedUntilIndexed(t *testing.T) {
	boiDir := t.TempDir()
	if err := EnsureCapabilityIndexes(boiDir); err != nil {
		t.Fatal(err)
	}
	skillDir := filepath.Join(boiDir, "skills")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("---\nname: loose\ndescription: Loose skill\nversion: 1\n---\n\nsecret instructions")
	if err := os.WriteFile(filepath.Join(skillDir, "loose.skill.md"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	env := coreblock.AgentEnvironment{ToolCalling: true, SkillCalling: true, ContextBytes: 4096}
	set, err := SelectCapabilities(boiDir, "loose", env)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Skills.Active) != 0 || set.SkillSummaryPrompt() != "" {
		t.Fatal("unindexed Skill was exposed")
	}
	entry := capability.Entry{Name: "loose", Source: "loose.skill.md", Summary: "Loose skill", Enabled: true, Tags: []string{"loose"}, ContextCost: 100}
	if err := capability.AddEntry(capability.IndexPath(boiDir, capability.KindSkill), capability.KindSkill, entry); err != nil {
		t.Fatal(err)
	}
	set, err = SelectCapabilities(boiDir, "loose", env)
	if err != nil || len(set.Skills.Active) != 1 || set.Skills.Active[0] != "loose" {
		t.Fatalf("indexed Skill not active: %#v %v", set, err)
	}
	if full, err := set.Skill("loose"); err != nil || full.Prompt != "secret instructions" {
		t.Fatalf("active Skill instructions unavailable: %#v %v", full, err)
	}
	if _, err := set.Skill("not-active"); err == nil {
		t.Fatal("inactive Skill instructions were exposed")
	}
}

func TestDefaultToolIndexIsExplicitAndBounded(t *testing.T) {
	boiDir := t.TempDir()
	if err := EnsureCapabilityIndexes(boiDir); err != nil {
		t.Fatal(err)
	}
	index, err := capability.LoadIndex(capability.IndexPath(boiDir, capability.KindTool), capability.KindTool)
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Entries) != 4 {
		t.Fatalf("default Tool entries = %d", len(index.Entries))
	}
	set, err := SelectCapabilities(boiDir, "read files", coreblock.AgentEnvironment{ToolCalling: true})
	if err != nil || len(set.Tools.Active) != 4 {
		t.Fatalf("default active Tools: %#v %v", set, err)
	}
}
