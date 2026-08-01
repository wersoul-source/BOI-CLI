package factory

import (
	"fmt"
	"os"

	"github.com/boi-family/boi-cli/internal/llm"
	"github.com/boi-family/boi-cli/internal/llm/providers"
)

type ProviderConfig struct {
	Name    string
	APIKey  string
	BaseURL string
	Model   string
}

const maxProviders = 20

func LoadProvidersFromEnv() ([]llm.Provider, error) {
	var result []llm.Provider
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
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no valid providers found in environment (PSC_1..PSC_%d)", maxProviders)
	}
	return result, nil
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
		"ollama", "openrouter", "together":
		return providers.NewOpenAIProvider(cfg.Name, cfg.APIKey, cfg.BaseURL, cfg.Model)
	case "anthropic":
		return providers.NewAnthropicProvider(cfg.Name, cfg.APIKey, cfg.BaseURL, cfg.Model)
	case "google":
		return providers.NewGoogleProvider(cfg.Name, cfg.APIKey, cfg.BaseURL, cfg.Model)
	default:
		return nil
	}
}
