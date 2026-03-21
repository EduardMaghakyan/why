package hook

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTranscript(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	var content string
	for _, line := range lines {
		content += line + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractLastReasoningMCP(t *testing.T) {
	path := writeTranscript(t,
		`{"type":"user","message":{"content":"fix the bug"}}`,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"mcp__why-tracker__record_why","input":{"file_path":"src/main.go","reasoning":"Token refresh race condition fix"}}]}}`,
	)

	got := ExtractLastReasoning(path)
	if got != "Token refresh race condition fix" {
		t.Errorf("want 'Token refresh race condition fix', got %q", got)
	}
}

func TestExtractLastReasoningBash(t *testing.T) {
	path := writeTranscript(t,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"why record src/main.go 'Added guard for race condition'"}}]}}`,
	)

	got := ExtractLastReasoning(path)
	if got != "Added guard for race condition" {
		t.Errorf("want 'Added guard for race condition', got %q", got)
	}
}

func TestExtractLastReasoningMultiple(t *testing.T) {
	path := writeTranscript(t,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"mcp__why-tracker__record_why","input":{"file_path":"a.go","reasoning":"first reason"}}]}}`,
		`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"mcp__why-tracker__record_why","input":{"file_path":"b.go","reasoning":"second reason"}}]}}`,
	)

	got := ExtractLastReasoning(path)
	if got != "second reason" {
		t.Errorf("want 'second reason', got %q", got)
	}
}

func TestExtractLastReasoningEmpty(t *testing.T) {
	path := writeTranscript(t,
		`{"type":"user","message":{"content":"hello"}}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Hi there!"}]}}`,
	)

	got := ExtractLastReasoning(path)
	if got != "" {
		t.Errorf("want empty, got %q", got)
	}
}

func TestExtractLastReasoningMissingFile(t *testing.T) {
	got := ExtractLastReasoning("/nonexistent/path/transcript.jsonl")
	if got != "" {
		t.Errorf("want empty, got %q", got)
	}
}

func TestExtractLastReasoningEmptyPath(t *testing.T) {
	got := ExtractLastReasoning("")
	if got != "" {
		t.Errorf("want empty, got %q", got)
	}
}

func TestParseWhyRecordCmd(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{
			name: "single quoted reasoning",
			cmd:  "why record src/main.go 'Fix race condition'",
			want: "Fix race condition",
		},
		{
			name: "double quoted reasoning",
			cmd:  `why record src/main.go "Fix race condition"`,
			want: "Fix race condition",
		},
		{
			name: "unquoted reasoning",
			cmd:  "why record src/main.go Fix race condition",
			want: "Fix race condition",
		},
		{
			name: "quoted file path",
			cmd:  `why record "src/my file.go" 'reasoning here'`,
			want: "reasoning here",
		},
		{
			name: "single quoted file path",
			cmd:  "why record 'src/my file.go' 'reasoning here'",
			want: "reasoning here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseWhyRecordCmd(tt.cmd)
			if got != tt.want {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		})
	}
}

func TestParseWhyRecordCmdEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{
			name: "no reasoning",
			cmd:  "why record src/main.go",
			want: "",
		},
		{
			name: "empty after prefix",
			cmd:  "why record ",
			want: "",
		},
		{
			name: "just the prefix",
			cmd:  "why record",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseWhyRecordCmd(tt.cmd)
			if got != tt.want {
				t.Errorf("want %q, got %q", tt.want, got)
			}
		})
	}
}
