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
// using the Myers diff algorithm. Returns a sequence of Edit operations.
// Time complexity is O(n*d) where d is the edit distance, making it
// efficient for the common case of small edits in large files.
func Diff(old, new []string) []Edit {
	n := len(old)
	m := len(new)

	if n == 0 && m == 0 {
		return nil
	}
	if n == 0 {
		edits := make([]Edit, m)
		for j := 0; j < m; j++ {
			edits[j] = Edit{Op: OpInsert, OldIdx: -1, NewIdx: j}
		}
		return edits
	}
	if m == 0 {
		edits := make([]Edit, n)
		for i := 0; i < n; i++ {
			edits[i] = Edit{Op: OpDelete, OldIdx: i, NewIdx: -1}
		}
		return edits
	}

	max := n + m
	v := make([]int, 2*max+1)
	var trace [][]int

	var found bool
	for d := 0; d <= max; d++ {
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[k-1+max] < v[k+1+max]) {
				x = v[k+1+max]
			} else {
				x = v[k-1+max] + 1
			}
			y := x - k

			for x < n && y < m && old[x] == new[y] {
				x++
				y++
			}

			v[k+max] = x

			if x >= n && y >= m {
				found = true
				break
			}
		}

		snapshot := make([]int, len(v))
		copy(snapshot, v)
		trace = append(trace, snapshot)

		if found {
			break
		}
	}

	return buildEdits(old, new, trace)
}

type editStep struct {
	op         EditOp
	editIdx    int
	sx, sy     int // snake start
	ex, ey     int // snake end
}

func buildEdits(old, new []string, trace [][]int) []Edit {
	n := len(old)
	m := len(new)
	max := n + m

	x, y := n, m
	steps := make([]editStep, 0, len(trace)-1)

	for d := len(trace) - 1; d > 0; d-- {
		vPrev := trace[d-1]
		k := x - y

		var prevK int
		if k == -d || (k != d && vPrev[k-1+max] < vPrev[k+1+max]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}

		prevX := vPrev[prevK+max]
		prevY := prevX - prevK

		var midX, midY int
		var op EditOp
		var idx int
		if prevK == k+1 {
			op = OpInsert
			idx = prevY
			midX = prevX
			midY = prevY + 1
		} else {
			op = OpDelete
			idx = prevX
			midX = prevX + 1
			midY = prevY
		}

		steps = append(steps, editStep{op, idx, midX, midY, x, y})
		x, y = prevX, prevY
	}

	// Reverse to forward order
	for i, j := 0, len(steps)-1; i < j; i, j = i+1, j-1 {
		steps[i], steps[j] = steps[j], steps[i]
	}

	cap := len(old)
	if len(new) > cap {
		cap = len(new)
	}
	edits := make([]Edit, 0, cap)

	// Initial snake from (0,0) to (x, y) — the d=0 diagonal
	for i := 0; i < x; i++ {
		edits = append(edits, Edit{Op: OpEqual, OldIdx: i, NewIdx: i})
	}

	for _, s := range steps {
		if s.op == OpInsert {
			edits = append(edits, Edit{Op: OpInsert, OldIdx: -1, NewIdx: s.editIdx})
		} else {
			edits = append(edits, Edit{Op: OpDelete, OldIdx: s.editIdx, NewIdx: -1})
		}

		// Snake
		sx, sy := s.sx, s.sy
		for sx < s.ex {
			edits = append(edits, Edit{Op: OpEqual, OldIdx: sx, NewIdx: sy})
			sx++
			sy++
		}
	}

	return edits
}
