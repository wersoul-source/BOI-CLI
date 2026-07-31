package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type ProviderOption struct {
	Name            string
	Label           string
	DefaultEndpoint string
	DefaultModel    string
}

var providers = []ProviderOption{
	{Name: "openai", Label: "OpenAI (GPT-4.1, GPT-4o)", DefaultEndpoint: "https://api.openai.com/v1", DefaultModel: "gpt-4.1-mini"},
	{Name: "anthropic", Label: "Anthropic (Claude Sonnet, Opus)", DefaultEndpoint: "https://api.anthropic.com/v1", DefaultModel: "claude-sonnet-5"},
	{Name: "google", Label: "Google (Gemini 2.5)", DefaultEndpoint: "https://generativelanguage.googleapis.com/v1beta", DefaultModel: "gemini-2.5-flash"},
	{Name: "groq", Label: "Groq (fast inference)", DefaultEndpoint: "https://api.groq.com/openai/v1", DefaultModel: "llama-3.3-70b"},
	{Name: "deepseek", Label: "DeepSeek (V3, R1)", DefaultEndpoint: "https://api.deepseek.com/v1", DefaultModel: "deepseek-chat"},
	{Name: "mistral", Label: "Mistral (Large, Small)", DefaultEndpoint: "https://api.mistral.ai/v1", DefaultModel: "mistral-small"},
	{Name: "xai", Label: "xAI (Grok)", DefaultEndpoint: "https://api.x.ai/v1", DefaultModel: "grok-4.5"},
	{Name: "ollama", Label: "Ollama (local models)", DefaultEndpoint: "http://localhost:11434/v1", DefaultModel: "llama3.3"},
	{Name: "openrouter", Label: "OpenRouter (multi-provider)", DefaultEndpoint: "https://openrouter.ai/api/v1", DefaultModel: "openai/gpt-4.1-mini"},
	{Name: "together", Label: "Together AI", DefaultEndpoint: "https://api.together.xyz/v1", DefaultModel: "meta-llama/Llama-3.3-70B"},
	{Name: "other", Label: "Other (custom endpoint)", DefaultEndpoint: "", DefaultModel: ""},
}

type selectedProvider struct {
	Name     string
	APIKey   string
	Endpoint string
	Model    string
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure AI providers interactively",
	Long:  `Interactive wizard to configure AI provider API keys for auto-fallback.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		RunSetupWizard()
		return nil
	},
}

func RunSetupWizard() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║        Configure AI Providers               ║")
	fmt.Println("║        BOI CLI — Provider Setup              ║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Welcome to BOI CLI! Let's set up your AI providers.")
	fmt.Println("You can configure multiple providers for auto-fallback.")
	fmt.Println()

	fmt.Println("  Choose providers (comma-separated numbers):")
	fmt.Println("  ─────────────────────────────────────────")
	for i, p := range providers {
		fmt.Printf("  %2d. %s\n", i+1, p.Label)
	}
	fmt.Println()

	defaultChoice := "1"
	fmt.Printf("  Selection [%s]: ", defaultChoice)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		input = defaultChoice
	}

	selections := parseSelections(input)
	if len(selections) == 0 {
		fmt.Println("  No providers selected. Setup cancelled.")
		return
	}

	var configured []selectedProvider

	for _, idx := range selections {
		prov := providers[idx]

		fmt.Println()
		fmt.Printf("  ── Provider: %s ──\n", prov.Label)

		if prov.Name == "other" {
			fmt.Print("  Provider name: ")
			name, _ := reader.ReadString('\n')
			name = strings.TrimSpace(name)
			if name == "" {
				fmt.Println("  Skipped.")
				continue
			}

			fmt.Print("  API endpoint URL: ")
			endpoint, _ := reader.ReadString('\n')
			endpoint = strings.TrimSpace(endpoint)

			fmt.Print("  Default model: ")
			model, _ := reader.ReadString('\n')
			model = strings.TrimSpace(model)

			fmt.Print("  API Key: ")
			apiKey, _ := reader.ReadString('\n')
			apiKey = strings.TrimSpace(apiKey)

			if apiKey == "" {
				fmt.Println("  No API key provided. Skipped.")
				continue
			}

			configured = append(configured, selectedProvider{
				Name:     name,
				APIKey:   apiKey,
				Endpoint: endpoint,
				Model:    model,
			})
		} else {
			fmt.Printf("  Endpoint: %s\n", prov.DefaultEndpoint)
			fmt.Printf("  Default model: %s\n", prov.DefaultModel)

			fmt.Print("  API Key: ")
			apiKey, _ := reader.ReadString('\n')
			apiKey = strings.TrimSpace(apiKey)

			if apiKey == "" {
				fmt.Println("  No API key provided. Skipped.")
				continue
			}

			configured = append(configured, selectedProvider{
				Name:     prov.Name,
				APIKey:   apiKey,
				Endpoint: prov.DefaultEndpoint,
				Model:    prov.DefaultModel,
			})
		}
	}

	if len(configured) == 0 {
		fmt.Println()
		fmt.Println("  No providers configured. Setup complete.")
		return
	}

	writeEnvFile(configured)

	fmt.Println()
	if len(configured) >= 2 {
		fmt.Println("✅ Auto-fallback enabled! Providers will be tried in order:")
	} else {
		fmt.Println("✅ Provider configured:")
	}
	for i, sp := range configured {
		fmt.Printf("   %d. %s (%s)\n", i+1, sp.Name, sp.Model)
	}
	fmt.Println()
	fmt.Println("   .env file created.")
	fmt.Println("   Run 'boi' to start using BOI CLI!")
}

func parseSelections(input string) []int {
	var result []int
	seen := make(map[int]bool)
	parts := strings.Split(input, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > len(providers) {
			continue
		}
		idx := n - 1
		if !seen[idx] {
			seen[idx] = true
			result = append(result, idx)
		}
	}
	return result
}

func writeEnvFile(configured []selectedProvider) {
	var lines []string
	lines = append(lines, "# =====================================================")
	lines = append(lines, "#  BOI CLI — Provider Supply Chain")
	lines = append(lines, "#  Auto-fallback configured via 'boi setup'")
	lines = append(lines, "# =====================================================")
	lines = append(lines, "")

	for i, sp := range configured {
		n := i + 1
		lines = append(lines, fmt.Sprintf("# --- Provider %d ---", n))
		lines = append(lines, fmt.Sprintf("PSC_%d_NAME=%s", n, sp.Name))
		lines = append(lines, fmt.Sprintf("PSC_%d_API_KEY=%s", n, sp.APIKey))
		lines = append(lines, fmt.Sprintf("PSC_%d_BASE_URL=%s", n, sp.Endpoint))
		lines = append(lines, fmt.Sprintf("PSC_%d_MODEL=%s", n, sp.Model))
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")

	cwd, _ := os.Getwd()
	envPath := filepath.Join(cwd, ".env")

	if _, err := os.Stat(envPath); err == nil {
		fmt.Println()
		fmt.Print("  ⚠  .env already exists. Overwrite? [y/N]: ")
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			fmt.Println("  Setup cancelled. Existing .env preserved.")
			return
		}
	}

	os.WriteFile(envPath, []byte(content), 0644)
}
