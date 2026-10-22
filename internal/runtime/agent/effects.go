package agent

import (
	"sort"
	"sync"
)

// Outcome says, from what the host recorded, what a task did to the world. It
// never comes from the model's own words.
type Outcome string

const (
	// OutcomeAnswered: no Tool ran.
	OutcomeAnswered Outcome = "answered"
	// OutcomeNoEffect: Tools ran (reads, or calls that failed or only
	// delegated), but none changed or ran anything that passed verification.
	OutcomeNoEffect Outcome = "no_effect"
	// OutcomeExecuted: a command or an outside call ran and passed
	// verification, with no changed file recorded.
	OutcomeExecuted Outcome = "executed"
	// OutcomeChanged: at least one file was changed and verified.
	OutcomeChanged Outcome = "changed"
)

// effectLedger records the Tool calls of a whole task, SubAgent children
// included, that changed files, ran commands or reached outside and then
// passed verification. A Tool call that was only requested, denied, or failed
// is not an effect, and neither is a delegation: the child's own calls are
// recorded when they happen, so a delegation proves nothing by itself.
type effectLedger struct {
	mu      sync.Mutex
	calls   int
	changed map[string]bool
}

func (l *effectLedger) record(call *ToolCall, result *ToolResult) {
	if call == nil || call.Risk == RiskRead || call.Tool == DelegateTool {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls++
	if l.changed == nil {
		l.changed = map[string]bool{}
	}
	if result != nil {
		for _, path := range result.ChangedPaths {
			l.changed[path] = true
		}
	}
}

// snapshot returns the number of effect calls and the changed paths, sorted.
func (l *effectLedger) snapshot() (int, []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	paths := make([]string, 0, len(l.changed))
	for path := range l.changed {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return l.calls, paths
}

func outcomeFor(toolCalls, effectCalls, changedPaths int) Outcome {
	switch {
	case changedPaths > 0:
		return OutcomeChanged
	case effectCalls > 0:
		return OutcomeExecuted
	case toolCalls > 0:
		return OutcomeNoEffect
	default:
		return OutcomeAnswered
	}
}
