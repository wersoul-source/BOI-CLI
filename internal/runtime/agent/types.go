package agent

import "time"

type AgentState struct {
	ID          string
	PersonaName string
	Phase       AgentPhase
	StopReason  StopReason
	Task        string
	Steps       []AgentStep
	StartedAt   time.Time
	MemoryUsed  int
	Plan        *TaskPlan
}

type AgentStep struct {
	Number     int
	Phase      AgentPhase
	Thought    string
	Action     string
	Result     string
	ToolCall   *ToolCall
	ToolResult *ToolResult
	Success    bool
	Duration   time.Duration
}

type PlanStatus string

const (
	PlanPending   PlanStatus = "pending"
	PlanRunning   PlanStatus = "running"
	PlanCompleted PlanStatus = "completed"
	PlanFailed    PlanStatus = "failed"
	PlanSkipped   PlanStatus = "skipped"
)

type TaskPlan struct {
	SchemaVersion int            `json:"schema_version"`
	ID            string         `json:"id"`
	Revision      int            `json:"revision"`
	Goal          string         `json:"goal"`
	Status        PlanStatus     `json:"status"`
	Steps         []PlannedStep  `json:"steps"`
	Revisions     []PlanRevision `json:"revisions,omitempty"`
}

type PlanRevision struct {
	Number int        `json:"number"`
	Phase  AgentPhase `json:"phase"`
	Reason string     `json:"reason"`
}

type PlannedStep struct {
	Number      int        `json:"number"`
	Description string     `json:"description"`
	Phase       AgentPhase `json:"phase"`
	DependsOn   []int      `json:"depends_on,omitempty"`
	Status      PlanStatus `json:"status"`
}

type AgentResult struct {
	TaskID             string
	Response           string
	Steps              int
	Tokens             int
	Duration           time.Duration
	Memory             []string
	Provider           string
	Model              string
	StopReason         StopReason
	Usage              Usage
	Error              string
	Plan               *TaskPlan
	Trace              []AgentStep
	Manifest           string
	Artifacts          []ArtifactReference
	ProviderProfileRef string
	IdempotencyKeyHash string
}
