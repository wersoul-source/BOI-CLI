package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/app"
	coreblock "github.com/boi-family/boi-cli/internal/core"
	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
)

func TestEnvPrintsEachProviderGap(t *testing.T) {
	var out bytes.Buffer
	printProviderChecks(&out, []app.ProviderCheck{
		{Provider: "maxplus", Model: "deepseek-v4.1-flash", Profile: app.ProfileMeasured, Gaps: []app.CapabilityGap{{Capability: "tool_observation", Status: "failed", Detail: "model replied: OBSERVATION_public"}}},
		{Provider: "fresh", Model: "m", Profile: app.ProfileMissing, ProfileError: "not qualified yet; run 'boi provider qualify fresh'"},
	})
	text := out.String()
	for _, want := range []string{"maxplus/deepseek-v4.1-flash   profile: measured   tools: no", "tool_observation", "OBSERVATION_public", "fresh/m   profile: missing", "boi provider qualify fresh"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
}

// refusingToolsProvider rejects tool definitions with HTTP 400, so the native
// probe is "unsupported", and answers every other probe badly.
type refusingToolsProvider struct{}

func (refusingToolsProvider) Name() string              { return "refusing" }
func (refusingToolsProvider) SupportsNativeTools() bool { return true }
func (refusingToolsProvider) Complete(_ context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	if len(req.Tools) > 0 {
		return nil, &llm.ProviderError{Provider: "refusing", StatusCode: 400, Class: llm.ErrorInvalidRequest, Message: "tools not supported"}
	}
	return &llm.CompletionResponse{Content: "no"}, nil
}

// An "unsupported" result is printed with its reason. Only failures used to
// carry one, so a refused tool definition looked the same as no reason at all.
func TestQualifyPrintsTheReasonForEveryResult(t *testing.T) {
	t.Setenv("BOI_CONFIG_DIR", t.TempDir())
	runtime := &app.Runtime{BoiDir: t.TempDir()}
	item := llmfactory.ConfiguredProvider{Name: "refusing", Model: "m", EndpointClass: "custom", Provider: refusingToolsProvider{}}
	var out bytes.Buffer
	if err := qualifyProvider(context.Background(), runtime, item, &out, 0); err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.Contains(line, "native_tool_schema") {
			if !strings.Contains(line, "unsupported") || !strings.Contains(line, "endpoint rejected tool definitions") {
				t.Fatalf("native line lacks its reason: %q", line)
			}
			return
		}
	}
	t.Fatalf("no native_tool_schema line in:\n%s", out.String())
}

// A Provider whose Tools work only through a recovered dialect says so: "tools:
// yes (assisted)" and the capability with its compensation, never a bare yes.
func TestEnvMarksAssistedTools(t *testing.T) {
	var out bytes.Buffer
	printProviderChecks(&out, []app.ProviderCheck{
		{Provider: "maxplus", Model: "m", Profile: app.ProfileMeasured, ToolCalling: true, Assisted: []app.CapabilityGap{{Capability: "native_tool_schema", Status: "assisted (dialect:dsml)", Detail: "recovered from dsml markup"}}},
	})
	text := out.String()
	for _, want := range []string{"tools: yes (assisted)", "native_tool_schema", "dialect:dsml"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
}

// boi env says in plain words that a Provider keeps its Tools under the
// approval floor, and what that means for a run that cannot ask.
func TestEnvShowsTheApprovalFloor(t *testing.T) {
	var out bytes.Buffer
	printReport(&out, &app.AgentReport{ToolCalling: true, NativeTools: true, ApprovalFloor: true, Providers: []string{"maxplus"}, ProviderChecks: []app.ProviderCheck{
		{Provider: "maxplus", Model: "m", Profile: app.ProfileMeasured, ToolCalling: true, ToolState: "ASSISTED", ToolCompensation: "approval-floor", ToolProtocol: "native", ApprovalFloor: true,
			Gaps: []app.CapabilityGap{{Capability: "tool_observation", Status: "failed", Detail: "model replied: OBSERVATION_public"}}},
		{Provider: "strong", Model: "m", Profile: app.ProfileMeasured, ToolCalling: true, ToolState: "NATIVE", ToolProtocol: "native"},
		{Provider: "nothing", Model: "m", Profile: app.ProfileMeasured, ToolState: "BLOCKED"},
	}})
	text := out.String()
	for _, want := range []string{
		"every Tool call needs your approval, because maxplus did not pass a safety check",
		"'boi ask'",
		"maxplus/m   profile: measured   tools: yes (assisted: approval-floor)",
		"strong/m   profile: measured   tools: yes\n",
		"nothing/m   profile: measured   tools: no (blocked)",
		"OBSERVATION_public",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
}

// After qualify the person reads what BOI will do with the model, not only
// which probes failed.
func TestQualifyPrintsTheRecipe(t *testing.T) {
	passed := func(name string) coreblock.CapabilityResult {
		return coreblock.CapabilityResult{Capability: name, Status: coreblock.CapabilityPassed}
	}
	target := coreblock.ProviderTarget{Provider: "p", Model: "m", EndpointClass: "custom"}
	profile := &coreblock.CapabilityProfile{SchemaVersion: 1, ProbeVersion: coreblock.ProbeSuiteVersion, Fingerprint: coreblock.TargetFingerprint(target, coreblock.ProbeSuiteVersion), Target: target, GeneratedAt: time.Now(), Results: []coreblock.CapabilityResult{
		passed("native_tool_schema"), passed("completion"), passed("authority"),
		{Capability: "tool_observation", Status: coreblock.CapabilityFailed, Detail: "model replied: OBSERVATION_public"},
	}}
	var out bytes.Buffer
	printRecipe(&out, app.RecipeForProfile(profile))
	for _, want := range []string{"Tools: ASSISTED (approval-floor)", "protocol: native", "every Tool call will need your approval", "OBSERVATION_public"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in:\n%s", want, out.String())
		}
	}
	out.Reset()
	dead := *profile
	dead.Results = []coreblock.CapabilityResult{{Capability: "completion", Status: coreblock.CapabilityFailed}}
	printRecipe(&out, app.RecipeForProfile(&dead))
	if !strings.Contains(out.String(), "Not usable") {
		t.Errorf("a dead endpoint must be called not usable:\n%s", out.String())
	}
}
