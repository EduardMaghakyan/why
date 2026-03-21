package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Object is a single immutable reasoning entry.
type Object struct {
	Timestamp string `json:"ts"`
	Commit    string `json:"commit"`
	Reasoning string `json:"reasoning"`
}

// Store manages the .why/objects/ content-addressable store.
type Store struct {
	Root string // path to .why/
}

// New creates a Store rooted at the given .why directory.
func New(root string) *Store {
	return &Store{Root: root}
}

// Put writes an Object and returns its SHA-256 hex hash.
func (s *Store) Put(obj *Object) (string, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("marshal object: %w", err)
	}

	hash := objectHash(obj)
	path := s.objectPath(hash)

	// Idempotent: skip if already exists
	if _, err := os.Stat(path); err == nil {
		return hash, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("create object dir: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return "", fmt.Errorf("write object: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("rename object: %w", err)
	}

	return hash, nil
}

// Get reads an Object by its full hex hash.
func (s *Store) Get(hash string) (*Object, error) {
	data, err := os.ReadFile(s.objectPath(hash))
	if err != nil {
		return nil, fmt.Errorf("object %s not found", truncHash(hash))
	}
	var obj Object
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("unmarshal object %s: %w", truncHash(hash), err)
	}
	return &obj, nil
}

// ObjectEntry pairs a hash with its deserialized Object.
type ObjectEntry struct {
	Hash   string
	Object *Object
}

// ListAll walks .why/objects/ and returns all reasoning entries sorted by timestamp.
func (s *Store) ListAll() ([]ObjectEntry, error) {
	objDir := filepath.Join(s.Root, "objects")
	var entries []ObjectEntry

	filepath.Walk(objDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		dir := filepath.Base(filepath.Dir(path))
		if len(dir) != 2 {
			return nil
		}
		hash := dir + info.Name()
		obj, err := s.Get(hash)
		if err != nil {
			return nil
		}
		entries = append(entries, ObjectEntry{Hash: hash, Object: obj})
		return nil
	})

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Object.Timestamp < entries[j].Object.Timestamp
	})

	return entries, nil
}

func (s *Store) objectPath(hash string) string {
	return filepath.Join(s.Root, "objects", hash[:2], hash[2:])
}

func objectHash(obj *Object) string {
	h := sha256.New()
	h.Write([]byte(obj.Reasoning))
	h.Write([]byte(obj.Timestamp))
	h.Write([]byte(obj.Commit))
	return hex.EncodeToString(h.Sum(nil))
}

func truncHash(hash string) string {
	if len(hash) > 12 {
		return hash[:12]
	}
	return hash
}
