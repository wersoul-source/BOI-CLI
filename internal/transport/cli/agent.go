package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/agent"
	"github.com/boi-family/boi-cli/internal/app"
	coreblock "github.com/boi-family/boi-cli/internal/block/core"
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
	Long:  "Runs the bounded BOI Agent loop with memory and the fixed Core Persona boi.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")

		runtime, ok := app.RuntimeFromContext(cmd.Context())
		if !ok || runtime == nil {
			return fmt.Errorf("application runtime is not configured")
		}
		if !strings.EqualFold(strings.TrimSpace(agentPersona), coreblock.CorePersonaName) {
			return fmt.Errorf("Persona selection is retired; Core Persona is fixed to %s", coreblock.CorePersonaName)
		}
		p := persona.CorePersona()

		configured, err := llmfactory.LoadConfiguredProvidersFromEnv()
		if err != nil {
			return fmt.Errorf("load providers: %w", err)
		}
		qualified := app.QualifiedProviders(runtime.BoiDir, configured)
		providers := make([]llm.Provider, 0, len(qualified))
		for _, item := range qualified {
			providers = append(providers, item.Provider)
		}
		if len(providers) == 0 {
			return fmt.Errorf("no qualified providers; run 'boi provider qualify <name>'")
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
		service.SetTaskRecorder(runtime.AgentFolder)
		app.ConfigureProviderProfileReferences(service, runtime.WorkspaceRoot, runtime.BoiDir, qualified)
		environment := app.ProviderEnvironment(runtime.BoiDir, qualified)
		service.SetToolCallingAllowed(environment.ToolCalling)
		capabilities, err := app.SelectCapabilities(runtime.BoiDir, query, environment)
		if err != nil {
			return fmt.Errorf("select capability registry: %w (run 'boi registry init')", err)
		}
		if err := service.SetActiveTools(capabilities.Tools.Active); err != nil {
			return err
		}
		service.SetSkills(capabilities.LoadedSkills)
		limits := agent.DefaultEngineLimits()
		limits.MaxSteps = agentSteps
		service.SetLimits(limits)

		if agentVerbose {
			agentName := coreblock.DefaultAgentName
			if identity, loadErr := coreblock.LoadIdentity(runtime.IdentityPath); loadErr == nil {
				agentName = identity.Name
			}
			fmt.Printf("Agent: %s | Core Persona: %s | Model preference: %s\n", agentName, p.Name, p.Model)
			fmt.Printf("Qualified environment: completion=%t tools=%t skills=%t context-bytes=%d\n", environment.Completion, environment.ToolCalling, environment.SkillCalling, environment.ContextBytes)
			fmt.Printf("Active registry: tools=%v skills=%v\n", capabilities.Tools.Active, capabilities.Skills.Active)
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
			fmt.Printf("Task: %s | Manifest: %s\n", result.TaskID, result.Manifest)
			if len(result.Memory) > 0 {
				fmt.Printf("Memories saved: %v\n", result.Memory)
			}
		}

		return nil
	},
}

func init() {
	askCmd.Flags().StringVarP(&agentPersona, "persona", "p", "boi", "Compatibility flag; Core Persona is fixed to boi")
	_ = askCmd.Flags().MarkDeprecated("persona", "Core Persona is fixed to boi in Work 1")
	askCmd.Flags().IntVarP(&agentSteps, "steps", "s", 15, "Max agent steps")
	askCmd.Flags().BoolVarP(&agentVerbose, "verbose", "v", false, "Verbose output")
	rootCmd.AddCommand(askCmd)
}
