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
	DecisionRespond  DecisionKind = "respond"
	DecisionUseTool  DecisionKind = "use_tool"
	DecisionUseSkill DecisionKind = "use_skill"
)

type DecisionInput struct {
	Task       string
	Step       int
	Steps      []AgentStep
	LastResult *ToolResult
	LastSkill  *SkillObservation
	Plan       *TaskPlan
}

type Decision struct {
	Kind      DecisionKind
	Response  string
	ToolCall  *ToolCall
	SkillName string
	Usage     Usage
	Provider  string
	Model     string
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
		if d.SkillName != "" {
			return errors.New("response decision must not contain a Skill")
		}
	case DecisionUseTool:
		if d.ToolCall == nil {
			return errors.New("tool decision is missing a tool call")
		}
		if err := d.ToolCall.Validate(); err != nil {
			return fmt.Errorf("tool decision: %w", err)
		}
		if d.SkillName != "" {
			return errors.New("Tool decision must not contain a Skill")
		}
	case DecisionUseSkill:
		if strings.TrimSpace(d.SkillName) == "" || d.ToolCall != nil {
			return errors.New("Skill decision requires only a Skill name")
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

type SkillLoader interface{ LoadSkill(string) (string, error) }
type SkillObservation struct {
	Name         string
	Instructions string
}

type EngineEvent struct {
	Kind         string
	Phase        AgentPhase
	Step         int
	ToolCall     *ToolCall
	ToolResult   *ToolResult
	StopReason   StopReason
	Verification *Verification
	Plan         *TaskPlan
}

type EngineLimits struct {
	MaxSteps      int
	MaxToolCalls  int
	MaxRecoveries int
	MaxSkillCalls int
	MaxTokens     int
	Timeout       time.Duration
}

func DefaultEngineLimits() EngineLimits {
	return EngineLimits{
		MaxSteps:      12,
		MaxToolCalls:  8,
		MaxRecoveries: 2,
		MaxSkillCalls: 4,
		MaxTokens:     64_000,
		Timeout:       2 * time.Minute,
	}
}

type Engine struct {
	Decider     Decider
	Authorizer  Authorizer
	Actor       Actor
	Verifier    Verifier
	Recoverer   Recoverer
	Limits      EngineLimits
	OnEvent     func(EngineEvent)
	Plan        *TaskPlan
	SkillLoader SkillLoader
	TaskID      string
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
	plan := e.Plan
	if plan == nil {
		plan = NewPlanner().Plan(task)
	}
	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("validate Task Plan: %w", err)
	}
	plan.Status = PlanRunning
	setPlanPhase(plan, PhaseObserve, PlanCompleted)
	taskID := strings.TrimSpace(e.TaskID)
	if taskID == "" {
		taskID = fmt.Sprintf("agent_%d", started.UnixNano())
	}
	state := &AgentState{
		ID:        taskID,
		Phase:     PhaseObserve,
		Task:      task,
		StartedAt: started,
		Plan:      plan,
	}
	e.emit(EngineEvent{Kind: "task", Phase: PhaseObserve, Plan: plan})
	if err := e.transition(state, PhaseDecide, 0); err != nil {
		return e.invalidStateResult(state, started, err), nil
	}

	var usage Usage
	var lastResult *ToolResult
	var lastSkill *SkillObservation
	toolCalls := 0
	skillCalls := 0
	recoveries := 0
	provider, model := "", ""

	for step := 1; step <= limits.MaxSteps; step++ {
		if reason, err := contextStop(runCtx); err != nil {
			return e.stop(state, started, usage, provider, model, reason, err.Error()), nil
		}
		setPlanPhase(plan, PhaseDecide, PlanRunning)
		decision, err := e.Decider.Decide(runCtx, DecisionInput{
			Task:       task,
			Step:       step,
			Steps:      append([]AgentStep(nil), state.Steps...),
			LastResult: lastResult,
			LastSkill:  lastSkill,
			Plan:       plan,
		})
		usage = addUsage(usage, decision.Usage)
		provider, model = preferNonEmpty(decision.Provider, provider), preferNonEmpty(decision.Model, model)
		if usage.TotalTokens() > limits.MaxTokens {
			return e.stop(state, started, usage, provider, model, StopBudgetExhausted, "token budget exhausted"), nil
		}
		if err != nil {
			if reason, contextErr := contextStop(runCtx); contextErr != nil {
				return e.stop(state, started, usage, provider, model, reason, contextErr.Error()), nil
			}
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
		setPlanPhase(plan, PhaseDecide, PlanCompleted)
		if decision.Kind == DecisionUseSkill {
			if skillCalls >= limits.MaxSkillCalls {
				return e.stop(state, started, usage, provider, model, StopBudgetExhausted, "Skill-call budget exhausted"), nil
			}
			if e.SkillLoader == nil {
				return e.stop(state, started, usage, provider, model, StopSafetyBlocked, "Skill loader is not configured"), nil
			}
			instructions, loadErr := e.SkillLoader.LoadSkill(decision.SkillName)
			if loadErr != nil {
				return e.stop(state, started, usage, provider, model, StopSafetyBlocked, loadErr.Error()), nil
			}
			skillCalls++
			lastSkill = &SkillObservation{Name: decision.SkillName, Instructions: instructions}
			lastResult = nil
			state.Steps = append(state.Steps, AgentStep{Number: step, Phase: PhaseObserve, Action: "skill:" + decision.SkillName, Result: "active Skill instructions loaded as untrusted context", Success: true, Duration: time.Since(started)})
			if err := e.transition(state, PhaseObserve, step); err != nil {
				return e.invalidStateResult(state, started, err), nil
			}
			setPlanPhase(plan, PhaseObserve, PlanCompleted)
			if err := e.transition(state, PhaseDecide, step); err != nil {
				return e.invalidStateResult(state, started, err), nil
			}
			continue
		}

		if decision.Kind == DecisionRespond {
			if toolCalls == 0 {
				setPlanPhase(plan, PhaseAct, PlanSkipped)
			}
			if err := e.transition(state, PhaseVerify, step); err != nil {
				return e.invalidStateResult(state, started, err), nil
			}
			verification, verifyErr := e.verify(runCtx, VerificationInput{Task: task, Response: decision.Response})
			e.emit(EngineEvent{Kind: "verification", Phase: PhaseVerify, Step: step, Verification: &verification})
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
			setPlanPhase(plan, PhaseVerify, PlanCompleted)
			state.Steps = append(state.Steps, AgentStep{Number: step, Phase: PhaseVerify, Action: "respond", Result: decision.Response, Success: true, Duration: time.Since(started)})
			plan.Status = PlanCompleted
			setPlanPhase(plan, PhaseStopped, PlanCompleted)
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
		setPlanPhase(plan, PhaseAct, PlanRunning)
		e.emit(EngineEvent{Kind: "tool", Phase: PhaseAuthorize, Step: step, ToolCall: call})
		if err := e.transition(state, PhaseAuthorize, step); err != nil {
			return e.invalidStateResult(state, started, err), nil
		}
		authorization, err := e.authorize(runCtx, *call)
		e.emit(EngineEvent{Kind: "approval", Phase: PhaseAuthorize, Step: step, ToolCall: call})
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
		e.emit(EngineEvent{Kind: "tool_result", Phase: PhaseAct, Step: step, ToolCall: call, ToolResult: &toolResult})
		actContextErr := actCtx.Err()
		actCancel()
		if actErr != nil || actContextErr != nil {
			err := actErr
			if actContextErr != nil {
				err = actContextErr
			}
			if reason, contextErr := contextStop(runCtx); contextErr != nil {
				return e.stop(state, started, usage, provider, model, reason, contextErr.Error()), nil
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
		e.emit(EngineEvent{Kind: "verification", Phase: PhaseVerify, Step: step, ToolCall: call, ToolResult: &toolResult, Verification: &verification})
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
		toolResult.Evidence = append(toolResult.Evidence, verification.Evidence...)
		state.Steps = append(state.Steps, AgentStep{Number: step, Phase: PhaseVerify, Action: call.Tool, Result: toolResult.Output, ToolCall: call, ToolResult: &toolResult, Success: true, Duration: time.Since(started)})
		setPlanPhase(plan, PhaseAct, PlanCompleted)
		setPlanPhase(plan, PhaseVerify, PlanCompleted)
		lastResult = &toolResult
		lastSkill = nil
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
	state.Plan.Revise(failure.Phase)
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
	setPlanPhase(state.Plan, next, PlanRunning)
	e.emit(EngineEvent{Kind: "phase", Phase: next, Step: step, Plan: state.Plan})
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
	if reason != StopCompleted {
		state.Plan.Status = PlanFailed
	}
	e.emit(EngineEvent{Kind: "stop", Phase: PhaseStopped, Step: len(state.Steps), StopReason: reason, Plan: state.Plan})
	return &AgentResult{TaskID: state.ID, Steps: len(state.Steps), Tokens: usage.TotalTokens(), Duration: usage.Elapsed, Provider: provider, Model: model, StopReason: reason, Usage: usage, Error: message, Plan: state.Plan, Trace: append([]AgentStep(nil), state.Steps...)}
}

func setPlanPhase(plan *TaskPlan, phase AgentPhase, status PlanStatus) {
	if plan == nil {
		return
	}
	for index := range plan.Steps {
		if plan.Steps[index].Phase == phase {
			plan.Steps[index].Status = status
		}
	}
}

func (e *Engine) invalidStateResult(state *AgentState, started time.Time, err error) *AgentResult {
	return e.stop(state, started, Usage{}, "", "", StopInvalidState, err.Error())
}

func (e *Engine) emit(event EngineEvent) {
	if e.OnEvent != nil {
		e.OnEvent(cloneEngineEvent(event))
	}
}

func cloneEngineEvent(event EngineEvent) EngineEvent {
	if event.Plan != nil {
		plan := *event.Plan
		plan.Steps = append([]PlannedStep(nil), event.Plan.Steps...)
		for index := range plan.Steps {
			plan.Steps[index].DependsOn = append([]int(nil), plan.Steps[index].DependsOn...)
		}
		plan.Revisions = append([]PlanRevision(nil), event.Plan.Revisions...)
		event.Plan = &plan
	}
	if event.ToolCall != nil {
		call := *event.ToolCall
		event.ToolCall = &call
	}
	if event.ToolResult != nil {
		result := *event.ToolResult
		result.ChangedPaths = append([]string(nil), result.ChangedPaths...)
		result.Evidence = append([]Evidence(nil), result.Evidence...)
		event.ToolResult = &result
	}
	if event.Verification != nil {
		verification := *event.Verification
		verification.Evidence = append([]Evidence(nil), event.Verification.Evidence...)
		event.Verification = &verification
	}
	return event
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
	if limits.MaxSkillCalls <= 0 {
		limits.MaxSkillCalls = defaults.MaxSkillCalls
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
