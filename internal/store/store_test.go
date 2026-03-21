package store

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, ".why")
	os.MkdirAll(filepath.Join(root, "objects"), 0755)
	return New(root)
}

func TestPutAndGet(t *testing.T) {
	s := newTestStore(t)
	obj := &Object{
		Timestamp: "2026-03-20 21:38",
		Commit:    "abc1234",
		TurnID:    "sess:123",
		Reasoning: "Fix race condition in token refresh",
	}

	hash, err := s.Put(obj)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if hash == "" {
		t.Fatal("Put returned empty hash")
	}

	got, err := s.Get(hash)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Timestamp != obj.Timestamp {
		t.Errorf("Timestamp: want %s, got %s", obj.Timestamp, got.Timestamp)
	}
	if got.Commit != obj.Commit {
		t.Errorf("Commit: want %s, got %s", obj.Commit, got.Commit)
	}
	if got.TurnID != obj.TurnID {
		t.Errorf("TurnID: want %s, got %s", obj.TurnID, got.TurnID)
	}
	if got.Reasoning != obj.Reasoning {
		t.Errorf("Reasoning: want %s, got %s", obj.Reasoning, got.Reasoning)
	}
}

func TestPutIdempotent(t *testing.T) {
	s := newTestStore(t)
	obj := &Object{
		Timestamp: "2026-03-20 21:38",
		Commit:    "abc1234",
		Reasoning: "Same reasoning",
	}

	hash1, err := s.Put(obj)
	if err != nil {
		t.Fatalf("Put 1: %v", err)
	}
	hash2, err := s.Put(obj)
	if err != nil {
		t.Fatalf("Put 2: %v", err)
	}
	if hash1 != hash2 {
		t.Errorf("idempotent Put returned different hashes: %s vs %s", hash1, hash2)
	}
}

func TestGetMissing(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Get("0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Fatal("expected error for missing object")
	}
}

func TestListAll(t *testing.T) {
	s := newTestStore(t)

	objs := []*Object{
		{Timestamp: "2026-03-20 22:00", Commit: "aaa", Reasoning: "third"},
		{Timestamp: "2026-03-20 21:00", Commit: "bbb", Reasoning: "first"},
		{Timestamp: "2026-03-20 21:30", Commit: "ccc", Reasoning: "second"},
	}
	for _, obj := range objs {
		if _, err := s.Put(obj); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}

	entries, err := s.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("want 3 entries, got %d", len(entries))
	}

	// Should be sorted by timestamp
	if entries[0].Object.Reasoning != "first" {
		t.Errorf("entries[0]: want 'first', got %q", entries[0].Object.Reasoning)
	}
	if entries[1].Object.Reasoning != "second" {
		t.Errorf("entries[1]: want 'second', got %q", entries[1].Object.Reasoning)
	}
	if entries[2].Object.Reasoning != "third" {
		t.Errorf("entries[2]: want 'third', got %q", entries[2].Object.Reasoning)
	}
}

func TestListAllEmpty(t *testing.T) {
	s := newTestStore(t)
	entries, err := s.ListAll()
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("want 0 entries, got %d", len(entries))
	}
}
