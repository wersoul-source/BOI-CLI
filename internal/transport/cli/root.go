package cli

import (
	"context"
	"os"
	"os/signal"

	"github.com/boi-family/boi-cli/internal/app"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "boi",
	Short: "BOI CLI — BOI Family AI Agent Runtime",
	Long: `BOI CLI is the BOI Family's AI Agent Runtime.

Built with Go, inspired by Chimera Architecture.
DNA from: OpenCode, Hermes, Claude Code, Codex CLI, Antigravity, Agent Zero, ZeroClaw.`,
	Version:       Version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command with the shared process runtime.
func Execute(runtime *app.Runtime) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return ExecuteContext(ctx, runtime)
}

// ExecuteContext supports deterministic cancellation in tests, embedding, and
// automation hosts while Execute supplies the process interrupt context.
func ExecuteContext(ctx context.Context, runtime *app.Runtime) error {
	ctx = app.WithRuntime(ctx, runtime)
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(personaCmd)
}
