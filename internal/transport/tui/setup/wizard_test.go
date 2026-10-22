package setup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	registry "github.com/boi-family/boi-cli/internal/service/provider/catalog"
)

func press(t *testing.T, m *WizardModel, keys ...tea.KeyType) {
	t.Helper()
	for _, key := range keys {
		m.Update(tea.KeyMsg{Type: key})
	}
}

func singleWizard() *WizardModel {
	w := NewWizard(registry.LoadEmbedded())
	w.single, w.totalCount = true, 1
	w.buildProviderList()
	w.listMode = "provider"
	w.step = stepSelect
	return w
}

func selectOther(t *testing.T, w *WizardModel) {
	t.Helper()
	for range w.listItems[1:] {
		press(t, w, tea.KeyDown)
	}
	press(t, w, tea.KeyEnter)
}

func TestCustomEndpointCarriesItsAddressToTheSavedProvider(t *testing.T) {
	w := singleWizard()
	selectOther(t, w)
	if w.step != stepEndpoint || !w.curCustom {
		t.Fatalf("Custom must ask for an address first: step=%v custom=%v", w.step, w.curCustom)
	}
	for _, bad := range []string{"", "nope", "ftp://host/v1", "https://"} {
		w.textInput.SetValue(bad)
		press(t, w, tea.KeyEnter)
		if w.step != stepEndpoint || w.errorMsg == "" {
			t.Fatalf("address %q was accepted", bad)
		}
	}
	w.textInput.SetValue("https://api.cloudflare.com/client/v4/accounts/abc/ai/v1/")
	press(t, w, tea.KeyEnter)
	if w.step != stepAPIKey || w.curBaseURL != "https://api.cloudflare.com/client/v4/accounts/abc/ai/v1" {
		t.Fatalf("step=%v base=%q", w.step, w.curBaseURL)
	}
	w.textInput.SetValue("secret-token")
	press(t, w, tea.KeyEnter)
	if w.step != stepModel || !w.curCustomText {
		t.Fatalf("a custom endpoint needs a typed model name: step=%v", w.step)
	}
	w.textInput.SetValue("@cf/qwen/qwen2.5-coder-32b-instruct")
	press(t, w, tea.KeyEnter)
	if !w.done || len(w.providers) != 1 {
		t.Fatalf("done=%v providers=%v", w.done, w.providers)
	}
	got := w.providers[0]
	// It is built by the OpenAI-compatible adapter, so its name must be one the
	// factory knows; "other" produced a provider that was silently dropped.
	if got.Name != "openai" || got.Endpoint != "https://api.cloudflare.com/client/v4/accounts/abc/ai/v1" || got.APIKey != "secret-token" || got.Model != "@cf/qwen/qwen2.5-coder-32b-instruct" {
		t.Fatalf("provider = %+v", got)
	}
}

func TestListedProviderKeepsItsCatalogEndpoint(t *testing.T) {
	w := singleWizard()
	press(t, w, tea.KeyEnter)
	if w.step != stepAPIKey || w.curCustom || w.curName == "" || w.curBaseURL == "" {
		t.Fatalf("step=%v name=%q base=%q custom=%v", w.step, w.curName, w.curBaseURL, w.curCustom)
	}
}

func TestEscapeWalksBackFromTheEndpointAndCancelsAtTheListInSingleMode(t *testing.T) {
	w := singleWizard()
	selectOther(t, w)
	press(t, w, tea.KeyEsc)
	if w.step != stepSelect {
		t.Fatalf("Esc at the address must return to the list, step=%v", w.step)
	}
	selectOther(t, w)
	w.textInput.SetValue("http://localhost:8080/v1")
	press(t, w, tea.KeyEnter)
	press(t, w, tea.KeyEsc)
	if w.step != stepEndpoint || w.textInput.Value() != "http://localhost:8080/v1" {
		t.Fatalf("Esc at the key must return to the address with it kept: step=%v value=%q", w.step, w.textInput.Value())
	}
	press(t, w, tea.KeyEsc, tea.KeyEsc)
	if !w.done || !w.cancelled {
		t.Fatalf("Esc at the list of a single-provider wizard must cancel: done=%v cancelled=%v", w.done, w.cancelled)
	}
}

func TestLocalCustomEndpointNeedsNoKey(t *testing.T) {
	w := singleWizard()
	selectOther(t, w)
	w.textInput.SetValue("http://127.0.0.1:1234/v1")
	press(t, w, tea.KeyEnter)
	press(t, w, tea.KeyEnter) // empty key
	if w.step != stepModel {
		t.Fatalf("a local endpoint must accept an empty key, step=%v", w.step)
	}
}

// A model typed into the shared text box used to overwrite the API key with
// the model name, so every provider with a custom model got a wrong key.
func TestTypedModelNameDoesNotReplaceTheAPIKey(t *testing.T) {
	w := singleWizard()
	press(t, w, tea.KeyEnter) // first listed provider
	w.textInput.SetValue("real-key")
	press(t, w, tea.KeyEnter)
	if w.step != stepModel {
		t.Fatalf("step=%v", w.step)
	}
	for range w.listItems[1:] {
		press(t, w, tea.KeyDown)
	}
	press(t, w, tea.KeyEnter) // the last item is "Custom model"
	if !w.curCustomText {
		t.Fatal("the last model item must switch to typing a name")
	}
	w.textInput.SetValue("my-model")
	press(t, w, tea.KeyEnter)
	if len(w.providers) != 1 || w.providers[0].APIKey != "real-key" || w.providers[0].Model != "my-model" {
		t.Fatalf("providers = %+v", w.providers)
	}
}

func TestGoingBackToTheKeyShowsTheKeyNotTheModelName(t *testing.T) {
	w := singleWizard()
	press(t, w, tea.KeyEnter)
	w.textInput.SetValue("real-key")
	press(t, w, tea.KeyEnter)
	for range w.listItems[1:] {
		press(t, w, tea.KeyDown)
	}
	press(t, w, tea.KeyEnter)
	w.textInput.SetValue("typed-model")
	press(t, w, tea.KeyEsc)
	if w.step != stepAPIKey || w.textInput.Value() != "real-key" {
		t.Fatalf("step=%v value=%q", w.step, w.textInput.Value())
	}
}
