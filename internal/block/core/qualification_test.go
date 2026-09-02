package core

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	llm "github.com/boi-family/boi-cli/internal/provider"
)

type probeProvider struct{ fail string }

func (p probeProvider) Name() string { return "probe" }
func (p probeProvider) Complete(_ context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	prompt := req.Messages[len(req.Messages)-1].Content
	if p.fail != "" && strings.Contains(prompt, "BOI_PROBE "+p.fail) {
		return nil, errors.New("probe failure")
	}
	for _, probe := range defaultProbeCases() {
		if prompt == probe.prompt {
			return &llm.CompletionResponse{Content: probe.expected, Provider: "probe", Model: "model"}, nil
		}
	}
	return &llm.CompletionResponse{Content: "unexpected"}, nil
}
func (p probeProvider) Stream(context.Context, llm.CompletionRequest) (<-chan llm.Token, error) {
	return nil, errors.New("unused")
}

func TestQualifierProducesDeterministicFailClosedProfile(t *testing.T) {
	now := time.Date(2026, 9, 2, 15, 0, 0, 0, time.UTC)
	target := ProviderTarget{Provider: "probe", Model: "model", EndpointClass: "local"}
	profile, err := (Qualifier{Now: func() time.Time { return now }}).Run(context.Background(), target, probeProvider{fail: "authority"})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Status("completion") != CapabilityPassed {
		t.Fatal("completion should pass")
	}
	if profile.Status("authority") != CapabilityFailed || profile.Allows("authority") {
		t.Fatal("failed authority probe must fail closed")
	}
	if profile.Status("missing") != CapabilityUnverified || profile.Allows("missing") {
		t.Fatal("missing capability must be unverified and denied")
	}
	if profile.Status("native_tool_schema") != CapabilityUnsupported {
		t.Fatal("native tool schema must be explicit unsupported")
	}
	if ComposeEnvironment(profile).ToolCalling {
		t.Fatal("failed authority probe must disable Tool Calling environment")
	}
	if ComposeEnvironment(nil).Completion {
		t.Fatal("missing profile must compose an empty environment")
	}
}

func TestCapabilityProfileRoundTripAndNoCredentials(t *testing.T) {
	target := ProviderTarget{Provider: "openai", Model: "model", EndpointClass: "custom"}
	profile, err := (Qualifier{Now: func() time.Time { return time.Unix(1, 0) }}).Run(context.Background(), target, probeProvider{})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "profile.json")
	if err := SaveCapabilityProfile(path, profile); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadCapabilityProfile(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Fingerprint != profile.Fingerprint {
		t.Fatal("fingerprint changed after round trip")
	}
	data, _ := json.Marshal(loaded)
	if strings.Contains(string(data), "api_key") || strings.Contains(string(data), "base_url") {
		t.Fatal("profile contains credential or raw endpoint fields")
	}
}

func TestTargetFingerprintEquivalentForSameInputs(t *testing.T) {
	target := ProviderTarget{Provider: "OpenAI", Model: "m", EndpointClass: "Official"}
	if TargetFingerprint(target, ProbeSuiteVersion) != TargetFingerprint(target, ProbeSuiteVersion) {
		t.Fatal("fingerprint is not deterministic")
	}
}

func TestRepeatedQualificationIsEquivalentIgnoringTimestamp(t *testing.T) {
	target := ProviderTarget{Provider: "probe", Model: "model", EndpointClass: "local"}
	first, err := (Qualifier{Now: func() time.Time { return time.Unix(1, 0) }}).Run(context.Background(), target, probeProvider{})
	if err != nil {
		t.Fatal(err)
	}
	second, err := (Qualifier{Now: func() time.Time { return time.Unix(2, 0) }}).Run(context.Background(), target, probeProvider{})
	if err != nil {
		t.Fatal(err)
	}
	if !first.Equivalent(*second) {
		t.Fatal("unchanged probes should produce an equivalent profile")
	}
}
