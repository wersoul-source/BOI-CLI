package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPrefetchFencesUntrustedMemoryAndTruncates(t *testing.T) {
	s, _ := newStore(t)
	_ = s.Save(&MemoryEntry{MemID: "m", Type: "fact", Key: "deploy", Content: strings.Repeat("x", 500), CreatedAt: time.Now()})
	out := s.Prefetch("deploy", 5)
	if !strings.HasPrefix(out, "<memory-context>") || !strings.HasSuffix(out, "</memory-context>") {
		t.Fatalf("missing fence: %q", out)
	}
	if !strings.Contains(out, "untrusted recalled data") || !strings.Contains(out, "...") {
		t.Fatal("recalled memory must be marked untrusted and truncated")
	}
	if s.Prefetch("nothing-matches", 5) != "" {
		t.Fatal("no match must return empty")
	}
}

func TestSetScoreAndReWeight(t *testing.T) {
	s, _ := newStore(t)
	_ = s.Save(&MemoryEntry{MemID: "old", Score: 0.5})
	_ = s.Save(&MemoryEntry{MemID: "new", Score: 0.5, Content: "c"})

	if err := s.SetScore("old", 0.9); err != nil {
		t.Fatal(err)
	}
	if err := s.SetScore("ghost", 1); err == nil {
		t.Fatal("unknown id must error")
	}
	if err := s.ReWeight("old", "new", "proof"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Query("", "", 0)
	scores := map[string]MemoryEntry{}
	for _, m := range got {
		scores[m.MemID] = m
	}
	if scores["new"].Score <= 0.5 || scores["old"].Score >= 0.9 || !strings.Contains(scores["new"].Content, "[Evidence: proof]") {
		t.Fatalf("evidence reweight wrong: %+v", scores)
	}
	if err := s.ReWeight("old", "new", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.ReWeight("old", "ghost", ""); err == nil {
		t.Fatal("missing pair must error")
	}
}

type fixedExtractor struct{ facts []ExtractedFact }

func (f fixedExtractor) Extract(ExtractRequest) ([]ExtractedFact, error) { return f.facts, nil }

func TestHookStoresExtractedFactsAndInjects(t *testing.T) {
	s, _ := newStore(t)
	h := NewMemoryHook(s, fixedExtractor{[]ExtractedFact{{Key: "lang", Content: "go", Type: "fact", Weight: 0.7}}})
	h.AfterTurn("q", "a")
	got, _ := s.Query("lang", "", 0)
	if len(got) != 1 || got[0].Score != 0.7 {
		t.Fatalf("fact not stored: %+v", got)
	}
	if !strings.Contains(h.BeforeTurn("lang"), "lang") {
		t.Fatal("BeforeTurn must recall stored fact")
	}
	// the default extractor must be inert, never panic
	NewMemoryHook(s, &SimpleExtractor{}).AfterTurn("q", "a")
}

func TestScanRepoSkipsToolingDirs(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"main.go", "pkg/a.go", ".git/config", "node_modules/x.js", ".boi/mem.json"} {
		full := filepath.Join(root, p)
		_ = os.MkdirAll(filepath.Dir(full), 0o755)
		_ = os.WriteFile(full, []byte("x"), 0o600)
	}
	m, err := ScanRepo(root)
	if err != nil || m.FileCount != 2 {
		t.Fatalf("files %d err %v", m.FileCount, err)
	}
	if !strings.Contains(m.Summary(), "Files: 2") {
		t.Fatalf("summary %q", m.Summary())
	}
}

func TestMemoryFilesRoundTripAndSearchOrder(t *testing.T) {
	root := t.TempDir()
	if got, err := LoadMemoryFiles(root); err != nil || got != "" {
		t.Fatalf("empty root: %q %v", got, err)
	}
	_ = os.WriteFile(filepath.Join(root, "BOI_MEMORY.md"), []byte("legacy"), 0o600)
	if got, _ := LoadMemoryFiles(root); got != "legacy" {
		t.Fatalf("fallback: %q", got)
	}
	if err := SaveMemoryFile(root, "primary"); err != nil {
		t.Fatal(err)
	}
	if got, _ := LoadMemoryFiles(root); got != "primary" {
		t.Fatalf(".boi/memory.md must win: %q", got)
	}
}

func TestEntityWeightsByType(t *testing.T) {
	if (&MemoryEntry{Type: "solution"}).Importance() <= (&MemoryEntry{Type: "fact"}).Importance() {
		t.Fatal("solution must outrank fact")
	}
	if (&MemoryEntry{CreatedAt: time.Now()}).Recency() <= (&MemoryEntry{CreatedAt: time.Now().Add(-30 * 24 * time.Hour)}).Recency() {
		t.Fatal("recent must outrank old")
	}
	if (&MemoryEntry{Content: "x"}).Confidence() <= (&MemoryEntry{}).Confidence() {
		t.Fatal("content must raise confidence")
	}
	if DefaultNudgeConfig().Interval <= 0 {
		t.Fatal("nudge interval")
	}
}
