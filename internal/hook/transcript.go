package hook

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
)

// ExtractLastReasoning reads the transcript JSONL and returns the reasoning
// from the most recent record_why call (MCP or Bash). Returns empty string if not found.
func ExtractLastReasoning(transcriptPath string) string {
	if transcriptPath == "" {
		return ""
	}

	f, err := os.Open(transcriptPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	var lastReasoning string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for long lines

	for scanner.Scan() {
		line := scanner.Bytes()

		var entry struct {
			Type    string `json:"type"`
			Message struct {
				Content []json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &entry) != nil || entry.Type != "assistant" {
			continue
		}

		for _, raw := range entry.Message.Content {
			var block struct {
				Type  string `json:"type"`
				Name  string `json:"name"`
				Input struct {
					Reasoning string `json:"reasoning"`
					Command   string `json:"command"`
				} `json:"input"`
			}
			if json.Unmarshal(raw, &block) != nil || block.Type != "tool_use" {
				continue
			}

			// MCP record_why
			if block.Name == "mcp__why-tracker__record_why" && block.Input.Reasoning != "" {
				lastReasoning = block.Input.Reasoning
			}

			// Bash: why record <file> '<reasoning>'
			if block.Name == "Bash" && strings.HasPrefix(block.Input.Command, "why record ") {
				if r := parseWhyRecordCmd(block.Input.Command); r != "" {
					lastReasoning = r
				}
			}
		}
	}

	return lastReasoning
}

// parseWhyRecordCmd extracts the reasoning argument from a "why record <file> '<reasoning>'" command.
func parseWhyRecordCmd(cmd string) string {
	// Strip "why record "
	if !strings.HasPrefix(cmd, "why record ") {
		return ""
	}
	rest := cmd[len("why record "):]

	// Skip the file argument (first non-quoted token)
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return ""
	}

	// Skip file arg
	if rest[0] == '\'' || rest[0] == '"' {
		// Quoted file path — find closing quote
		q := rest[0]
		end := strings.IndexByte(rest[1:], q)
		if end < 0 {
			return ""
		}
		rest = strings.TrimSpace(rest[end+2:])
	} else {
		// Unquoted file path
		idx := strings.IndexByte(rest, ' ')
		if idx < 0 {
			return ""
		}
		rest = strings.TrimSpace(rest[idx+1:])
	}

	// The rest is the reasoning, possibly quoted
	if len(rest) >= 2 && (rest[0] == '\'' || rest[0] == '"') {
		q := rest[0]
		if rest[len(rest)-1] == byte(q) {
			return rest[1 : len(rest)-1]
		}
	}

	return rest
}
