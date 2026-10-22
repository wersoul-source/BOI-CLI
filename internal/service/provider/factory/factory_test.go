package factory

import "testing"

func setProvider(t *testing.T, name, key, baseURL, model string) {
	t.Helper()
	t.Setenv("PSC_1_NAME", name)
	t.Setenv("PSC_1_API_KEY", key)
	t.Setenv("PSC_1_BASE_URL", baseURL)
	t.Setenv("PSC_1_MODEL", model)
}

func TestHostedProviderNeedsAnAPIKey(t *testing.T) {
	setProvider(t, "openai", "", "", "gpt-x")
	if _, err := LoadConfiguredProvidersFromEnv(); err == nil || CountProvidersFromEnv() != 0 {
		t.Fatalf("a hosted provider without a key must be ignored: %v", err)
	}
	setProvider(t, "openai", "", "https://api.example.com/v1", "m")
	if CountProvidersFromEnv() != 0 {
		t.Fatal("a remote endpoint without a key must be ignored")
	}
}

func TestLocalServerNeedsNoAPIKey(t *testing.T) {
	for name, baseURL := range map[string]string{
		"ollama default":    "",
		"llama.cpp":         "http://localhost:8080/v1",
		"lm studio":         "http://127.0.0.1:1234/v1",
		"ipv6 loopback":     "http://[::1]:8080/v1",
		"ollama other port": "http://localhost:11500/v1",
	} {
		providerName := "ollama"
		if baseURL != "" && name != "ollama other port" {
			providerName = "openai"
		}
		setProvider(t, providerName, "", baseURL, "qwen2.5-coder:7b")
		configured, err := LoadConfiguredProvidersFromEnv()
		if err != nil || len(configured) != 1 || CountProvidersFromEnv() != 1 {
			t.Errorf("%s: %v %d", name, err, len(configured))
			continue
		}
		if configured[0].EndpointClass != "local" {
			t.Errorf("%s: endpoint class %q, want local", name, configured[0].EndpointClass)
		}
	}
}

func TestOllamaWithoutAnEndpointIsNotSentToOpenAI(t *testing.T) {
	setProvider(t, "ollama", "", "", "qwen2.5-coder:7b")
	cfg, ok := providerConfigFromEnv(1)
	if !ok || cfg.BaseURL != ollamaBaseURL {
		t.Fatalf("cfg = %+v ok=%v", cfg, ok)
	}
}
