package app

import (
	"fmt"
	"path/filepath"

	"github.com/boi-family/boi-cli/internal/block/port"
	coreblock "github.com/boi-family/boi-cli/internal/core"
	"github.com/boi-family/boi-cli/internal/core/persona"
	"github.com/boi-family/boi-cli/internal/equipment/memory"
	"github.com/boi-family/boi-cli/internal/equipment/tools/filesystem"
	"github.com/boi-family/boi-cli/internal/equipment/tools/process"
	"github.com/boi-family/boi-cli/internal/runtime/agent"
	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
)

// BuiltinTools is the compile-time Equipment wired into every Agent. Adding a
// Tool means adding it here and to DefaultCapabilityIndexes; the Runtime is
// not edited.
func BuiltinTools(sandbox *workspace.Sandbox) []port.Tool {
	if sandbox == nil {
		return nil
	}
	return append(filesystem.Tools(sandbox), process.Tool(sandbox))
}

// Agent is one composed BOI Agent: the fixed Runtime Service plus the Blocks
// plugged into it for this workspace.
type Agent struct {
	Service     *agent.Service
	Qualified   []llmfactory.ConfiguredProvider
	Router      *llm.Router
	Environment coreblock.AgentEnvironment
	boiDir      string
}

// BuildAgent is the single composition point shared by TUI and CLI.
// configured are the Providers found in the environment; only qualified
// ones enter the Router. A nil Router is allowed so the TUI can still start.
func (r *Runtime) BuildAgent(configured []llmfactory.ConfiguredProvider) (*Agent, error) {
	if r == nil {
		return nil, fmt.Errorf("application runtime is not configured")
	}
	qualified := QualifiedProviders(r.BoiDir, configured)
	var router *llm.Router
	if len(qualified) > 0 {
		providers := make([]llm.Provider, 0, len(qualified))
		for _, item := range qualified {
			providers = append(providers, item.Provider)
		}
		router = llm.NewRouter(providers)
	}

	var memoryHook agent.MemoryHook
	if store, err := memory.Open(filepath.Join(r.BoiDir, "memory")); err == nil {
		memoryHook = memory.NewMemoryHook(store, &memory.SimpleExtractor{})
	}

	service := agent.NewService(persona.CorePersona(), router, memoryHook, r.Sandbox)
	tools := BuiltinTools(r.Sandbox)
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		if err := service.RegisterTool(tool); err != nil {
			return nil, fmt.Errorf("register Tool: %w", err)
		}
		names = append(names, tool.Spec().Name)
	}
	if err := service.SetActiveTools(names); err != nil {
		return nil, err
	}
	if r.AgentFolder != nil {
		service.SetTaskRecorder(r.AgentFolder)
	}
	ConfigureProviderProfileReferences(service, r.WorkspaceRoot, r.BoiDir, qualified)
	environment := ProviderEnvironment(r.BoiDir, qualified)
	service.SetToolCallingAllowed(environment.ToolCalling)
	return &Agent{Service: service, Qualified: qualified, Router: router, Environment: environment, boiDir: r.BoiDir}, nil
}

// PrepareTask selects the bounded working set for one task and activates it.
func (a *Agent) PrepareTask(task string) (*CapabilitySet, error) {
	capabilities, err := SelectCapabilities(a.boiDir, task, a.Environment, a.Service.RegisteredTools())
	if err != nil {
		return nil, err
	}
	if err := a.Service.SetActiveTools(capabilities.Tools.Active); err != nil {
		return nil, err
	}
	a.Service.SetSkills(capabilities.SkillDocs())
	return capabilities, nil
}
