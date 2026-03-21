package hook

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

var (
	pendingDir   = filepath.Join(os.TempDir(), ".why-pending")
	hookStateDir = filepath.Join(os.TempDir(), ".why-hook-state")
)

// PreState holds the state saved between pre and post hooks.
type PreState struct {
	FilePath      string `json:"file_path"`
	ReasoningHash string `json:"reasoning_hash"`
	Snapshot      string `json:"snapshot"`
}

// FileKey returns a short hash key for a file path.
func FileKey(absPath string) string {
	h := sha256.Sum256([]byte(absPath))
	return hex.EncodeToString(h[:])[:16]
}

// ReadPending reads the latest pending reasoning hash (non-destructive).
func ReadPending() string {
	data, err := os.ReadFile(filepath.Join(pendingDir, "latest"))
	if err != nil {
		return ""
	}
	return string(data)
}

// WritePending writes a reasoning hash as the latest pending.
func WritePending(hash string) error {
	if err := os.MkdirAll(pendingDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(pendingDir, "latest"), []byte(hash), 0644)
}

// SaveState saves pre-hook state for the post-hook to consume.
func (s *PreState) Save(key string) error {
	if err := os.MkdirAll(hookStateDir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(hookStateDir, key), data, 0644)
}

// TempDirs returns the pending and hook-state temp directory paths.
func TempDirs() (string, string) {
	return pendingDir, hookStateDir
}

// LoadState loads and deletes pre-hook state.
func LoadState(key string) (*PreState, error) {
	path := filepath.Join(hookStateDir, key)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	os.Remove(path)
	var state PreState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}
