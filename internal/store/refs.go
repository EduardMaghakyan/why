package store

import (
	"os"
	"path/filepath"
	"strings"

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
func (r *Refs) Write(sourcePath string, hashes []string) error {
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

func (r *Refs) refPath(sourcePath string) string {
	return filepath.Join(r.Root, "refs", sourcePath)
}
