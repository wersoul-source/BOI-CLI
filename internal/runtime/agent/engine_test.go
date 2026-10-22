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

func ungrounded(suspect bool) func(string, string) bool {
	return func(string, string) bool { return suspect }
}

func TestEngineChallengesAnAnswerWithoutAnyToolOnce(t *testing.T) {
	decider := &sequenceDecider{decisions: []Decision{
		{Kind: DecisionRespond, Response: "I fixed it"},
		{Kind: DecisionRespond, Response: "I fixed it"},
	}}
	engine := &Engine{Decider: decider, Limits: DefaultEngineLimits(), Recoverer: BoundedRecoverer{}, Ungrounded: ungrounded(true)}
	result, err := engine.Run(context.Background(), "fix the bug")
	if err != nil {
		t.Fatal(err)
	}
	if decider.calls != 2 || decider.inputs[1].LastError != unevidencedReason {
		t.Fatalf("the model was not asked again: calls=%d inputs=%+v", decider.calls, decider.inputs)
	}
	if result.StopReason != StopCompleted || result.Response != "I fixed it" || result.Notice != unevidencedNotice {
		t.Fatalf("a repeated answer must complete with the notice: %#v", result)
	}
}

func TestEngineAcceptsAToolCallAfterTheChallenge(t *testing.T) {
	decider := &sequenceDecider{decisions: []Decision{
		{Kind: DecisionRespond, Response: "done"},
		{Kind: DecisionUseTool, ToolCall: readCall()},
		{Kind: DecisionRespond, Response: "fixed"},
	}}
	engine := &Engine{Decider: decider, Limits: DefaultEngineLimits(), Recoverer: BoundedRecoverer{}, Ungrounded: ungrounded(true),
		Actor: actorFunc(func(_ context.Context, call ToolCall, _ Authorization) (ToolResult, error) {
			return successfulToolResult(call.ID), nil
		})}
	result, err := engine.Run(context.Background(), "fix the bug")
	if err != nil {
		t.Fatal(err)
	}
	if result.StopReason != StopCompleted || result.Response != "fixed" || result.Notice != "" || result.Usage.ToolCalls != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestEngineLeavesAnswersAloneWhenNothingIsSuspect(t *testing.T) {
	decider := &sequenceDecider{decisions: []Decision{{Kind: DecisionRespond, Response: "READY"}}}
	engine := &Engine{Decider: decider, Limits: DefaultEngineLimits(), Recoverer: BoundedRecoverer{}, Ungrounded: ungrounded(false)}
	result, err := engine.Run(context.Background(), "say ready")
	if err != nil {
		t.Fatal(err)
	}
	if decider.calls != 1 || result.Notice != "" || result.StopReason != StopCompleted {
		t.Fatalf("calls=%d result=%#v", decider.calls, result)
	}
}

func TestEngineDoesNotChallengeAnAnswerAfterATool(t *testing.T) {
	decider := &sequenceDecider{decisions: []Decision{
		{Kind: DecisionUseTool, ToolCall: readCall()},
		{Kind: DecisionRespond, Response: "answer"},
	}}
	engine := &Engine{Decider: decider, Limits: DefaultEngineLimits(), Recoverer: BoundedRecoverer{}, Ungrounded: ungrounded(true),
		Actor: actorFunc(func(_ context.Context, call ToolCall, _ Authorization) (ToolResult, error) {
			return successfulToolResult(call.ID), nil
		})}
	result, err := engine.Run(context.Background(), "read then answer")
	if err != nil || decider.calls != 2 || result.Notice != "" {
		t.Fatalf("calls=%d result=%#v err=%v", decider.calls, result, err)
	}
}

func writeCall(path string) *ToolCall {
	return &ToolCall{
		ID: "call_w", Tool: "workspace.write", Purpose: "apply the fix",
		Arguments: map[string]any{"path": path, "content": "x"}, Target: path,
		Risk: RiskChange, Approval: ApprovalConfirm, Timeout: time.Second,
	}
}

func succeedingActor() actorFunc {
	return func(_ context.Context, call ToolCall, _ Authorization) (ToolResult, error) {
		result := successfulToolResult(call.ID)
		if call.Risk != RiskRead {
			result.ChangedPaths = []string{call.Target}
		}
		return result, nil
	}
}

type grantAll struct{}

func (grantAll) Authorize(_ context.Context, call ToolCall) (Authorization, error) {
	return Authorization{Allowed: true, State: ApprovalApproved}, nil
}

func TestReadingThenOfferingCodeAsTextIsChallengedUntilItIsApplied(t *testing.T) {
	decider := &sequenceDecider{decisions: []Decision{
		{Kind: DecisionUseTool, ToolCall: readCall()},
		{Kind: DecisionRespond, Response: "Here is the fix:\n```go\nreturn a + b\n```"},
		{Kind: DecisionUseTool, ToolCall: writeCall("calc.go")},
		{Kind: DecisionRespond, Response: "I applied it to calc.go"},
	}}
	engine := &Engine{Decider: decider, Authorizer: grantAll{}, Actor: succeedingActor(), Limits: DefaultEngineLimits(), Recoverer: BoundedRecoverer{}, Ungrounded: ungrounded(true)}
	result, err := engine.Run(context.Background(), "fix calc.go")
	if err != nil {
		t.Fatal(err)
	}
	if decider.inputs[2].LastError != unevidencedReason {
		t.Fatalf("code offered as text was not sent back: %+v", decider.inputs[2])
	}
	if result.StopReason != StopCompleted || result.Notice != "" || result.Outcome != OutcomeChanged || len(result.ChangedPaths) != 1 || result.ChangedPaths[0] != "calc.go" {
		t.Fatalf("result = %#v", result)
	}
}

func TestReadingThenAnsweringInProseIsNotChallenged(t *testing.T) {
	decider := &sequenceDecider{decisions: []Decision{
		{Kind: DecisionUseTool, ToolCall: readCall()},
		{Kind: DecisionRespond, Response: "calc.go adds two numbers."},
	}}
	engine := &Engine{Decider: decider, Actor: succeedingActor(), Limits: DefaultEngineLimits(), Recoverer: BoundedRecoverer{}, Ungrounded: ungrounded(true)}
	result, err := engine.Run(context.Background(), "what does calc.go do?")
	if err != nil || decider.calls != 2 || result.Notice != "" || result.Outcome != OutcomeNoEffect || len(result.ChangedPaths) != 0 {
		t.Fatalf("calls=%d result=%#v err=%v", decider.calls, result, err)
	}
}

func TestOutcomeOfAnAnswerWithoutToolsIsAnswered(t *testing.T) {
	engine := &Engine{Decider: &sequenceDecider{decisions: []Decision{{Kind: DecisionRespond, Response: "hello"}}}, Limits: DefaultEngineLimits()}
	result, err := engine.Run(context.Background(), "hi")
	if err != nil || result.Outcome != OutcomeAnswered {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestAFailedChangeIsNotAnEffect(t *testing.T) {
	decider := &sequenceDecider{decisions: []Decision{
		{Kind: DecisionUseTool, ToolCall: writeCall("calc.go")},
		{Kind: DecisionRespond, Response: "done, calc.go is fixed"},
		{Kind: DecisionRespond, Response: "done, calc.go is fixed"},
	}}
	failing := actorFunc(func(_ context.Context, call ToolCall, _ Authorization) (ToolResult, error) {
		return ToolResult{CallID: call.ID}, errors.New("disk is read-only")
	})
	engine := &Engine{Decider: decider, Authorizer: grantAll{}, Actor: failing, Limits: DefaultEngineLimits(), Recoverer: recovererFunc(func(context.Context, Failure) Recovery {
		return Recovery{Retry: true, Reason: "retry"}
	}), Ungrounded: ungrounded(true)}
	result, err := engine.Run(context.Background(), "fix calc.go")
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome == OutcomeChanged || len(result.ChangedPaths) != 0 {
		t.Fatalf("a change that failed was counted: %#v", result)
	}
}
