package weight

import (
	"strings"
	"time"
)

type WeightExplanation struct {
	EntityID   string       `json:"entity_id"`
	Breakdown  []WeightStep `json:"breakdown"`
	FinalScore float64      `json:"final_score"`
}

type WeightStep struct {
	Dimension    string  `json:"dimension"`
	RawValue     float64 `json:"raw_value"`
	PolicyWeight float64 `json:"policy_weight"`
	Contribution float64 `json:"contribution"`
	Reason       string  `json:"reason"`
}

func Explain(e Entity, policy *WeightPolicy) *WeightExplanation {
	exp := &WeightExplanation{EntityID: e.ID()}

	steps := []struct {
		dim    string
		value  float64
		weight float64
		reason string
	}{
		{"truth", e.Truth(), policy.Truth, evidenceReason(e.EvidenceCount())},
		{"confidence", e.Confidence(), policy.Confidence, confidenceReason(e.ConflictCount())},
		{"importance", e.Importance(), policy.Importance, "importance score"},
		{"recency", e.Recency(), policy.Recency, recencyReason(e.LastSeen())},
		{"usage", e.Usage(), policy.Usage, "usage frequency"},
	}

	psum := policy.Truth + policy.Confidence + policy.Importance + policy.Recency + policy.Usage
	total := 0.0

	for _, s := range steps {
		contrib := 0.0
		if psum > 0 {
			contrib = s.value * s.weight / psum
		}
		total += contrib
		exp.Breakdown = append(exp.Breakdown, WeightStep{
			Dimension:    s.dim,
			RawValue:     s.value,
			PolicyWeight: s.weight,
			Contribution: contrib,
			Reason:       s.reason,
		})
	}

	exp.FinalScore = total
	return exp
}

func (exp *WeightExplanation) String() string {
	var sb strings.Builder
	for _, s := range exp.Breakdown {
		sign := "+"
		if s.Contribution < 0 {
			sign = ""
		}
		sb.WriteString(sign)
		sb.WriteString(dimLabel(s.Dimension))
		sb.WriteString(": ")
		sb.WriteString(s.Reason)
		sb.WriteString("\n")
	}
	return sb.String()
}

func evidenceReason(count int) string {
	if count >= 10 {
		return "strong evidence"
	}
	if count >= 3 {
		return "moderate evidence"
	}
	if count > 0 {
		return "some evidence"
	}
	return "no evidence"
}

func confidenceReason(conflicts int) string {
	if conflicts == 0 {
		return "no conflicts"
	}
	if conflicts < 3 {
		return "few conflicts"
	}
	return "many conflicts"
}

func recencyReason(lastSeen time.Time) string {
	hours := time.Since(lastSeen).Hours()
	if hours < 1 {
		return "seen recently"
	}
	if hours < 24 {
		return "seen today"
	}
	return "not seen recently"
}

func dimLabel(dim string) string {
	switch dim {
	case "truth":
		return "Truth       "
	case "confidence":
		return "Confidence  "
	case "importance":
		return "Importance  "
	case "recency":
		return "Recency     "
	case "usage":
		return "Usage       "
	}
	return dim
}
