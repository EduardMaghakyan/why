package diff

import (
	"testing"
)

func TestDiffIdentical(t *testing.T) {
	old := []string{"a", "b", "c"}
	edits := Diff(old, old)
	for _, e := range edits {
		if e.Op != OpEqual {
			t.Errorf("expected all OpEqual, got %v", e.Op)
		}
	}
}

func TestDiffInsert(t *testing.T) {
	old := []string{"a", "c"}
	new := []string{"a", "b", "c"}
	edits := Diff(old, new)

	want := []Edit{
		{Op: OpEqual, OldIdx: 0, NewIdx: 0},
		{Op: OpInsert, OldIdx: -1, NewIdx: 1},
		{Op: OpEqual, OldIdx: 1, NewIdx: 2},
	}
	assertEdits(t, want, edits)
}

func TestDiffDelete(t *testing.T) {
	old := []string{"a", "b", "c"}
	new := []string{"a", "c"}
	edits := Diff(old, new)

	want := []Edit{
		{Op: OpEqual, OldIdx: 0, NewIdx: 0},
		{Op: OpDelete, OldIdx: 1, NewIdx: -1},
		{Op: OpEqual, OldIdx: 2, NewIdx: 1},
	}
	assertEdits(t, want, edits)
}

func TestDiffReplace(t *testing.T) {
	old := []string{"a", "b", "c"}
	new := []string{"a", "x", "c"}
	edits := Diff(old, new)

	// b deleted, x inserted
	hasDelete := false
	hasInsert := false
	for _, e := range edits {
		if e.Op == OpDelete && old[e.OldIdx] == "b" {
			hasDelete = true
		}
		if e.Op == OpInsert && new[e.NewIdx] == "x" {
			hasInsert = true
		}
	}
	if !hasDelete || !hasInsert {
		t.Errorf("expected delete of 'b' and insert of 'x', got %v", edits)
	}
}

func TestDiffEmpty(t *testing.T) {
	edits := Diff(nil, []string{"a", "b"})
	for _, e := range edits {
		if e.Op != OpInsert {
			t.Errorf("expected all inserts for empty->non-empty, got %v", e.Op)
		}
	}
}

func assertEdits(t *testing.T, want, got []Edit) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("edit count: want %d, got %d\nwant: %v\ngot:  %v", len(want), len(got), want, got)
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("edit[%d]: want %v, got %v", i, want[i], got[i])
		}
	}
}
