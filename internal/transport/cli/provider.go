package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/app"
	coreblock "github.com/boi-family/boi-cli/internal/block/core"
	"github.com/boi-family/boi-cli/internal/config"
	llmfactory "github.com/boi-family/boi-cli/internal/provider/factory"
	"github.com/boi-family/boi-cli/internal/workspace"
	"github.com/spf13/cobra"
)

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage LLM providers",
}

var providerProbeTimeout time.Duration

var providerQualifyCmd = &cobra.Command{
	Use:   "qualify <name>",
	Short: "Run the BOI Provider capability probe suite",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, ok := app.RuntimeFromContext(cmd.Context())
		if !ok || runtime == nil {
			return fmt.Errorf("application runtime is not configured")
		}
		configured, err := llmfactory.LoadConfiguredProvidersFromEnv()
		if err != nil {
			return err
		}
		name := strings.ToLower(strings.TrimSpace(args[0]))
		for _, item := range configured {
			if strings.ToLower(item.Name) != name {
				continue
			}
			target := coreblock.ProviderTarget{Provider: item.Name, Model: item.Model, EndpointClass: item.EndpointClass}
			profile, err := (coreblock.Qualifier{Timeout: providerProbeTimeout}).Run(cmd.Context(), target, item.Provider)
			if err != nil {
				return fmt.Errorf("qualify provider: %w", err)
			}
			path := coreblock.CapabilityProfilePath(runtime.BoiDir, target)
			if err := coreblock.SaveCapabilityProfile(path, profile); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Provider: %s\nModel: %s\nEndpoint class: %s\n", target.Provider, target.Model, target.EndpointClass)
			for _, result := range profile.Results {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-20s %s\n", result.Capability, result.Status)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Profile: %s\n", path)
			return nil
		}
		return fmt.Errorf("provider %q is not configured", name)
	},
}

var providerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured providers with status",
	RunE: func(cmd *cobra.Command, args []string) error {
		providers, err := llmfactory.LoadProvidersFromEnv()
		if err != nil {
			fmt.Println("No providers configured. Run 'boi setup'.")
			return nil
		}
		fmt.Println("Configured providers:")
		for i, p := range providers {
			fmt.Printf("  %d. %s\n", i+1, p.Name())
		}
		return nil
	},
}

var providerSwitchCmd = &cobra.Command{
	Use:   "switch <name>",
	Short: "Switch active provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.ToLower(strings.TrimSpace(args[0]))
		providers, err := llmfactory.LoadProvidersFromEnv()
		if err != nil {
			return fmt.Errorf("no providers configured")
		}
		found := false
		for _, p := range providers {
			if p.Name() == name {
				found = true
				break
			}
		}
		if !found {
			names := make([]string, len(providers))
			for i, p := range providers {
				names[i] = p.Name()
			}
			return fmt.Errorf("provider '%s' not found. Available: %s", name, strings.Join(names, ", "))
		}

		// Save to config
		root, _ := workspace.DetectRoot()
		cfgPath := filepath.Join(workspace.GetBoiDir(root), "config.yaml")
		cfg := loadOrCreateConfig(cfgPath)
		cfg.Provider = name
		if err := cfg.SaveTo(cfgPath); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("Switched provider to: %s\n", name)
		fmt.Println("Note: Restart BOI TUI for the change to take effect.")
		return nil
	},
}

var modelCmd = &cobra.Command{
	Use:   "model <name>",
	Short: "Set default model for current provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		model := strings.TrimSpace(args[0])
		root, _ := workspace.DetectRoot()
		cfgPath := filepath.Join(workspace.GetBoiDir(root), "config.yaml")
		cfg := loadOrCreateConfig(cfgPath)
		cfg.Model = model
		if err := cfg.SaveTo(cfgPath); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Printf("Model set to: %s\n", model)
		return nil
	},
}

func loadOrCreateConfig(path string) *config.Config {
	cfg, err := config.LoadFrom(path)
	if err != nil {
		cfg = config.Default()
	}
	return cfg
}

func init() {
	providerQualifyCmd.Flags().DurationVar(&providerProbeTimeout, "timeout", 30*time.Second, "Timeout for each capability probe")
	providerCmd.AddCommand(providerListCmd)
	providerCmd.AddCommand(providerSwitchCmd)
	providerCmd.AddCommand(providerQualifyCmd)
	rootCmd.AddCommand(providerCmd)
	rootCmd.AddCommand(modelCmd)
}
