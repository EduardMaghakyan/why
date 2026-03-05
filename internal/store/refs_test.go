package store

import (
	"testing"
)

func TestRebuildUnchanged(t *testing.T) {
	refs := &Refs{}
	old := []string{"a", "b", "c"}
	hashes := []string{"h1", "h2", "h3"}
	result := refs.Rebuild(old, old, hashes, "new-hash")

	// All unchanged → keep old hashes
	for i, h := range result {
		if h != hashes[i] {
			t.Errorf("line %d: want %s, got %s", i, hashes[i], h)
		}
	}
}

func TestRebuildInsert(t *testing.T) {
	refs := &Refs{}
	old := []string{"a", "c"}
	new := []string{"a", "b", "c"}
	hashes := []string{"h1", "h3"}
	result := refs.Rebuild(old, new, hashes, "h-new")

	want := []string{"h1", "h-new", "h3"}
	assertHashes(t, want, result)
}

func TestRebuildDelete(t *testing.T) {
	refs := &Refs{}
	old := []string{"a", "b", "c"}
	new := []string{"a", "c"}
	hashes := []string{"h1", "h2", "h3"}
	result := refs.Rebuild(old, new, hashes, "h-new")

	want := []string{"h1", "h3"}
	assertHashes(t, want, result)
}

func TestRebuildFromEmpty(t *testing.T) {
	refs := &Refs{}
	result := refs.Rebuild(nil, []string{"a", "b"}, nil, "h-new")

	want := []string{"h-new", "h-new"}
	assertHashes(t, want, result)
}

func assertHashes(t *testing.T, want, got []string) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("length: want %d, got %d\nwant: %v\ngot:  %v", len(want), len(got), want, got)
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("[%d]: want %s, got %s", i, want[i], got[i])
		}
	}
}
