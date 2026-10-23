package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	Version   = "2.5.2"
	BuildGo   = runtime.Version()
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
