package cli

import (
	"fmt"
	"path/filepath"

	"github.com/boi-family/boi-cli/internal/memory"
	"github.com/boi-family/boi-cli/internal/weight"
	"github.com/boi-family/boi-cli/internal/workspace"
	"github.com/spf13/cobra"
)

var weightCmd = &cobra.Command{
	Use:   "weight",
	Short: "Weight Engine — explain scores",
}

var weightExplainCmd = &cobra.Command{
	Use:   "explain [memory-id-pattern]",
	Short: "Show weight breakdown for a memory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, _ := workspace.DetectRoot()
		dbDir := filepath.Join(workspace.GetBoiDir(root), "memory")

		store, err := memory.Open(dbDir)
		if err != nil {
			return err
		}
		defer store.Close()

		results, err := store.SearchMemory(args[0], 1)
		if err != nil || len(results) == 0 {
			return fmt.Errorf("no memory found for: %s", args[0])
		}

		mem := &results[0].Entry
		eng := weight.NewEngine(weight.DefaultPolicy())
		exp := eng.ComputeAndExplain(mem)

		fmt.Println(exp)
		fmt.Println("────────────────────")
		fmt.Printf("Final       %.2f\n", exp.FinalScore)
		return nil
	},
}

func init() {
	weightCmd.AddCommand(weightExplainCmd)
	rootCmd.AddCommand(weightCmd)
}
