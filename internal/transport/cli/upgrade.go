package cli

import (
	"fmt"

	platformupdate "github.com/boi-family/boi-cli/internal/platform/update"
	"github.com/spf13/cobra"
)

var checkOnly bool

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade BOI CLI to latest version",
	Long:  "Checks GitHub Releases for the latest version and upgrades if newer.",
	RunE: func(cmd *cobra.Command, args []string) error {
		currentVer := Version
		latest, err := platformupdate.FetchLatestVersion()
		if err != nil {
			return fmt.Errorf("failed to check latest version: %w", err)
		}

		fmt.Printf("Current: v%s\n", currentVer)
		fmt.Printf("Latest:  v%s\n", latest)

		if cmp := platformupdate.CompareVersions(latest, currentVer); cmp <= 0 {
			fmt.Println("\nAlready up to date.")
			return nil
		}

		if checkOnly {
			fmt.Println("\nUpdate available! Run 'boi upgrade' to install.")
			return nil
		}

		fmt.Println("\nDownloading update...")
		if err := platformupdate.DownloadAndReplace(latest); err != nil {
			return fmt.Errorf("upgrade failed: %w", err)
		}

		fmt.Println("Upgrade complete! Restarting...")
		platformupdate.RestartSelf()
		return nil
	},
}

func init() {
	upgradeCmd.Flags().BoolVarP(&checkOnly, "check", "c", false, "Only check for updates, do not install")
	rootCmd.AddCommand(upgradeCmd)
}
