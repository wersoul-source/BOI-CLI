package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "boi",
	Short: "BOI CLI — BOI Family AI Agent Runtime",
	Long: `BOI CLI is the BOI Family's AI Agent Runtime.

Built with Go, inspired by Chimera Architecture.
DNA from: OpenCode, Hermes, Claude Code, Codex CLI, Antigravity, Agent Zero, ZeroClaw.`,
	Version: "0.1.0",
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(personaCmd)
}
