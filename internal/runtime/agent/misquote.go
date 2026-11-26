package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Misquote is an identifier in a final answer that is a near copy of one a
// Tool returned, but not the same: a value the model read and then altered.
type Misquote struct {
	// Answer is what the answer says.
	Answer string `json:"answer"`
	// Observed is the value the Tool returned.
	Observed string `json:"observed"`
}

// The check looks for one failure and says so: a model that reads a value and
// writes back a corrupted copy of it (ZOO-MARKER-42 read, ZOO-MARKER-public
// answered; OBSERVATION_IGNORED expected, OBSERVATION_public replied). It does
// not judge whether an answer is right. A wrong summary, a wrong number or
// wrong prose is invisible to it, and it never repairs an answer: the host
// editing what the model said would be pretending the model got it right
// (ADR-006).
//
// Known false positives, kept on purpose: an identifier the model mentions
// that differs from an observed one in a single segment, such as `user_id`
// beside an observed `user_ids`. The cost is one extra model call and a
// notice, so a possible misquote is reported as possible.
const (
	minTokenLength     = 6
	minHashLikeLength  = 8
	maxMisquoteReports = 3
)

// tokenPattern finds identifier-like words. It stops at '/' and ':' so a path
// is read as its parts and a file name never looks like a corrupted value.
var tokenPattern = regexp.MustCompile(`[A-Za-z0-9](?:[A-Za-z0-9_.\-]*[A-Za-z0-9])?`)

func tokens(text string) []string {
	var out []string
	for _, token := range tokenPattern.FindAllString(text, -1) {
		if len(token) >= minTokenLength {
			out = append(out, token)
		}
	}
	return out
}

// segments splits on '-' and '_' only. A dot stays inside a segment, so
// `agent.go` and `report.go` are single words, not two near-equal pairs.
func segments(token string) []string {
	return strings.FieldsFunc(token, func(r rune) bool { return r == '-' || r == '_' })
}

func hasDigit(text string) bool {
	return strings.IndexFunc(text, unicode.IsDigit) >= 0
}

// misquotes returns the identifiers in response that are altered copies of
// identifiers in observed (what the Tools returned). Words the user wrote in
// the task, and words that appear unchanged in an observation, are never
// reported.
func misquotes(task, response string, observed []string) []Misquote {
	if len(observed) == 0 || strings.TrimSpace(response) == "" {
		return nil
	}
	text := strings.Join(observed, "\n")
	var candidates []string
	known := map[string]bool{}
	for _, token := range tokens(task) {
		known[token] = true
	}
	for _, token := range tokens(text) {
		if !known[token] {
			candidates = append(candidates, token)
		}
		known[token] = true
	}
	var found []Misquote
	reported := map[string]bool{}
	for _, token := range tokens(response) {
		if known[token] || reported[token] || strings.Contains(text, token) || strings.Contains(task, token) {
			continue
		}
		if near, ok := nearObserved(token, candidates); ok {
			reported[token] = true
			found = append(found, Misquote{Answer: token, Observed: near})
		}
	}
	return found
}

// nearObserved finds an observed identifier that token is an altered copy of:
// the same words with exactly one of them changed (two or more segments), or
// a hash-like value with one character changed.
func nearObserved(token string, observed []string) (string, bool) {
	own := segments(token)
	for _, candidate := range observed {
		other := segments(candidate)
		switch {
		case len(own) >= 2 && len(own) == len(other):
			different := 0
			for i := range own {
				if own[i] != other[i] {
					different++
				}
			}
			if different == 1 {
				return candidate, true
			}
		case len(own) == 1 && len(other) == 1 && len(token) >= minHashLikeLength && hasDigit(token) && hasDigit(candidate) && editDistanceIsOne(token, candidate):
			return candidate, true
		}
	}
	return "", false
}

// editDistanceIsOne reports whether the strings differ by one inserted,
// deleted or changed character.
func editDistanceIsOne(a, b string) bool {
	ra, rb := []rune(a), []rune(b)
	if len(ra) < len(rb) {
		ra, rb = rb, ra
	}
	if len(ra)-len(rb) > 1 {
		return false
	}
	i := 0
	for i < len(rb) && ra[i] == rb[i] {
		i++
	}
	if i == len(rb) {
		return len(ra) != len(rb)
	}
	if len(ra) == len(rb) {
		return string(ra[i+1:]) == string(rb[i+1:])
	}
	return string(ra[i+1:]) == string(rb[i:])
}

// observedTexts is what the Tools of a task returned, in the form the check
// compares against: the text inside a JSON output rather than its escaped
// source, so a file's lines are compared as the file wrote them.
func observedTexts(steps []AgentStep) []string {
	var out []string
	for _, step := range steps {
		if step.ToolResult == nil || step.ToolResult.Status != ToolSucceeded {
			continue
		}
		out = append(out, outputStrings(step.ToolResult.Output)...)
	}
	return out
}

func outputStrings(output string) []string {
	trimmed := strings.TrimSpace(output)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		var decoded any
		if json.Unmarshal([]byte(trimmed), &decoded) == nil {
			var out []string
			collectStrings(decoded, &out)
			return out
		}
	}
	return []string{output}
}

func collectStrings(value any, out *[]string) {
	switch typed := value.(type) {
	case string:
		*out = append(*out, typed)
	case []any:
		for _, item := range typed {
			collectStrings(item, out)
		}
	case map[string]any:
		for _, item := range typed {
			collectStrings(item, out)
		}
	}
}

// misquoteReason is what the model is told when it is asked once to answer
// again.
func misquoteReason(found []Misquote) string {
	first := found[0]
	return fmt.Sprintf("your reply says %q, but the Tool returned %q. Copy the value exactly as the Tool returned it and answer again.", first.Answer, first.Observed)
}

// misquoteNotice is what the person reads when the answer still carries one.
func misquoteNotice(found []Misquote) string {
	lines := []string{"This answer may misquote what was read:"}
	for i, item := range found {
		if i == maxMisquoteReports {
			lines = append(lines, fmt.Sprintf("  … and %d more", len(found)-maxMisquoteReports))
			break
		}
		lines = append(lines, fmt.Sprintf("  it says %q, the Tool returned %q", item.Answer, item.Observed))
	}
	return strings.Join(lines, "\n")
}
