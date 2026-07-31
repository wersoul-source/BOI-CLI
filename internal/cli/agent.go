package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/agent"
	llmfactory "github.com/boi-family/boi-cli/internal/llm/factory"
	"github.com/boi-family/boi-cli/internal/memory"
	"github.com/boi-family/boi-cli/internal/persona"
	"github.com/boi-family/boi-cli/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	agentPersona string
	agentSteps   int
	agentVerbose bool
)

var askCmd = &cobra.Command{
	Use:   "ask [query]",
	Short: "Ask the BOI agent",
	Long:  "Runs the full agent loop with memory, skills, and persona.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")

		root, _ := workspace.DetectRoot()
		personaDir := filepath.Join(workspace.GetBoiDir(root), "personas")
		reg := persona.NewRegistry()
		if loaded, err := persona.LoadDir(personaDir); err == nil {
			for _, p := range loaded {
				reg.Register(p)
			}
		}
		p, _ := reg.Get(agentPersona)
		if p == nil {
			p = persona.DefaultPersona()
		}

		providers, err := llmfactory.LoadProvidersFromEnv()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: no providers configured: %v\n", err)
			fmt.Fprintf(os.Stderr, "Set PSC_1_NAME, PSC_1_API_KEY, etc. in .env\n")
			fmt.Fprintf(os.Stderr, "Falling back to simulated response mode\n\n")
		}

		dbDir := filepath.Join(workspace.GetBoiDir(root), "memory")
		store, err := memory.Open(dbDir)
		var memHook *memory.MemoryHook
		if err == nil {
			extractor := &memory.SimpleExtractor{}
			memHook = memory.NewMemoryHook(store, extractor)
			defer store.Close()
		}

		loop := agent.NewLoop(p, providers, memHook)
		loop.MaxSteps = agentSteps

		if agentVerbose {
			fmt.Printf("Agent: %s (%s)\n", p.Name, p.Model)
			fmt.Printf("Steps: max %d\n", loop.MaxSteps)
			fmt.Println("---")
		}

		ctx := context.Background()
		start := time.Now()

		result, err := loop.Run(ctx, query)
		if err != nil {
			return fmt.Errorf("agent error: %w", err)
		}

		fmt.Println(result.Response)

		if agentVerbose {
			fmt.Println("---")
			fmt.Printf("Steps: %d | Tokens: %d | Time: %v\n",
				result.Steps, result.Tokens, time.Since(start))
			if len(result.Memory) > 0 {
				fmt.Printf("Memories saved: %v\n", result.Memory)
			}
		}

		return nil
	},
}

func init() {
	askCmd.Flags().StringVarP(&agentPersona, "persona", "p", "kamkaew", "Persona to use")
	askCmd.Flags().IntVarP(&agentSteps, "steps", "s", 15, "Max agent steps")
	askCmd.Flags().BoolVarP(&agentVerbose, "verbose", "v", false, "Verbose output")
	rootCmd.AddCommand(askCmd)
}
