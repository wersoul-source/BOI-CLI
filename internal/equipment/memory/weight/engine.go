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

func (eng *Engine) ComputeAndExplain(e Entity) *WeightExplanation {
	return Explain(e, eng.policy)
}
