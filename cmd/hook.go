package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"time"

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

var hookTurnStartCmd = &cobra.Command{
	Use:   "turn-start",
	Short: "UserPromptSubmit hook handler",
	RunE:  runHookTurnStart,
}

var hookTurnEndCmd = &cobra.Command{
	Use:   "turn-end",
	Short: "Stop hook handler",
	RunE:  runHookTurnEnd,
}

func init() {
	hookCmd.AddCommand(hookPreCmd)
	hookCmd.AddCommand(hookPostCmd)
	hookCmd.AddCommand(hookTurnStartCmd)
	hookCmd.AddCommand(hookTurnEndCmd)
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
	sharedHash := hook.ReadPending()
	turnID := hook.ReadTurnID()

	// Fallback: recover reasoning from transcript if pending hash is missing
	if sharedHash == "" && hookInput.TranscriptPath != "" {
		if reasoning := hook.ExtractLastReasoning(hookInput.TranscriptPath); reasoning != "" {
			whyStore := store.New(".why")
			obj := &store.Object{
				Timestamp: time.Now().Format("2006-01-02 15:04"),
				TurnID:    turnID,
				Reasoning: reasoning,
			}
			if hash, err := whyStore.Put(obj); err == nil {
				sharedHash = hash
			}
		}
	}

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

		content, _ := os.ReadFile(filePath)
		state := &hook.PreState{
			FilePath:      filePath,
			ReasoningHash: sharedHash,
			TurnID:        turnID,
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

		// Fallback: try reading pending hash if pre-hook missed it
		if state.ReasoningHash == "" {
			state.ReasoningHash = hook.ReadPending()
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

func runHookTurnStart(cmd *cobra.Command, args []string) error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	var hookInput struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(input, &hookInput); err != nil {
		return nil
	}

	turnID := fmt.Sprintf("%s:%d", hookInput.SessionID, time.Now().UnixMilli())
	hook.WriteTurnID(turnID)

	fmt.Println("{}")
	return nil
}

func runHookTurnEnd(cmd *cobra.Command, args []string) error {
	// Drain stdin (hook always sends input)
	io.ReadAll(os.Stdin)
	hook.ClearTurnID()
	fmt.Println("{}")
	return nil
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}
