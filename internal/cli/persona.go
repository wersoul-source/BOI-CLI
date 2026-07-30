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
	Short: "Manage BOI Family personas",
}

var personaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available personas",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := workspace.DetectRoot()
		if err != nil {
			return fmt.Errorf("detect workspace: %w", err)
		}

		personaDir := filepath.Join(workspace.GetBoiDir(root), "personas")
		reg, err := persona.Load(personaDir)
		if err != nil {
			return fmt.Errorf("load personas: %w", err)
		}

		cfgPath := filepath.Join(workspace.GetBoiDir(root), "config.yaml")
		cfg, _ := config.LoadFrom(cfgPath)
		activeName := ""
		if cfg != nil {
			activeName = cfg.Persona
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tMODEL\tTEMP\tDESCRIPTION")
		fmt.Fprintln(w, "────\t─────\t────\t───────────")

		for _, name := range reg.List() {
			p, _ := reg.Get(name)
			marker := " "
			if name == activeName || (activeName == "" && name == "kamkaew") {
				marker = "*"
			}
			desc := p.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			fmt.Fprintf(w, "%s %s\t%s\t%.1f\t%s\n", marker, p.Name, p.Model, p.Temperature, desc)
		}
		w.Flush()

		fmt.Println()
		fmt.Printf("Personas: %s\n", personaDir)
		fmt.Println("  * = active persona (default: kamkaew)")
		fmt.Println()
		fmt.Println("Use 'boi persona switch <name>' to change active persona")
		return nil
	},
}

var personaSwitchCmd = &cobra.Command{
	Use:   "switch [name]",
	Short: "Switch active persona",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.ToLower(strings.TrimSpace(args[0]))

		root, err := workspace.DetectRoot()
		if err != nil {
			return fmt.Errorf("detect workspace: %w", err)
		}

		personaDir := filepath.Join(workspace.GetBoiDir(root), "personas")
		reg, err := persona.Load(personaDir)
		if err != nil {
			return fmt.Errorf("load personas: %w", err)
		}

		p, err := reg.Get(name)
		if err != nil {
			return err
		}

		cfgPath := filepath.Join(workspace.GetBoiDir(root), "config.yaml")
		cfg, err := config.LoadFrom(cfgPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		cfg.Persona = p.Name
		if err := cfg.SaveTo(cfgPath); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("Switched to persona: %s (%s)\n", p.Name, p.Description)
		fmt.Printf("  Model: %s\n", p.Model)
		fmt.Printf("  Temperature: %.1f\n", p.Temperature)
		return nil
	},
}

var personaInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Copy default persona files to .boi/personas/",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := workspace.DetectRoot()
		if err != nil {
			return fmt.Errorf("detect workspace: %w", err)
		}

		personaDir := filepath.Join(workspace.GetBoiDir(root), "personas")

		if err := os.MkdirAll(personaDir, 0755); err != nil {
			return fmt.Errorf("create persona dir: %w", err)
		}

		count := 0
		for _, filename := range persona.DefaultPersonaFiles() {
			targetPath := filepath.Join(personaDir, filename)
			if _, err := os.Stat(targetPath); err == nil {
				fmt.Printf("  skip (exists): %s\n", filename)
				continue
			}

			data, err := persona.DefaultPersonas.ReadFile("defaults/" + filename)
			if err != nil {
				return fmt.Errorf("read default persona %s: %w", filename, err)
			}

			if err := os.WriteFile(targetPath, data, 0644); err != nil {
				return fmt.Errorf("write persona %s: %w", filename, err)
			}

			fmt.Printf("  created: %s\n", filename)
			count++
		}

		fmt.Printf("\n%d persona files initialized in %s\n", count, personaDir)
		return nil
	},
}

func init() {
	personaCmd.AddCommand(personaListCmd)
	personaCmd.AddCommand(personaSwitchCmd)
	personaCmd.AddCommand(personaInitCmd)
}
