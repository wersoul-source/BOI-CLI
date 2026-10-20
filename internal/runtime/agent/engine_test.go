package agent

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type sequenceDecider struct {
	decisions []Decision
	errors    []error
	inputs    []DecisionInput
	calls     int
}

func (d *sequenceDecider) Decide(_ context.Context, input DecisionInput) (Decision, error) {
	d.inputs = append(d.inputs, input)
	index := d.calls
	d.calls++
	var decision Decision
	if index < len(d.decisions) {
		decision = d.decisions[index]
	}
	if index < len(d.errors) {
		return decision, d.errors[index]
	}
	return decision, nil
}

type actorFunc func(context.Context, ToolCall, Authorization) (ToolResult, error)

func (f actorFunc) Act(ctx context.Context, call ToolCall, auth Authorization) (ToolResult, error) {
	return f(ctx, call, auth)
}

type verifierFunc func(context.Context, VerificationInput) (Verification, error)

func (f verifierFunc) Verify(ctx context.Context, input VerificationInput) (Verification, error) {
	return f(ctx, input)
}

type recovererFunc func(context.Context, Failure) Recovery

func (f recovererFunc) Recover(ctx context.Context, failure Failure) Recovery {
	return f(ctx, failure)
}

func readCall() *ToolCall {
	return &ToolCall{
		ID: "call_001", Tool: "workspace.read", Purpose: "inspect configuration",
		Arguments: map[string]any{"path": "config.yaml"}, Target: "config.yaml",
		Risk: RiskRead, Approval: ApprovalAuto, Timeout: time.Second,
	}
}

func successfulToolResult(callID string) ToolResult {
	now := time.Now()
	return ToolResult{CallID: callID, Status: ToolSucceeded, Output: "observed", StartedAt: now, FinishedAt: now.Add(time.Millisecond)}
}

func TestEngineCompletesDirectResponse(t *testing.T) {
	var phases []AgentPhase
	engine := &Engine{
		Decider: &sequenceDecider{decisions: []Decision{{
			Kind: DecisionRespond, Response: "done", Provider: "test", Model: "model",
			Usage: Usage{InputTokens: 2, OutputTokens: 3, ProviderCalls: 1},
		}}},
		Limits: DefaultEngineLimits(),
		OnEvent: func(event EngineEvent) {
			if event.Kind == "task" || event.Kind == "phase" || event.Kind == "stop" {
				phases = append(phases, event.Phase)
			}
		},
	}
	result, err := engine.Run(context.Background(), "answer this")
	if err != nil {
		t.Fatalf("run engine: %v", err)
	}
	if result.StopReason != StopCompleted || result.Response != "done" || result.Tokens != 5 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Plan == nil || result.Plan.Status != PlanCompleted {
		t.Fatalf("Task Plan not completed: %#v", result.Plan)
	}
	want := []AgentPhase{PhaseObserve, PhaseDecide, PhaseVerify, PhaseStopped}
	if !reflect.DeepEqual(phases, want) {
		t.Fatalf("phases = %v, want %v", phases, want)
	}
}

func TestEngineRunsAuthorizedToolThenResponds(t *testing.T) {
	call := readCall()
	decider := &sequenceDecider{decisions: []Decision{
		{Kind: DecisionUseTool, ToolCall: call},
		{Kind: DecisionRespond, Response: "verified answer"},
	}}
	actorCalls := 0
	engine := &Engine{
		Decider: decider,
		Actor: actorFunc(func(_ context.Context, got ToolCall, _ Authorization) (ToolResult, error) {
			actorCalls++
			return successfulToolResult(got.ID), nil
		}),
		Limits: DefaultEngineLimits(),
	}
	result, err := engine.Run(context.Background(), "inspect then answer")
	if err != nil {
		t.Fatalf("run engine: %v", err)
	}
	if result.StopReason != StopCompleted || actorCalls != 1 || result.Usage.ToolCalls != 1 {
		t.Fatalf("result=%#v actorCalls=%d", result, actorCalls)
	}
}

func TestEngineStopsForApprovalBeforeAction(t *testing.T) {
	call := readCall()
	call.Risk = RiskChange
	call.Approval = ApprovalConfirm
	actorCalled := false
	engine := &Engine{
		Decider: &sequenceDecider{decisions: []Decision{{Kind: DecisionUseTool, ToolCall: call}}},
		Actor: actorFunc(func(context.Context, ToolCall, Authorization) (ToolResult, error) {
			actorCalled = true
			return ToolResult{}, nil
		}),
		Limits: DefaultEngineLimits(),
	}
	result, err := engine.Run(context.Background(), "change file")
	if err != nil {
		t.Fatalf("run engine: %v", err)
	}
	if result.StopReason != StopNeedsApproval || actorCalled {
		t.Fatalf("result=%#v actorCalled=%v", result, actorCalled)
	}
}

func TestEngineRecoversFromToolFailure(t *testing.T) {
	call := readCall()
	decider := &sequenceDecider{decisions: []Decision{
		{Kind: DecisionUseTool, ToolCall: call},
		{Kind: DecisionUseTool, ToolCall: call},
		{Kind: DecisionRespond, Response: "recovered"},
	}}
	actorCalls := 0
	engine := &Engine{
		Decider: decider,
		Actor: actorFunc(func(_ context.Context, got ToolCall, _ Authorization) (ToolResult, error) {
			actorCalls++
			if actorCalls == 1 {
				return ToolResult{CallID: got.ID, Status: ToolFailed}, errors.New("temporary failure")
			}
			return successfulToolResult(got.ID), nil
		}),
		Recoverer: recovererFunc(func(_ context.Context, failure Failure) Recovery {
			return Recovery{Retry: failure.Attempt == 1}
		}),
		Limits: DefaultEngineLimits(),
	}
	result, err := engine.Run(context.Background(), "recover")
	if err != nil {
		t.Fatalf("run engine: %v", err)
	}
	if result.StopReason != StopCompleted || actorCalls != 2 {
		t.Fatalf("result=%#v actorCalls=%d", result, actorCalls)
	}
	if result.Plan == nil || result.Plan.Revision != 2 {
		t.Fatalf("recovery did not revise Plan: %#v", result.Plan)
	}
	observed := decider.inputs[1].LastResult
	if observed == nil || observed.Status != ToolFailed || observed.ErrorClass != "execution" || observed.ErrorMessage != "temporary failure" || observed.FinishedAt.IsZero() {
		t.Fatalf("tool failure observation was not normalized: %#v", observed)
	}
}

func TestEngineEnforcesBudgets(t *testing.T) {
	t.Run("tokens", func(t *testing.T) {
		engine := &Engine{
			Decider: &sequenceDecider{decisions: []Decision{{Kind: DecisionRespond, Response: "too expensive", Usage: Usage{InputTokens: 8, OutputTokens: 8}}}},
			Limits:  EngineLimits{MaxSteps: 2, MaxToolCalls: 1, MaxTokens: 10, Timeout: time.Second},
		}
		result, _ := engine.Run(context.Background(), "budget")
		if result.StopReason != StopBudgetExhausted {
			t.Fatalf("stop = %s, want budget_exhausted", result.StopReason)
		}
	})
	t.Run("steps", func(t *testing.T) {
		call := readCall()
		engine := &Engine{
			Decider: &sequenceDecider{decisions: []Decision{{Kind: DecisionUseTool, ToolCall: call}, {Kind: DecisionUseTool, ToolCall: call}}},
			Actor: actorFunc(func(_ context.Context, got ToolCall, _ Authorization) (ToolResult, error) {
				return successfulToolResult(got.ID), nil
			}),
			Limits: EngineLimits{MaxSteps: 2, MaxToolCalls: 3, MaxTokens: 100, Timeout: time.Second},
		}
		result, _ := engine.Run(context.Background(), "loop")
		if result.StopReason != StopMaxSteps {
			t.Fatalf("stop = %s, want max_steps", result.StopReason)
		}
	})
}

func TestEngineHonorsCancellationAndToolTimeout(t *testing.T) {
	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		engine := &Engine{Decider: &sequenceDecider{}, Limits: DefaultEngineLimits()}
		result, _ := engine.Run(ctx, "cancel")
		if result.StopReason != StopCancelled {
			t.Fatalf("stop = %s, want cancelled", result.StopReason)
		}
	})
	t.Run("tool timeout", func(t *testing.T) {
		call := readCall()
		call.Timeout = 10 * time.Millisecond
		engine := &Engine{
			Decider: &sequenceDecider{decisions: []Decision{{Kind: DecisionUseTool, ToolCall: call}}},
			Actor: actorFunc(func(ctx context.Context, _ ToolCall, _ Authorization) (ToolResult, error) {
				<-ctx.Done()
				return ToolResult{CallID: call.ID, Status: ToolTimedOut}, ctx.Err()
			}),
			Limits: DefaultEngineLimits(),
		}
		result, _ := engine.Run(context.Background(), "timeout")
		if result.StopReason != StopTimeout {
			t.Fatalf("stop = %s, want timeout", result.StopReason)
		}
	})
}

func TestEngineVerificationFailureStops(t *testing.T) {
	engine := &Engine{
		Decider: &sequenceDecider{decisions: []Decision{{Kind: DecisionRespond, Response: "unsupported"}}},
		Verifier: verifierFunc(func(context.Context, VerificationInput) (Verification, error) {
			return Verification{Passed: false, Reason: "missing evidence"}, nil
		}),
		Limits: DefaultEngineLimits(),
	}
	result, _ := engine.Run(context.Background(), "verify")
	if result.StopReason != StopVerificationFailed {
		t.Fatalf("result = %#v", result)
	}
}
