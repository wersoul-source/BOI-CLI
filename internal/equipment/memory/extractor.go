package memory

type ExtractRequest struct {
	UserQuery     string
	AgentResponse string
}

type ExtractedFact struct {
	Key     string
	Content string
	Type    string
	Weight  float64
}

type Extractor interface {
	Extract(req ExtractRequest) ([]ExtractedFact, error)
}

type SimpleExtractor struct{}

func (e *SimpleExtractor) Extract(req ExtractRequest) ([]ExtractedFact, error) {
	return nil, nil
}
