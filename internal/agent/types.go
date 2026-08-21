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

type TaskPlan struct {
	Goal  string
	Steps []PlannedStep
}

type PlannedStep struct {
	Number      int
	Description string
	Tool        string
	Args        string
	DependsOn   int
}

type AgentResult struct {
	Response   string
	Steps      int
	Tokens     int
	Duration   time.Duration
	Memory     []string
	Provider   string
	Model      string
	StopReason StopReason
	Usage      Usage
	Error      string
}
