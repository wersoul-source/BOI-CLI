package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/boi-family/boi-cli/internal/command"
	"github.com/boi-family/boi-cli/internal/logger"
	"github.com/boi-family/boi-cli/internal/workspace"
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

		exec := command.NewExecutor(command.WithVerbose(verbose))

		var result string
		var err error

		if workDir != "" {
			result, err = exec.RunWithDir(commandStr, workDir)
		} else {
			root, wserr := workspace.DetectRoot()
			if wserr == nil {
				result, err = exec.RunWithDir(commandStr, root)
			} else {
				result, err = exec.Run(commandStr)
			}
		}

		if err != nil {
			log.Error("command failed", "error", err)
			os.Exit(1)
		}

		fmt.Print(result)
		return nil
	},
}
