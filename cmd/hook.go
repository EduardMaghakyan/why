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
	for _, filePath := range paths {
		if shouldSkip(filePath) {
			continue
		}

		absPath, err := filepath.Abs(filePath)
		if err != nil {
			continue
		}
		key := hook.FileKey(absPath)

		// Read pending reasoning hash
		reasoningHash := hook.ReadPending(key)

		// Snapshot current file content
		content, _ := os.ReadFile(filePath)

		state := &hook.PreState{
			FilePath:      filePath,
			ReasoningHash: reasoningHash,
			Snapshot:      string(content),
		}
		state.Save(key)
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
	return strings.Split(s, "\n")
}
