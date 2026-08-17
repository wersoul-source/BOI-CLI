package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/boi-family/boi-cli/internal/app"
	logger "github.com/boi-family/boi-cli/internal/platform/logging"
	command "github.com/boi-family/boi-cli/internal/tool/process"
	"github.com/spf13/cobra"
)

var (
	verbose bool
	workDir string
)

func init() {
	runCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	runCmd.Flags().StringVarP(&workDir, "dir", "d", "", "Working directory")
}

var runCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "Execute a shell command",
	Long:  "Run executes a shell command with sandbox safety checks.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		commandStr := strings.Join(args, " ")

		log := logger.New()
		log.Info("boi run", "command", commandStr)

		runtime, ok := app.RuntimeFromContext(cmd.Context())
		if !ok || runtime.Sandbox == nil {
			return fmt.Errorf("workspace runtime is not configured")
		}
		exec := command.NewExecutor(
			command.WithVerbose(verbose),
			command.WithWorkspace(runtime.Sandbox),
		)

		var result string
		var err error

		if workDir != "" {
			result, err = exec.RunWithDir(commandStr, workDir)
		} else {
			result, err = exec.Run(commandStr)
		}

		if err != nil {
			log.Error("command failed", "error", err)
			os.Exit(1)
		}

		fmt.Print(result)
		return nil
	},
}
