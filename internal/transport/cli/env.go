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
  loaded    its Tools are registered, but none can be offered now
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

func init() {
	envCmd.Flags().BoolVar(&envJSON, "json", false, "write the report as JSON")
	rootCmd.AddCommand(envCmd)
}
