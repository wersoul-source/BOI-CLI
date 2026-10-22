package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/boi-family/boi-cli/internal/block"
	coreblock "github.com/boi-family/boi-cli/internal/core"
	"github.com/boi-family/boi-cli/internal/equipment/capability"
	"github.com/boi-family/boi-cli/internal/equipment/skill"
)

func writeSkill(t *testing.T, dir, file, name, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + name + " guide\nrequires:\n  - process.run\n---\n\n" + body
	if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func skillEnvironment(boiDir string, blocks ...block.EnvironmentBlock) block.Environment {
	environment := environmentOf(blocks...)
	environment.Dir = boiDir
	return environment
}

func skillPackBlock(id, dir string, required bool) block.EnvironmentBlock {
	return block.EnvironmentBlock{ID: id, Required: required, Skills: &block.SkillFolder{Dir: dir}}
}

func TestSkillPackJoinsTheWorkingSetWithoutTheRegistry(t *testing.T) {
	boiDir := t.TempDir()
	if err := EnsureCapabilityIndexes(boiDir); err != nil {
		t.Fatal(err)
	}
	writeSkill(t, filepath.Join(boiDir, "skills", "team"), "deploy.skill.md", "deploy", "deploy steps")
	writeSkill(t, filepath.Join(boiDir, "skills", "team"), "review.skill.md", "review", "review steps")
	composition, err := Compose(skillEnvironment(boiDir, skillPackBlock("team", "skills/team", true)), BuiltinPacks(), testSandbox(t), "2.4.0", coreblock.AgentEnvironment{})
	if err != nil {
		t.Fatal(err)
	}
	entry := composition.Entries[0]
	if entry.Status != PackActive || entry.Kind != "skill-pack" || strings.Join(entry.Skills, ",") != "deploy,review" || len(composition.Tools) != 0 {
		t.Fatalf("entry = %+v", entry)
	}
	environment := coreblock.AgentEnvironment{ToolCalling: true, SkillCalling: true, ContextBytes: 4096}
	// A Thai task shares no word with the names, and process.run is not
	// active: both Skills are still offered, without their Tool.
	set, err := selectCapabilities(boiDir, "ช่วยขึ้นระบบให้หน่อย", environment, []string{"workspace.read"}, composition.Skills)
	if err != nil || len(set.Skills.Active) != 2 {
		t.Fatalf("pack Skills not offered: %+v %v", set.Skills, err)
	}
	if full, err := set.Skill("deploy"); err != nil || full.Prompt != "deploy steps" {
		t.Fatalf("pack Skill instructions: %+v %v", full, err)
	}
	// Without Skill calling nothing is offered.
	set, _ = selectCapabilities(boiDir, "deploy", coreblock.AgentEnvironment{ToolCalling: true}, nil, composition.Skills)
	if len(set.Skills.Active) != 0 {
		t.Fatalf("Skills offered to a Provider that is not qualified for them: %v", set.Skills.Active)
	}
}

func TestRegistrySkillWinsOverAPackSkill(t *testing.T) {
	boiDir := t.TempDir()
	if err := EnsureCapabilityIndexes(boiDir); err != nil {
		t.Fatal(err)
	}
	writeSkill(t, filepath.Join(boiDir, "skills"), "deploy.skill.md", "deploy", "registry steps")
	if err := capability.AddEntry(capability.IndexPath(boiDir, capability.KindSkill), capability.KindSkill, capability.Entry{Name: "deploy", Source: "deploy.skill.md", Summary: "deploy", Enabled: true, Tags: []string{"deploy"}, ContextCost: 100}); err != nil {
		t.Fatal(err)
	}
	writeSkill(t, filepath.Join(boiDir, "skills", "team"), "deploy.skill.md", "deploy", "pack steps")
	composition, err := Compose(skillEnvironment(boiDir, skillPackBlock("team", "skills/team", true)), BuiltinPacks(), testSandbox(t), "2.4.0", coreblock.AgentEnvironment{})
	if err != nil {
		t.Fatal(err)
	}
	set, err := selectCapabilities(boiDir, "deploy", coreblock.AgentEnvironment{ToolCalling: true, SkillCalling: true, ContextBytes: 4096}, builtinToolNames, composition.Skills)
	if err != nil || len(set.Skills.Active) != 1 {
		t.Fatalf("skills = %+v %v", set.Skills, err)
	}
	if full, _ := set.Skill("deploy"); full == nil || full.Prompt != "registry steps" {
		t.Fatalf("the pack Skill replaced the registry's: %+v", full)
	}
}

func TestSkillPackRefusals(t *testing.T) {
	outside := t.TempDir()
	writeSkill(t, outside, "secret.skill.md", "secret", "outside .boi")
	cases := map[string]struct {
		prepare func(boiDir string)
		dir     string
		reason  string
	}{
		"missing folder": {func(string) {}, "skills/none", "does not exist"},
		"empty folder":   {func(b string) { _ = os.MkdirAll(filepath.Join(b, "skills", "e"), 0o755) }, "skills/e", "no *.skill.md files"},
		"parent escape":  {func(string) {}, "../outside", "must be a folder inside .boi"},
		"absolute path":  {func(string) {}, outside, "must be a folder inside .boi"},
		"symlinked folder": {func(b string) {
			_ = os.MkdirAll(filepath.Join(b, "skills"), 0o755)
			_ = os.Symlink(outside, filepath.Join(b, "skills", "link"))
		}, "skills/link", "leaves .boi"},
		"symlinked file": {func(b string) {
			_ = os.MkdirAll(filepath.Join(b, "skills", "f"), 0o755)
			_ = os.Symlink(filepath.Join(outside, "secret.skill.md"), filepath.Join(b, "skills", "f", "x.skill.md"))
		}, "skills/f", "not a regular file"},
		"too large": {func(b string) {
			writeSkill(t, filepath.Join(b, "skills", "big"), "big.skill.md", "big", strings.Repeat("x", skill.MaxFileBytes))
		}, "skills/big", "larger than"},
		"same name twice": {func(b string) {
			writeSkill(t, filepath.Join(b, "skills", "d"), "a.skill.md", "same", "a")
			writeSkill(t, filepath.Join(b, "skills", "d"), "b.skill.md", "same", "b")
		}, "skills/d", "defined in both"},
		"bad frontmatter": {func(b string) {
			_ = os.MkdirAll(filepath.Join(b, "skills", "bad"), 0o755)
			_ = os.WriteFile(filepath.Join(b, "skills", "bad", "bad.skill.md"), []byte("---\nname: [unclosed\n---\nbody"), 0o644)
		}, "skills/bad", "bad.skill.md"},
	}
	for name, tc := range cases {
		boiDir := t.TempDir()
		tc.prepare(boiDir)
		optional, err := Compose(skillEnvironment(boiDir, skillPackBlock("team", tc.dir, false)), BuiltinPacks(), testSandbox(t), "2.4.0", coreblock.AgentEnvironment{})
		if err != nil {
			t.Fatalf("%s: an optional Skill pack must never stop composition: %v", name, err)
		}
		if entry := optional.Entries[0]; entry.Status != PackSkipped || !strings.Contains(entry.Reason, tc.reason) || len(optional.Skills) != 0 {
			t.Errorf("%s: entry %+v, want skipped with %q", name, entry, tc.reason)
		}
		if _, err := Compose(skillEnvironment(boiDir, skillPackBlock("team", tc.dir, true)), BuiltinPacks(), testSandbox(t), "2.4.0", coreblock.AgentEnvironment{}); err == nil || !strings.Contains(err.Error(), tc.reason) {
			t.Errorf("%s: a required Skill pack must stop start-up with %q, got %v", name, tc.reason, err)
		}
	}
}

func TestSkillNameCannotBeProvidedByTwoPacks(t *testing.T) {
	boiDir := t.TempDir()
	writeSkill(t, filepath.Join(boiDir, "skills", "one"), "a.skill.md", "shared", "one")
	writeSkill(t, filepath.Join(boiDir, "skills", "two"), "a.skill.md", "shared", "two")
	composition, err := Compose(skillEnvironment(boiDir, skillPackBlock("one", "skills/one", true), skillPackBlock("two", "skills/two", false)), BuiltinPacks(), testSandbox(t), "2.4.0", coreblock.AgentEnvironment{})
	if err != nil {
		t.Fatal(err)
	}
	if second := composition.Entries[1]; second.Status != PackSkipped || !strings.Contains(second.Reason, "already provided by one") || len(composition.Skills) != 1 {
		t.Fatalf("second pack = %+v", second)
	}
}

func TestEnvironmentRejectsAnAmbiguousSkillBlock(t *testing.T) {
	both := environmentOf(block.EnvironmentBlock{ID: "x", MCP: &block.MCPServer{Command: "srv"}, Skills: &block.SkillFolder{Dir: "s"}})
	if err := both.Validate(); err == nil || !strings.Contains(err.Error(), "both an MCP server and a Skill folder") {
		t.Fatalf("both = %v", err)
	}
	empty := environmentOf(block.EnvironmentBlock{ID: "x", Skills: &block.SkillFolder{}})
	if err := empty.Validate(); err == nil || !strings.Contains(err.Error(), "no dir") {
		t.Fatalf("empty = %v", err)
	}
}

func TestReportShowsSkillPackStates(t *testing.T) {
	environment := &block.Environment{SchemaVersion: 1, Name: "skills", Blocks: []block.EnvironmentBlock{
		{ID: "workspace-tools", Required: true},
		skillPackBlock("team", "skills/team", false),
		skillPackBlock("gone", "skills/gone", false),
	}}
	runtime := reportRuntime(t, environment)
	writeSkill(t, filepath.Join(runtime.BoiDir, "skills", "team"), "deploy.skill.md", "deploy", "steps")
	configured := qualifiedFake(t, runtime.BoiDir)
	report, err := runtime.Report(configured)
	if err != nil {
		t.Fatal(err)
	}
	team, gone := blockByID(t, report, "team"), blockByID(t, report, "gone")
	// The fake profile passes no skill_calling probe: read, but not offered.
	if team.State != StateLoaded || team.Kind != "skill-pack" || strings.Join(team.Skills, ",") != "deploy" || !strings.Contains(team.Reason, "Skills") {
		t.Fatalf("team = %+v", team)
	}
	if gone.State != StateDeclared || !strings.Contains(gone.Reason, "does not exist") {
		t.Fatalf("gone = %+v", gone)
	}
	if !report.Ready {
		t.Fatalf("an optional Skill pack must not make the Agent not ready: %+v", report)
	}
}

func TestSkillPackIsActiveWhenTheProviderAllowsSkills(t *testing.T) {
	entry := CompositionEntry{Status: PackActive}
	if skillPackState(entry, true) != StateActive || skillPackState(entry, false) != StateLoaded || skillPackState(CompositionEntry{Status: PackSkipped}, true) != StateDeclared {
		t.Fatal("wrong Skill pack state")
	}
}

func TestPrepareTaskActivatesSkillPackSkills(t *testing.T) {
	root := t.TempDir()
	boiDir := filepath.Join(root, ".boi")
	if err := EnsureCapabilityIndexes(boiDir); err != nil {
		t.Fatal(err)
	}
	writeSkill(t, filepath.Join(boiDir, "skills", "team"), "deploy.skill.md", "deploy", "deploy steps")
	data, _ := json.Marshal(environmentOf(block.EnvironmentBlock{ID: "workspace-tools", Required: true}, skillPackBlock("team", "skills/team", false)))
	if err := os.WriteFile(filepath.Join(boiDir, EnvironmentFile), data, 0o600); err != nil {
		t.Fatal(err)
	}
	runtime := &Runtime{Version: "2.4.0", WorkspaceRoot: root, BoiDir: boiDir, Sandbox: testSandbox(t)}
	agent, err := runtime.BuildAgent(nil)
	if err != nil {
		t.Fatal(err)
	}
	agent.Environment.SkillCalling, agent.Environment.ContextBytes = true, 4096
	capabilities, err := agent.PrepareTask("how do we deploy?")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(capabilities.Skills.Active, "deploy") || len(capabilities.SkillDocs()) != 1 {
		t.Fatalf("the Skill pack's Skill was not activated: %+v", capabilities.Skills)
	}
}
