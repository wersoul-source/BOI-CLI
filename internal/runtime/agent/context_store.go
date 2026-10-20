package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/boi-family/boi-cli/internal/block/port"
)

// ContextExpandTool is the Runtime capability that reads back an observation
// the host shortened or released from the model context.
const ContextExpandTool = "context.expand"

// maxExpandBytes keeps one expansion within a single observation budget.
const maxExpandBytes = maxObservationBytes

// contextStore keeps the full output of every tool result in one task, so
// the model context can carry a short reference instead of the whole text
// and still recover exact content later. Compute once, reference many: the
// store is lossless and per task; nothing in it is summarized or rewritten.
type contextStore struct {
	mu      sync.Mutex
	entries map[string]storedOutput
	byCall  map[string]string
}

type storedOutput struct {
	tool   string
	text   string
	digest string
}

func newContextStore() *contextStore {
	return &contextStore{entries: map[string]storedOutput{}, byCall: map[string]string{}}
}

// put stores a tool output once per call and returns its reference.
func (s *contextStore) put(callID, tool, text string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ref, ok := s.byCall[callID]; ok {
		return ref
	}
	ref := "r" + strconv.Itoa(len(s.entries)+1)
	s.entries[ref] = storedOutput{tool: tool, text: text, digest: digest(text)}
	s.byCall[callID] = ref
	return ref
}

func (s *contextStore) get(ref string) (storedOutput, error) {
	s.mu.Lock()
	entry, ok := s.entries[ref]
	s.mu.Unlock()
	if !ok {
		return storedOutput{}, fmt.Errorf("unknown context reference %q", ref)
	}
	if digest(entry.text) != entry.digest {
		return storedOutput{}, fmt.Errorf("context reference %q failed its integrity check", ref)
	}
	return entry, nil
}

func digest(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:8])
}

type contextStoreKey struct{}

func withContextStore(ctx context.Context, store *contextStore) context.Context {
	return context.WithValue(ctx, contextStoreKey{}, store)
}

// contextExpand is a read-only Tool over the current task's context store.
// It is part of the Runtime, not Equipment, because the store lives only for
// one task inside the Runtime.
type contextExpand struct{}

func (contextExpand) Spec() port.ToolSpec {
	return port.ToolSpec{
		Name:        ContextExpandTool,
		Usage:       "context.expand(ref, lines, match)",
		Description: "Read back the exact text of an observation the host shortened or released, by its reference. Give match to get only the lines containing that text, or lines for a range.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ref":   map[string]any{"type": "string", "description": "reference such as r3 from a shortened observation"},
				"lines": map[string]any{"type": "string", "description": "optional 1-based line range such as 120-180"},
				"match": map[string]any{"type": "string", "description": "optional case-insensitive text; returns only matching lines with their numbers"},
			},
			"required": []string{"ref"},
		},
		Risk: port.RiskRead, Approval: port.ApprovalAuto, Timeout: 5 * time.Second,
	}
}

func (contextExpand) Describe(arguments map[string]any) port.ToolPreview {
	ref := port.OptionalString(arguments, "ref")
	return port.ToolPreview{Target: ref, Preview: "Read back stored observation " + ref + " " + port.OptionalString(arguments, "lines")}
}

func (contextExpand) Execute(ctx context.Context, arguments map[string]any) (port.ToolOutput, error) {
	store, _ := ctx.Value(contextStoreKey{}).(*contextStore)
	if store == nil {
		return port.ToolOutput{}, fmt.Errorf("no context store for this task")
	}
	ref, err := port.StringArgument(arguments, "ref")
	if err != nil {
		return port.ToolOutput{}, err
	}
	entry, err := store.get(strings.TrimSpace(ref))
	if err != nil {
		return port.ToolOutput{}, err
	}
	lines := strings.Split(entry.text, "\n")
	first, last := 1, len(lines)
	if span := strings.TrimSpace(port.OptionalString(arguments, "lines")); span != "" {
		if first, last, err = parseLineRange(span, len(lines)); err != nil {
			return port.ToolOutput{}, err
		}
	}
	if match := strings.TrimSpace(port.OptionalString(arguments, "match")); match != "" {
		return port.ToolOutput{Output: matchingLines(ref, entry, lines[first-1:last], first, match)}, nil
	}
	// Return the longest prefix of the range that fits, and say where to
	// continue, rather than failing: a model that asked for too much then
	// still makes progress with its next call.
	end, size := first-1, 0
	for end < last && (end == first-1 || size+len(lines[end])+1 <= maxExpandBytes) {
		size += len(lines[end]) + 1
		end++
	}
	body := strings.Join(lines[first-1:end], "\n")
	if len(body) > maxExpandBytes {
		body = clipMiddle(body, maxExpandBytes)
	}
	header := fmt.Sprintf("%s from %s, lines %d-%d of %d, sha256:%s verified", ref, entry.tool, first, end, len(lines), entry.digest)
	if end < last {
		header += fmt.Sprintf("; output limit reached, continue with lines %d-%d or use match", end+1, last)
	}
	return port.ToolOutput{Output: header + "\n" + body}, nil
}

// maxMatches bounds one match result.
const maxMatches = 50

func matchingLines(ref string, entry storedOutput, lines []string, offset int, match string) string {
	needle := strings.ToLower(match)
	var found []string
	count := 0
	for index, line := range lines {
		if !strings.Contains(strings.ToLower(line), needle) {
			continue
		}
		count++
		if len(found) < maxMatches {
			found = append(found, fmt.Sprintf("%d: %s", offset+index, clipMiddle(line, 512)))
		}
	}
	header := fmt.Sprintf("%s from %s, %d lines match %q, sha256:%s verified", ref, entry.tool, count, match, entry.digest)
	if count > len(found) {
		header += fmt.Sprintf("; showing the first %d", len(found))
	}
	return strings.Join(append([]string{header}, found...), "\n")
}

func parseLineRange(span string, total int) (int, int, error) {
	start, end, found := strings.Cut(span, "-")
	first, err := strconv.Atoi(strings.TrimSpace(start))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid line range %q", span)
	}
	last := first
	if found {
		if last, err = strconv.Atoi(strings.TrimSpace(end)); err != nil {
			return 0, 0, fmt.Errorf("invalid line range %q", span)
		}
	}
	if first < 1 || last < first || first > total {
		return 0, 0, fmt.Errorf("line range %q is outside 1-%d", span, total)
	}
	if last > total {
		last = total
	}
	return first, last, nil
}
