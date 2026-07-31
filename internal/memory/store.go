package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Store is the Phantom DB — file-based memory store
type Store struct {
	mu  sync.RWMutex
	dir string
}

// MemoryEntry represents a stored memory
type MemoryEntry struct {
	MemID     string    `json:"id"`
	SessionID string    `json:"session_id"`
	Type      string    `json:"type"`
	Key       string    `json:"key"`
	Content   string    `json:"content"`
	Score     float64   `json:"score"`
	CreatedAt time.Time `json:"created_at"`
	TTL       int64     `json:"ttl"`
}

// Open creates or opens the Phantom DB directory
func Open(dbDir string) (*Store, error) {
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("create memory dir: %w", err)
	}
	s := &Store{dir: dbDir}
	return s, nil
}

// Save stores a memory entry as JSON file
func (s *Store) Save(entry *MemoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.dir, entry.MemID+".json")
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// loadAll reads all memory entries from disk
func (s *Store) loadAll() ([]MemoryEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, nil
	}

	var results []MemoryEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(s.dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var m MemoryEntry
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		results = append(results, m)
	}

	return results, nil
}

// Query searches memory by key or type
func (s *Store) Query(key, mtype string, limit int) ([]MemoryEntry, error) {
	all, err := s.loadAll()
	if err != nil {
		return nil, err
	}

	var results []MemoryEntry
	for _, m := range all {
		if mtype != "" && m.Type != mtype {
			continue
		}
		if key != "" && !strings.Contains(strings.ToLower(m.Key), strings.ToLower(key)) {
			continue
		}
		results = append(results, m)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].CreatedAt.After(results[j].CreatedAt)
		}
		return results[i].Score > results[j].Score
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// QueryBySession returns all memories from a session
func (s *Store) QueryBySession(sessionID string) ([]MemoryEntry, error) {
	all, err := s.loadAll()
	if err != nil {
		return nil, err
	}
	var results []MemoryEntry
	for _, m := range all {
		if m.SessionID == sessionID {
			results = append(results, m)
		}
	}
	return results, nil
}

// Delete removes a memory entry
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.dir, id+".json")
	return os.Remove(path)
}

// CleanExpired removes expired memory entries
func (s *Store) CleanExpired() (int64, error) {
	all, err := s.loadAll()
	if err != nil {
		return 0, err
	}

	var removed int64
	now := time.Now().Unix()
	for _, m := range all {
		if m.TTL > 0 && (m.CreatedAt.Unix()+m.TTL) < now {
			path := filepath.Join(s.dir, m.MemID+".json")
			if os.Remove(path) == nil {
				removed++
			}
		}
	}
	return removed, nil
}

// Stats returns memory statistics
func (s *Store) Stats() (map[string]interface{}, error) {
	all, err := s.loadAll()
	if err != nil {
		return nil, err
	}

	total := len(all)
	facts := 0
	patterns := 0
	solutions := 0
	for _, m := range all {
		switch m.Type {
		case "fact":
			facts++
		case "pattern":
			patterns++
		case "solution":
			solutions++
		}
	}

	return map[string]interface{}{
		"total":     total,
		"facts":     facts,
		"patterns":  patterns,
		"solutions": solutions,
	}, nil
}

// SearchMemory performs keyword search on memory entries
func (s *Store) SearchMemory(query string, limit int) ([]SearchResult, error) {
	keywords := strings.Fields(strings.ToLower(query))
	all, err := s.loadAll()
	if err != nil {
		return nil, err
	}

	type result struct {
		entry MemoryEntry
		score float64
	}

	var scored []result
	for _, m := range all {
		score := 0.0
		lowerKey := strings.ToLower(m.Key)
		lowerContent := strings.ToLower(m.Content)
		for _, kw := range keywords {
			if strings.Contains(lowerKey, kw) {
				score += 3.0
			}
			if strings.Contains(lowerContent, kw) {
				score += 1.0
			}
		}
		if score > 0 {
			scored = append(scored, result{m, score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	if limit > 0 && len(scored) > limit {
		scored = scored[:limit]
	}

	var results []SearchResult
	for _, r := range scored {
		results = append(results, SearchResult{
			Entry:  r.entry,
			Score:  r.score,
			Source: "keyword",
		})
	}
	return results, nil
}

// Close is a no-op for file-based store
func (s *Store) Close() error {
	return nil
}

// SearchResult type
type SearchResult struct {
	Entry  MemoryEntry
	Score  float64
	Source string
}
