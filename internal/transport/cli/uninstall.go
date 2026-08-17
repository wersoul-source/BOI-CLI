package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	uninstallForce    bool
	uninstallKeepData bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall BOI CLI",
	Long:  "Removes BOI CLI binary and data. Requires confirmation unless --force is used.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !uninstallForce {
			fmt.Print("This will remove BOI CLI. Continue? [y/N]: ")
			scanner := bufio.NewScanner(os.Stdin)
			if !scanner.Scan() {
				fmt.Println("\nUninstall cancelled.")
				return nil
			}
			answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if answer != "y" && answer != "yes" {
				fmt.Println("Uninstall cancelled.")
				return nil
			}
		}

		if !uninstallKeepData {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot find home directory: %w", err)
			}
			boiDataDir := filepath.Join(homeDir, ".boi")
			if _, err := os.Stat(boiDataDir); err == nil {
				fmt.Printf("Removing data: %s\n", boiDataDir)
				if err := os.RemoveAll(boiDataDir); err != nil {
					return fmt.Errorf("remove data directory: %w", err)
				}
				fmt.Println("  Data removed.")
			} else {
				fmt.Println("  No data directory found.")
			}
		} else {
			fmt.Println("  Keeping data directory (--keep-data)")
		}

		binPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot find binary: %w", err)
		}

		fmt.Printf("Removing binary: %s\n", binPath)
		absBin, _ := filepath.Abs(binPath)
		batchFile := filepath.Join(os.TempDir(), "boi-uninstall.bat")

		batch := fmt.Sprintf(`@echo off
timeout /t 1 /nobreak >nul
del /f /q "%s"
del /f /q "%%~f0"
`, absBin)

		if err := os.WriteFile(batchFile, []byte(batch), 0644); err != nil {
			return fmt.Errorf("write uninstall script: %w", err)
		}

		exec.Command("cmd", "/c", batchFile).Start()
		fmt.Println("  Binary removal scheduled.")
		fmt.Println("\nUninstall complete. Goodbye!")

		os.Exit(0)
		return nil
	},
}

func init() {
	uninstallCmd.Flags().BoolVarP(&uninstallForce, "force", "f", false, "Skip confirmation prompt")
	uninstallCmd.Flags().BoolVar(&uninstallKeepData, "keep-data", false, "Preserve ~/.boi/ data directory")
	rootCmd.AddCommand(uninstallCmd)
}
