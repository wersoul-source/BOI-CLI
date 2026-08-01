package cli

import (
	"fmt"

	"github.com/boi-family/boi-cli/internal/registry"
	"github.com/boi-family/boi-cli/internal/tui/setup"
	"github.com/spf13/cobra"
)

var (
	setupRefresh bool
)

func init() {
	setupCmd.Flags().BoolVar(&setupRefresh, "refresh", false, "Refresh endpoint registry from remote source")
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure AI providers interactively (TUI wizard)",
	Long:  `Interactive TUI wizard to configure AI provider API keys for auto-fallback.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		RunSetupWizard(setupRefresh)
		return nil
	},
}

// RunSetupWizard launches the interactive provider setup TUI.
func RunSetupWizard(refresh bool) {
	reg := registry.LoadEmbedded()

	if refresh {
		fmt.Println("  Fetching latest provider registry...")
		// Future: fetch from GitHub and merge
		// For now, use embedded
		fmt.Println("  Using embedded registry.")
	}

	result := setup.Run(reg)
	if result.Cancelled {
		fmt.Println("  Setup cancelled.")
		return
	}
	if len(result.Providers) == 0 {
		fmt.Println("  No providers configured.")
		return
	}
}
