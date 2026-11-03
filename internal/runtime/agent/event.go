package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// Event is the envelope of the Event contract, version 0 of Architecture IV
// (§5.4, F1 and F2). It says what happened, in a form a person, a Transport
// and a later session can all rely on. The Runtime writes it; nothing reads a
// Tool argument, a Tool output or model text out of it, because an event
// never carries them: a log of events is metadata and may be shared. Ids are
// assigned by the host, never by the model.
type Event struct {
	// SchemaVersion is EventSchemaVersion when the event was written.
	SchemaVersion int `json:"schema_version"`
	// Seq starts at 1 for each task and rises by one with every event.
	Seq  uint64    `json:"seq"`
	Time time.Time `json:"time"`
	// TaskID is the Agent Folder task the event belongs to.
	TaskID string `json:"task_id"`
	// AgentID names the agent that acted. A SubAgent has its own and carries
	// its parent's in ParentAgentID.
	AgentID       string `json:"agent_id"`
	ParentAgentID string `json:"parent_agent_id,omitempty"`
	Step          int    `json:"step"`
	// CallID links the event to a Tool call, when it is about one.
	CallID string `json:"call_id,omitempty"`
	// RecipeVersion is the Recipe in force. It stays empty until a Recipe
	// exists (Phase 2); the field is reserved now so the log needs no rewrite.
	RecipeVersion string `json:"recipe_version,omitempty"`
	Kind          string `json:"kind"`
	// Payload is specific to Kind and versioned with SchemaVersion.
	Payload json.RawMessage `json:"payload,omitempty"`
}

// EventSchemaVersion is the version of the envelope this build writes. A
// reader accepts that version and the one before it, and skips what is newer
// or older instead of failing (see ReadEvents).
const EventSchemaVersion = 1

// Event kinds. The first group is emitted today; the second is reserved by
// the contract so a later phase adds it without a new schema (ARCHITECTURE_IV
// §5.5, F2). A reader skips kinds it does not know.
const (
	EventTaskStarted          = "task.started"
	EventPhaseChanged         = "phase.changed"
	EventToolProposed         = "tool.proposed"
	EventApprovalRequested    = "approval.requested"
	EventToolResult           = "tool.result"
	EventVerificationRecorded = "verification.recorded"
	EventTaskStopped          = "task.stopped"
	EventRouteChanged         = "route.changed"

	EventModelSwitched    = "model.switched"
	EventContextCompacted = "context.compacted"
	EventApprovalDecided  = "approval.decided"
	EventEffectRecorded   = "effect.recorded"
	EventModeChanged      = "mode.changed"
	EventCapabilityUsed   = "capability.used"
)

var knownEventKinds = map[string]bool{
	EventTaskStarted: true, EventPhaseChanged: true, EventToolProposed: true, EventApprovalRequested: true,
	EventToolResult: true, EventVerificationRecorded: true, EventTaskStopped: true, EventRouteChanged: true,
	EventModelSwitched: true, EventContextCompacted: true, EventApprovalDecided: true,
	EventEffectRecorded: true, EventModeChanged: true, EventCapabilityUsed: true,
}

// KnownEventKind reports whether this build knows the kind.
func KnownEventKind(kind string) bool { return knownEventKinds[kind] }

// acceptsEventVersion is the compatibility rule: a reader at version r reads
// events of version r and r-1.
func acceptsEventVersion(reader, event int) bool {
	return event >= 1 && (event == reader || event == reader-1)
}

// EventRecorder is implemented by a TaskRecorder that also keeps the event
// log. It is optional: a recorder without it keeps only its checkpoint.
type EventRecorder interface {
	RecordStamped(session *TaskSession, event Event) error
}

// eventStamper numbers one agent's events and gives them their ids.
type eventStamper struct {
	mu            sync.Mutex
	seq           uint64
	taskID        string
	agentID       string
	parentAgentID string
	now           func() time.Time
}

func newEventStamper(taskID, agentID, parentAgentID string) *eventStamper {
	return &eventStamper{taskID: taskID, agentID: agentID, parentAgentID: parentAgentID, now: time.Now}
}

// stamp builds the next event. A payload that cannot be encoded is a
// programming error, so it is reported rather than dropped.
func (s *eventStamper) stamp(kind string, step int, callID string, payload any) (Event, error) {
	var raw json.RawMessage
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return Event{}, fmt.Errorf("encode %s payload: %w", kind, err)
		}
		raw = encoded
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return Event{SchemaVersion: EventSchemaVersion, Seq: s.seq, Time: s.now().UTC(), TaskID: s.taskID, AgentID: s.agentID, ParentAgentID: s.parentAgentID, Step: step, CallID: callID, Kind: kind, Payload: raw}, nil
}

// maxReasonRunes bounds free text copied from an endpoint into an event.
const maxReasonRunes = 200

func clip(text string) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) > maxReasonRunes {
		return string(runes[:maxReasonRunes]) + "…"
	}
	return string(runes)
}

// eventFor maps an engine event to the envelope kind and a payload that holds
// metadata only: names, classes, counts and statuses. It never copies a Tool
// argument, a Tool output or an error message, which may carry file content
// or a secret.
func eventFor(event EngineEvent) (kind, callID string, payload any) {
	if event.ToolCall != nil {
		callID = event.ToolCall.ID
	}
	switch event.Kind {
	case "task":
		body := struct {
			PlanID       string `json:"plan_id,omitempty"`
			PlanRevision int    `json:"plan_revision,omitempty"`
		}{}
		if event.Plan != nil {
			body.PlanID, body.PlanRevision = event.Plan.ID, event.Plan.Revision
		}
		return EventTaskStarted, callID, body
	case "phase":
		return EventPhaseChanged, callID, struct {
			Phase AgentPhase `json:"phase"`
		}{event.Phase}
	case "tool", "approval":
		body := struct {
			Tool     string `json:"tool"`
			Risk     string `json:"risk,omitempty"`
			Approval string `json:"approval,omitempty"`
		}{}
		if event.ToolCall != nil {
			body.Tool, body.Risk, body.Approval = event.ToolCall.Tool, string(event.ToolCall.Risk), string(event.ToolCall.Approval)
		}
		if event.Kind == "approval" {
			return EventApprovalRequested, callID, body
		}
		return EventToolProposed, callID, body
	case "tool_result":
		body := struct {
			Tool         string `json:"tool,omitempty"`
			Status       string `json:"status"`
			ChangedPaths int    `json:"changed_paths"`
			ErrorClass   string `json:"error_class,omitempty"`
		}{}
		if event.ToolCall != nil {
			body.Tool = event.ToolCall.Tool
		}
		if event.ToolResult != nil {
			body.Status, body.ChangedPaths, body.ErrorClass = string(event.ToolResult.Status), len(event.ToolResult.ChangedPaths), event.ToolResult.ErrorClass
		}
		return EventToolResult, callID, body
	case "verification":
		body := struct {
			Passed bool `json:"passed"`
		}{}
		if event.Verification != nil {
			body.Passed = event.Verification.Passed
		}
		return EventVerificationRecorded, callID, body
	case "stop":
		return EventTaskStopped, callID, struct {
			StopReason StopReason `json:"stop_reason"`
		}{event.StopReason}
	}
	// A kind the engine adds later still gets recorded, under a name no
	// reader knows, so it is skipped rather than lost or misread.
	return "engine." + event.Kind, callID, nil
}

// routeChangedPayload is the payload of EventRouteChanged (P1.5).
func routeChangedPayload(kind, reason string) any {
	return struct {
		Route  string `json:"route"`
		Reason string `json:"reason"`
	}{kind, clip(reason)}
}

// SkippedEvent is an event ReadEvents did not return, with the reason.
type SkippedEvent struct {
	Line   int
	Seq    uint64
	Kind   string
	Reason string
}

// EventLog is what ReadEvents found.
type EventLog struct {
	Events  []Event
	Skipped []SkippedEvent
}

// ReadEvents reads a JSON-lines event log. It returns the events this build
// understands and lists the rest as skipped, each with its reason: an event
// of an unknown kind, or of a schema version this reader does not accept
// (older than the previous one, or newer than its own). A skipped event is
// never fatal, so an old reader can replay a newer log and a newer reader an
// older one. Only a line that is not an event at all is an error.
func ReadEvents(r io.Reader) (EventLog, error) {
	var log EventLog
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64<<10), 1<<20)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		var event Event
		if err := json.Unmarshal([]byte(text), &event); err != nil {
			return log, fmt.Errorf("event log line %d is not an event: %w", line, err)
		}
		switch {
		case !acceptsEventVersion(EventSchemaVersion, event.SchemaVersion):
			log.Skipped = append(log.Skipped, SkippedEvent{Line: line, Seq: event.Seq, Kind: event.Kind, Reason: fmt.Sprintf("schema version %d is not readable by version %d", event.SchemaVersion, EventSchemaVersion)})
		case !KnownEventKind(event.Kind):
			log.Skipped = append(log.Skipped, SkippedEvent{Line: line, Seq: event.Seq, Kind: event.Kind, Reason: "unknown kind"})
		default:
			log.Events = append(log.Events, event)
		}
	}
	return log, scanner.Err()
}
