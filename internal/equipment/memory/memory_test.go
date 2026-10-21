package memory

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "db")
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s, dir
}

func TestStoreSaveQueryDelete(t *testing.T) {
	s, _ := newStore(t)
	now := time.Now()
	for _, e := range []*MemoryEntry{
		{MemID: "a", SessionID: "s1", Type: "fact", Key: "Go Version", Content: "uses go 1.24", Score: 1, CreatedAt: now},
		{MemID: "b", SessionID: "s2", Type: "pattern", Key: "naming", Content: "snake", Score: 5, CreatedAt: now},
	} {
		if err := s.Save(e); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.Query("go", "", 0)
	if err != nil || len(got) != 1 || got[0].MemID != "a" {
		t.Fatalf("key query: %v %v", got, err)
	}
	got, _ = s.Query("", "", 1)
	if len(got) != 1 || got[0].MemID != "b" {
		t.Fatalf("limit/score order wrong: %v", got)
	}
	if bs, _ := s.QueryBySession("s2"); len(bs) != 1 {
		t.Fatalf("session query: %v", bs)
	}
	if err := s.Delete("a"); err != nil {
		t.Fatal(err)
	}
	if got, _ = s.Query("", "", 0); len(got) != 1 {
		t.Fatalf("delete failed: %v", got)
	}
	st, _ := s.Stats()
	if st["total"] != 1 || st["patterns"] != 1 {
		t.Fatalf("stats: %v", st)
	}
}

func TestStoreRejectsPathTraversalIDs(t *testing.T) {
	s, dir := newStore(t)
	outside := filepath.Join(filepath.Dir(dir), "victim.json")
	if err := os.WriteFile(outside, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"../victim", "..", ".", "", "a/b", `a\b`} {
		if err := s.Save(&MemoryEntry{MemID: id}); err == nil {
			t.Errorf("Save accepted id %q", id)
		}
		if err := s.Delete(id); err == nil {
			t.Errorf("Delete accepted id %q", id)
		}
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("file outside store was touched: %v", err)
	}
}

func TestStoreCleanExpired(t *testing.T) {
	s, _ := newStore(t)
	old := time.Now().Add(-time.Hour)
	_ = s.Save(&MemoryEntry{MemID: "old", CreatedAt: old, TTL: 60})
	_ = s.Save(&MemoryEntry{MemID: "forever", CreatedAt: old})
	n, err := s.CleanExpired()
	if err != nil || n != 1 {
		t.Fatalf("removed %d, err %v", n, err)
	}
	if got, _ := s.Query("", "", 0); len(got) != 1 || got[0].MemID != "forever" {
		t.Fatalf("wrong survivors: %v", got)
	}
}

func TestStoreSkipsCorruptFiles(t *testing.T) {
	s, dir := newStore(t)
	_ = s.Save(&MemoryEntry{MemID: "ok", Key: "k"})
	_ = os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{not json"), 0o600)
	got, err := s.Query("", "", 0)
	if err != nil || len(got) != 1 {
		t.Fatalf("corrupt file broke query: %v %v", got, err)
	}
}

func TestSearchMemoryRanksKeyAboveContent(t *testing.T) {
	s, _ := newStore(t)
	_ = s.Save(&MemoryEntry{MemID: "content", Key: "x", Content: "deploy steps"})
	_ = s.Save(&MemoryEntry{MemID: "key", Key: "deploy", Content: "y"})
	_ = s.Save(&MemoryEntry{MemID: "none", Key: "z", Content: "z"})
	res, err := s.SearchMemory("deploy", 0)
	if err != nil || len(res) != 2 || res[0].Entry.MemID != "key" {
		t.Fatalf("ranking wrong: %+v %v", res, err)
	}
}
