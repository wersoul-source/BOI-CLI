package agent

type AgentPhase string

const (
	PhaseObserve   AgentPhase = "observe"
	PhaseDecide    AgentPhase = "decide"
	PhaseAuthorize AgentPhase = "authorize"
	PhaseAct       AgentPhase = "act"
	PhaseVerify    AgentPhase = "verify"
	PhaseRecover   AgentPhase = "recover"
	PhaseStopped   AgentPhase = "stopped"
)

type StopReason string

const (
	StopNone               StopReason = ""
	StopCompleted          StopReason = "completed"
	StopNeedsApproval      StopReason = "needs_approval"
	StopRejected           StopReason = "rejected"
	StopCancelled          StopReason = "cancelled"
	StopTimeout            StopReason = "timeout"
	StopMaxSteps           StopReason = "max_steps"
	StopBudgetExhausted    StopReason = "budget_exhausted"
	StopToolFailed         StopReason = "tool_failed"
	StopProviderFailed     StopReason = "provider_failed"
	StopVerificationFailed StopReason = "verification_failed"
	StopSafetyBlocked      StopReason = "safety_blocked"
	StopInvalidState       StopReason = "invalid_state"
)

func (r StopReason) Valid() bool {
	switch r {
	case StopNone, StopCompleted, StopNeedsApproval, StopRejected, StopCancelled,
		StopTimeout, StopMaxSteps, StopBudgetExhausted, StopToolFailed,
		StopProviderFailed, StopVerificationFailed, StopSafetyBlocked, StopInvalidState:
		return true
	default:
		return false
	}
}

func CanTransition(from, to AgentPhase) bool {
	if to == PhaseStopped {
		return from != PhaseStopped
	}
	switch from {
	case PhaseObserve:
		return to == PhaseDecide
	case PhaseDecide:
		return to == PhaseAuthorize || to == PhaseVerify
	case PhaseAuthorize:
		return to == PhaseAct || to == PhaseRecover
	case PhaseAct:
		return to == PhaseVerify || to == PhaseRecover
	case PhaseVerify:
		return to == PhaseDecide || to == PhaseRecover
	case PhaseRecover:
		return to == PhaseObserve || to == PhaseDecide || to == PhaseAuthorize
	default:
		return false
	}
}
