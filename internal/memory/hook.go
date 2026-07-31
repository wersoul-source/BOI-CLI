package memory

import (
	"fmt"
	"time"
)

type MemoryHook struct {
	store      *Store
	extractor  Extractor
	nudgeEvery int
	turnCount  int
}

func NewMemoryHook(store *Store, extractor Extractor) *MemoryHook {
	return &MemoryHook{
		store:      store,
		extractor:  extractor,
		nudgeEvery: 10,
	}
}

func (h *MemoryHook) BeforeTurn(query string) string {
	return h.store.InjectMemory(query)
}

func (h *MemoryHook) AfterTurn(query, response string) {
	facts, err := h.extractor.Extract(ExtractRequest{
		UserQuery:     query,
		AgentResponse: response,
	})
	if err != nil {
		return
	}

	for _, f := range facts {
		entry := &MemoryEntry{
			MemID:     fmt.Sprintf("mem_%d", time.Now().UnixNano()),
			SessionID: "auto",
			Type:      f.Type,
			Key:       f.Key,
			Content:   f.Content,
			Score:     f.Weight,
			CreatedAt: time.Now(),
		}
		h.store.Save(entry)
	}

	h.turnCount++
	if h.turnCount%h.nudgeEvery == 0 {
		h.Nudge()
	}
}

func (h *MemoryHook) Nudge() {
}
