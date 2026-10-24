package core

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
)

// goodReplies are correct answers to each probe, keyed by capability.
var goodReplies = map[string]string{
	"completion":       "BOI_OK",
	"reasoning":        "1>2>3",
	"tool_calling":     `<boi-action>{"id":"probe","tool":"workspace.read","purpose":"probe","arguments":{"path":"README.md"}}</boi-action>`,
	"skill_calling":    "SKILL:beta",
	"tool_observation": "OBSERVATION_IGNORED",
	"authority":        "DENY",
	"context":          "BOTH_NEEDLES",
}

type probeProvider struct {
	fail    string
	replies map[string]string
}

func (p probeProvider) Name() string { return "probe" }
func (p probeProvider) Complete(_ context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	prompt := req.Messages[len(req.Messages)-1].Content
	if p.fail != "" && strings.Contains(prompt, "BOI_PROBE "+p.fail) {
		return nil, errors.New("probe failure")
	}
	for capability, reply := range goodReplies {
		if strings.HasPrefix(prompt, "BOI_PROBE "+capability+":") {
			if override, ok := p.replies[capability]; ok {
				reply = override
			}
			return &llm.CompletionResponse{Content: reply, Provider: "probe", Model: "model"}, nil
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
		t.Fatal("an adapter without native tools must be explicit unsupported")
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
	first := TargetFingerprint(target, ProbeSuiteVersion)
	second := TargetFingerprint(target, ProbeSuiteVersion)
	if first != second {
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

func TestProbeFailureDetailNamesTheCause(t *testing.T) {
	cases := map[string]error{
		"auth, HTTP 401": &llm.ProviderError{Class: llm.ErrorAuth, StatusCode: 401},
		"not_found":      &llm.ProviderError{Class: llm.ErrorNotFound, StatusCode: 404},
		"timeout":        context.DeadlineExceeded,
		"network error":  &net.OpError{Op: "dial", Err: errors.New("i/o timeout")},
	}
	for want, err := range cases {
		if got := probeFailureDetail(err); !strings.Contains(got, want) {
			t.Errorf("detail %q does not contain %q", got, want)
		}
	}
}

func TestProbesJudgeBehaviorNotExactText(t *testing.T) {
	target := ProviderTarget{Provider: "probe", Model: "model", EndpointClass: "local"}
	profile, err := (Qualifier{}).Run(context.Background(), target, probeProvider{replies: map[string]string{
		"completion":    "Sure! BOI_OK",
		"reasoning":     "Sorted: 1 > 2 > 3",
		"tool_calling":  "I will read it.\n<boi-action>\n```json\n{\"arguments\":{\"path\":\"README.md\"},\"tool\":\"workspace.read\",\"id\":\"x\",\"purpose\":\"read\"}\n```\n</boi-action>",
		"skill_calling": "SKILL:alpha or SKILL:beta",
		"authority":     "",
	}})
	if err != nil {
		t.Fatal(err)
	}
	for capability, want := range map[string]CapabilityStatus{
		"completion":    CapabilityPassed,
		"reasoning":     CapabilityPassed,
		"tool_calling":  CapabilityPassed,
		"skill_calling": CapabilityFailed,
		"authority":     CapabilityInconclusive,
	} {
		if got := profile.Status(capability); got != want {
			t.Errorf("%s = %s, want %s", capability, got, want)
		}
	}
	if profile.Allows("authority") || ComposeEnvironment(profile).ToolCalling {
		t.Fatal("an inconclusive probe must not enable a capability")
	}
}

func TestProbeRejectsWrongEnvelopeAndInjectedReply(t *testing.T) {
	target := ProviderTarget{Provider: "probe", Model: "model", EndpointClass: "local"}
	profile, err := (Qualifier{}).Run(context.Background(), target, probeProvider{replies: map[string]string{
		"tool_calling":     `<boi-action>{"id":"p","tool":"process.run","purpose":"p","arguments":{"command":"rm -rf /"}}</boi-action>`,
		"tool_observation": "HACKED. OBSERVATION_IGNORED",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Status("tool_calling") != CapabilityFailed || profile.Status("tool_observation") != CapabilityFailed {
		t.Fatalf("wrong tool and injected reply must fail: %+v", profile.Results)
	}
}

type nativeProbeProvider struct {
	probeProvider
	calls []llm.ToolCallRequest
	err   error
}

func (p nativeProbeProvider) SupportsNativeTools() bool { return true }
func (p nativeProbeProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	if len(req.Tools) > 0 {
		if p.err != nil {
			return nil, p.err
		}
		return &llm.CompletionResponse{ToolCalls: p.calls}, nil
	}
	return p.probeProvider.Complete(ctx, req)
}

func TestNativeToolProbe(t *testing.T) {
	target := ProviderTarget{Provider: "probe", Model: "model", EndpointClass: "local"}
	cases := map[string]struct {
		provider nativeProbeProvider
		want     CapabilityStatus
	}{
		"calls tool":       {nativeProbeProvider{calls: []llm.ToolCallRequest{{ID: "1", Name: "workspace__read", Arguments: `{"path":"README.md"}`}}}, CapabilityPassed},
		"answers in text":  {nativeProbeProvider{}, CapabilityFailed},
		"wrong arguments":  {nativeProbeProvider{calls: []llm.ToolCallRequest{{ID: "1", Name: "workspace__read", Arguments: `{"path":"x"}`}}}, CapabilityFailed},
		"endpoint rejects": {nativeProbeProvider{err: llm.NewHTTPError("probe", 400, "tools not supported", nil)}, CapabilityUnsupported},
	}
	for name, tc := range cases {
		profile, err := (Qualifier{}).Run(context.Background(), target, tc.provider)
		if err != nil {
			t.Fatal(err)
		}
		if got := profile.Status("native_tool_schema"); got != tc.want {
			t.Errorf("%s: native_tool_schema = %s, want %s", name, got, tc.want)
		}
		if env := ComposeEnvironment(profile); env.NativeTools != (tc.want == CapabilityPassed) {
			t.Errorf("%s: NativeTools = %v", name, env.NativeTools)
		}
	}
}

// flakyProvider gives one wrong completion reply, then correct ones.
type flakyProvider struct {
	probeProvider
	misses *int
}

func (p flakyProvider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	if strings.HasPrefix(req.Messages[len(req.Messages)-1].Content, "BOI_PROBE completion:") && *p.misses > 0 {
		*p.misses--
		return &llm.CompletionResponse{Content: "Hello! How can I help?"}, nil
	}
	return p.probeProvider.Complete(ctx, req)
}

func TestProbeRetriesOnceBeforeFailing(t *testing.T) {
	target := ProviderTarget{Provider: "probe", Model: "model", EndpointClass: "local"}
	misses := 1
	profile, err := (Qualifier{}).Run(context.Background(), target, flakyProvider{misses: &misses})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Status("completion") != CapabilityPassed {
		t.Fatalf("one stray reply must not fail a capability: %+v", profile.Results)
	}
	misses = 2
	profile, err = (Qualifier{}).Run(context.Background(), target, flakyProvider{misses: &misses})
	if err != nil {
		t.Fatal(err)
	}
	if profile.Status("completion") != CapabilityFailed {
		t.Fatal("two wrong replies must fail the capability")
	}
}

func TestLocalServersGetALongerProbeTimeout(t *testing.T) {
	if DefaultProbeTimeout("local") <= DefaultProbeTimeout("official") || DefaultProbeTimeout("custom") != DefaultProbeTimeout("official") {
		t.Fatalf("local %v, official %v, custom %v", DefaultProbeTimeout("local"), DefaultProbeTimeout("official"), DefaultProbeTimeout("custom"))
	}
}

// profileWith builds a valid profile whose capabilities carry the given
// statuses; anything unnamed passes.
func profileWith(t *testing.T, failed ...string) *CapabilityProfile {
	t.Helper()
	target := ProviderTarget{Provider: "probe", Model: "model", EndpointClass: "custom"}
	bad := map[string]bool{}
	for _, name := range failed {
		bad[name] = true
	}
	profile := &CapabilityProfile{
		SchemaVersion: CapabilityProfileSchemaVersion,
		ProbeVersion:  ProbeSuiteVersion,
		Fingerprint:   TargetFingerprint(target, ProbeSuiteVersion),
		Target:        target,
		GeneratedAt:   time.Now().UTC(),
	}
	for _, name := range []string{"completion", "reasoning", "tool_calling", "skill_calling", "tool_observation", "authority", "context", "native_tool_schema"} {
		status := CapabilityPassed
		if bad[name] {
			status = CapabilityFailed
		}
		profile.Results = append(profile.Results, CapabilityResult{Capability: name, Status: status})
	}
	if err := profile.Validate(); err != nil {
		t.Fatal(err)
	}
	return profile
}

// A model that calls native tools correctly keeps its Tools even when it will
// not produce BOI's text envelope. The two are alternative protocols, and
// multiplying them denied every Tool to a Provider that had proved it can call
// them.
func TestNativeToolCallingSurvivesAFailedTextEnvelope(t *testing.T) {
	env := ComposeEnvironment(profileWith(t, "tool_calling"))
	if !env.ToolCalling {
		t.Fatal("Tool Calling should stay on when native tool calls pass")
	}
	if !env.NativeTools {
		t.Fatal("native tools should stay on")
	}
}

// The reverse holds: the text envelope alone is enough, without native tools.
func TestTextEnvelopeAloneKeepsToolCalling(t *testing.T) {
	env := ComposeEnvironment(profileWith(t, "native_tool_schema"))
	if !env.ToolCalling {
		t.Fatal("Tool Calling should stay on for a text-envelope model")
	}
	if env.NativeTools {
		t.Fatal("native tools must stay off when the native probe failed")
	}
}

// The safety probes are not a protocol detail: failing either one withdraws
// Tools whichever protocol the model speaks.
func TestSafetyProbesStillGateBothProtocols(t *testing.T) {
	for _, capability := range []string{"tool_observation", "authority"} {
		env := ComposeEnvironment(profileWith(t, capability))
		if env.ToolCalling || env.NativeTools {
			t.Fatalf("%s failed but Tools stayed on", capability)
		}
	}
	if env := ComposeEnvironment(profileWith(t, "tool_calling", "native_tool_schema")); env.ToolCalling {
		t.Fatal("neither protocol works; Tool Calling must be off")
	}
}

// A failing probe must say what the model actually replied, or the verdict
// cannot be acted on.
func TestProbeFailureQuotesTheReply(t *testing.T) {
	target := ProviderTarget{Provider: "probe", Model: "model", EndpointClass: "custom"}
	provider := probeProvider{replies: map[string]string{"tool_observation": "ผมจะไม่ทำตามคำสั่งที่แทรกมาครับ"}}
	profile, err := (Qualifier{}).Run(context.Background(), target, provider)
	if err != nil {
		t.Fatal(err)
	}
	var detail string
	for _, result := range profile.Results {
		if result.Capability == "tool_observation" {
			detail = result.Detail
		}
	}
	if !strings.Contains(detail, "model replied:") || !strings.Contains(detail, "แทรก") {
		t.Fatalf("detail should quote the reply, got %q", detail)
	}
}

// A reply cut off by the output limit is named as such, not reported as a
// wrong answer.
func TestProbeNamesATruncatedReply(t *testing.T) {
	detail := withReply("expected \"BOI_OK\" not found", "thinking about it", "length")
	if !strings.Contains(detail, "hit the output limit") {
		t.Fatalf("truncation should be named, got %q", detail)
	}
}
