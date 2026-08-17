package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "0.3.0"
	BuildGo   = "1.24"
	BuildArch = "Chimera"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show BOI CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("BOI CLI v" + Version)
		fmt.Println("Build: Go " + BuildGo)
		fmt.Println("Arch: " + BuildArch)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
