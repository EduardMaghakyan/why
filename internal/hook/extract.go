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
	// Try "file_path" (Edit, Write)
	var single struct {
		FilePath string `json:"file_path"`
	}
	if json.Unmarshal(raw, &single) == nil && single.FilePath != "" {
		return []string{single.FilePath}
	}

	// Try "edits" array (MultiEdit)
	var multi struct {
		Edits []struct {
			FilePath string `json:"file_path"`
		} `json:"edits"`
	}
	if json.Unmarshal(raw, &multi) == nil && len(multi.Edits) > 0 {
		seen := map[string]bool{}
		var paths []string
		for _, e := range multi.Edits {
			if e.FilePath != "" && !seen[e.FilePath] {
				paths = append(paths, e.FilePath)
				seen[e.FilePath] = true
			}
		}
		return paths
	}

	// Fallback: try "path" for forward compatibility
	var fallback struct {
		Path string `json:"path"`
	}
	if json.Unmarshal(raw, &fallback) == nil && fallback.Path != "" {
		return []string{fallback.Path}
	}

	return nil
}
