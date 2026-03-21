package hook

import (
	"bufio"
	"encoding/json"
	"os"
)

// ExtractLastReasoning reads the transcript JSONL and returns the reasoning
// from the most recent record_why call. Returns empty string if not found.
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
				} `json:"input"`
			}
			if json.Unmarshal(raw, &block) != nil {
				continue
			}
			if block.Type == "tool_use" && block.Name == "mcp__why-tracker__record_why" && block.Input.Reasoning != "" {
				lastReasoning = block.Input.Reasoning
			}
		}
	}

	return lastReasoning
}
