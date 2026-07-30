package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/boi-family/boi-cli/internal/config"
	"github.com/boi-family/boi-cli/internal/logger"
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
		log := logger.New()

		boiDir := ".boi"
		configPath := filepath.Join(boiDir, "config.yaml")

		// Check if already initialized
		if !force {
			if _, err := os.Stat(configPath); err == nil {
				return fmt.Errorf("already initialized. Use --force to overwrite")
			}
		}

		// Create .boi/ directory
		if err := os.MkdirAll(boiDir, 0755); err != nil {
			return fmt.Errorf("create .boi/ failed: %w", err)
		}

		// Write default config
		cfg := config.Default()
		if err := cfg.SaveTo(configPath); err != nil {
			return fmt.Errorf("save config failed: %w", err)
		}

		// Write .gitignore
		gitignore := ".boi/\n"
		if err := os.WriteFile(filepath.Join(boiDir, ".gitignore"), []byte(gitignore), 0644); err != nil {
			log.Warn("failed to create .gitignore", "error", err)
		}

		log.Info("workspace initialized", "path", boiDir)

		fmt.Println("✅ BOI CLI workspace initialized!")
		fmt.Printf("   Config: %s\n", configPath)
		fmt.Println("")
		fmt.Println("   Next steps:")
		fmt.Println("     boi config           — view configuration")
		fmt.Println("     boi run 'ls -la'     — test command execution")
		return nil
	},
}
