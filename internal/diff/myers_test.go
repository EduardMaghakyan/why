package diff

import (
	"fmt"
	"testing"
	"time"
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

func TestDiffBothEmpty(t *testing.T) {
	edits := Diff(nil, nil)
	if edits != nil {
		t.Errorf("expected nil for both-empty, got %v", edits)
	}
}

func TestDiffNewEmpty(t *testing.T) {
	edits := Diff([]string{"a", "b", "c"}, nil)
	if len(edits) != 3 {
		t.Fatalf("expected 3 deletes, got %d edits", len(edits))
	}
	for i, e := range edits {
		if e.Op != OpDelete {
			t.Errorf("edit[%d]: expected OpDelete, got %v", i, e.Op)
		}
		if e.OldIdx != i {
			t.Errorf("edit[%d]: expected OldIdx=%d, got %d", i, i, e.OldIdx)
		}
		if e.NewIdx != -1 {
			t.Errorf("edit[%d]: expected NewIdx=-1, got %d", i, e.NewIdx)
		}
	}
}

func TestDiffCompletelyDifferent(t *testing.T) {
	old := []string{"a", "b", "c"}
	new := []string{"x", "y", "z"}
	edits := Diff(old, new)

	// Verify round-trip: applying edits to old should produce new
	result := applyEdits(old, new, edits)
	assertSliceEqual(t, new, result)
}

func TestDiffLargeIdentical(t *testing.T) {
	lines := make([]string, 20000)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i)
	}

	start := time.Now()
	edits := Diff(lines, lines)
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Errorf("large identical diff took %v, expected < 2s", elapsed)
	}
	for _, e := range edits {
		if e.Op != OpEqual {
			t.Errorf("expected all OpEqual for identical files, got %v", e.Op)
			break
		}
	}
	if len(edits) != 20000 {
		t.Errorf("expected 20000 edits, got %d", len(edits))
	}
}

func TestDiffSingleEditLargeFile(t *testing.T) {
	n := 10000
	old := make([]string, n)
	new := make([]string, n)
	for i := 0; i < n; i++ {
		old[i] = fmt.Sprintf("line %d", i)
		new[i] = fmt.Sprintf("line %d", i)
	}
	new[n/2] = "CHANGED LINE"

	start := time.Now()
	edits := Diff(old, new)
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Errorf("single-edit large diff took %v, expected < 2s", elapsed)
	}

	result := applyEdits(old, new, edits)
	assertSliceEqual(t, new, result)
}

func TestDiffPrefixSuffix(t *testing.T) {
	old := []string{"a", "b", "c", "d", "e"}
	new := []string{"X", "b", "c", "d", "Y"}
	edits := Diff(old, new)

	result := applyEdits(old, new, edits)
	assertSliceEqual(t, new, result)
}

func TestDiffRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		old  []string
		new  []string
	}{
		{"insert-middle", []string{"a", "c"}, []string{"a", "b", "c"}},
		{"delete-middle", []string{"a", "b", "c"}, []string{"a", "c"}},
		{"replace", []string{"a", "b", "c"}, []string{"a", "x", "c"}},
		{"prepend", []string{"b", "c"}, []string{"a", "b", "c"}},
		{"append", []string{"a", "b"}, []string{"a", "b", "c"}},
		{"empty-to-full", nil, []string{"a", "b"}},
		{"full-to-empty", []string{"a", "b"}, nil},
		{"complete-replace", []string{"a", "b"}, []string{"x", "y"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			edits := Diff(tc.old, tc.new)
			result := applyEdits(tc.old, tc.new, edits)
			assertSliceEqual(t, tc.new, result)
		})
	}
}

// applyEdits reconstructs the new slice by walking the edit sequence.
func applyEdits(old, new []string, edits []Edit) []string {
	var result []string
	for _, e := range edits {
		switch e.Op {
		case OpEqual:
			result = append(result, old[e.OldIdx])
		case OpInsert:
			result = append(result, new[e.NewIdx])
		case OpDelete:
			// skip
		}
	}
	return result
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

func assertSliceEqual(t *testing.T, want, got []string) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("length: want %d, got %d\nwant: %v\ngot:  %v", len(want), len(got), want, got)
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("index %d: want %q, got %q", i, want[i], got[i])
		}
	}
}
