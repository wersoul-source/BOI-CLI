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
		Usage:       "context.expand(ref, lines)",
		Description: "Read back the exact text of an observation the host shortened or released, by its reference and an optional line range.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ref":   map[string]any{"type": "string", "description": "reference such as r3 from a shortened observation"},
				"lines": map[string]any{"type": "string", "description": "optional 1-based line range such as 120-180"},
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
	body := strings.Join(lines[first-1:last], "\n")
	if len(body) > maxExpandBytes {
		return port.ToolOutput{}, fmt.Errorf("lines %d-%d are %d bytes; request a narrower range (limit %d bytes)", first, last, len(body), maxExpandBytes)
	}
	header := fmt.Sprintf("%s from %s, lines %d-%d of %d, sha256:%s verified\n", ref, entry.tool, first, last, len(lines), entry.digest)
	return port.ToolOutput{Output: header + body}, nil
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
