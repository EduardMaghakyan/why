package store

import (
	"path/filepath"
	"testing"
)

func newTestSymbolRefs(t *testing.T) *SymbolRefs {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, ".why")
	return NewSymbolRefs(root)
}

func TestSymbolAppendAndRead(t *testing.T) {
	s := newTestSymbolRefs(t)

	if err := s.Append("src/main.go", "processPayment", "hash1", "2026-03-20 21:00"); err != nil {
		t.Fatalf("Append: %v", err)
	}

	refs := s.Read("src/main.go")
	if refs == nil {
		t.Fatal("Read returned nil")
	}
	entries := refs["processPayment"]
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	if entries[0].Hash != "hash1" {
		t.Errorf("want hash1, got %s", entries[0].Hash)
	}
	if entries[0].Timestamp != "2026-03-20 21:00" {
		t.Errorf("want 2026-03-20 21:00, got %s", entries[0].Timestamp)
	}
}

func TestSymbolAppendMultiple(t *testing.T) {
	s := newTestSymbolRefs(t)

	s.Append("src/main.go", "processPayment", "hash1", "2026-03-20 21:00")
	s.Append("src/main.go", "processPayment", "hash2", "2026-03-20 22:00")
	s.Append("src/main.go", "refund", "hash3", "2026-03-20 22:30")

	refs := s.Read("src/main.go")
	if len(refs["processPayment"]) != 2 {
		t.Errorf("processPayment: want 2 entries, got %d", len(refs["processPayment"]))
	}
	if len(refs["refund"]) != 1 {
		t.Errorf("refund: want 1 entry, got %d", len(refs["refund"]))
	}
}

func TestSymbolAppendDedup(t *testing.T) {
	s := newTestSymbolRefs(t)

	s.Append("src/main.go", "processPayment", "hash1", "2026-03-20 21:00")
	s.Append("src/main.go", "processPayment", "hash1", "2026-03-20 21:00") // duplicate

	refs := s.Read("src/main.go")
	if len(refs["processPayment"]) != 1 {
		t.Errorf("want 1 entry (deduped), got %d", len(refs["processPayment"]))
	}
}

func TestSymbolReadMissing(t *testing.T) {
	s := newTestSymbolRefs(t)
	refs := s.Read("nonexistent.go")
	if refs != nil {
		t.Errorf("want nil, got %v", refs)
	}
}

func TestSymbolListSymbols(t *testing.T) {
	s := newTestSymbolRefs(t)

	s.Append("src/main.go", "zebra", "h1", "2026-03-20 21:00")
	s.Append("src/main.go", "alpha", "h2", "2026-03-20 21:00")
	s.Append("src/main.go", "middle", "h3", "2026-03-20 21:00")

	names := s.ListSymbols("src/main.go")
	if len(names) != 3 {
		t.Fatalf("want 3 symbols, got %d", len(names))
	}
	// Should be sorted
	if names[0] != "alpha" || names[1] != "middle" || names[2] != "zebra" {
		t.Errorf("want [alpha middle zebra], got %v", names)
	}
}
