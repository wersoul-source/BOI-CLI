package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/boi-family/boi-cli/internal/app"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
	"github.com/spf13/cobra"
)

var envJSON bool

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Show which Blocks this workspace's Agent is composed of",
	Long: `Composes the Agent the way 'boi' and 'boi ask' do and reports what the
Runtime actually holds. Each Block is:

  active    at least one of its Tools is offered to the model
            (a Skill pack: the qualified Provider allows Skills)
  loaded    its Tools are registered, but none can be offered now
            (a Skill pack: Skills are read, but no qualified Provider allows Skills)
  declared  listed in .boi/environment.json but not loaded (see reason)

MCP servers are started once to list their tools.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		runtime, ok := app.RuntimeFromContext(cmd.Context())
		if !ok || runtime == nil {
			return fmt.Errorf("application runtime is not configured")
		}
		configured, _ := llmfactory.LoadConfiguredProvidersFromEnv()
		report, err := runtime.Report(configured)
		if err != nil {
			return err
		}
		if envJSON {
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(report)
		}
		printReport(cmd.OutOrStdout(), report)
		if !report.Ready {
			return fmt.Errorf("agent is not ready: %s", report.Problem)
		}
		return nil
	},
}

func printReport(out io.Writer, report *app.AgentReport) {
	name := report.Environment
	if name == "" {
		name = "(unreadable)"
	}
	fmt.Fprintf(out, "Environment: %s\n", name)
	providers := "none qualified"
	if len(report.Providers) > 0 {
		providers = strings.Join(report.Providers, ", ")
	}
	fmt.Fprintf(out, "Providers:   %s\n", providers)
	fmt.Fprintf(out, "Tool calling: %s   Native tools: %s\n\n", yesNo(report.ToolCalling), yesNo(report.NativeTools))
	for _, block := range report.Blocks {
		required := "optional"
		if block.Required {
			required = "required"
		}
		fmt.Fprintf(out, "%-9s %s (%s, %s)\n", block.State, block.ID, block.Kind, required)
		if block.Reason != "" {
			fmt.Fprintf(out, "          reason: %s\n", block.Reason)
		}
		for _, tool := range block.Tools {
			fmt.Fprintf(out, "          %-30s %s\n", tool.Name, toolState(tool))
		}
		for _, name := range block.Skills {
			fmt.Fprintf(out, "          %-30s Skill\n", name)
		}
	}
	if len(report.RuntimeTools) > 0 {
		fmt.Fprintln(out, "runtime   (built in)")
		for _, tool := range report.RuntimeTools {
			fmt.Fprintf(out, "          %-30s %s\n", tool.Name, toolState(tool))
		}
	}
	if report.Problem != "" {
		fmt.Fprintf(out, "\nProblem: %s\n", report.Problem)
	}
}

func toolState(tool app.ToolReport) string {
	switch {
	case tool.Offered:
		return "offered" + approvalNote(tool)
	case tool.Registered:
		return "registered, not offered"
	default:
		return "not registered"
	}
}

func approvalNote(tool app.ToolReport) string {
	if tool.Approval != "" && tool.Approval != "auto" {
		return " (needs approval)"
	}
	return ""
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

var envReplace bool

var envPresetsCmd = &cobra.Command{
	Use:   "presets",
	Short: "List ready-made environments",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		presets := app.Presets()
		for _, name := range app.PresetNames() {
			fmt.Fprintf(cmd.OutOrStdout(), "%-9s %s\n", name, presets[name].Description)
		}
		return nil
	},
}

var envUseCmd = &cobra.Command{
	Use:   "use <preset>",
	Short: "Write a preset as this workspace's .boi/environment.json",
	Long: `Writes the preset as .boi/environment.json. An existing file is kept unless
--replace is given; then it is backed up next to itself first. Run 'boi env'
afterwards to see the result.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runtime, ok := app.RuntimeFromContext(cmd.Context())
		if !ok || runtime == nil {
			return fmt.Errorf("application runtime is not configured")
		}
		path, err := app.UsePreset(runtime.BoiDir, args[0], envReplace)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Environment %q written to %s\n", args[0], path)
		return nil
	},
}

var envTrustCmd = &cobra.Command{
	Use:   "trust",
	Short: "Allow this workspace's environment.json to start its MCP servers",
	Long: `MCP servers listed in a workspace's .boi/environment.json start only after
you trust that exact file. Review it first: each MCP entry is a command that
runs on this machine. Any later edit to the file withdraws the trust.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		runtime, ok := app.RuntimeFromContext(cmd.Context())
		if !ok || runtime == nil {
			return fmt.Errorf("application runtime is not configured")
		}
		commands, err := app.TrustEnvironment(runtime.BoiDir)
		if err != nil {
			return err
		}
		if len(commands) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "Trusted. This environment starts no MCP servers.")
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Trusted. These commands may now start when BOI starts:")
		for _, line := range commands {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", line)
		}
		return nil
	},
}

func init() {
	envCmd.Flags().BoolVar(&envJSON, "json", false, "write the report as JSON")
	envUseCmd.Flags().BoolVar(&envReplace, "replace", false, "back up and replace an existing environment.json")
	envCmd.AddCommand(envPresetsCmd, envUseCmd, envTrustCmd)
	rootCmd.AddCommand(envCmd)
}
