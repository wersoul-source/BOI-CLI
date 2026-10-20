package weight

type WeightPolicy struct {
	Truth      float64 `yaml:"truth" json:"truth"`
	Confidence float64 `yaml:"confidence" json:"confidence"`
	Importance float64 `yaml:"importance" json:"importance"`
	Recency    float64 `yaml:"recency" json:"recency"`
	Usage      float64 `yaml:"usage" json:"usage"`
	DecayRate  float64 `yaml:"decay_rate" json:"decay_rate"`
}

func DefaultPolicy() *WeightPolicy {
	return &WeightPolicy{
		Truth:      0.35,
		Confidence: 0.25,
		Importance: 0.10,
		Recency:    0.10,
		Usage:      0.20,
		DecayRate:  0.05,
	}
}
