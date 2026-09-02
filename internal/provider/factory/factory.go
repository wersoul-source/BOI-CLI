package factory

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	llm "github.com/boi-family/boi-cli/internal/provider"
	providers "github.com/boi-family/boi-cli/internal/provider/adapters"
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
		prefix := fmt.Sprintf("PSC_%d_", i)
		cfg := ProviderConfig{
			Name:    os.Getenv(prefix + "NAME"),
			APIKey:  os.Getenv(prefix + "API_KEY"),
			BaseURL: os.Getenv(prefix + "BASE_URL"),
			Model:   os.Getenv(prefix + "MODEL"),
		}
		if cfg.Name == "" {
			continue
		}
		if cfg.APIKey == "" || cfg.Model == "" {
			continue
		}
		p := createProvider(cfg)
		if p != nil {
			result = append(result, ConfiguredProvider{Provider: p, Name: cfg.Name, Model: cfg.Model, EndpointClass: classifyEndpoint(cfg.Name, cfg.BaseURL)})
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid providers found in environment (PSC_1..PSC_%d)", maxProviders)
	}
	return result, nil
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
		name := os.Getenv(fmt.Sprintf("PSC_%d_NAME", i))
		if name == "" {
			continue
		}
		key := os.Getenv(fmt.Sprintf("PSC_%d_API_KEY", i))
		model := os.Getenv(fmt.Sprintf("PSC_%d_MODEL", i))
		if key != "" && model != "" {
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
		"novita", "runpod":
		return providers.NewOpenAIProvider(cfg.Name, cfg.APIKey, cfg.BaseURL, cfg.Model)
	case "anthropic":
		return providers.NewAnthropicProvider(cfg.Name, cfg.APIKey, cfg.BaseURL, cfg.Model)
	case "google":
		return providers.NewGoogleProvider(cfg.Name, cfg.APIKey, cfg.BaseURL, cfg.Model)
	default:
		return nil
	}
}
