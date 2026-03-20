package hook

import (
	"encoding/json"
	"testing"
)

func TestExtractPaths_FilePath(t *testing.T) {
	raw := json.RawMessage(`{"file_path": "src/main.go", "old_string": "foo", "new_string": "bar"}`)
	got := ExtractPaths(raw)
	if len(got) != 1 || got[0] != "src/main.go" {
		t.Fatalf("expected [src/main.go], got %v", got)
	}
}

func TestExtractPaths_MultiEdit(t *testing.T) {
	raw := json.RawMessage(`{"edits": [
		{"file_path": "a.go", "old_string": "x", "new_string": "y"},
		{"file_path": "b.go", "old_string": "x", "new_string": "y"},
		{"file_path": "a.go", "old_string": "z", "new_string": "w"}
	]}`)
	got := ExtractPaths(raw)
	if len(got) != 2 || got[0] != "a.go" || got[1] != "b.go" {
		t.Fatalf("expected [a.go b.go] (deduped), got %v", got)
	}
}

func TestExtractPaths_PathFallback(t *testing.T) {
	raw := json.RawMessage(`{"path": "/tmp/output.txt"}`)
	got := ExtractPaths(raw)
	if len(got) != 1 || got[0] != "/tmp/output.txt" {
		t.Fatalf("expected [/tmp/output.txt], got %v", got)
	}
}

func TestExtractPaths_Empty(t *testing.T) {
	got := ExtractPaths(json.RawMessage(`{}`))
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestExtractPaths_Malformed(t *testing.T) {
	got := ExtractPaths(json.RawMessage(`not json`))
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}
