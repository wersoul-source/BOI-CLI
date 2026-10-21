package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	coreblock "github.com/boi-family/boi-cli/internal/core"
	"github.com/boi-family/boi-cli/internal/equipment/capability"
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
	set, err := SelectCapabilities(boiDir, "loose", env, builtinToolNames)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Skills.Active) != 0 || len(set.SkillDocs()) != 0 {
		t.Fatal("unindexed Skill was exposed")
	}
	entry := capability.Entry{Name: "loose", Source: "loose.skill.md", Summary: "Loose skill", Enabled: true, Tags: []string{"loose"}, ContextCost: 100}
	if err := capability.AddEntry(capability.IndexPath(boiDir, capability.KindSkill), capability.KindSkill, entry); err != nil {
		t.Fatal(err)
	}
	set, err = SelectCapabilities(boiDir, "loose", env, builtinToolNames)
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
	if len(index.Entries) != 6 {
		t.Fatalf("default Tool entries = %d", len(index.Entries))
	}
	set, err := SelectCapabilities(boiDir, "read files", coreblock.AgentEnvironment{ToolCalling: true}, builtinToolNames)
	if err != nil || len(set.Tools.Active) != 6 {
		t.Fatalf("default active Tools: %#v %v", set, err)
	}
}

var builtinToolNames = []string{"workspace.list", "workspace.read", "workspace.search", "workspace.edit", "workspace.write", "process.run"}

func TestSkillsAreOfferedWithoutKeywordsOrTheirTools(t *testing.T) {
	boiDir := t.TempDir()
	if err := EnsureCapabilityIndexes(boiDir); err != nil {
		t.Fatal(err)
	}
	skillDir := filepath.Join(boiDir, "skills")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "deploy.skill.md"), []byte("---\nname: deploy\ndescription: How to deploy\n---\n\nsteps"), 0o644); err != nil {
		t.Fatal(err)
	}
	entry := capability.Entry{Name: "deploy", Source: "deploy.skill.md", Summary: "Deploy the service", Enabled: true, Tags: []string{"deploy"}, Requires: []string{"process.run"}, ContextCost: 100}
	if err := capability.AddEntry(capability.IndexPath(boiDir, capability.KindSkill), capability.KindSkill, entry); err != nil {
		t.Fatal(err)
	}
	env := coreblock.AgentEnvironment{ToolCalling: true, SkillCalling: true, ContextBytes: 4096}
	readOnly := []string{"workspace.list", "workspace.read"}
	// A Thai task shares no word with the Skill's tags, and process.run is
	// not installed: the Skill is still offered, marked as running without it.
	set, err := SelectCapabilities(boiDir, "ช่วยขึ้นระบบให้หน่อย", env, readOnly)
	if err != nil || len(set.Skills.Active) != 1 {
		t.Fatalf("Skill hidden: %#v %v", set.Skills, err)
	}
	docs := set.SkillDocs()
	if len(docs) != 1 || !strings.Contains(docs[0].Description, "process.run") {
		t.Fatalf("missing Tool not named: %#v", docs)
	}
	set, err = SelectCapabilities(boiDir, "deploy", env, builtinToolNames)
	if err != nil || set.SkillDocs()[0].Description != "How to deploy" {
		t.Fatalf("Skill with its Tools: %#v %v", set.SkillDocs(), err)
	}
}
