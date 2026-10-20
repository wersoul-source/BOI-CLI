// Package port defines the contracts between the fixed BOI spine and its
// pluggable Blocks. It imports no BOI package, so any Block can implement a
// port without creating a dependency on another Block.
package port

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Risk classifies what a Tool can do to the host.
type Risk string

const (
	RiskRead     Risk = "read"
	RiskChange   Risk = "change"
	RiskExecute  Risk = "execute"
	RiskExternal Risk = "external"
	RiskCritical Risk = "critical"
)

func (r Risk) Valid() bool {
	switch r {
	case RiskRead, RiskChange, RiskExecute, RiskExternal, RiskCritical:
		return true
	default:
		return false
	}
}

// Approval is the host approval a Tool requires before it may act.
type Approval string

const (
	ApprovalAuto     Approval = "auto"
	ApprovalConfirm  Approval = "confirm"
	ApprovalCritical Approval = "critical"
	ApprovalDenied   Approval = "denied"
)

func (a Approval) Valid() bool {
	switch a {
	case ApprovalAuto, ApprovalConfirm, ApprovalCritical, ApprovalDenied:
		return true
	default:
		return false
	}
}

// ToolSpec is the host-owned policy of a Tool. The model never supplies it.
type ToolSpec struct {
	// Name is the capability name the model uses, e.g. "workspace.read".
	Name string
	// Usage is the call signature shown to the model, e.g. "workspace.read(path)".
	Usage    string
	Risk     Risk
	Approval Approval
	Timeout  time.Duration
}

// Validate rejects a spec the Broker could not enforce safely.
func (s ToolSpec) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("tool name is required")
	}
	if !s.Risk.Valid() {
		return fmt.Errorf("tool %s has invalid risk %q", s.Name, s.Risk)
	}
	if !s.Approval.Valid() {
		return fmt.Errorf("tool %s has invalid approval %q", s.Name, s.Approval)
	}
	if s.Approval == ApprovalAuto && s.Risk != RiskRead {
		return fmt.Errorf("tool %s: risk %q cannot use automatic approval", s.Name, s.Risk)
	}
	if s.Timeout <= 0 {
		return fmt.Errorf("tool %s timeout must be positive", s.Name)
	}
	return nil
}

// ToolPreview is what the user sees when approving a call.
type ToolPreview struct {
	Target  string
	Preview string
}

// ToolOutput is the observation a Tool returns to the Runtime.
type ToolOutput struct {
	Output       string
	ChangedPaths []string
}

// Tool is a capability the Runtime Broker can authorize and execute.
// Implementations live in Blocks; the Broker never knows concrete Tools.
type Tool interface {
	Spec() ToolSpec
	// Describe renders the approval preview. It must not cause side effects.
	Describe(arguments map[string]any) ToolPreview
	// Execute performs the call after the Broker has authorized it.
	Execute(ctx context.Context, arguments map[string]any) (ToolOutput, error)
}

// StringArgument returns a non-empty string argument or an error.
func StringArgument(arguments map[string]any, name string) (string, error) {
	value, ok := arguments[name].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("argument %q must be a non-empty string", name)
	}
	return value, nil
}

// OptionalString returns a string argument, or "" when absent.
func OptionalString(arguments map[string]any, name string) string {
	value, _ := arguments[name].(string)
	return value
}

// Evidence is a host observation that supports a verification verdict.
type Evidence struct {
	Kind    string
	Summary string
	Ref     string
}

// VerifiableTool is optionally implemented by a Tool that can re-observe its
// own effect, for example reading back a written file. Tools without it are
// verified only by returning non-empty output.
type VerifiableTool interface {
	Tool
	Verify(ctx context.Context, arguments map[string]any, output ToolOutput) (Evidence, error)
}
