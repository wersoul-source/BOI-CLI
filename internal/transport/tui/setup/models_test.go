package setup

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	// The wizard tests must not reach the network.
	fetchModels = func(string, string) []string { return nil }
}

func TestParseModelIDsDropsNonChatModels(t *testing.T) {
	got := parseModelIDs(strings.NewReader(`{"data":[{"id":"b-chat"},{"id":"text-embedding-3"},{"id":"a-chat"},{"id":""},{"id":"whisper-1"}]}`))
	if strings.Join(got, ",") != "a-chat,b-chat" {
		t.Fatalf("models = %v", got)
	}
	if parseModelIDs(strings.NewReader("not json")) != nil {
		t.Fatal("bad body must give no models")
	}
}

func TestLiveModelsSkipsOtherProtocols(t *testing.T) {
	called := false
	old := fetchModels
	fetchModels = func(string, string) []string { called = true; return []string{"x"} }
	defer func() { fetchModels = old }()
	if liveModels("anthropic", "https://api.anthropic.com/v1", "k") != nil || called {
		t.Fatal("anthropic must not be asked for an OpenAI-style list")
	}
	if got := liveModels("openai", "https://example.test/v1", "k"); len(got) != 1 {
		t.Fatalf("got %v", got)
	}
}

func TestWizardOffersLiveModelsAndStillAllowsTyping(t *testing.T) {
	old := fetchModels
	fetchModels = func(string, string) []string { return []string{"zen-a", "zen-b"} }
	defer func() { fetchModels = old }()

	w := singleWizard()
	selectOther(t, w)
	typeText(w, "https://example.test/v1")
	press(t, w, tea.KeyEnter)
	typeText(w, "key")
	press(t, w, tea.KeyEnter)
	if w.step != stepModel || w.curCustomText || len(w.listItems) != 3 || w.listItems[0] != "zen-a" {
		t.Fatalf("step=%v typed=%v items=%v", w.step, w.curCustomText, w.listItems)
	}
}
