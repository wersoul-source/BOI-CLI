package app

import (
	"sort"

	"github.com/boi-family/boi-cli/internal/block"
	"github.com/boi-family/boi-cli/internal/runtime/agent"
	llmfactory "github.com/boi-family/boi-cli/internal/service/provider/factory"
)

// Block states in an Agent report, from weakest to strongest. Each is read
// from what the Runtime actually holds, never from the manifest alone.
const (
	// StateDeclared: listed in the environment but not loaded; Reason says why.
	StateDeclared = "declared"
	// StateLoaded: its Tools are registered in the Broker, but none of them
	// can be offered to the model now; Reason says why.
	StateLoaded = "loaded"
	// StateActive: at least one of its Tools is offered to the model.
	StateActive = "active"
)

// AgentReport describes the Agent a workspace composes, as the Runtime holds it.
type AgentReport struct {
	Environment string        `json:"environment"`
	Providers   []string      `json:"providers"`
	ToolCalling bool          `json:"tool_calling"`
	NativeTools bool          `json:"native_tools"`
	Ready       bool          `json:"ready"`
	Problem     string        `json:"problem,omitempty"`
	Blocks      []BlockReport `json:"blocks"`
	// RuntimeTools are offered by the Runtime itself, not by a Block.
	RuntimeTools []ToolReport `json:"runtime_tools"`
}

type BlockReport struct {
	ID       string       `json:"id"`
	Kind     string       `json:"kind"`
	Required bool         `json:"required"`
	State    string       `json:"state"`
	Reason   string       `json:"reason,omitempty"`
	Tools    []ToolReport `json:"tools"`
	// Skills lists the Skills of a skill-pack Block.
	Skills []string `json:"skills,omitempty"`
}

type ToolReport struct {
	Name       string `json:"name"`
	Registered bool   `json:"registered"`
	Offered    bool   `json:"offered"`
	Risk       string `json:"risk,omitempty"`
	Approval   string `json:"approval,omitempty"`
}

// Report composes the workspace's Agent the way BuildAgent does, without
// stopping at the first problem, and reports each Block's real state.
// Offered Tools are those of the baseline working set (an empty task); a
// task can select a different subset of the registered Tools.
func (r *Runtime) Report(configured []llmfactory.ConfiguredProvider) (*AgentReport, error) {
	qualified := QualifiedProviders(r.BoiDir, configured)
	environment := ProviderEnvironment(r.BoiDir, qualified)
	report := &AgentReport{ToolCalling: environment.ToolCalling, NativeTools: environment.NativeTools, Blocks: []BlockReport{}, RuntimeTools: []ToolReport{}}
	for _, item := range qualified {
		report.Providers = append(report.Providers, item.Name+"/"+item.Model)
	}
	selected, err := LoadEnvironment(r.BoiDir)
	if err != nil {
		report.Problem = err.Error()
		return report, nil
	}
	report.Environment = selected.Name
	composition, err := compose(selected, BuiltinPacks(), r.Sandbox, r.Version, environment, false)
	if err != nil {
		report.Problem = err.Error()
		return report, nil
	}

	var registered, offered map[string]bool
	var service *agent.Service
	blocking := ""
	for _, entry := range composition.Entries {
		if entry.Required && entry.Status != PackActive && blocking == "" {
			blocking = "required Block " + entry.ID + ": " + entry.Reason
		}
	}
	if blocking == "" {
		built, err := r.assemble(qualified, routerFor(qualified), environment, composition)
		if err != nil {
			report.Problem = err.Error()
			return report, nil
		}
		if err := EnsureCapabilityIndexes(r.BoiDir); err == nil {
			_, err = built.PrepareTask("")
			if err != nil {
				report.Problem = err.Error()
			}
		}
		service = built.Service
		registered, offered = setOf(service.RegisteredTools()), setOf(service.OfferedTools())
	} else {
		report.Problem = blocking
	}

	owned := map[string]bool{}
	for _, entry := range composition.Entries {
		block := BlockReport{ID: entry.ID, Kind: entry.Kind, Required: entry.Required, State: StateDeclared, Reason: entry.Reason, Tools: []ToolReport{}}
		for _, tool := range composition.Tools {
			name := tool.Spec().Name
			if !contains(entry.Tools, name) {
				continue
			}
			owned[name] = true
			spec := tool.Spec()
			block.Tools = append(block.Tools, ToolReport{Name: name, Registered: registered[name], Offered: offered[name], Risk: string(spec.Risk), Approval: string(spec.Approval)})
		}
		if entry.Kind == skillPackKind {
			block.Skills = entry.Skills
			block.State = skillPackState(entry, environment.SkillCalling)
			if block.State == StateLoaded && block.Reason == "" {
				block.Reason = "no qualified Provider allows Skills"
			}
			report.Blocks = append(report.Blocks, block)
			continue
		}
		block.State = blockState(entry, block.Tools, service != nil)
		if block.State == StateLoaded && block.Reason == "" {
			block.Reason = loadedReason(environment.ToolCalling)
		}
		report.Blocks = append(report.Blocks, block)
	}
	for name := range registered {
		if !owned[name] {
			report.RuntimeTools = append(report.RuntimeTools, ToolReport{Name: name, Registered: true, Offered: offered[name]})
		}
	}
	sort.Slice(report.RuntimeTools, func(i, j int) bool { return report.RuntimeTools[i].Name < report.RuntimeTools[j].Name })
	report.Ready = report.Problem == "" && service != nil
	return report, nil
}

const skillPackKind = string(block.SkillPack)

// skillPackState: a Skill pack is active when the qualified Provider allows
// Skills, loaded when its Skills are read but cannot be offered, declared
// when it was refused.
func skillPackState(entry CompositionEntry, skillCalling bool) string {
	if entry.Status != PackActive {
		return StateDeclared
	}
	if skillCalling {
		return StateActive
	}
	return StateLoaded
}

func blockState(entry CompositionEntry, tools []ToolReport, assembled bool) string {
	if entry.Status != PackActive || !assembled {
		return StateDeclared
	}
	for _, tool := range tools {
		if !tool.Registered {
			return StateDeclared
		}
	}
	for _, tool := range tools {
		if tool.Offered {
			return StateActive
		}
	}
	return StateLoaded
}

func loadedReason(toolCalling bool) string {
	if !toolCalling {
		return "no qualified Provider allows Tool calling"
	}
	return "none of its Tools is in the baseline working set (.boi/registry/tools.json)"
}

func setOf(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}
	return set
}

func contains(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}
