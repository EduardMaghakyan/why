package store

import (
	"os"
	"path/filepath"
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

func newTestRefs(t *testing.T) (*Refs, *Store) {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, ".why")
	os.MkdirAll(filepath.Join(root, "objects"), 0755)
	os.MkdirAll(filepath.Join(root, "refs"), 0755)
	return NewRefs(root), New(root)
}

func TestWriteAndRead(t *testing.T) {
	refs, _ := newTestRefs(t)
	hashes := []string{"h1", "", "h3"}

	if err := refs.Write("src/main.go", hashes); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := refs.Read("src/main.go")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	assertHashes(t, hashes, got)
}

func TestWriteSkipsAllEmpty(t *testing.T) {
	refs, _ := newTestRefs(t)
	hashes := []string{"", "", ""}

	if err := refs.Write("src/empty.go", hashes); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// File should not exist
	path := filepath.Join(refs.Root, "refs", "src/empty.go")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected refs file to not be created for all-empty hashes")
	}
}

func TestReadMissing(t *testing.T) {
	refs, _ := newTestRefs(t)
	got, err := refs.Read("nonexistent.go")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got != nil {
		t.Errorf("want nil, got %v", got)
	}
}

func TestFindRelatedByTurnID(t *testing.T) {
	refs, store := newTestRefs(t)

	// Create two objects with same turn ID
	obj1 := &Object{Timestamp: "2026-03-20 21:00", Commit: "aaa", TurnID: "sess:100", Reasoning: "reason A"}
	obj2 := &Object{Timestamp: "2026-03-20 21:00", Commit: "aaa", TurnID: "sess:100", Reasoning: "reason B"}

	h1, _ := store.Put(obj1)
	h2, _ := store.Put(obj2)

	refs.Write("file_a.go", []string{h1})
	refs.Write("file_b.go", []string{h2})

	related := refs.FindRelated("file_a.go", store)
	if len(related) != 1 || related[0] != "file_b.go" {
		t.Errorf("want [file_b.go], got %v", related)
	}
}

func TestFindRelatedByTimestamp(t *testing.T) {
	refs, store := newTestRefs(t)

	// No turn IDs, but timestamps within 5 min
	obj1 := &Object{Timestamp: "2026-03-20 21:00", Commit: "aaa", Reasoning: "reason A"}
	obj2 := &Object{Timestamp: "2026-03-20 21:03", Commit: "aaa", Reasoning: "reason B"}

	h1, _ := store.Put(obj1)
	h2, _ := store.Put(obj2)

	refs.Write("file_a.go", []string{h1})
	refs.Write("file_b.go", []string{h2})

	related := refs.FindRelated("file_a.go", store)
	if len(related) != 1 || related[0] != "file_b.go" {
		t.Errorf("want [file_b.go], got %v", related)
	}
}

func TestFindRelatedNoMatch(t *testing.T) {
	refs, store := newTestRefs(t)

	// Timestamps more than 5 min apart, no turn IDs
	obj1 := &Object{Timestamp: "2026-03-20 21:00", Commit: "aaa", Reasoning: "reason A"}
	obj2 := &Object{Timestamp: "2026-03-20 22:00", Commit: "bbb", Reasoning: "reason B"}

	h1, _ := store.Put(obj1)
	h2, _ := store.Put(obj2)

	refs.Write("file_a.go", []string{h1})
	refs.Write("file_b.go", []string{h2})

	related := refs.FindRelated("file_a.go", store)
	if len(related) != 0 {
		t.Errorf("want no related, got %v", related)
	}
}

func TestFindRelatedExcludesSelf(t *testing.T) {
	refs, store := newTestRefs(t)

	obj := &Object{Timestamp: "2026-03-20 21:00", Commit: "aaa", TurnID: "sess:100", Reasoning: "reason"}
	h, _ := store.Put(obj)

	refs.Write("file_a.go", []string{h})

	related := refs.FindRelated("file_a.go", store)
	if len(related) != 0 {
		t.Errorf("want no related (self excluded), got %v", related)
	}
}
