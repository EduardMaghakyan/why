package hook

import "encoding/json"

// HookInput matches the JSON structure Claude Code sends to hooks.
type HookInput struct {
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
}

// ExtractPaths extracts file path(s) from various tool_input shapes.
// Handles Edit (file_path), Write (file_path), MultiEdit (edits[].file_path).
func ExtractPaths(raw json.RawMessage) []string {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return nil
	}

	// "file_path" → Edit, Write
	if fp, ok := fields["file_path"]; ok {
		var s string
		if json.Unmarshal(fp, &s) == nil && s != "" {
			return []string{s}
		}
	}

	// "edits" → MultiEdit
	if editsRaw, ok := fields["edits"]; ok {
		var edits []struct {
			FilePath string `json:"file_path"`
		}
		if json.Unmarshal(editsRaw, &edits) == nil && len(edits) > 0 {
			seen := map[string]bool{}
			var paths []string
			for _, e := range edits {
				if e.FilePath != "" && !seen[e.FilePath] {
					paths = append(paths, e.FilePath)
					seen[e.FilePath] = true
				}
			}
			return paths
		}
	}

	// "path" → fallback
	if p, ok := fields["path"]; ok {
		var s string
		if json.Unmarshal(p, &s) == nil && s != "" {
			return []string{s}
		}
	}

	return nil
}
