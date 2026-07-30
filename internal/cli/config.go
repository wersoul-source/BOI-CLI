package cli

import (
	"fmt"
	"os"

	"github.com/boi-family/boi-cli/internal/config"
	"github.com/boi-family/boi-cli/internal/workspace"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	showAll bool
)

func init() {
	configCmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show full config including API keys (masked)")
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or edit configuration",
	Long:  `Displays the current BOI CLI configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := workspace.DetectRoot()
		if err != nil {
			root, _ = os.Getwd()
		}

		configPath := workspace.GetBoiDir(root) + "/config.yaml"

		cfg, err := config.LoadFrom(configPath)
		if err != nil {
			return fmt.Errorf("no config found. Run 'boi init' first: %w", err)
		}

		if showAll {
			// Show full YAML
			data, _ := yaml.Marshal(cfg)
			fmt.Print(string(data))
			return nil
		}

		// Summary view
		fmt.Println("┌─────────────────────────────────────┐")
		fmt.Println("│        BOI CLI Config           │")
		fmt.Println("├─────────────────────────────────────┤")
		fmt.Printf("│ Provider:   %-24s │\n", cfg.Provider)
		fmt.Printf("│ Model:      %-24s │\n", cfg.Model)
		fmt.Printf("│ Log Level:  %-24s │\n", cfg.LogLevel)
		fmt.Printf("│ Workspace:  %-24s │\n", root)

		apiCount := 0
		for range cfg.APIKeys {
			apiCount++
		}
		fmt.Printf("│ API Keys:   %d configured             │\n", apiCount)
		fmt.Println("└─────────────────────────────────────┘")
		fmt.Println("")
		fmt.Println("Config file:", configPath)
		fmt.Println("Use --all to see full YAML output.")
		return nil
	},
}
