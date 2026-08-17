package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boi-family/boi-cli/internal/skill"
	"github.com/boi-family/boi-cli/internal/workspace"
	"github.com/spf13/cobra"
)

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage skills",
	Long:  "List, load, and manage skills for BOI CLI.",
}

var skillListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all loaded skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		reg := skill.NewRegistry()

		root, _ := workspace.DetectRoot()
		skillDir := filepath.Join(workspace.GetBoiDir(root), "skills")

		skills, err := skill.LoadDir(skillDir)
		if err != nil {
			return fmt.Errorf("load skills: %w", err)
		}

		for _, s := range skills {
			reg.Register(s)
		}

		all := reg.All()
		if len(all) == 0 {
			fmt.Println("No skills loaded.")
			fmt.Println("")
			fmt.Println("Add skills to: " + skillDir)
			fmt.Println("Use 'boi skill init' to create default skills.")
			return nil
		}

		fmt.Println("BOI Skills")
		fmt.Println("----------")
		for _, s := range all {
			desc := s.Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}
			fmt.Printf("  %-12s — %s\n", s.Name, desc)
		}
		fmt.Println("")
		fmt.Printf("Skills dir: %s\n", skillDir)
		fmt.Printf("Total: %d skills\n", len(all))
		return nil
	},
}

var skillInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize default skills in .boi/skills/",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, _ := workspace.DetectRoot()
		skillDir := filepath.Join(workspace.GetBoiDir(root), "skills")

		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return err
		}

		defaults := map[string]string{
			"git": "" +
				"---\n" +
				"name: git-helper\n" +
				"description: Git operations assistant\n" +
				"version: \"1.0\"\n" +
				"requires:\n" +
				"  - shell\n" +
				"---\n\n" +
				"# Git Helper Skill\n" +
				"1. git status\n" +
				"2. git diff\n" +
				"3. git add + git commit\n",
			"web": "" +
				"---\n" +
				"name: web-search\n" +
				"description: Web search and fetch\n" +
				"version: \"1.0\"\n" +
				"requires:\n" +
				"  - shell\n" +
				"---\n\n" +
				"# Web Search Skill\n" +
				"Use configured search tools.\n",
		}

		for name, content := range defaults {
			path := filepath.Join(skillDir, name+".skill.md")
			if _, err := os.Stat(path); err == nil {
				fmt.Printf("  skip: %s (already exists)\n", name)
				continue
			}
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return fmt.Errorf("write %s: %w", name, err)
			}
			fmt.Printf("  created: %s.skill.md\n", name)
		}

		fmt.Println("Skills initialized!")
		return nil
	},
}

var skillShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show skill details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reg := skill.NewRegistry()
		root, _ := workspace.DetectRoot()
		skillDir := filepath.Join(workspace.GetBoiDir(root), "skills")

		skills, err := skill.LoadDir(skillDir)
		if err != nil {
			return err
		}
		for _, s := range skills {
			reg.Register(s)
		}

		s, err := reg.Get(args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Name:        %s\n", s.Name)
		fmt.Printf("Description: %s\n", s.Description)
		fmt.Printf("Version:     %s\n", s.Version)
		fmt.Printf("Path:        %s\n", s.Path)
		fmt.Printf("BuiltIn:     %v\n", s.BuiltIn)
		if len(s.Requires) > 0 {
			fmt.Printf("Requires:    %v\n", s.Requires)
		}
		fmt.Println("\n--- Content ---")
		fmt.Println(s.Prompt)
		return nil
	},
}

func init() {
	skillCmd.AddCommand(skillListCmd)
	skillCmd.AddCommand(skillInitCmd)
	skillCmd.AddCommand(skillShowCmd)
	rootCmd.AddCommand(skillCmd)
}
