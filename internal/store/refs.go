package store

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/eduardmaghakyan/why/internal/diff"
)

// Refs manages .why/refs/<source-path> files.
// Each file has one hash per line, aligned 1:1 with source lines.
type Refs struct {
	Root string // path to .why/
}

// NewRefs creates a Refs manager rooted at the given .why directory.
func NewRefs(root string) *Refs {
	return &Refs{Root: root}
}

// Read returns the slice of hashes for a given source file.
// Empty string means no reasoning for that line.
func (r *Refs) Read(sourcePath string) ([]string, error) {
	path := r.refPath(sourcePath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	content := string(data)
	if content == "" {
		return nil, nil
	}
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	return lines, nil
}

// Write atomically writes the hash slice to the refs file.
// Skips writing if all hashes are empty (no reasoning to store).
func (r *Refs) Write(sourcePath string, hashes []string) error {
	allEmpty := true
	for _, h := range hashes {
		if h != "" {
			allEmpty = false
			break
		}
	}
	if allEmpty {
		return nil
	}

	path := r.refPath(sourcePath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	content := strings.Join(hashes, "\n") + "\n"
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Rebuild takes old/new source lines, old hashes, and a reasoning hash.
// Returns new hashes aligned with new source lines.
func (r *Refs) Rebuild(oldLines, newLines, oldHashes []string, reasoningHash string) []string {
	// Pad oldHashes to match oldLines length
	for len(oldHashes) < len(oldLines) {
		oldHashes = append(oldHashes, "")
	}

	edits := diff.Diff(oldLines, newLines)
	newHashes := make([]string, 0, len(newLines))

	for _, e := range edits {
		switch e.Op {
		case diff.OpEqual:
			newHashes = append(newHashes, oldHashes[e.OldIdx])
		case diff.OpInsert:
			newHashes = append(newHashes, reasoningHash)
		case diff.OpDelete:
			// dropped — no entry in new hashes
		}
	}

	return newHashes
}

// FindRelated returns file paths that were edited together with the given file.
// Primary: matches by turn ID. Fallback: timestamp clustering (5 min window).
func (r *Refs) FindRelated(filePath string, s *Store) []string {
	// Collect turn IDs and timestamps from the target file
	targetHashes := r.uniqueHashes(filePath)
	if len(targetHashes) == 0 {
		return nil
	}

	var targetTurnIDs []string
	var targetTimes []time.Time
	for _, hash := range targetHashes {
		obj, err := s.Get(hash)
		if err != nil {
			continue
		}
		if obj.TurnID != "" {
			targetTurnIDs = append(targetTurnIDs, obj.TurnID)
		}
		if t, err := time.Parse("2006-01-02 15:04", obj.Timestamp); err == nil {
			targetTimes = append(targetTimes, t)
		}
	}

	turnIDSet := make(map[string]bool)
	for _, id := range targetTurnIDs {
		turnIDSet[id] = true
	}

	// Walk all refs files and find matches
	refsDir := filepath.Join(r.Root, "refs")
	related := make(map[string]bool)

	filepath.Walk(refsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(refsDir, path)
		if err != nil || rel == filePath {
			return nil
		}

		hashes := r.uniqueHashes(rel)
		for _, hash := range hashes {
			obj, err := s.Get(hash)
			if err != nil {
				continue
			}

			// Primary: match by turn ID
			if obj.TurnID != "" && turnIDSet[obj.TurnID] {
				related[rel] = true
				return nil
			}

			// Fallback: timestamp proximity (5 min window)
			if len(targetTurnIDs) == 0 {
				if t, err := time.Parse("2006-01-02 15:04", obj.Timestamp); err == nil {
					for _, tt := range targetTimes {
						if absDuration(t.Sub(tt)) <= 5*time.Minute {
							related[rel] = true
							return nil
						}
					}
				}
			}
		}
		return nil
	})

	result := make([]string, 0, len(related))
	for f := range related {
		result = append(result, f)
	}
	sort.Strings(result)
	return result
}

// uniqueHashes returns deduplicated non-empty hashes from a refs file.
func (r *Refs) uniqueHashes(filePath string) []string {
	hashes, _ := r.Read(filePath)
	seen := make(map[string]bool)
	var unique []string
	for _, h := range hashes {
		if h != "" && !seen[h] {
			seen[h] = true
			unique = append(unique, h)
		}
	}
	return unique
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func (r *Refs) refPath(sourcePath string) string {
	return filepath.Join(r.Root, "refs", sourcePath)
}
