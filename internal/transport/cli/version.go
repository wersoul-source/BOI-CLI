package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "1.5.0"
	BuildGo   = "1.24"
	BuildArch = "Chimera"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show BOI Agent Suite version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("BOI Agent Suite v" + Version)
		fmt.Println("Build: Go " + BuildGo)
		fmt.Println("Arch: " + BuildArch)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
