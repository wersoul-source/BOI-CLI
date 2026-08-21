package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/agent"
	"github.com/boi-family/boi-cli/internal/app"
	"github.com/boi-family/boi-cli/internal/memory"
	"github.com/boi-family/boi-cli/internal/persona"
	llm "github.com/boi-family/boi-cli/internal/provider"
	llmfactory "github.com/boi-family/boi-cli/internal/provider/factory"
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

		runtime, ok := app.RuntimeFromContext(cmd.Context())
		if !ok || runtime == nil {
			return fmt.Errorf("application runtime is not configured")
		}
		personaDir := filepath.Join(runtime.BoiDir, "personas")
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
			return fmt.Errorf("load providers: %w", err)
		}

		dbDir := filepath.Join(runtime.BoiDir, "memory")
		store, err := memory.Open(dbDir)
		var memHook *memory.MemoryHook
		if err == nil {
			extractor := &memory.SimpleExtractor{}
			memHook = memory.NewMemoryHook(store, extractor)
			defer store.Close()
		}

		service := agent.NewService(p, llm.NewRouter(providers), memHook, runtime.Sandbox)
		limits := agent.DefaultEngineLimits()
		limits.MaxSteps = agentSteps
		service.SetLimits(limits)

		if agentVerbose {
			fmt.Printf("Agent: %s (%s)\n", p.Name, p.Model)
			fmt.Printf("Steps: max %d\n", limits.MaxSteps)
			fmt.Println("---")
		}

		ctx := context.Background()
		start := time.Now()

		result, err := service.Run(ctx, query)
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
