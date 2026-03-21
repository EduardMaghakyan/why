package hook

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadPending(t *testing.T) {
	// Use temp dir to avoid interfering with real state
	old := pendingDir
	pendingDir = filepath.Join(t.TempDir(), ".why-pending")
	defer func() { pendingDir = old }()

	if err := WritePending("abc123"); err != nil {
		t.Fatalf("WritePending: %v", err)
	}

	got := ReadPending()
	if got != "abc123" {
		t.Errorf("want abc123, got %q", got)
	}

	// Non-destructive: read again should return same value
	got2 := ReadPending()
	if got2 != "abc123" {
		t.Errorf("second read: want abc123, got %q", got2)
	}
}

func TestReadPendingMissing(t *testing.T) {
	old := pendingDir
	pendingDir = filepath.Join(t.TempDir(), ".why-pending-missing")
	defer func() { pendingDir = old }()

	got := ReadPending()
	if got != "" {
		t.Errorf("want empty, got %q", got)
	}
}

func TestSaveAndLoadState(t *testing.T) {
	old := hookStateDir
	hookStateDir = filepath.Join(t.TempDir(), ".why-hook-state")
	defer func() { hookStateDir = old }()

	state := &PreState{
		FilePath:      "src/main.go",
		ReasoningHash: "hashABC",
		TurnID:        "sess:456",
		Snapshot:      "package main\n",
	}
	key := "testkey123"

	if err := state.Save(key); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := LoadState(key)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if loaded.FilePath != state.FilePath {
		t.Errorf("FilePath: want %s, got %s", state.FilePath, loaded.FilePath)
	}
	if loaded.ReasoningHash != state.ReasoningHash {
		t.Errorf("ReasoningHash: want %s, got %s", state.ReasoningHash, loaded.ReasoningHash)
	}
	if loaded.TurnID != state.TurnID {
		t.Errorf("TurnID: want %s, got %s", state.TurnID, loaded.TurnID)
	}
	if loaded.Snapshot != state.Snapshot {
		t.Errorf("Snapshot: want %q, got %q", state.Snapshot, loaded.Snapshot)
	}

	// LoadState should delete the file
	_, err = LoadState(key)
	if err == nil {
		t.Error("expected error on second LoadState (file should be deleted)")
	}
}

func TestLoadStateMissing(t *testing.T) {
	old := hookStateDir
	hookStateDir = filepath.Join(t.TempDir(), ".why-hook-state-missing")
	defer func() { hookStateDir = old }()

	_, err := LoadState("nonexistent")
	if err == nil {
		t.Error("expected error for missing state")
	}
}

func TestTurnIDLifecycle(t *testing.T) {
	old := turnDir
	turnDir = filepath.Join(t.TempDir(), ".why-turn")
	defer func() { turnDir = old }()

	// Initially empty
	if got := ReadTurnID(); got != "" {
		t.Errorf("initial: want empty, got %q", got)
	}

	// Write
	if err := WriteTurnID("sess:789:1234"); err != nil {
		t.Fatalf("WriteTurnID: %v", err)
	}
	if got := ReadTurnID(); got != "sess:789:1234" {
		t.Errorf("after write: want sess:789:1234, got %q", got)
	}

	// Clear
	ClearTurnID()
	if got := ReadTurnID(); got != "" {
		t.Errorf("after clear: want empty, got %q", got)
	}
}

func TestFileKey(t *testing.T) {
	key1 := FileKey("/Users/ed/project/src/main.go")
	key2 := FileKey("/Users/ed/project/src/main.go")
	key3 := FileKey("/Users/ed/project/src/other.go")

	if key1 != key2 {
		t.Errorf("same path should produce same key: %s vs %s", key1, key2)
	}
	if key1 == key3 {
		t.Error("different paths should produce different keys")
	}
	if len(key1) != 16 {
		t.Errorf("key length: want 16, got %d", len(key1))
	}

	// Verify it doesn't contain path separators
	if _, err := os.Stat(filepath.Join(t.TempDir(), key1)); err == nil {
		t.Error("key should not create files")
	}
}
