package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/boi-family/boi-cli/internal/block/port"
)

// RiskClass and ApprovalClass are owned by the plug-in port so Tools in other
// Blocks share the exact policy vocabulary the Broker enforces.
type RiskClass = port.Risk

const (
	RiskRead     = port.RiskRead
	RiskChange   = port.RiskChange
	RiskExecute  = port.RiskExecute
	RiskExternal = port.RiskExternal
	RiskCritical = port.RiskCritical
)

type ApprovalClass = port.Approval

const (
	ApprovalAuto     = port.ApprovalAuto
	ApprovalConfirm  = port.ApprovalConfirm
	ApprovalCritical = port.ApprovalCritical
	ApprovalDenied   = port.ApprovalDenied
)

// ToolCall is an untrusted proposal from an Agent. It cannot execute itself;
// a host capability broker must validate and authorize it first.
type ToolCall struct {
	ID             string
	Tool           string
	Purpose        string
	Arguments      map[string]any
	Target         string
	ExpectedResult string
	Preview        string
	Risk           RiskClass
	Approval       ApprovalClass
	Timeout        time.Duration
	IdempotencyKey string
}

func (c ToolCall) Validate() error {
	if strings.TrimSpace(c.ID) == "" {
		return errors.New("tool call id is required")
	}
	if strings.TrimSpace(c.Tool) == "" {
		return errors.New("tool name is required")
	}
	if strings.TrimSpace(c.Purpose) == "" {
		return errors.New("tool purpose is required")
	}
	if c.Timeout <= 0 {
		return errors.New("tool timeout must be positive")
	}
	if !c.Risk.Valid() {
		return fmt.Errorf("invalid tool risk class: %q", c.Risk)
	}
	if !c.Approval.Valid() {
		return fmt.Errorf("invalid approval class: %q", c.Approval)
	}
	if c.Approval == ApprovalAuto && c.Risk != RiskRead {
		return fmt.Errorf("risk %q cannot use automatic approval", c.Risk)
	}
	if c.Risk == RiskCritical && c.Approval != ApprovalCritical && c.Approval != ApprovalDenied {
		return errors.New("critical tool call requires critical approval")
	}
	if _, err := c.Fingerprint(); err != nil {
		return fmt.Errorf("tool call cannot be fingerprinted: %w", err)
	}
	return nil
}

// Fingerprint binds approval to the exact action shown to the user. Any
// argument, target, preview, risk, or timeout change produces a new digest.
func (c ToolCall) Fingerprint() (string, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

type ToolResultStatus string

const (
	ToolSucceeded ToolResultStatus = "succeeded"
	ToolFailed    ToolResultStatus = "failed"
	ToolPartial   ToolResultStatus = "partial"
	ToolCancelled ToolResultStatus = "cancelled"
	ToolTimedOut  ToolResultStatus = "timed_out"
	ToolDenied    ToolResultStatus = "denied"
)

// ToolResult is a host observation. Agent responses may cite a side effect
// only when the corresponding result and evidence confirm it.
type ToolResult struct {
	CallID       string
	Status       ToolResultStatus
	Output       string
	ChangedPaths []string
	Evidence     []Evidence
	ErrorClass   string
	ErrorMessage string
	StartedAt    time.Time
	FinishedAt   time.Time
}

type Evidence = port.Evidence

func (r ToolResult) Validate() error {
	if strings.TrimSpace(r.CallID) == "" {
		return errors.New("tool result call id is required")
	}
	if !r.Status.Valid() {
		return fmt.Errorf("invalid tool result status: %q", r.Status)
	}
	if !r.StartedAt.IsZero() && !r.FinishedAt.IsZero() && r.FinishedAt.Before(r.StartedAt) {
		return errors.New("tool result finished before it started")
	}
	return nil
}

func (s ToolResultStatus) Valid() bool {
	switch s {
	case ToolSucceeded, ToolFailed, ToolPartial, ToolCancelled, ToolTimedOut, ToolDenied:
		return true
	default:
		return false
	}
}

type ApprovalState string

const (
	ApprovalRequested ApprovalState = "requested"
	ApprovalApproved  ApprovalState = "approved"
	ApprovalRejected  ApprovalState = "rejected"
	ApprovalExpired   ApprovalState = "expired"
	ApprovalCancelled ApprovalState = "cancelled"
)

type ApprovalRequest struct {
	ID              string
	Call            ToolCall
	CallFingerprint string
	State           ApprovalState
	RequestedAt     time.Time
	ExpiresAt       time.Time
}

func NewApprovalRequest(id string, call ToolCall, requestedAt, expiresAt time.Time) (ApprovalRequest, error) {
	fingerprint, err := call.Fingerprint()
	if err != nil {
		return ApprovalRequest{}, fmt.Errorf("fingerprint approval tool call: %w", err)
	}
	request := ApprovalRequest{
		ID:              id,
		Call:            call,
		CallFingerprint: fingerprint,
		State:           ApprovalRequested,
		RequestedAt:     requestedAt,
		ExpiresAt:       expiresAt,
	}
	if err := request.Validate(requestedAt); err != nil {
		return ApprovalRequest{}, err
	}
	return request, nil
}

func (r ApprovalRequest) Validate(now time.Time) error {
	if strings.TrimSpace(r.ID) == "" {
		return errors.New("approval request id is required")
	}
	if err := r.Call.Validate(); err != nil {
		return fmt.Errorf("approval tool call: %w", err)
	}
	if r.Call.Approval != ApprovalConfirm && r.Call.Approval != ApprovalCritical {
		return fmt.Errorf("tool call with approval class %q must not open an approval request", r.Call.Approval)
	}
	fingerprint, err := r.Call.Fingerprint()
	if err != nil {
		return fmt.Errorf("fingerprint approval tool call: %w", err)
	}
	if r.CallFingerprint == "" || r.CallFingerprint != fingerprint {
		return errors.New("approval tool call changed after request creation")
	}
	if r.State != ApprovalRequested {
		return fmt.Errorf("new approval request must be requested, got %q", r.State)
	}
	if r.RequestedAt.IsZero() {
		return errors.New("approval request time is required")
	}
	if !r.ExpiresAt.IsZero() && !r.ExpiresAt.After(r.RequestedAt) {
		return errors.New("approval expiry must be after request time")
	}
	if !r.ExpiresAt.IsZero() && !now.Before(r.ExpiresAt) {
		return errors.New("approval request is expired")
	}
	return nil
}

type ApprovalDecision struct {
	RequestID string
	State     ApprovalState
	Reason    string
	DecidedAt time.Time
}

func (d ApprovalDecision) Validate() error {
	if strings.TrimSpace(d.RequestID) == "" {
		return errors.New("approval decision request id is required")
	}
	if d.DecidedAt.IsZero() {
		return errors.New("approval decision time is required")
	}
	switch d.State {
	case ApprovalApproved, ApprovalRejected, ApprovalExpired, ApprovalCancelled:
		return nil
	default:
		return fmt.Errorf("invalid approval decision state: %q", d.State)
	}
}

type Usage struct {
	InputTokens   int
	OutputTokens  int
	ToolCalls     int
	ProviderCalls int
	Elapsed       time.Duration
}

func (u Usage) TotalTokens() int {
	return u.InputTokens + u.OutputTokens
}
