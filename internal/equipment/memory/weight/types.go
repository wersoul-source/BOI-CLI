package weight

import "time"

type Entity interface {
	ID() string
	Truth() float64
	Confidence() float64
	Importance() float64
	Recency() float64
	Usage() float64
	LastSeen() time.Time
	EvidenceCount() int
	ConflictCount() int
	SetWeights(w *Weights)
	Data() interface{}
}

type Weights struct {
	Truth      float64 `json:"truth"`
	Confidence float64 `json:"confidence"`
	Importance float64 `json:"importance"`
	Recency    float64 `json:"recency"`
	Usage      float64 `json:"usage"`
}

func (w *Weights) Sum(policy *WeightPolicy) float64 {
	sum := w.Truth*policy.Truth +
		w.Confidence*policy.Confidence +
		w.Importance*policy.Importance +
		w.Recency*policy.Recency +
		w.Usage*policy.Usage
	psum := policy.Truth + policy.Confidence + policy.Importance + policy.Recency + policy.Usage
	if psum == 0 {
		return 0
	}
	return sum / psum
}
