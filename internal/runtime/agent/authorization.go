package agent

import (
	"context"
	"fmt"
	"time"
)

type ApprovalEvent struct {
	Request   ApprovalRequest
	Decisions chan ApprovalDecision
}

type InteractiveAuthorizer struct {
	Emit func(ApprovalEvent) error
	TTL  time.Duration
}

func (a *InteractiveAuthorizer) Authorize(ctx context.Context, call ToolCall) (Authorization, error) {
	if call.Risk == RiskRead && call.Approval == ApprovalAuto {
		return Authorization{Allowed: true, State: ApprovalApproved}, nil
	}
	if call.Approval == ApprovalDenied {
		return Authorization{Allowed: false, State: ApprovalRejected, Reason: "capability denied by host policy"}, nil
	}
	ttl := a.TTL
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	now := time.Now()
	request, err := NewApprovalRequest("approval_"+call.ID, call, now, now.Add(ttl))
	if err != nil {
		return Authorization{}, err
	}
	decisions := make(chan ApprovalDecision, 1)
	if a.Emit == nil {
		return Authorization{Allowed: false, State: ApprovalRequested, Request: &request, Reason: "interactive approval is unavailable"}, nil
	}
	if err := a.Emit(ApprovalEvent{Request: request, Decisions: decisions}); err != nil {
		return Authorization{}, err
	}
	timer := time.NewTimer(ttl)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return Authorization{}, ctx.Err()
	case decision := <-decisions:
		if err := decision.Validate(); err != nil {
			return Authorization{}, err
		}
		if decision.RequestID != request.ID {
			return Authorization{}, fmt.Errorf("approval decision does not match request")
		}
		allowed := decision.State == ApprovalApproved
		return Authorization{Allowed: allowed, State: decision.State, Request: &request, Decision: &decision, Reason: decision.Reason}, nil
	case <-timer.C:
		return Authorization{Allowed: false, State: ApprovalExpired, Request: &request, Reason: "approval request expired"}, nil
	}
}

type RejectingAuthorizer struct{}

func (RejectingAuthorizer) Authorize(_ context.Context, call ToolCall) (Authorization, error) {
	if call.Risk == RiskRead && call.Approval == ApprovalAuto {
		return Authorization{Allowed: true, State: ApprovalApproved}, nil
	}
	return Authorization{Allowed: false, State: ApprovalRequested, Reason: "interactive approval required"}, nil
}
