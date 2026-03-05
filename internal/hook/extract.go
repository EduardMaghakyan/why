package hook

import "encoding/json"

// HookInput matches the JSON structure Claude Code sends to hooks.
type HookInput struct {
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
}

// ExtractPaths extracts file path(s) from various tool_input shapes.
// Handles Edit (path), Write (file_path), MultiEdit (edits[].path).
func ExtractPaths(raw json.RawMessage) []string {
	// Try "path" (Edit)
	var edit struct {
		Path string `json:"path"`
	}
	if json.Unmarshal(raw, &edit) == nil && edit.Path != "" {
		return []string{edit.Path}
	}

	// Try "file_path" (Write)
	var write struct {
		FilePath string `json:"file_path"`
	}
	if json.Unmarshal(raw, &write) == nil && write.FilePath != "" {
		return []string{write.FilePath}
	}

	// Try "edits" array (MultiEdit)
	var multi struct {
		Edits []struct {
			Path string `json:"path"`
		} `json:"edits"`
	}
	if json.Unmarshal(raw, &multi) == nil && len(multi.Edits) > 0 {
		seen := map[string]bool{}
		var paths []string
		for _, e := range multi.Edits {
			if e.Path != "" && !seen[e.Path] {
				paths = append(paths, e.Path)
				seen[e.Path] = true
			}
		}
		return paths
	}

	return nil
}
