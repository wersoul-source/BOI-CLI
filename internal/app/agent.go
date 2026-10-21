package app

import (
	"fmt"
	"path/filepath"

	"github.com/boi-family/boi-cli/internal/block/port"
	coreblock "github.com/boi-family/boi-cli/internal/core"
	"github.com/boi-family/boi-cli/internal/core/persona"
	"github.com/boi-family/boi-cli/internal/equipment/memory"
	"github.com/boi-family/boi-cli/internal/runtime/agent"
	llm "github.com/boi-family/boi-cli/internal/runtime/llm"
	"github.com/boi-family/boi-cli/internal/runtime/workspace"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
)

// BuiltinTools are the Tools of every built-in pack, as composed by the
// default environment. Adding a Tool means adding it to a pack in
// BuiltinPacks and to DefaultCapabilityIndexes; the Runtime is not edited.
func BuiltinTools(sandbox *workspace.Sandbox) []port.Tool {
	if sandbox == nil {
		return nil
	}
	composition, err := Compose(DefaultEnvironment(), BuiltinPacks(), sandbox, "", coreblock.AgentEnvironment{})
	if err != nil {
		return nil
	}
	return composition.Tools
}

// Agent is one composed BOI Agent: the fixed Runtime Service plus the Blocks
// plugged into it for this workspace.
type Agent struct {
	Service     *agent.Service
	Qualified   []llmfactory.ConfiguredProvider
	Router      *llm.Router
	Environment coreblock.AgentEnvironment
	// Composition reports which Blocks were plugged in and why others were not.
	Composition *Composition
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
	router := routerFor(qualified)
	environment := ProviderEnvironment(r.BoiDir, qualified)
	selected, err := LoadEnvironment(r.BoiDir)
	if err != nil {
		return nil, err
	}
	composition, err := Compose(selected, BuiltinPacks(), r.Sandbox, r.Version, environment)
	if err != nil {
		return nil, fmt.Errorf("compose Agent: %w", err)
	}
	return r.assemble(qualified, router, environment, composition)
}

// routerFor builds the failover Router of the qualified Providers, or nil.
func routerFor(qualified []llmfactory.ConfiguredProvider) *llm.Router {
	if len(qualified) == 0 {
		return nil
	}
	providers := make([]llm.Provider, 0, len(qualified))
	for _, item := range qualified {
		providers = append(providers, item.Provider)
	}
	return llm.NewRouter(providers)
}

// assemble plugs a resolved composition into a new Runtime Service.
func (r *Runtime) assemble(qualified []llmfactory.ConfiguredProvider, router *llm.Router, environment coreblock.AgentEnvironment, composition *Composition) (*Agent, error) {
	var memoryHook agent.MemoryHook
	if store, err := memory.Open(filepath.Join(r.BoiDir, "memory")); err == nil {
		memoryHook = memory.NewMemoryHook(store, &memory.SimpleExtractor{})
	}

	service := agent.NewService(persona.CorePersona(), router, memoryHook, r.Sandbox)
	tools := composition.Tools
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
	service.SetToolCallingAllowed(environment.ToolCalling)
	service.SetNativeToolCalling(environment.NativeTools)
	return &Agent{Service: service, Qualified: qualified, Router: router, Environment: environment, Composition: composition, boiDir: r.BoiDir}, nil
}

// PrepareTask selects the bounded working set for one task and activates it.
func (a *Agent) PrepareTask(task string) (*CapabilitySet, error) {
	capabilities, err := SelectCapabilities(a.boiDir, task, a.Environment, a.Service.RegisteredTools())
	if err != nil {
		return nil, err
	}
	// MCP Tools are chosen by the environment, not the Tool index, so every
	// one an active MCP Block plugged in joins the working set while there is
	// room under the active-Tool limit.
	if a.Composition != nil && a.Environment.ToolCalling {
		for _, name := range a.Composition.MCPTools() {
			if len(capabilities.Tools.Active) >= agent.MaxActiveTools-1 {
				break
			}
			capabilities.Tools.Active = append(capabilities.Tools.Active, name)
		}
	}
	if err := a.Service.SetActiveTools(capabilities.Tools.Active); err != nil {
		return nil, err
	}
	a.Service.SetSkills(capabilities.SkillDocs())
	return capabilities, nil
}
