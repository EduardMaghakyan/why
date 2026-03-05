package diff

// EditOp represents a single diff operation.
type EditOp int

const (
	OpEqual  EditOp = iota // line unchanged
	OpInsert               // line added in new
	OpDelete               // line removed from old
)

// Edit represents a single line operation in a diff.
type Edit struct {
	Op     EditOp
	OldIdx int // index in old slice (-1 for inserts)
	NewIdx int // index in new slice (-1 for deletes)
}

// Diff computes a line-level diff between old and new string slices
// using a simple LCS-based algorithm. Returns a sequence of Edit operations.
func Diff(old, new []string) []Edit {
	n := len(old)
	m := len(new)

	// Build LCS table
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if old[i] == new[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	// Walk the LCS table to produce edits
	var edits []Edit
	i, j := 0, 0
	for i < n && j < m {
		if old[i] == new[j] {
			edits = append(edits, Edit{Op: OpEqual, OldIdx: i, NewIdx: j})
			i++
			j++
		} else if lcs[i+1][j] >= lcs[i][j+1] {
			edits = append(edits, Edit{Op: OpDelete, OldIdx: i, NewIdx: -1})
			i++
		} else {
			edits = append(edits, Edit{Op: OpInsert, OldIdx: -1, NewIdx: j})
			j++
		}
	}
	for i < n {
		edits = append(edits, Edit{Op: OpDelete, OldIdx: i, NewIdx: -1})
		i++
	}
	for j < m {
		edits = append(edits, Edit{Op: OpInsert, OldIdx: -1, NewIdx: j})
		j++
	}

	return edits
}
