package app

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	coreblock "github.com/boi-family/boi-cli/internal/core"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
)

func TestProviderEnvironmentFailsClosedAndRequiresEveryProvider(t *testing.T) {
	boiDir := t.TempDir()
	items := []llmfactory.ConfiguredProvider{{Name: "one", Model: "m1", EndpointClass: "official"}, {Name: "two", Model: "m2", EndpointClass: "custom"}}
	if ProviderEnvironment(boiDir, items).ToolCalling {
		t.Fatal("missing profiles must disable Tool Calling")
	}
	if len(QualifiedProviders(boiDir, items)) != 0 {
		t.Fatal("missing profiles must not enter Router")
	}
	for index, item := range items {
		target := coreblock.ProviderTarget{Provider: item.Name, Model: item.Model, EndpointClass: item.EndpointClass}
		profile := &coreblock.CapabilityProfile{SchemaVersion: 1, ProbeVersion: coreblock.ProbeSuiteVersion, Fingerprint: coreblock.TargetFingerprint(target, coreblock.ProbeSuiteVersion), Target: target, GeneratedAt: time.Now(), Results: []coreblock.CapabilityResult{
			{Capability: "completion", Status: coreblock.CapabilityPassed},
			{Capability: "tool_calling", Status: coreblock.CapabilityPassed},
			{Capability: "tool_observation", Status: coreblock.CapabilityPassed},
			{Capability: "authority", Status: coreblock.CapabilityPassed},
			{Capability: "skill_calling", Status: coreblock.CapabilityPassed},
			{Capability: "context", Status: coreblock.CapabilityPassed, InputBytes: 1024 + index},
		}}
		if err := coreblock.SaveCapabilityProfile(coreblock.CapabilityProfilePath(boiDir, target), profile); err != nil {
			t.Fatal(err)
		}
		if index == 0 && len(QualifiedProviders(boiDir, items)) != 1 {
			t.Fatal("only profiled Provider should enter Router")
		}
	}
	env := ProviderEnvironment(boiDir, items)
	if !env.ToolCalling || !env.SkillCalling || env.ContextBytes != 1024 {
		t.Fatalf("unexpected environment: %#v", env)
	}
	if len(QualifiedProviders(boiDir, items)) != 2 {
		t.Fatal("both passing Providers should enter Router")
	}
}

func TestQualificationProfileIsSharedThroughGlobalDir(t *testing.T) {
	global := t.TempDir()
	t.Setenv("BOI_CONFIG_DIR", global)
	target := coreblock.ProviderTarget{Provider: "openai", Model: "m", EndpointClass: "custom"}
	profile := &coreblock.CapabilityProfile{SchemaVersion: 1, ProbeVersion: coreblock.ProbeSuiteVersion, Fingerprint: coreblock.TargetFingerprint(target, coreblock.ProbeSuiteVersion), Target: target, GeneratedAt: time.Now(), Results: []coreblock.CapabilityResult{{Capability: "completion", Status: coreblock.CapabilityPassed}}}
	first, second := filepath.Join(t.TempDir(), ".boi"), filepath.Join(t.TempDir(), ".boi")
	if _, _, err := LoadProfile(second, target); err == nil {
		t.Fatal("profile found before any qualification")
	}
	if _, err := SaveProfile(first, target, profile); err != nil {
		t.Fatal(err)
	}
	_, path, err := LoadProfile(second, target)
	if err != nil || !strings.HasPrefix(path, global) {
		t.Fatalf("second workspace did not reuse the shared profile: path=%q err=%v", path, err)
	}
}

func TestExpiredQualificationKeepsProviderOut(t *testing.T) {
	boiDir := t.TempDir()
	item := llmfactory.ConfiguredProvider{Name: "old", Model: "m", EndpointClass: "custom"}
	target := coreblock.ProviderTarget{Provider: item.Name, Model: item.Model, EndpointClass: item.EndpointClass}
	save := func(generated time.Time) {
		profile := &coreblock.CapabilityProfile{SchemaVersion: 1, ProbeVersion: coreblock.ProbeSuiteVersion, Fingerprint: coreblock.TargetFingerprint(target, coreblock.ProbeSuiteVersion), Target: target, GeneratedAt: generated, Results: []coreblock.CapabilityResult{{Capability: "completion", Status: coreblock.CapabilityPassed}}}
		if err := coreblock.SaveCapabilityProfile(coreblock.CapabilityProfilePath(boiDir, target), profile); err != nil {
			t.Fatal(err)
		}
	}
	save(time.Now().Add(-coreblock.ProfileMaxAge - time.Hour))
	if len(QualifiedProviders(boiDir, []llmfactory.ConfiguredProvider{item})) != 0 {
		t.Fatal("an expired profile must not let the Provider into the Router")
	}
	if _, _, err := LoadProfile(boiDir, target); err == nil || !strings.Contains(err.Error(), "boi provider qualify old") {
		t.Fatalf("the error must say how to fix it: %v", err)
	}
	save(time.Now().Add(-coreblock.ProfileMaxAge + time.Hour))
	if len(QualifiedProviders(boiDir, []llmfactory.ConfiguredProvider{item})) != 1 {
		t.Fatal("a profile inside the window must still count")
	}
}

func recipeTestProfile(t *testing.T, boiDir string, item llmfactory.ConfiguredProvider, changed ...coreblock.CapabilityResult) {
	t.Helper()
	results := []coreblock.CapabilityResult{
		{Capability: "native_tool_schema", Status: coreblock.CapabilityPassed},
		{Capability: "completion", Status: coreblock.CapabilityPassed},
		{Capability: "tool_calling", Status: coreblock.CapabilityPassed},
		{Capability: "skill_calling", Status: coreblock.CapabilityPassed},
		{Capability: "tool_observation", Status: coreblock.CapabilityPassed},
		{Capability: "authority", Status: coreblock.CapabilityPassed},
		{Capability: "context", Status: coreblock.CapabilityPassed, InputBytes: 2048},
	}
	for _, change := range changed {
		for i := range results {
			if results[i].Capability == change.Capability {
				results[i] = change
			}
		}
	}
	target := coreblock.ProviderTarget{Provider: item.Name, Model: item.Model, EndpointClass: item.EndpointClass}
	profile := &coreblock.CapabilityProfile{SchemaVersion: 1, ProbeVersion: coreblock.ProbeSuiteVersion, Fingerprint: coreblock.TargetFingerprint(target, coreblock.ProbeSuiteVersion), Target: target, GeneratedAt: time.Now(), Results: results}
	if err := coreblock.SaveCapabilityProfile(coreblock.CapabilityProfilePath(boiDir, target), profile); err != nil {
		t.Fatal(err)
	}
}

// A failed safety probe no longer withholds every Tool. The Provider keeps
// its Tool route under the approval floor (ADR-006 D1, P2.5).
func TestAFailedSafetyProbePutsTheAgentUnderTheApprovalFloor(t *testing.T) {
	t.Setenv("BOI_CONFIG_DIR", t.TempDir())
	boiDir := t.TempDir()
	weak := llmfactory.ConfiguredProvider{Name: "weak", Model: "m", EndpointClass: "custom"}
	recipeTestProfile(t, boiDir, weak, coreblock.CapabilityResult{Capability: "tool_observation", Status: coreblock.CapabilityFailed, Detail: "model replied: OBSERVATION_public"})
	env := ProviderEnvironment(boiDir, []llmfactory.ConfiguredProvider{weak})
	if !env.Completion || !env.ToolCalling || !env.NativeTools || !env.ApprovalFloor {
		t.Fatalf("environment = %+v", env)
	}
	if len(QualifiedProviders(boiDir, []llmfactory.ConfiguredProvider{weak})) != 1 {
		t.Fatal("a Provider that completes must enter the Router")
	}
}

// With several Providers in the Router the floor is shared and strictest: one
// Provider that needs it puts the whole Agent under it (D3). Without such a
// Provider nothing changes for anyone.
func TestTheApprovalFloorIsSharedAcrossTheRouter(t *testing.T) {
	t.Setenv("BOI_CONFIG_DIR", t.TempDir())
	boiDir := t.TempDir()
	strong := llmfactory.ConfiguredProvider{Name: "strong", Model: "m", EndpointClass: "custom"}
	weak := llmfactory.ConfiguredProvider{Name: "weak", Model: "m", EndpointClass: "custom"}
	recipeTestProfile(t, boiDir, strong)
	recipeTestProfile(t, boiDir, weak, coreblock.CapabilityResult{Capability: "authority", Status: coreblock.CapabilityFailed})

	alone := ProviderEnvironment(boiDir, []llmfactory.ConfiguredProvider{strong})
	if !alone.ToolCalling || alone.ApprovalFloor || !alone.SkillCalling {
		t.Fatalf("a Provider that passed everything must be unchanged: %+v", alone)
	}
	for name, order := range map[string][]llmfactory.ConfiguredProvider{"weak first": {weak, strong}, "strong first": {strong, weak}} {
		mixed := ProviderEnvironment(boiDir, order)
		if !mixed.ToolCalling || !mixed.ApprovalFloor {
			t.Fatalf("%s: environment = %+v", name, mixed)
		}
		// Skills need the authority probe; no compensation exists for them.
		if mixed.SkillCalling {
			t.Fatalf("%s: Skills are on although one Provider failed the authority probe", name)
		}
	}
	if single, both := RecipeVersion(boiDir, []llmfactory.ConfiguredProvider{strong}), RecipeVersion(boiDir, []llmfactory.ConfiguredProvider{strong, weak}); single == "" || both == "" || single == both || !strings.HasPrefix(both, "multi-") {
		t.Fatalf("recipe versions: %q and %q", single, both)
	}
}

// The two exclusions of ADR-006 still hold: no profile, and completion that
// does not work. Neither gets a floor to hide behind.
func TestExclusionsAreUnchanged(t *testing.T) {
	t.Setenv("BOI_CONFIG_DIR", t.TempDir())
	boiDir := t.TempDir()
	unmeasured := llmfactory.ConfiguredProvider{Name: "unmeasured", Model: "m", EndpointClass: "custom"}
	dead := llmfactory.ConfiguredProvider{Name: "dead", Model: "m", EndpointClass: "custom"}
	recipeTestProfile(t, boiDir, dead, coreblock.CapabilityResult{Capability: "completion", Status: coreblock.CapabilityFailed}, coreblock.CapabilityResult{Capability: "tool_observation", Status: coreblock.CapabilityFailed})
	all := []llmfactory.ConfiguredProvider{unmeasured, dead}
	if got := QualifiedProviders(boiDir, all); len(got) != 0 {
		t.Fatalf("excluded Providers entered the Router: %+v", got)
	}
	if env := ProviderEnvironment(boiDir, all); env != (coreblock.AgentEnvironment{}) {
		t.Fatalf("environment = %+v", env)
	}
}
