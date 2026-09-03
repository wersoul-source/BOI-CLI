package agent

import "time"

// TaskSession is the host-owned filesystem scope assigned to one Agent run.
// The model may use these paths as data, but cannot redefine their authority.
type TaskSession struct {
	ID        string
	BinDir    string
	OutputDir string
	StartedAt time.Time
}

type ArtifactReference struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

// TaskRecorder owns durable task checkpoints and the final artifact manifest.
// Implementations must not persist prompts, model output, or secrets as logs.
type TaskRecorder interface {
	Begin(task string, plan *TaskPlan) (*TaskSession, error)
	RecordEvent(session *TaskSession, event EngineEvent) error
	Finalize(session *TaskSession, result *AgentResult) error
}
