package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/boi-family/boi-cli/internal/memory"
	"github.com/boi-family/boi-cli/internal/workspace"
	"github.com/spf13/cobra"
)

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Manage Phantom DB memory",
	Long:  "Search, view, and manage cross-session memory.",
}

var memorySearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search memory entries",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, _ := workspace.DetectRoot()
		dbDir := filepath.Join(workspace.GetBoiDir(root), "memory")

		store, err := memory.Open(dbDir)
		if err != nil {
			return fmt.Errorf("open memory: %w", err)
		}
		defer store.Close()

		results, err := store.SearchMemory(args[0], 10)
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}

		if len(results) == 0 {
			fmt.Println("No memories found for:", args[0])
			return nil
		}

		fmt.Printf("Found %d memories:\n\n", len(results))
		for i, r := range results {
			fmt.Printf("%d. [%s] %s (score: %.1f)\n", i+1, r.Entry.Type, r.Entry.Key, r.Score)
			content := r.Entry.Content
			if len(content) > 100 {
				content = content[:97] + "..."
			}
			fmt.Println("  ", content)
			fmt.Println()
		}
		return nil
	},
}

var memoryStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show memory statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, _ := workspace.DetectRoot()
		dbDir := filepath.Join(workspace.GetBoiDir(root), "memory")

		store, err := memory.Open(dbDir)
		if err != nil {
			return fmt.Errorf("open memory: %w", err)
		}
		defer store.Close()

		stats, _ := store.Stats()
		fmt.Println("Phantom DB Stats")
		fmt.Println("----------------")
		for k, v := range stats {
			fmt.Printf("  %s: %v\n", k, v)
		}

		fmt.Println("")
		entries, _ := os.ReadDir(dbDir)
		fmt.Printf("  Files: %d\n", len(entries))
		fmt.Printf("  Path: %s\n", dbDir)
		return nil
	},
}

var memorySaveCmd = &cobra.Command{
	Use:   "save [type] [key] [content]",
	Short: "Save a memory entry",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, _ := workspace.DetectRoot()
		dbDir := filepath.Join(workspace.GetBoiDir(root), "memory")

		store, err := memory.Open(dbDir)
		if err != nil {
			return fmt.Errorf("open memory: %w", err)
		}
		defer store.Close()

		entry := &memory.MemoryEntry{
			MemID:     fmt.Sprintf("mem_%d", time.Now().UnixNano()),
			SessionID: "cli",
			Type:      args[0],
			Key:       args[1],
			Content:   args[2],
			Score:     1.0,
			CreatedAt: time.Now(),
		}

		if err := store.Save(entry); err != nil {
			return fmt.Errorf("save: %w", err)
		}

		fmt.Printf("Saved: [%s] %s\n", entry.Type, entry.Key)
		return nil
	},
}

var memoryRepoMapCmd = &cobra.Command{
	Use:   "repomap",
	Short: "Scan and display project structure",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, _ := workspace.DetectRoot()

		m, err := memory.ScanRepo(root)
		if err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		fmt.Println(m.Summary())
		fmt.Printf("\n  Total size: %.1f KB\n", float64(m.TotalSize)/1024)
		fmt.Printf("  Files: %d\n", m.FileCount)

		fmt.Println("\n  Largest files:")
		limit := m.FileCount
		if limit > 10 {
			limit = 10
		}
		for i := 0; i < limit; i++ {
			f := m.Files[i]
			fmt.Printf("    %s (%d bytes)\n", f.Path, f.Size)
		}
		return nil
	},
}

var memoryInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project memory file",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, _ := workspace.DetectRoot()

		content := "# Project Memory\n\n## Architecture\n\n## Conventions\n\n## Known Issues\n\n## Key Decisions\n"
		if err := memory.SaveMemoryFile(root, content); err != nil {
			return err
		}
		fmt.Println("Created .boi/memory.md")
		return nil
	},
}

func init() {
	memoryCmd.AddCommand(memorySearchCmd)
	memoryCmd.AddCommand(memoryStatsCmd)
	memoryCmd.AddCommand(memorySaveCmd)
	memoryCmd.AddCommand(memoryRepoMapCmd)
	memoryCmd.AddCommand(memoryInitCmd)
	rootCmd.AddCommand(memoryCmd)
}
