package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/app"
	coreblock "github.com/boi-family/boi-cli/internal/core"
	"github.com/boi-family/boi-cli/internal/core/persona"
	"github.com/boi-family/boi-cli/internal/runtime/agent"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
	"github.com/spf13/cobra"
)

var (
	agentPersona        string
	agentSteps          int
	agentVerbose        bool
	agentJSON           bool
	agentIdempotencyKey string
)

var askCmd = &cobra.Command{
	Use:   "ask [query...]",
	Short: "Ask the BOI agent",
	Long: `Runs the bounded BOI Agent loop with memory and the fixed Core Persona boi.

Query input comes from argv when present, otherwise from piped UTF-8 stdin.
The command never prompts for a missing query. --json writes exactly one
versioned result object to stdout and keeps mutation non-interactive/denied.`,
	Example: "  boi ask explain this repository\n  Get-Content task.txt | boi ask --json --idempotency-key build-001",
	RunE: func(cmd *cobra.Command, args []string) error {
		query, err := resolveAskQuery(args, cmd.InOrStdin())
		if err != nil {
			return finishAsk(cmd, nil, err)
		}
		if err := validateIdempotencyKey(agentIdempotencyKey); err != nil {
			return finishAsk(cmd, nil, err)
		}
		if agentIdempotencyKey != "" && !agentJSON {
			return finishAsk(cmd, nil, invalidInput("--idempotency-key requires --json"))
		}
		if agentSteps <= 0 {
			return finishAsk(cmd, nil, invalidInput("--steps must be greater than zero"))
		}

		runtime, ok := app.RuntimeFromContext(cmd.Context())
		if !ok || runtime == nil {
			return finishAsk(cmd, nil, &CommandError{Code: ExitInternal, Class: "internal", Message: "application runtime is not configured"})
		}
		if !strings.EqualFold(strings.TrimSpace(agentPersona), coreblock.CorePersonaName) {
			return finishAsk(cmd, nil, invalidInput(fmt.Sprintf("Persona selection is retired; Core Persona is fixed to %s", coreblock.CorePersonaName)))
		}
		p := persona.CorePersona()

		configured, err := llmfactory.LoadConfiguredProvidersFromEnv()
		if err != nil {
			return finishAsk(cmd, nil, unavailable("load providers", err))
		}
		if len(app.QualifiedProviders(runtime.BoiDir, configured)) == 0 {
			return finishAsk(cmd, nil, unavailable("no qualified providers; run 'boi provider qualify <name>'", nil))
		}
		if err := runtime.EnsureWorkspaceState(); err != nil {
			return finishAsk(cmd, nil, unavailable("initialize workspace state", err))
		}
		composed, err := runtime.BuildAgent(configured)
		if err != nil {
			return finishAsk(cmd, nil, &CommandError{Code: ExitInternal, Class: "internal", Message: "compose Agent", Cause: err})
		}
		service, environment := composed.Service, composed.Environment
		capabilities, err := composed.PrepareTask(query)
		if err != nil {
			return finishAsk(cmd, nil, unavailable("select capability registry; run 'boi registry init'", err))
		}
		limits := agent.DefaultEngineLimits()
		limits.MaxSteps = agentSteps
		service.SetLimits(limits)

		if agentVerbose {
			agentName := coreblock.DefaultAgentName
			if identity, loadErr := coreblock.LoadIdentity(runtime.IdentityPath); loadErr == nil {
				agentName = identity.Name
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Agent: %s | Core Persona: %s | Model preference: %s\n", agentName, p.Name, p.Model)
			fmt.Fprintf(cmd.ErrOrStderr(), "Qualified environment: completion=%t tools=%t skills=%t context-bytes=%d\n", environment.Completion, environment.ToolCalling, environment.SkillCalling, environment.ContextBytes)
			fmt.Fprintf(cmd.ErrOrStderr(), "Active registry: tools=%v skills=%v\n", capabilities.Tools.Active, capabilities.Skills.Active)
			fmt.Fprintf(cmd.ErrOrStderr(), "Steps: max %d\n---\n", limits.MaxSteps)
		}

		ctx := cmd.Context()
		start := time.Now()

		var result *agent.AgentResult
		if agentJSON {
			result, err = service.RunAutomation(ctx, query, agentIdempotencyKey)
		} else {
			result, err = service.Run(ctx, query)
		}
		if agentJSON {
			return writeAutomationResult(cmd.OutOrStdout(), result, err)
		}
		if err != nil {
			return finishAsk(cmd, result, err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), result.Response)

		if agentVerbose {
			fmt.Fprintln(cmd.ErrOrStderr(), "---")
			fmt.Fprintf(cmd.ErrOrStderr(), "Steps: %d | Tokens: %d | Time: %v\n",
				result.Steps, result.Tokens, time.Since(start))
			fmt.Fprintf(cmd.ErrOrStderr(), "Task: %s | Manifest: %s\n", result.TaskID, result.Manifest)
			if len(result.Memory) > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "Memories saved: %v\n", result.Memory)
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
	askCmd.Flags().BoolVar(&agentJSON, "json", false, "Write one versioned JSON result to stdout")
	askCmd.Flags().StringVar(&agentIdempotencyKey, "idempotency-key", "", "Automation key (1-128 safe characters; requires --json)")
	rootCmd.AddCommand(askCmd)
}

func finishAsk(cmd *cobra.Command, result *agent.AgentResult, err error) error {
	if agentJSON {
		return writeAutomationResult(cmd.OutOrStdout(), result, err)
	}
	code, class := classifyAutomationFailure(result, err)
	if code == ExitCompleted {
		return nil
	}
	message := "command failed"
	if err != nil {
		message = err.Error()
	}
	return &CommandError{Code: code, Class: class, Message: message, Cause: err}
}

func unavailable(message string, cause error) error {
	return &CommandError{Code: ExitUnavailable, Class: "unavailable", Message: message, Cause: cause}
}
