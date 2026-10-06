package memory

type NudgeConfig struct {
	Interval int
	MinTurns int
}

func DefaultNudgeConfig() *NudgeConfig {
	return &NudgeConfig{
		Interval: 10,
		MinTurns: 6,
	}
}
