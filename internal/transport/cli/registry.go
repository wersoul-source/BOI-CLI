package cli

import (
	"fmt"
	"strings"

	"github.com/boi-family/boi-cli/internal/app"
	"github.com/boi-family/boi-cli/internal/capability"
	"github.com/spf13/cobra"
)

var registryCmd = &cobra.Command{Use: "registry", Short: "Manage explicit active capability indexes"}

var registryInitCmd = &cobra.Command{Use: "init", Short: "Create missing Skill and Tool indexes without overwriting", RunE: func(cmd *cobra.Command, _ []string) error {
	runtime, ok := app.RuntimeFromContext(cmd.Context())
	if !ok {
		return fmt.Errorf("application runtime is not configured")
	}
	if err := app.EnsureCapabilityIndexes(runtime.BoiDir); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Capability indexes ready: %s\n", runtime.BoiDir)
	return nil
}}

var registryListCmd = &cobra.Command{Use: "list <skill|tool>", Short: "List registered capabilities, excluding loose files", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
	runtime, ok := app.RuntimeFromContext(cmd.Context())
	if !ok {
		return fmt.Errorf("application runtime is not configured")
	}
	kind, err := parseCapabilityKind(args[0])
	if err != nil {
		return err
	}
	index, err := capability.LoadIndex(capability.IndexPath(runtime.BoiDir, kind), kind)
	if err != nil {
		return err
	}
	for _, entry := range index.Entries {
		fmt.Fprintf(cmd.OutOrStdout(), "%-24s enabled=%-5t priority=%d source=%s\n", entry.Name, entry.Enabled, entry.Priority, entry.Source)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Registered: %d | Active limit: 15\n", len(index.Entries))
	return nil
}}

var registrySource, registrySummary string
var registryPriority, registryContextCost int
var registryRequires, registryTags []string

var registryAddCmd = &cobra.Command{Use: "add <skill|tool> <name>", Short: "Register one installed capability explicitly", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
	runtime, ok := app.RuntimeFromContext(cmd.Context())
	if !ok {
		return fmt.Errorf("application runtime is not configured")
	}
	kind, err := parseCapabilityKind(args[0])
	if err != nil {
		return err
	}
	name, source := strings.TrimSpace(args[1]), strings.TrimSpace(registrySource)
	if source == "" && kind == capability.KindSkill {
		source = name + ".skill.md"
	} else if source == "" {
		source = "builtin"
	}
	if strings.TrimSpace(registrySummary) == "" {
		return fmt.Errorf("--summary is required")
	}
	entry := capability.Entry{Name: name, Source: source, Summary: registrySummary, Enabled: true, Priority: registryPriority, Tags: registryTags, Requires: registryRequires, ContextCost: registryContextCost}
	if err := capability.AddEntry(capability.IndexPath(runtime.BoiDir, kind), kind, entry); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Registered %s: %s\n", kind, name)
	return nil
}}

func parseCapabilityKind(value string) (capability.Kind, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "skill", "skills":
		return capability.KindSkill, nil
	case "tool", "tools":
		return capability.KindTool, nil
	default:
		return "", fmt.Errorf("capability kind must be skill or tool")
	}
}

func init() {
	registryAddCmd.Flags().StringVar(&registrySource, "source", "", "Relative Skill file or Tool source")
	registryAddCmd.Flags().StringVar(&registrySummary, "summary", "", "Short capability summary")
	registryAddCmd.Flags().IntVar(&registryPriority, "priority", 0, "Deterministic selection priority")
	registryAddCmd.Flags().IntVar(&registryContextCost, "context-cost", 0, "Estimated Context bytes")
	registryAddCmd.Flags().StringSliceVar(&registryRequires, "requires", nil, "Required active Tool names")
	registryAddCmd.Flags().StringSliceVar(&registryTags, "tags", nil, "Task matching tags")
	registryCmd.AddCommand(registryInitCmd, registryListCmd, registryAddCmd)
	rootCmd.AddCommand(registryCmd)
}
