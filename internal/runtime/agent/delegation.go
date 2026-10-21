package agent

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/boi-family/boi-cli/internal/block/port"
)

// DelegateTool is the name of the Tool a SubAgent pack adds (ADR-003).
const DelegateTool = "agent.delegate"

// sharedBudget is the token and Tool-call budget of one top-level task. A
// child run charges the same budget, so parent plus child never exceed the
// parent's limits, and the task reports what both really used.
type sharedBudget struct {
	mu           sync.Mutex
	maxTokens    int
	maxToolCalls int
	used         Usage
}

// charge records use and reports whether the budget still holds.
func (b *sharedBudget) charge(use Usage) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.used = addUsage(b.used, use)
	b.used.ToolCalls += use.ToolCalls
	return b.used.TotalTokens() <= b.maxTokens && b.used.ToolCalls <= b.maxToolCalls
}

func (b *sharedBudget) total() Usage {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.used
}

type budgetKey struct{}

func budgetFrom(ctx context.Context) *sharedBudget {
	budget, _ := ctx.Value(budgetKey{}).(*sharedBudget)
	return budget
}

// runScope is what a child run inherits from the task that delegates to it.
type runScope struct {
	authorizer Authorizer
	depth      int
}

type runScopeKey struct{}

func scopeFrom(ctx context.Context) (runScope, bool) {
	scope, ok := ctx.Value(runScopeKey{}).(runScope)
	return scope, ok
}

// ChildLimits bound one delegated task, inside the parent's shared budget.
var ChildLimits = EngineLimits{MaxSteps: 6, MaxToolCalls: 4, MaxRecoveries: 2, MaxSkillCalls: 1, MaxTokens: 16_000, Timeout: 90 * time.Second}

// DelegableSpecs returns the specs of the named Tools if the parent may hand
// every one of them to a child: each must be offered to the model now, and
// the delegate Tool itself never is (depth 1).
func (s *Service) DelegableSpecs(names []string) ([]port.ToolSpec, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("delegation needs at least one Tool")
	}
	offered := map[string]bool{}
	for _, name := range s.broker.CapabilityNames() {
		offered[name] = true
	}
	specs := make([]port.ToolSpec, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		if name == DelegateTool {
			return nil, fmt.Errorf("a SubAgent cannot delegate further")
		}
		// Every child has its own context.expand; granting it again adds nothing.
		if seen[name] || name == ContextExpandTool {
			continue
		}
		seen[name] = true
		tool, ok := s.broker.Tool(name)
		if !ok || !offered[name] {
			return nil, fmt.Errorf("Tool %s is not available to the parent task, so it cannot be delegated", name)
		}
		specs = append(specs, tool.Spec())
	}
	return specs, nil
}

// RunChild runs a delegated task with a subset of the parent's Tools, the
// parent's authorizer and Providers, and the parent's shared budget. The
// child sees only its task: no conversation, memory or Skills.
func (s *Service) RunChild(ctx context.Context, task string, tools []string) (*AgentResult, error) {
	scope, ok := scopeFrom(ctx)
	if !ok {
		return nil, fmt.Errorf("delegation is only possible inside a running task")
	}
	if scope.depth > 0 {
		return nil, fmt.Errorf("a SubAgent cannot delegate further")
	}
	specs, err := s.DelegableSpecs(tools)
	if err != nil {
		return nil, err
	}
	s.mu.RLock()
	parentPersona, native := s.persona, s.nativeTools
	s.mu.RUnlock()
	child := NewService(parentPersona, s.router, nil, s.sandbox)
	names := make([]string, 0, len(specs))
	for _, spec := range specs {
		tool, _ := s.broker.Tool(spec.Name)
		if err := child.RegisterTool(tool); err != nil {
			return nil, err
		}
		names = append(names, spec.Name)
	}
	sort.Strings(names)
	if err := child.broker.SetActiveCapabilities(append(names, ContextExpandTool)); err != nil {
		return nil, err
	}
	child.nativeTools = native
	child.limits = ChildLimits
	return child.runScoped(ctx, task, scope.authorizer, scope.depth+1)
}
