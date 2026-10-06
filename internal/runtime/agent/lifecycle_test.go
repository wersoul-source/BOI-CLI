package agent

import "testing"

func TestAgentPhaseTransitions(t *testing.T) {
	valid := [][2]AgentPhase{
		{PhaseObserve, PhaseDecide},
		{PhaseDecide, PhaseAuthorize},
		{PhaseDecide, PhaseRecover},
		{PhaseAuthorize, PhaseAct},
		{PhaseAct, PhaseVerify},
		{PhaseVerify, PhaseDecide},
		{PhaseAct, PhaseRecover},
		{PhaseRecover, PhaseObserve},
		{PhaseVerify, PhaseStopped},
	}
	for _, transition := range valid {
		if !CanTransition(transition[0], transition[1]) {
			t.Errorf("expected transition %s -> %s", transition[0], transition[1])
		}
	}

	invalid := [][2]AgentPhase{
		{PhaseObserve, PhaseAct},
		{PhaseAuthorize, PhaseVerify},
		{PhaseStopped, PhaseObserve},
	}
	for _, transition := range invalid {
		if CanTransition(transition[0], transition[1]) {
			t.Errorf("unexpected transition %s -> %s", transition[0], transition[1])
		}
	}
}

func TestStopReasonsAreTypedAndValid(t *testing.T) {
	for _, reason := range []StopReason{
		StopCompleted, StopNeedsApproval, StopRejected, StopCancelled,
		StopTimeout, StopMaxSteps, StopBudgetExhausted, StopToolFailed,
		StopProviderFailed, StopVerificationFailed, StopSafetyBlocked,
	} {
		if !reason.Valid() {
			t.Errorf("stop reason %q should be valid", reason)
		}
	}
	if StopReason("made_up").Valid() {
		t.Fatal("unexpected custom stop reason accepted")
	}
}
