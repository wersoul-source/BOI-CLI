package agent

import "time"

type AgentState struct {
	ID          string
	PersonaName string
	Status      string
	Task        string
	Steps       []AgentStep
	StartedAt   time.Time
	MemoryUsed  int
}

type AgentStep struct {
	Number   int
	Thought  string
	Action   string
	Result   string
	Success  bool
	Duration time.Duration
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
	Response string
	Steps    int
	Tokens   int
	Duration time.Duration
	Memory   []string
	Provider string
	Model    string
}
