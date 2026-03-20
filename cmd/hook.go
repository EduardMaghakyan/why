package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/eduardmaghakyan/why/internal/hook"
	"github.com/eduardmaghakyan/why/internal/store"
	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Hook handlers for Claude Code",
}

var hookPreCmd = &cobra.Command{
	Use:   "pre",
	Short: "PreToolUse hook handler",
	RunE:  runHookPre,
}

var hookPostCmd = &cobra.Command{
	Use:   "post",
	Short: "PostToolUse hook handler",
	RunE:  runHookPost,
}

func init() {
	hookCmd.AddCommand(hookPreCmd)
	hookCmd.AddCommand(hookPostCmd)
	rootCmd.AddCommand(hookCmd)
}

func runHookPre(cmd *cobra.Command, args []string) error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	var hookInput hook.HookInput
	if err := json.Unmarshal(input, &hookInput); err != nil {
		return nil // silently skip malformed input
	}

	paths := hook.ExtractPaths(hookInput.ToolInput)

	// First pass: collect pending hashes for all files, find shared hash for MultiEdit
	type fileInfo struct {
		relPath string
		absPath string
		key     string
	}
	var files []fileInfo
	var sharedHash string

	for _, filePath := range paths {
		filePath = relPath(filePath)
		if shouldSkip(filePath) {
			continue
		}
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			continue
		}
		key := hook.FileKey(absPath)
		files = append(files, fileInfo{relPath: filePath, absPath: absPath, key: key})

		// Read pending reasoning hash (consumed on read)
		if h := hook.ReadPending(key); h != "" && sharedHash == "" {
			sharedHash = h
		}
	}

	// Second pass: save pre-state using the shared hash for all files
	for _, f := range files {
		reasoningHash := sharedHash
		if reasoningHash == "" {
			fmt.Fprintf(os.Stderr, "warning: no reasoning recorded for %s (was record_why called?)\n", f.relPath)
		}

		content, _ := os.ReadFile(f.relPath)
		state := &hook.PreState{
			FilePath:      f.relPath,
			ReasoningHash: reasoningHash,
			Snapshot:      string(content),
		}
		state.Save(f.key)
	}

	fmt.Println("{}")
	return nil
}

func runHookPost(cmd *cobra.Command, args []string) error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	var hookInput hook.HookInput
	if err := json.Unmarshal(input, &hookInput); err != nil {
		return nil
	}

	paths := hook.ExtractPaths(hookInput.ToolInput)
	refs := store.NewRefs(".why")

	for _, filePath := range paths {
		filePath = relPath(filePath)
		if shouldSkip(filePath) {
			continue
		}

		absPath, err := filepath.Abs(filePath)
		if err != nil {
			continue
		}
		key := hook.FileKey(absPath)

		// Load pre-hook state
		state, err := hook.LoadState(key)
		if err != nil {
			continue
		}

		// Read current file
		newContent, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		oldLines := splitLines(state.Snapshot)
		newLines := splitLines(string(newContent))

		// Read existing refs
		oldHashes, _ := refs.Read(filePath)

		// Rebuild refs
		newHashes := refs.Rebuild(oldLines, newLines, oldHashes, state.ReasoningHash)

		// Write refs
		refs.Write(filePath, newHashes)
	}

	fmt.Println("{}")
	return nil
}

// relPath normalizes a path to be relative to cwd.
// Claude Code often sends absolute paths in tool_input;
// refs must use relative paths so blame lookups work.
func relPath(path string) string {
	if !filepath.IsAbs(path) {
		return path
	}
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}
	return rel
}

func shouldSkip(path string) bool {
	if strings.HasPrefix(path, ".why/") || strings.HasPrefix(path, ".why\\") {
		return true
	}
	if strings.Contains(path, ".claude/") || strings.Contains(path, ".claude\\") {
		return true
	}
	return false
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}
