package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type DecisionKind string

const (
	DecisionRespond DecisionKind = "respond"
	DecisionUseTool DecisionKind = "use_tool"
)

type DecisionInput struct {
	Task       string
	Step       int
	Steps      []AgentStep
	LastResult *ToolResult
}

type Decision struct {
	Kind     DecisionKind
	Response string
	ToolCall *ToolCall
	Usage    Usage
	Provider string
	Model    string
}

func (d Decision) Validate() error {
	switch d.Kind {
	case DecisionRespond:
		if strings.TrimSpace(d.Response) == "" {
			return errors.New("response decision is empty")
		}
		if d.ToolCall != nil {
			return errors.New("response decision must not contain a tool call")
		}
	case DecisionUseTool:
		if d.ToolCall == nil {
			return errors.New("tool decision is missing a tool call")
		}
		if err := d.ToolCall.Validate(); err != nil {
			return fmt.Errorf("tool decision: %w", err)
		}
	default:
		return fmt.Errorf("invalid decision kind: %q", d.Kind)
	}
	return nil
}

type Authorization struct {
	Allowed  bool
	State    ApprovalState
	Request  *ApprovalRequest
	Decision *ApprovalDecision
	Reason   string
}

type VerificationInput struct {
	Task       string
	Response   string
	ToolCall   *ToolCall
	ToolResult *ToolResult
}

type Verification struct {
	Passed   bool
	Reason   string
	Evidence []Evidence
}

type Failure struct {
	Phase      AgentPhase
	Err        error
	ToolCall   *ToolCall
	ToolResult *ToolResult
	Attempt    int
}

type Recovery struct {
	Retry      bool
	StopReason StopReason
	Reason     string
}

type Decider interface {
	Decide(context.Context, DecisionInput) (Decision, error)
}

type Authorizer interface {
	Authorize(context.Context, ToolCall) (Authorization, error)
}

type Actor interface {
	Act(context.Context, ToolCall, Authorization) (ToolResult, error)
}

type Verifier interface {
	Verify(context.Context, VerificationInput) (Verification, error)
}

type Recoverer interface {
	Recover(context.Context, Failure) Recovery
}

type EngineEvent struct {
	Phase      AgentPhase
	Step       int
	ToolCall   *ToolCall
	ToolResult *ToolResult
	StopReason StopReason
}

type EngineLimits struct {
	MaxSteps      int
	MaxToolCalls  int
	MaxRecoveries int
	MaxTokens     int
	Timeout       time.Duration
}

func DefaultEngineLimits() EngineLimits {
	return EngineLimits{
		MaxSteps:      12,
		MaxToolCalls:  8,
		MaxRecoveries: 2,
		MaxTokens:     64_000,
		Timeout:       2 * time.Minute,
	}
}

type Engine struct {
	Decider    Decider
	Authorizer Authorizer
	Actor      Actor
	Verifier   Verifier
	Recoverer  Recoverer
	Limits     EngineLimits
	OnEvent    func(EngineEvent)
}

func (e *Engine) Run(ctx context.Context, task string) (*AgentResult, error) {
	if e == nil || e.Decider == nil {
		return nil, errors.New("agent engine decider is required")
	}
	task = strings.TrimSpace(task)
	if task == "" {
		return nil, errors.New("agent task is empty")
	}
	limits := normalizeEngineLimits(e.Limits)
	runCtx, cancel := context.WithTimeout(ctx, limits.Timeout)
	defer cancel()

	started := time.Now()
	state := &AgentState{
		ID:        fmt.Sprintf("agent_%d", started.UnixNano()),
		Phase:     PhaseObserve,
		Task:      task,
		StartedAt: started,
	}
	e.emit(EngineEvent{Phase: PhaseObserve})
	if err := e.transition(state, PhaseDecide, 0); err != nil {
		return e.invalidStateResult(state, started, err), nil
	}

	var usage Usage
	var lastResult *ToolResult
	toolCalls := 0
	recoveries := 0
	provider, model := "", ""

	for step := 1; step <= limits.MaxSteps; step++ {
		if reason, err := contextStop(runCtx); err != nil {
			return e.stop(state, started, usage, provider, model, reason, err.Error()), nil
		}
		decision, err := e.Decider.Decide(runCtx, DecisionInput{
			Task:       task,
			Step:       step,
			Steps:      append([]AgentStep(nil), state.Steps...),
			LastResult: lastResult,
		})
		usage = addUsage(usage, decision.Usage)
		provider, model = preferNonEmpty(decision.Provider, provider), preferNonEmpty(decision.Model, model)
		if usage.TotalTokens() > limits.MaxTokens {
			return e.stop(state, started, usage, provider, model, StopBudgetExhausted, "token budget exhausted"), nil
		}
		if err != nil {
			if !e.tryRecover(runCtx, state, Failure{Phase: PhaseDecide, Err: err, Attempt: recoveries + 1}, &recoveries, limits, step) {
				return e.stop(state, started, usage, provider, model, StopProviderFailed, err.Error()), nil
			}
			continue
		}
		if err := decision.Validate(); err != nil {
			if !e.tryRecover(runCtx, state, Failure{Phase: PhaseDecide, Err: err, Attempt: recoveries + 1}, &recoveries, limits, step) {
				return e.stop(state, started, usage, provider, model, StopInvalidState, err.Error()), nil
			}
			continue
		}

		if decision.Kind == DecisionRespond {
			if err := e.transition(state, PhaseVerify, step); err != nil {
				return e.invalidStateResult(state, started, err), nil
			}
			verification, verifyErr := e.verify(runCtx, VerificationInput{Task: task, Response: decision.Response})
			if verifyErr != nil || !verification.Passed {
				err := verifyErr
				if err == nil {
					err = errors.New(verification.Reason)
				}
				if !e.tryRecover(runCtx, state, Failure{Phase: PhaseVerify, Err: err, Attempt: recoveries + 1}, &recoveries, limits, step) {
					return e.stop(state, started, usage, provider, model, StopVerificationFailed, err.Error()), nil
				}
				continue
			}
			state.Steps = append(state.Steps, AgentStep{Number: step, Phase: PhaseVerify, Action: "respond", Result: decision.Response, Success: true, Duration: time.Since(started)})
			result := e.stop(state, started, usage, provider, model, StopCompleted, "")
			result.Response = decision.Response
			return result, nil
		}

		if toolCalls >= limits.MaxToolCalls {
			return e.stop(state, started, usage, provider, model, StopBudgetExhausted, "tool-call budget exhausted"), nil
		}
		toolCalls++
		usage.ToolCalls = toolCalls
		call := decision.ToolCall
		if err := e.transition(state, PhaseAuthorize, step); err != nil {
			return e.invalidStateResult(state, started, err), nil
		}
		authorization, err := e.authorize(runCtx, *call)
		if err != nil {
			if !e.tryRecover(runCtx, state, Failure{Phase: PhaseAuthorize, Err: err, ToolCall: call, Attempt: recoveries + 1}, &recoveries, limits, step) {
				return e.stop(state, started, usage, provider, model, StopSafetyBlocked, err.Error()), nil
			}
			continue
		}
		if !authorization.Allowed {
			reason := stopReasonForAuthorization(authorization)
			return e.stop(state, started, usage, provider, model, reason, authorization.Reason), nil
		}
		if e.Actor == nil {
			return e.stop(state, started, usage, provider, model, StopSafetyBlocked, "agent actor is not configured"), nil
		}
		if err := e.transition(state, PhaseAct, step); err != nil {
			return e.invalidStateResult(state, started, err), nil
		}
		actCtx, actCancel := context.WithTimeout(runCtx, call.Timeout)
		toolResult, actErr := e.Actor.Act(actCtx, *call, authorization)
		actContextErr := actCtx.Err()
		actCancel()
		if actErr != nil || actContextErr != nil {
			err := actErr
			if actContextErr != nil {
				err = actContextErr
			}
			if !e.tryRecover(runCtx, state, Failure{Phase: PhaseAct, Err: err, ToolCall: call, ToolResult: &toolResult, Attempt: recoveries + 1}, &recoveries, limits, step) {
				reason := StopToolFailed
				if errors.Is(err, context.DeadlineExceeded) {
					reason = StopTimeout
				}
				return e.stop(state, started, usage, provider, model, reason, err.Error()), nil
			}
			lastResult = &toolResult
			continue
		}
		if err := toolResult.Validate(); err != nil {
			if !e.tryRecover(runCtx, state, Failure{Phase: PhaseAct, Err: err, ToolCall: call, ToolResult: &toolResult, Attempt: recoveries + 1}, &recoveries, limits, step) {
				return e.stop(state, started, usage, provider, model, StopToolFailed, err.Error()), nil
			}
			lastResult = &toolResult
			continue
		}
		if err := e.transition(state, PhaseVerify, step); err != nil {
			return e.invalidStateResult(state, started, err), nil
		}
		verification, verifyErr := e.verify(runCtx, VerificationInput{Task: task, ToolCall: call, ToolResult: &toolResult})
		if verifyErr != nil || !verification.Passed {
			err := verifyErr
			if err == nil {
				err = errors.New(verification.Reason)
			}
			if !e.tryRecover(runCtx, state, Failure{Phase: PhaseVerify, Err: err, ToolCall: call, ToolResult: &toolResult, Attempt: recoveries + 1}, &recoveries, limits, step) {
				return e.stop(state, started, usage, provider, model, StopVerificationFailed, err.Error()), nil
			}
			lastResult = &toolResult
			continue
		}
		state.Steps = append(state.Steps, AgentStep{Number: step, Phase: PhaseVerify, Action: call.Tool, Result: toolResult.Output, ToolCall: call, ToolResult: &toolResult, Success: true, Duration: time.Since(started)})
		lastResult = &toolResult
		recoveries = 0
		if err := e.transition(state, PhaseDecide, step); err != nil {
			return e.invalidStateResult(state, started, err), nil
		}
	}
	return e.stop(state, started, usage, provider, model, StopMaxSteps, "maximum Agent steps reached"), nil
}

func (e *Engine) authorize(ctx context.Context, call ToolCall) (Authorization, error) {
	if e.Authorizer != nil {
		return e.Authorizer.Authorize(ctx, call)
	}
	if call.Approval == ApprovalAuto && call.Risk == RiskRead {
		return Authorization{Allowed: true, State: ApprovalApproved}, nil
	}
	return Authorization{Allowed: false, State: ApprovalRequested, Reason: "explicit approval is required"}, nil
}

func (e *Engine) verify(ctx context.Context, input VerificationInput) (Verification, error) {
	if e.Verifier != nil {
		return e.Verifier.Verify(ctx, input)
	}
	if input.ToolResult != nil && input.ToolResult.Status != ToolSucceeded {
		return Verification{Passed: false, Reason: "tool result did not succeed"}, nil
	}
	return Verification{Passed: true}, nil
}

func (e *Engine) tryRecover(ctx context.Context, state *AgentState, failure Failure, recoveries *int, limits EngineLimits, step int) bool {
	if e.Recoverer == nil || *recoveries >= limits.MaxRecoveries {
		return false
	}
	recovery := e.Recoverer.Recover(ctx, failure)
	if !recovery.Retry {
		if recovery.StopReason.Valid() && recovery.StopReason != StopNone {
			state.StopReason = recovery.StopReason
		}
		return false
	}
	*recoveries++
	if err := e.transition(state, PhaseRecover, step); err != nil {
		return false
	}
	return e.transition(state, PhaseDecide, step) == nil
}

func (e *Engine) transition(state *AgentState, next AgentPhase, step int) error {
	if !CanTransition(state.Phase, next) {
		return fmt.Errorf("invalid Agent phase transition: %s -> %s", state.Phase, next)
	}
	state.Phase = next
	e.emit(EngineEvent{Phase: next, Step: step})
	return nil
}

func (e *Engine) stop(state *AgentState, started time.Time, usage Usage, provider, model string, reason StopReason, message string) *AgentResult {
	if state.StopReason.Valid() && state.StopReason != StopNone && reason != StopCompleted {
		reason = state.StopReason
	}
	if state.Phase != PhaseStopped && CanTransition(state.Phase, PhaseStopped) {
		state.Phase = PhaseStopped
	}
	state.StopReason = reason
	usage.Elapsed = time.Since(started)
	e.emit(EngineEvent{Phase: PhaseStopped, Step: len(state.Steps), StopReason: reason})
	return &AgentResult{Steps: len(state.Steps), Tokens: usage.TotalTokens(), Duration: usage.Elapsed, Provider: provider, Model: model, StopReason: reason, Usage: usage, Error: message}
}

func (e *Engine) invalidStateResult(state *AgentState, started time.Time, err error) *AgentResult {
	return e.stop(state, started, Usage{}, "", "", StopInvalidState, err.Error())
}

func (e *Engine) emit(event EngineEvent) {
	if e.OnEvent != nil {
		e.OnEvent(event)
	}
}

func normalizeEngineLimits(limits EngineLimits) EngineLimits {
	defaults := DefaultEngineLimits()
	if limits.MaxSteps <= 0 {
		limits.MaxSteps = defaults.MaxSteps
	}
	if limits.MaxToolCalls <= 0 {
		limits.MaxToolCalls = defaults.MaxToolCalls
	}
	if limits.MaxRecoveries < 0 {
		limits.MaxRecoveries = defaults.MaxRecoveries
	}
	if limits.MaxTokens <= 0 {
		limits.MaxTokens = defaults.MaxTokens
	}
	if limits.Timeout <= 0 {
		limits.Timeout = defaults.Timeout
	}
	return limits
}

func addUsage(total, next Usage) Usage {
	total.InputTokens += next.InputTokens
	total.OutputTokens += next.OutputTokens
	total.ProviderCalls += next.ProviderCalls
	return total
}

func preferNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func contextStop(ctx context.Context) (StopReason, error) {
	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return StopTimeout, err
		}
		return StopCancelled, err
	}
	return StopNone, nil
}

func stopReasonForAuthorization(auth Authorization) StopReason {
	switch auth.State {
	case ApprovalRequested:
		return StopNeedsApproval
	case ApprovalRejected:
		return StopRejected
	case ApprovalCancelled:
		return StopCancelled
	case ApprovalExpired:
		return StopTimeout
	default:
		return StopSafetyBlocked
	}
}
