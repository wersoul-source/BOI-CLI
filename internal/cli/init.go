package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boi-family/boi-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	force bool
)

func init() {
	initCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing config")
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize BOI CLI workspace",
	Long:  `Creates a .boi/ directory with default configuration in the current directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunInit(force)
	},
}

func RunInit(force bool) error {
	boiDir := ".boi"
	configPath := filepath.Join(boiDir, "config.yaml")

	if !force {
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("already initialized. Use --force to overwrite")
		}
	}

	if err := os.MkdirAll(boiDir, 0755); err != nil {
		return fmt.Errorf("create .boi/ failed: %w", err)
	}

	cfg := config.Default()
	if err := cfg.SaveTo(configPath); err != nil {
		return fmt.Errorf("save config failed: %w", err)
	}

	gitignore := ".boi/\n"
	os.WriteFile(filepath.Join(boiDir, ".gitignore"), []byte(gitignore), 0644)

	skillsDir := filepath.Join(boiDir, "skills")
	os.MkdirAll(skillsDir, 0755)

	memoryDir := filepath.Join(boiDir, "memory")
	os.MkdirAll(memoryDir, 0755)

	return nil
}
