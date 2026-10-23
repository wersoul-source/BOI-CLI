package cli

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/boi-family/boi-cli/internal/app"
	coreblock "github.com/boi-family/boi-cli/internal/core"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
	"github.com/boi-family/boi-cli/internal/transport/tui/setup"
)

// modelServer answers GET /v1/models like an OpenAI-compatible local server.
func modelServer(t *testing.T, ids ...string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		items := make([]string, 0, len(ids))
		for _, id := range ids {
			items = append(items, `{"id":"`+id+`"}`)
		}
		_, _ = w.Write([]byte(`{"data":[` + strings.Join(items, ",") + `]}`))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestDetectLocalServersListsChatModelsOnly(t *testing.T) {
	good := modelServer(t, "qwen2.5-coder:7b", "nomic-embed-text", "llama3.2:3b")
	empty := modelServer(t)
	candidates := []localCandidate{
		{"Ollama", good.URL + "/v1"},
		{"Empty", empty.URL + "/v1"},
		{"Closed", "http://127.0.0.1:1/v1"},
		{"Not a model server", good.URL + "/other"},
	}
	found := detectLocalServers(context.Background(), http.DefaultClient, candidates)
	if len(found) != 1 || found[0].Label != "Ollama" || strings.Join(found[0].Models, ",") != "llama3.2:3b,qwen2.5-coder:7b" {
		t.Fatalf("found = %+v", found)
	}
}

// quickstartFixture gives a project folder, isolated settings and stubs for
// everything that would touch the network or the real .env.
type quickstartFixture struct {
	runtime   *app.Runtime
	saved     []setup.ConfiguredProvider
	qualified []time.Duration
	hosted    int
	profile   []coreblock.CapabilityResult
}

func newQuickstartFixture(t *testing.T, candidates []localCandidate) *quickstartFixture {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	t.Setenv("BOI_CONFIG_DIR", t.TempDir())
	for _, name := range []string{"NAME", "API_KEY", "BASE_URL", "MODEL"} {
		t.Setenv("PSC_1_"+name, "")
	}
	runtime, err := app.NewRuntime("test")
	if err != nil {
		t.Fatal(err)
	}
	fixture := &quickstartFixture{runtime: runtime, profile: []coreblock.CapabilityResult{
		{Capability: "completion", Status: coreblock.CapabilityPassed},
		{Capability: "tool_calling", Status: coreblock.CapabilityPassed},
		{Capability: "tool_observation", Status: coreblock.CapabilityPassed},
		{Capability: "authority", Status: coreblock.CapabilityPassed},
	}}
	oldCandidates, oldHosted, oldQualify, oldSave := quickstartCandidates, quickstartHosted, quickstartQualify, quickstartSave
	t.Cleanup(func() {
		quickstartCandidates, quickstartHosted, quickstartQualify, quickstartSave = oldCandidates, oldHosted, oldQualify, oldSave
	})
	quickstartCandidates = candidates
	quickstartHosted = func() error { fixture.hosted++; return nil }
	quickstartSave = func(providers []setup.ConfiguredProvider) (string, string, error) {
		fixture.saved = providers
		t.Setenv("PSC_1_NAME", providers[0].Name)
		t.Setenv("PSC_1_BASE_URL", providers[0].Endpoint)
		t.Setenv("PSC_1_MODEL", providers[0].Model)
		return filepath.Join(root, ".env"), "", nil
	}
	quickstartQualify = func(_ context.Context, runtime *app.Runtime, item llmfactory.ConfiguredProvider, _ io.Writer, timeout time.Duration) error {
		fixture.qualified = append(fixture.qualified, timeout)
		target := coreblock.ProviderTarget{Provider: item.Name, Model: item.Model, EndpointClass: item.EndpointClass}
		profile := &coreblock.CapabilityProfile{SchemaVersion: 1, ProbeVersion: coreblock.ProbeSuiteVersion, Fingerprint: coreblock.TargetFingerprint(target, coreblock.ProbeSuiteVersion), Target: target, GeneratedAt: time.Now(), Results: fixture.profile}
		_, err := app.SaveProfile(runtime.BoiDir, target, profile)
		return err
	}
	return fixture
}

func runQuickstart(t *testing.T, fixture *quickstartFixture, input string) (string, error) {
	t.Helper()
	var out strings.Builder
	err := RunQuickstart(context.Background(), fixture.runtime, bufio.NewReader(strings.NewReader(input)), &out)
	return out.String(), err
}

func TestQuickstartUsesTheLocalModelItFinds(t *testing.T) {
	server := modelServer(t, "qwen2.5-coder:7b")
	fixture := newQuickstartFixture(t, []localCandidate{{"Ollama", server.URL + "/v1"}})
	output, err := runQuickstart(t, fixture, "\n")
	if err != nil {
		t.Fatalf("%v\n%s", err, output)
	}
	if len(fixture.saved) != 1 || fixture.saved[0].Name != "ollama" || fixture.saved[0].Model != "qwen2.5-coder:7b" || fixture.saved[0].Endpoint != server.URL+"/v1" || fixture.saved[0].APIKey != "" {
		t.Fatalf("saved = %+v", fixture.saved)
	}
	if len(fixture.qualified) != 1 || fixture.qualified[0] != coreblock.DefaultProbeTimeout("local") {
		t.Fatalf("a local model must be checked once with the local timeout: %v", fixture.qualified)
	}
	for _, want := range []string{"Ollama: qwen2.5-coder:7b", "Ready.", "every change asks you first", "boi ask"} {
		if !strings.Contains(output, want) {
			t.Errorf("output lacks %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "paid") {
		t.Errorf("a local model must not be described as paid:\n%s", output)
	}
}

func TestQuickstartSaysSoWhenTheModelCannotUseTools(t *testing.T) {
	server := modelServer(t, "tiny:1b")
	fixture := newQuickstartFixture(t, []localCandidate{{"Ollama", server.URL + "/v1"}})
	fixture.profile = []coreblock.CapabilityResult{
		{Capability: "completion", Status: coreblock.CapabilityPassed},
		{Capability: "tool_calling", Status: coreblock.CapabilityFailed, Detail: "no call"},
	}
	output, err := runQuickstart(t, fixture, "1\n")
	if err != nil || !strings.Contains(output, "did not prove it can use tools") || strings.Contains(output, "Ready.") {
		t.Fatalf("err=%v\n%s", err, output)
	}
}

func TestQuickstartOffersHostedProvidersWhenNothingIsRunning(t *testing.T) {
	fixture := newQuickstartFixture(t, nil)
	output, err := runQuickstart(t, fixture, "1\n")
	if err != nil {
		t.Fatal(err)
	}
	if fixture.hosted != 1 || len(fixture.saved) != 0 || len(fixture.qualified) != 0 {
		t.Fatalf("hosted=%d saved=%v qualified=%v", fixture.hosted, fixture.saved, fixture.qualified)
	}
	if !strings.Contains(output, "None found") || !strings.Contains(output, "No model is configured yet") {
		t.Fatalf("output:\n%s", output)
	}
}

func TestQuickstartAcceptsALocalServerAtAnotherAddress(t *testing.T) {
	server := modelServer(t, "mistral:7b")
	fixture := newQuickstartFixture(t, nil)
	output, err := runQuickstart(t, fixture, "2\n"+server.URL+"/v1\n\n")
	if err != nil {
		t.Fatalf("%v\n%s", err, output)
	}
	if len(fixture.saved) != 1 || fixture.saved[0].Model != "mistral:7b" || fixture.saved[0].Endpoint != server.URL+"/v1" {
		t.Fatalf("saved = %+v", fixture.saved)
	}
	if _, err := runQuickstart(t, newQuickstartFixture(t, nil), "2\nnot an address\n"); err == nil {
		t.Fatal("a bad address must be refused")
	}
}

func TestQuickstartGivesUpAfterThreeInvalidAnswers(t *testing.T) {
	fixture := newQuickstartFixture(t, nil)
	if _, err := runQuickstart(t, fixture, "x\ny\nz\n"); err == nil || len(fixture.saved) != 0 {
		t.Fatalf("err=%v saved=%v", err, fixture.saved)
	}
	if _, err := runQuickstart(t, newQuickstartFixture(t, nil), ""); err == nil {
		t.Fatal("the end of input with no choice must stop, not loop")
	}
}

func TestQuickstartSkipsTheChoiceWhenAModelIsAlreadyConfigured(t *testing.T) {
	fixture := newQuickstartFixture(t, nil)
	t.Setenv("PSC_1_NAME", "ollama")
	t.Setenv("PSC_1_BASE_URL", "http://localhost:11434/v1")
	t.Setenv("PSC_1_MODEL", "qwen2.5-coder:7b")
	output, err := runQuickstart(t, fixture, "")
	if err != nil || len(fixture.saved) != 0 || len(fixture.qualified) != 1 || !strings.Contains(output, "Ready.") {
		t.Fatalf("err=%v saved=%v qualified=%v\n%s", err, fixture.saved, fixture.qualified, output)
	}
	// Run again: the model is already checked, nothing is sent.
	output, err = runQuickstart(t, fixture, "")
	if err != nil || len(fixture.qualified) != 1 || !strings.Contains(output, "Already checked.") {
		t.Fatalf("a second run must not repeat the check: err=%v qualified=%v\n%s", err, fixture.qualified, output)
	}
}

func TestAuthFailureHintNamesTheServerOnlyWhenEveryCheckWasUnauthorized(t *testing.T) {
	unauthorized := func(name string) coreblock.CapabilityResult {
		return coreblock.CapabilityResult{Capability: name, Status: coreblock.CapabilityFailed, Detail: "provider request failed: auth, HTTP 401 (check the API key)"}
	}
	all := []coreblock.CapabilityResult{unauthorized("completion"), unauthorized("reasoning")}
	hint := authFailureHint(all, "api.openai.com")
	if !strings.Contains(hint, "api.openai.com") || !strings.Contains(hint, "401") || !strings.Contains(hint, "only") {
		t.Fatalf("hint = %q", hint)
	}
	if !strings.Contains(authFailureHint(all, ""), "the provider") {
		t.Fatal("without a host the hint must still read well")
	}
	mixed := append([]coreblock.CapabilityResult{{Capability: "context", Status: coreblock.CapabilityPassed}}, all...)
	other := []coreblock.CapabilityResult{unauthorized("completion"), {Capability: "reasoning", Status: coreblock.CapabilityFailed, Detail: "timeout"}}
	for name, results := range map[string][]coreblock.CapabilityResult{"one passed": mixed, "another cause": other, "none": nil} {
		if hint := authFailureHint(results, "x"); hint != "" {
			t.Errorf("%s: unexpected hint %q", name, hint)
		}
	}
}
