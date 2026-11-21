package core

import (
	"strings"
	"testing"
	"time"
)

func recipeProfile(results ...CapabilityResult) *CapabilityProfile {
	target := ProviderTarget{Provider: "p", Model: "m", EndpointClass: "custom"}
	return &CapabilityProfile{SchemaVersion: CapabilityProfileSchemaVersion, ProbeVersion: ProbeSuiteVersion, Fingerprint: TargetFingerprint(target, ProbeSuiteVersion), Target: target, GeneratedAt: time.Now(), Results: results}
}

func res(capability string, status CapabilityStatus) CapabilityResult {
	return CapabilityResult{Capability: capability, Status: status}
}

func allPassed() []CapabilityResult {
	return []CapabilityResult{
		res("native_tool_schema", CapabilityPassed), res("completion", CapabilityPassed), res("reasoning", CapabilityPassed),
		res("tool_calling", CapabilityPassed), res("skill_calling", CapabilityPassed), res("tool_observation", CapabilityPassed),
		res("authority", CapabilityPassed), res("context", CapabilityPassed),
	}
}

func with(results []CapabilityResult, changed ...CapabilityResult) []CapabilityResult {
	out := append([]CapabilityResult{}, results...)
	for _, change := range changed {
		for i := range out {
			if out[i].Capability == change.Capability {
				out[i] = change
			}
		}
	}
	return out
}

// The Recipe for each kind of model the zoo reproduces.
func TestComposeRecipeByModelKind(t *testing.T) {
	failedObservation := CapabilityResult{Capability: "tool_observation", Status: CapabilityFailed, Detail: `expected "OBSERVATION_IGNORED" not found; model replied: OBSERVATION_public`}
	cases := []struct {
		name         string
		results      []CapabilityResult
		verified     VerifiedCompensations
		tools        CapabilityState
		protocol     string
		floor        string
		compensation string
		usable       bool
	}{
		{"native model", allPassed(), VerifiedCompensations{}, StateNative, ProtocolNative, FloorNone, "", true},
		{"dialect model", with(allPassed(), CapabilityResult{Capability: "native_tool_schema", Status: CapabilityPassed, Assisted: "dialect:dsml"}, res("tool_calling", CapabilityFailed)), VerifiedCompensations{}, StateAssisted, "dialect:dsml", FloorNone, "dialect:dsml", true},
		{"envelope only", with(allPassed(), res("native_tool_schema", CapabilityUnsupported)), VerifiedCompensations{}, StateAssisted, ProtocolEnvelope, FloorNone, CompensationEnvelope, true},
		{"no protocol works", with(allPassed(), res("native_tool_schema", CapabilityFailed), res("tool_calling", CapabilityFailed)), VerifiedCompensations{}, StateBlocked, ProtocolNone, FloorNone, "", true},
		{"corrupted token, floor not verified", with(allPassed(), failedObservation), VerifiedCompensations{}, StateUnverified, ProtocolNative, FloorConfirm, CompensationApprovalFloor, true},
		{"corrupted token, floor verified", with(allPassed(), failedObservation), VerifiedCompensations{ApprovalFloor: true}, StateAssisted, ProtocolNative, FloorConfirm, CompensationApprovalFloor, true},
		{"dialect under the floor", with(allPassed(), CapabilityResult{Capability: "native_tool_schema", Status: CapabilityPassed, Assisted: "dialect:hermes"}, res("authority", CapabilityFailed)), VerifiedCompensations{ApprovalFloor: true}, StateAssisted, "dialect:hermes", FloorConfirm, "dialect:hermes+approval-floor", true},
		{"completion fails", with(allPassed(), res("completion", CapabilityFailed)), VerifiedCompensations{ApprovalFloor: true}, StateBlocked, ProtocolNone, FloorNone, "", false},
		{"safety probes never ran", []CapabilityResult{res("completion", CapabilityPassed), res("native_tool_schema", CapabilityPassed)}, VerifiedCompensations{ApprovalFloor: true}, StateUnverified, ProtocolNative, FloorNone, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recipe := ComposeRecipe(recipeProfile(tc.results...), tc.verified)
			tools, _ := recipe.Capability(RecipeTools)
			if tools.State != tc.tools || recipe.ToolProtocol != tc.protocol || recipe.ApprovalFloor != tc.floor || tools.Compensation != tc.compensation || recipe.Usable() != tc.usable {
				t.Fatalf("tools=%s protocol=%s floor=%s compensation=%q usable=%v\nwant tools=%s protocol=%s floor=%s compensation=%q usable=%v\nevidence: %s",
					tools.State, recipe.ToolProtocol, recipe.ApprovalFloor, tools.Compensation, recipe.Usable(),
					tc.tools, tc.protocol, tc.floor, tc.compensation, tc.usable, tools.Evidence)
			}
			if tools.State != StateNative && tools.Evidence == "" {
				t.Fatal("a capability that is not NATIVE must say why")
			}
		})
	}
}

// The probe's own words reach the Recipe, so the gap can be explained.
func TestRecipeKeepsTheProbeEvidence(t *testing.T) {
	recipe := ComposeRecipe(recipeProfile(with(allPassed(), CapabilityResult{Capability: "tool_observation", Status: CapabilityFailed, Detail: "model replied: OBSERVATION_public"})...), VerifiedCompensations{})
	tools, _ := recipe.Capability(RecipeTools)
	if !strings.Contains(tools.Evidence, "OBSERVATION_public") || !strings.Contains(tools.Evidence, "not verified in this build") {
		t.Fatalf("evidence = %q", tools.Evidence)
	}
	if probe, ok := recipe.Capability("tool_observation"); !ok || probe.State != StateBlocked {
		t.Fatalf("the probe itself is not listed as blocked: %+v", probe)
	}
}

// Without a measured profile nothing is usable, and nothing is BLOCKED:
// there is no evidence either way.
func TestRecipeWithoutAProfile(t *testing.T) {
	broken := recipeProfile(allPassed()...)
	broken.Fingerprint = "tampered"
	for name, profile := range map[string]*CapabilityProfile{"nil": nil, "invalid": broken} {
		recipe := ComposeRecipe(profile, VerifiedCompensations{ApprovalFloor: true})
		if recipe.Usable() || recipe.ToolProtocol != ProtocolNone {
			t.Fatalf("%s profile produced a usable Recipe: %+v", name, recipe)
		}
		for _, capability := range recipe.Capabilities {
			if capability.State != StateUnverified {
				t.Fatalf("%s profile: %s is %s", name, capability.Name, capability.State)
			}
		}
	}
}

var everyStatus = []CapabilityStatus{CapabilityPassed, CapabilityFailed, CapabilityUnsupported, CapabilityUnverified, CapabilityInconclusive}

// forEveryProfile calls fn with every combination of statuses for the six
// probes that decide completion, Tools and Skills, each with and without the
// native probe being a dialect recovery.
func forEveryProfile(fn func(profile *CapabilityProfile)) {
	probes := []string{"completion", "native_tool_schema", "tool_calling", "tool_observation", "authority", "skill_calling"}
	index := make([]int, len(probes))
	for {
		for _, assisted := range []string{"", "dialect:dsml"} {
			results := make([]CapabilityResult, len(probes))
			for i, probe := range probes {
				results[i] = res(probe, everyStatus[index[i]])
				if probe == "native_tool_schema" && results[i].Status == CapabilityPassed {
					results[i].Assisted = assisted
				}
			}
			fn(recipeProfile(results...))
		}
		position := 0
		for position < len(index) {
			index[position]++
			if index[position] < len(everyStatus) {
				break
			}
			index[position] = 0
			position++
		}
		if position == len(index) {
			return
		}
	}
}

// With no compensation verified, a Recipe allows exactly what today's
// ComposeEnvironment allows, for every possible profile. Driving the Runtime
// from the Recipe (P2.5) therefore changes nothing until a compensation is
// proven.
func TestRecipeMatchesTodaysEnvironmentUntilACompensationIsVerified(t *testing.T) {
	count := 0
	forEveryProfile(func(profile *CapabilityProfile) {
		count++
		env, recipe := ComposeEnvironment(profile), ComposeRecipe(profile, VerifiedCompensations{})
		if recipe.Usable() != env.Completion || recipe.State(RecipeTools).Usable() != env.ToolCalling || recipe.State(RecipeSkills).Usable() != env.SkillCalling {
			t.Fatalf("recipe (usable=%v tools=%s skills=%s) differs from environment %+v for %+v", recipe.Usable(), recipe.State(RecipeTools), recipe.State(RecipeSkills), env, profile.Results)
		}
		if (recipe.ToolProtocol == ProtocolNative || strings.HasPrefix(recipe.ToolProtocol, ProtocolDialectPrefix)) != env.NativeTools && env.ToolCalling {
			t.Fatalf("protocol %s differs from NativeTools=%v for %+v", recipe.ToolProtocol, env.NativeTools, profile.Results)
		}
	})
	if count != 2*5*5*5*5*5*5 {
		t.Fatalf("checked %d profiles", count)
	}
}

// What a verified approval floor adds, and what it can never add (ADR-006).
func TestTheApprovalFloorOnlyOpensWhatItShould(t *testing.T) {
	opened := 0
	forEveryProfile(func(profile *CapabilityProfile) {
		before, after := ComposeRecipe(profile, VerifiedCompensations{}), ComposeRecipe(profile, VerifiedCompensations{ApprovalFloor: true})
		tools, _ := after.Capability(RecipeTools)
		if before.State(RecipeTools).Usable() && !tools.State.Usable() {
			t.Fatalf("verifying a compensation closed a route: %+v", profile.Results)
		}
		// Nothing else moves.
		if before.Usable() != after.Usable() || before.State(RecipeSkills) != after.State(RecipeSkills) || before.ToolProtocol != after.ToolProtocol || before.ApprovalFloor != after.ApprovalFloor {
			t.Fatalf("the floor changed more than the Tool state: %+v", profile.Results)
		}
		if !before.State(RecipeTools).Usable() && tools.State.Usable() {
			opened++
			// A route opened by the floor is assisted, floored, has a working
			// protocol, and rests on completion.
			if tools.State != StateAssisted || after.ApprovalFloor != FloorConfirm || after.ToolProtocol == ProtocolNone || !after.Usable() || !strings.Contains(tools.Compensation, CompensationApprovalFloor) {
				t.Fatalf("the floor opened a route it should not: %+v\n%+v", after, profile.Results)
			}
		}
		// A capability that needed help is never reported as the model's own.
		if tools.State == StateNative && (tools.Compensation != "" || after.ApprovalFloor != FloorNone || after.ToolProtocol != ProtocolNative) {
			t.Fatalf("an assisted route reads as NATIVE: %+v", after)
		}
	})
	if opened == 0 {
		t.Fatal("the approval floor opened nothing; the test does not exercise it")
	}
}

// The version names the decisions: equal Recipes share it, and any changed
// decision changes it.
func TestRecipeVersionFollowsItsContent(t *testing.T) {
	profile := recipeProfile(allPassed()...)
	first, again := ComposeRecipe(profile, VerifiedCompensations{}), ComposeRecipe(profile, VerifiedCompensations{})
	if first.Version == "" || first.Version != again.Version {
		t.Fatalf("versions %q and %q", first.Version, again.Version)
	}
	floored := ComposeRecipe(recipeProfile(with(allPassed(), res("authority", CapabilityFailed))...), VerifiedCompensations{})
	verified := ComposeRecipe(recipeProfile(with(allPassed(), res("authority", CapabilityFailed))...), VerifiedCompensations{ApprovalFloor: true})
	if floored.Version == first.Version || verified.Version == floored.Version {
		t.Fatalf("different decisions share a version: %s %s %s", first.Version, floored.Version, verified.Version)
	}
}
