package weight

type Engine struct {
	policy *WeightPolicy
}

func NewEngine(policy *WeightPolicy) *Engine {
	if policy == nil {
		policy = DefaultPolicy()
	}
	return &Engine{policy: policy}
}

func (eng *Engine) Compute(e Entity) float64 {
	w := &Weights{
		Truth:      e.Truth(),
		Confidence: e.Confidence(),
		Importance: e.Importance(),
		Recency:    e.Recency(),
		Usage:      e.Usage(),
	}
	return w.Sum(eng.policy)
}

func (eng *Engine) ComputeAndExplain(e Entity) *WeightExplanation {
	return Explain(e, eng.policy)
}

func ApplyDecay(score float64, hoursSince float64, decayRate float64) float64 {
	decay := hoursSince * decayRate
	result := score - decay
	if result < 0.1 {
		return 0.1
	}
	return result
}
