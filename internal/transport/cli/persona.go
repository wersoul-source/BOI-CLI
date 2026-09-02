package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/boi-family/boi-cli/internal/config"
	"github.com/boi-family/boi-cli/internal/persona"
	"github.com/boi-family/boi-cli/internal/workspace"
	"github.com/spf13/cobra"
)

var personaCmd = &cobra.Command{
	Use:   "persona",
	Short: "Show BOI Core Persona compatibility information",
	Long:  "Work 1 has one fixed Core Persona: boi. Legacy Persona files are preserved but are not selectable runtime identities.",
}

var personaListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show the fixed BOI Core Persona",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := persona.CorePersona()
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tMODEL\tTEMP\tDESCRIPTION")
		fmt.Fprintln(w, "────\t─────\t────\t───────────")
		fmt.Fprintf(w, "* %s\t%s\t%.1f\t%s\n", p.Name, p.Model, p.Temperature, p.Description)
		if err := w.Flush(); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "\nCore Persona is fixed by BOI Core. Agent name is stored separately in .boi/agent.yaml.")
		return nil
	},
}

var personaSwitchCmd = &cobra.Command{
	Use:   "switch [name]",
	Short: "Compatibility command; only the Core Persona boi is accepted",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.ToLower(strings.TrimSpace(args[0]))
		if name != "boi" {
			return fmt.Errorf("Persona switching is retired in Work 1; Core Persona is fixed to boi")
		}
		root, err := workspace.DetectRoot()
		if err != nil {
			return fmt.Errorf("detect workspace: %w", err)
		}
		cfgPath := filepath.Join(workspace.GetBoiDir(root), "config.yaml")
		cfg, err := config.LoadFrom(cfgPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		cfg.Persona = "boi"
		if err := cfg.SaveTo(cfgPath); err != nil {
			return fmt.Errorf("save compatibility config: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Core Persona is already fixed to: boi")
		return nil
	},
}

var personaInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Install the BOI Core Persona file without overwriting",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := workspace.DetectRoot()
		if err != nil {
			return fmt.Errorf("detect workspace: %w", err)
		}
		personaDir := filepath.Join(workspace.GetBoiDir(root), "personas")
		if err := os.MkdirAll(personaDir, 0o755); err != nil {
			return fmt.Errorf("create persona dir: %w", err)
		}
		targetPath := filepath.Join(personaDir, "boi.yaml")
		if _, err := os.Stat(targetPath); err == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "skip (exists): boi.yaml")
			return nil
		}
		data, err := persona.DefaultPersonas.ReadFile("defaults/boi.yaml")
		if err != nil {
			return fmt.Errorf("read Core Persona: %w", err)
		}
		if err := os.WriteFile(targetPath, data, 0o644); err != nil {
			return fmt.Errorf("write Core Persona: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "created: %s\n", targetPath)
		return nil
	},
}

var personaWizardCmd = &cobra.Command{
	Use:   "wizard",
	Short: "Compatibility notice for the retired Persona wizard",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.OutOrStdout(), "Persona selection is retired. Core Persona is fixed to boi.")
		fmt.Fprintln(cmd.OutOrStdout(), "Launch 'boi' without arguments to create or load your Agent identity.")
	},
}

func init() {
	personaCmd.AddCommand(personaListCmd)
	personaCmd.AddCommand(personaSwitchCmd)
	personaCmd.AddCommand(personaInitCmd)
	personaCmd.AddCommand(personaWizardCmd)
}
