package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/boi-family/boi-cli/internal/app"
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
