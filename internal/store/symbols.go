package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// SymbolEntry records a reasoning hash attached to a symbol.
type SymbolEntry struct {
	Hash      string `json:"hash"`
	Timestamp string `json:"ts"`
}

// SymbolRefs manages .why/symbols/<file>.json files.
type SymbolRefs struct {
	Root string // path to .why/
}

// NewSymbolRefs creates a SymbolRefs manager.
func NewSymbolRefs(root string) *SymbolRefs {
	return &SymbolRefs{Root: root}
}

// Read returns the symbol→entries map for a file.
func (s *SymbolRefs) Read(filePath string) map[string][]SymbolEntry {
	data, err := os.ReadFile(s.symbolPath(filePath))
	if err != nil {
		return nil
	}
	var result map[string][]SymbolEntry
	if json.Unmarshal(data, &result) != nil {
		return nil
	}
	return result
}

// Append adds a reasoning entry for a symbol in a file.
func (s *SymbolRefs) Append(filePath, symbolName, hash, timestamp string) error {
	existing := s.Read(filePath)
	if existing == nil {
		existing = make(map[string][]SymbolEntry)
	}

	// Deduplicate: don't add same hash twice for same symbol
	for _, e := range existing[symbolName] {
		if e.Hash == hash {
			return nil
		}
	}

	existing[symbolName] = append(existing[symbolName], SymbolEntry{
		Hash:      hash,
		Timestamp: timestamp,
	})

	path := s.symbolPath(filePath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

// ListSymbols returns all symbol names for a file, sorted alphabetically.
func (s *SymbolRefs) ListSymbols(filePath string) []string {
	refs := s.Read(filePath)
	if refs == nil {
		return nil
	}
	names := make([]string, 0, len(refs))
	for name := range refs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *SymbolRefs) symbolPath(filePath string) string {
	return filepath.Join(s.Root, "symbols", filePath+".json")
}
