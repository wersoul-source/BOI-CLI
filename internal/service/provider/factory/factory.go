package factory

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
	providers "github.com/boi-family/boi-cli/internal/service/provider/adapters"
)

type ProviderConfig struct {
	Name    string
	APIKey  string
	BaseURL string
	Model   string
}

const maxProviders = 20

type ConfiguredProvider struct {
	Provider      llm.Provider
	Name          string
	Model         string
	EndpointClass string
	// Host is where requests go (host and port only), so a check can say which
	// server rejected a key. It is empty when the Provider's own default is used.
	Host string
	// BaseURL is the address requests are built on, without any user name,
	// query or fragment, so it can be shown to the user to check.
	BaseURL string
}

func LoadProvidersFromEnv() ([]llm.Provider, error) {
	configured, err := LoadConfiguredProvidersFromEnv()
	if err != nil {
		return nil, err
	}
	result := make([]llm.Provider, 0, len(configured))
	for _, item := range configured {
		result = append(result, item.Provider)
	}
	return result, nil
}

func LoadConfiguredProvidersFromEnv() ([]ConfiguredProvider, error) {
	var result []ConfiguredProvider
	for i := 1; i <= maxProviders; i++ {
		cfg, ok := providerConfigFromEnv(i)
		if !ok {
			continue
		}
		p := createProvider(cfg)
		if p != nil {
			result = append(result, ConfiguredProvider{Provider: p, Name: cfg.Name, Model: cfg.Model, EndpointClass: classifyEndpoint(cfg.Name, cfg.BaseURL), Host: endpointHost(cfg.BaseURL), BaseURL: displayURL(cfg.BaseURL)})
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid providers found in environment (PSC_1..PSC_%d)", maxProviders)
	}
	return result, nil
}

// ollamaBaseURL is where Ollama listens unless told otherwise.
const ollamaBaseURL = "http://localhost:11434/v1"

// providerConfigFromEnv reads PSC_<n>_*. A provider needs a name and a model,
// and an API key unless it runs on this machine: a local server such as
// Ollama, llama.cpp or LM Studio has no key to give.
func providerConfigFromEnv(n int) (ProviderConfig, bool) {
	prefix := fmt.Sprintf("PSC_%d_", n)
	cfg := ProviderConfig{
		Name:    os.Getenv(prefix + "NAME"),
		APIKey:  os.Getenv(prefix + "API_KEY"),
		BaseURL: os.Getenv(prefix + "BASE_URL"),
		Model:   os.Getenv(prefix + "MODEL"),
	}
	if cfg.Name == "" || cfg.Model == "" {
		return cfg, false
	}
	if cfg.Name == "ollama" && strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = ollamaBaseURL
	}
	if cfg.APIKey == "" && classifyEndpoint(cfg.Name, cfg.BaseURL) != "local" {
		return cfg, false
	}
	return cfg, true
}

// endpointHost returns the host (and port) of a base URL, or "" when there is
// none. It never includes a path, a user name or a query.
func endpointHost(baseURL string) string {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return ""
	}
	return u.Host
}

// displayURL returns scheme, host and path of a base URL: nothing a user
// would not want to read back, and no credentials.
func displayURL(baseURL string) string {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || u.Host == "" {
		return ""
	}
	return strings.TrimRight(u.Scheme+"://"+u.Host+u.EscapedPath(), "/")
}

func classifyEndpoint(name, baseURL string) string {
	if strings.TrimSpace(baseURL) == "" {
		return "official"
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Hostname() == "" {
		return "custom"
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return "local"
	}
	return "custom"
}

func CountProvidersFromEnv() int {
	count := 0
	for i := 1; i <= maxProviders; i++ {
		if _, ok := providerConfigFromEnv(i); ok {
			count++
		}
	}
	return count
}

func createProvider(cfg ProviderConfig) llm.Provider {
	switch cfg.Name {
	case "openai", "groq", "deepseek", "mistral", "xai",
		"ollama", "openrouter", "together",
		"cerebras", "fireworks", "hyperbolic", "cohere",
		"perplexity", "replicate", "nvidia", "deepinfra",
		"novita", "runpod", "opencode", "maxplus":
		return providers.NewOpenAIProvider(cfg.Name, cfg.APIKey, cfg.BaseURL, cfg.Model)
	case "anthropic":
		return providers.NewAnthropicProvider(cfg.Name, cfg.APIKey, cfg.BaseURL, cfg.Model)
	case "google":
		return providers.NewGoogleProvider(cfg.Name, cfg.APIKey, cfg.BaseURL, cfg.Model)
	default:
		return nil
	}
}
