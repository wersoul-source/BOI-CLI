package agent

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEventVersionRule(t *testing.T) {
	cases := []struct {
		reader, event int
		want          bool
	}{
		{1, 1, true}, {1, 0, false}, {1, 2, false},
		{2, 2, true}, {2, 1, true}, {2, 0, false}, {2, 3, false},
		{5, 4, true}, {5, 3, false},
	}
	for _, c := range cases {
		if got := acceptsEventVersion(c.reader, c.event); got != c.want {
			t.Errorf("reader v%d reading v%d = %v, want %v", c.reader, c.event, got, c.want)
		}
	}
}

// A reader never fails on what it does not know: an unknown kind and a schema
// version it cannot read are skipped and listed with the reason.
func TestReadEventsSkipsWhatItCannotRead(t *testing.T) {
	log := strings.Join([]string{
		`{"schema_version":1,"seq":1,"task_id":"t","agent_id":"a","kind":"task.started"}`,
		`{"schema_version":1,"seq":2,"task_id":"t","agent_id":"a","kind":"kind.from.the.future","payload":{"x":1}}`,
		`{"schema_version":2,"seq":3,"task_id":"t","agent_id":"a","kind":"task.stopped"}`,
		`{"seq":4,"task_id":"t","agent_id":"a","kind":"task.stopped"}`,
		``,
		`{"schema_version":1,"seq":5,"task_id":"t","agent_id":"a","kind":"task.stopped","payload":{"stop_reason":"completed"}}`,
	}, "\n")
	got, err := ReadEvents(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Events) != 2 || got.Events[0].Seq != 1 || got.Events[1].Seq != 5 {
		t.Fatalf("events = %+v", got.Events)
	}
	if len(got.Skipped) != 3 {
		t.Fatalf("skipped = %+v", got.Skipped)
	}
	reasons := map[uint64]string{}
	for _, s := range got.Skipped {
		reasons[s.Seq] = s.Reason
	}
	if reasons[2] != "unknown kind" || !strings.Contains(reasons[3], "version 2") || !strings.Contains(reasons[4], "version 0") {
		t.Fatalf("reasons = %v", reasons)
	}
}

func TestReadEventsRejectsAnythingThatIsNotAnEvent(t *testing.T) {
	if _, err := ReadEvents(strings.NewReader("{\"schema_version\":1}\nnot json\n")); err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("err = %v", err)
	}
}

func TestReservedKindsAreKnown(t *testing.T) {
	for _, kind := range []string{EventModelSwitched, EventContextCompacted, EventApprovalDecided, EventEffectRecorded, EventModeChanged, EventCapabilityUsed, EventRouteChanged} {
		if !KnownEventKind(kind) {
			t.Errorf("%s is reserved by the contract but unknown", kind)
		}
	}
	if KnownEventKind("engine.new_thing") {
		t.Error("an engine kind the contract has not named must read as unknown")
	}
}

// An event is metadata. A Tool argument, a Tool output and an error message
// may hold file content or a secret and must never reach an event.
func TestEventsNeverCarryArgumentsOutputOrErrorText(t *testing.T) {
	secret := "sk-live-SECRET"
	call := &ToolCall{ID: "c1", Tool: "process.run", Purpose: "p " + secret, Arguments: map[string]any{"command": "echo " + secret}, Target: secret, Preview: secret, Risk: RiskExecute}
	result := &ToolResult{CallID: "c1", Status: ToolFailed, Output: "out " + secret, ErrorClass: "exit", ErrorMessage: "err " + secret, ChangedPaths: []string{secret}}
	events := []EngineEvent{
		{Kind: "tool", ToolCall: call},
		{Kind: "approval", ToolCall: call},
		{Kind: "tool_result", ToolCall: call, ToolResult: result},
		{Kind: "verification", ToolCall: call, ToolResult: result, Verification: &Verification{Passed: false, Reason: secret}},
		{Kind: "stop", StopReason: StopCompleted},
		{Kind: "something_new", ToolCall: call, ToolResult: result},
	}
	for _, event := range events {
		kind, callID, payload := eventFor(event)
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "SECRET") || strings.Contains(kind, "SECRET") || strings.Contains(callID, "SECRET") {
			t.Errorf("%s leaked: kind=%s payload=%s", event.Kind, kind, raw)
		}
	}
	if kind, _, _ := eventFor(EngineEvent{Kind: "something_new"}); KnownEventKind(kind) {
		t.Errorf("an unmapped engine kind got the known kind %q", kind)
	}
}

func TestRouteChangedReasonIsBounded(t *testing.T) {
	raw, _ := json.Marshal(routeChangedPayload("native→envelope", strings.Repeat("x", 5000)))
	if len(raw) > 400 {
		t.Fatalf("a route.changed payload of %d bytes is not bounded", len(raw))
	}
}

func TestStamperNumbersEventsAndKeepsTheParentLink(t *testing.T) {
	stamper := newEventStamper("task-1", "agent-task-1", "agent-parent")
	fixed := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	stamper.now = func() time.Time { return fixed }
	first, err := stamper.stamp(EventTaskStarted, 0, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := stamper.stamp(EventToolProposed, 1, "call-1", struct{ Tool string }{"workspace.read"})
	if first.Seq != 1 || second.Seq != 2 || second.CallID != "call-1" || second.Step != 1 {
		t.Fatalf("events = %+v %+v", first, second)
	}
	if second.TaskID != "task-1" || second.AgentID != "agent-task-1" || second.ParentAgentID != "agent-parent" || second.SchemaVersion != EventSchemaVersion || !second.Time.Equal(fixed) {
		t.Fatalf("ids = %+v", second)
	}
	if _, err := stamper.stamp(EventTaskStarted, 0, "", make(chan int)); err == nil {
		t.Fatal("an unencodable payload must be reported, not dropped")
	}
}
