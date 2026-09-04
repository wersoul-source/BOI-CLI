package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boi-family/boi-cli/internal/app"
	"github.com/boi-family/boi-cli/internal/config"
	"github.com/boi-family/boi-cli/internal/workspace"
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
	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve current directory: %w", err)
	}
	return RunInitAt(root, force)
}

func RunInitAt(root string, force bool) error {
	boiDir := filepath.Join(root, ".boi")
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

	gitignore := "*\n!.gitignore\n"
	if err := os.WriteFile(filepath.Join(boiDir, ".gitignore"), []byte(gitignore), 0o644); err != nil {
		return fmt.Errorf("protect BOI workspace data: %w", err)
	}
	if err := workspace.EnsureLocalGitExcludes(root, ".boi/", ".env", ".env.boi-backup-*"); err != nil {
		return fmt.Errorf("protect local BOI data from Git: %w", err)
	}

	skillsDir := filepath.Join(boiDir, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return fmt.Errorf("create Skill directory: %w", err)
	}

	memoryDir := filepath.Join(boiDir, "memory")
	if err := os.MkdirAll(memoryDir, 0o755); err != nil {
		return fmt.Errorf("create memory directory: %w", err)
	}
	if err := app.EnsureCapabilityIndexes(boiDir); err != nil {
		return fmt.Errorf("initialize capability indexes: %w", err)
	}

	return nil
}
