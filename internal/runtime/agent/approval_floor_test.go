package agent

import (
	"context"
	"strings"
	"testing"
	"time"
)

// These tests are the evidence behind ApprovalFloorEnforced (ADR-006, D1 and
// "How D1 meets Rule 3"). Removing or weakening one means that constant must
// become false.

func flooredBroker(t *testing.T) (brokerFixture, *fakeTool) {
	t.Helper()
	fixture := testBroker(t)
	fake := &fakeTool{name: "probe.read"}
	if err := fixture.Register(fake); err != nil {
		t.Fatal(err)
	}
	if err := fixture.SetActiveCapabilities(append(fixture.CapabilityNames(), fake.name)); err != nil {
		t.Fatal(err)
	}
	fixture.SetApprovalFloor(true)
	return fixture, fake
}

var automaticCalls = []ToolCall{
	{ID: "c1", Tool: "probe.read", Purpose: "p", Arguments: map[string]any{}},
	{ID: "c2", Tool: "workspace.list", Purpose: "p", Arguments: map[string]any{"path": "."}},
	{ID: "c3", Tool: "workspace.read", Purpose: "p", Arguments: map[string]any{"path": "a.txt"}},
	{ID: "c4", Tool: "workspace.search", Purpose: "p", Arguments: map[string]any{"pattern": "x"}},
}

// With the floor on, no call is automatic. Without it, the same calls are.
func TestApprovalFloorRaisesEveryAutomaticCall(t *testing.T) {
	fixture, _ := flooredBroker(t)
	for _, proposed := range automaticCalls {
		floored, err := fixture.Prepare(proposed)
		if err != nil {
			t.Fatalf("%s: %v", proposed.Tool, err)
		}
		if floored.Approval == ApprovalAuto {
			t.Errorf("%s is still automatic under the floor", proposed.Tool)
		}
		if floored.IdempotencyKey == "" {
			t.Errorf("%s needs approval but has no idempotency key", proposed.Tool)
		}
	}
	fixture.SetApprovalFloor(false)
	for _, proposed := range automaticCalls {
		plain, err := fixture.Prepare(proposed)
		if err != nil || plain.Approval != ApprovalAuto {
			t.Errorf("%s without the floor: approval=%s err=%v", proposed.Tool, plain.Approval, err)
		}
	}
}

// The floor only raises: a Tool that already needs approval keeps its class.
func TestApprovalFloorNeverLowersAClass(t *testing.T) {
	fixture, _ := flooredBroker(t)
	write := ToolCall{ID: "w1", Tool: "workspace.write", Purpose: "p", Arguments: map[string]any{"path": "n.txt", "content": "x"}}
	floored, err := fixture.Prepare(write)
	if err != nil {
		t.Fatal(err)
	}
	fixture.SetApprovalFloor(false)
	plain, err := fixture.Prepare(write)
	if err != nil {
		t.Fatal(err)
	}
	if floored.Approval != plain.Approval || floored.Risk != plain.Risk {
		t.Fatalf("the floor changed a non-automatic call: %s/%s vs %s/%s", floored.Approval, floored.Risk, plain.Approval, plain.Risk)
	}
}

// A floored call does not run on a bare "allowed": it needs the approval
// request a person answered, for exactly that call.
func TestFlooredCallDoesNotRunWithoutAnApprovalRequest(t *testing.T) {
	fixture, fake := flooredBroker(t)
	prepared, err := fixture.Prepare(automaticCalls[0])
	if err != nil {
		t.Fatal(err)
	}
	for name, authorization := range map[string]Authorization{
		"nothing":            {},
		"allowed, no ticket": {Allowed: true, State: ApprovalApproved},
	} {
		if _, err := fixture.Act(context.Background(), prepared, authorization); err == nil {
			t.Errorf("%s: the call ran", name)
		}
	}
	other, err := fixture.Prepare(ToolCall{ID: "other", Tool: "probe.read", Purpose: "different", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	foreign, err := NewApprovalRequest("approval_other", other, now, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.Act(context.Background(), prepared, Authorization{Allowed: true, State: ApprovalApproved, Request: &foreign}); err == nil {
		t.Error("an approval for another call ran this one")
	}
	if fake.calls != 0 {
		t.Fatalf("the Tool executed %d time(s) without approval", fake.calls)
	}
	request, err := NewApprovalRequest("approval_c1", prepared, now, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.Act(context.Background(), prepared, Authorization{Allowed: true, State: ApprovalApproved, Request: &request}); err != nil || fake.calls != 1 {
		t.Fatalf("an approved floored call must run once: err=%v calls=%d", err, fake.calls)
	}
}

// A call prepared before the floor came on cannot be used after it: the host
// prepares it again and the two no longer match.
func TestACallPreparedBeforeTheFloorCannotSlipUnderIt(t *testing.T) {
	fixture, fake := flooredBroker(t)
	fixture.SetApprovalFloor(false)
	automatic, err := fixture.Prepare(automaticCalls[0])
	if err != nil || automatic.Approval != ApprovalAuto {
		t.Fatalf("setup: %v %s", err, automatic.Approval)
	}
	fixture.SetApprovalFloor(true)
	if _, err := fixture.Act(context.Background(), automatic, Authorization{Allowed: true, State: ApprovalApproved}); err == nil || fake.calls != 0 {
		t.Fatalf("a pre-floor automatic call ran under the floor: err=%v calls=%d", err, fake.calls)
	}
}

// D2: a run that cannot ask refuses every floored call.
func TestUnattendedAuthorizerRefusesEveryFlooredCall(t *testing.T) {
	fixture, _ := flooredBroker(t)
	for _, proposed := range automaticCalls {
		prepared, err := fixture.Prepare(proposed)
		if err != nil {
			t.Fatal(err)
		}
		authorization, err := RejectingAuthorizer{}.Authorize(context.Background(), prepared)
		if err != nil || authorization.Allowed {
			t.Errorf("%s was allowed without a person: %+v err=%v", proposed.Tool, authorization, err)
		}
		if !strings.Contains(authorization.Reason, "approval") {
			t.Errorf("%s: the refusal does not say why: %q", proposed.Tool, authorization.Reason)
		}
	}
}

// The interactive authorizer asks for every floored call, reads included.
func TestInteractiveAuthorizerAsksForEveryFlooredCall(t *testing.T) {
	fixture, _ := flooredBroker(t)
	asked := 0
	authorizer := &InteractiveAuthorizer{Emit: func(event ApprovalEvent) error {
		asked++
		event.Decisions <- ApprovalDecision{RequestID: event.Request.ID, State: ApprovalApproved, DecidedAt: time.Now()}
		return nil
	}}
	for _, proposed := range automaticCalls {
		prepared, err := fixture.Prepare(proposed)
		if err != nil {
			t.Fatal(err)
		}
		if authorization, err := authorizer.Authorize(context.Background(), prepared); err != nil || !authorization.Allowed || authorization.Request == nil {
			t.Fatalf("%s: %+v err=%v", proposed.Tool, authorization, err)
		}
	}
	if asked != len(automaticCalls) {
		t.Fatalf("asked %d time(s) for %d calls", asked, len(automaticCalls))
	}
}
