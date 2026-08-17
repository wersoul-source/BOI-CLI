package memory

import (
	"time"

	"github.com/boi-family/boi-cli/internal/memory/weight"
)

var _ weight.Entity = (*MemoryEntry)(nil)

func (m *MemoryEntry) ID() string {
	return m.MemID
}

func (m *MemoryEntry) Truth() float64 {
	return m.Score
}

func (m *MemoryEntry) Confidence() float64 {
	if m.Content != "" {
		return 0.5
	}
	return 0.1
}

func (m *MemoryEntry) Importance() float64 {
	switch m.Type {
	case "solution":
		return 0.9
	case "pattern":
		return 0.7
	case "fact":
		return 0.5
	default:
		return 0.3
	}
}

func (m *MemoryEntry) Recency() float64 {
	hours := time.Since(m.CreatedAt).Hours()
	if hours < 1 {
		return 1.0
	}
	if hours < 24 {
		return 0.8
	}
	if hours < 168 {
		return 0.5
	}
	return 0.2
}

func (m *MemoryEntry) Usage() float64 {
	return 0.5
}

func (m *MemoryEntry) LastSeen() time.Time {
	return m.CreatedAt
}

func (m *MemoryEntry) EvidenceCount() int {
	return 1
}

func (m *MemoryEntry) ConflictCount() int {
	return 0
}

func (m *MemoryEntry) SetWeights(w *weight.Weights) {
	if w != nil {
		m.Score = w.Sum(weight.DefaultPolicy())
	}
}

func (m *MemoryEntry) Data() interface{} {
	return m
}
