package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eduardmaghakyan/why/internal/debug"
	"github.com/eduardmaghakyan/why/internal/hook"
	"github.com/eduardmaghakyan/why/internal/store"
	"github.com/eduardmaghakyan/why/internal/symbols"
	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:           "hook",
	Short:         "Hook handlers for Claude Code",
	SilenceErrors: true,
	SilenceUsage:  true,
}

var hookPreCmd = &cobra.Command{
	Use:           "pre",
	Short:         "PreToolUse hook handler",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE:          safeRunHook("hook-pre", runHookPre),
}

var hookPostCmd = &cobra.Command{
	Use:           "post",
	Short:         "PostToolUse hook handler",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE:          safeRunHook("hook-post", runHookPost),
}

var hookTurnStartCmd = &cobra.Command{
	Use:           "turn-start",
	Short:         "UserPromptSubmit hook handler",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE:          safeRunHook("hook-turn-start", runHookTurnStart),
}

var hookTurnEndCmd = &cobra.Command{
	Use:           "turn-end",
	Short:         "Stop hook handler",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE:          safeRunHook("hook-turn-end", runHookTurnEnd),
}

func init() {
	hookCmd.AddCommand(hookPreCmd)
	hookCmd.AddCommand(hookPostCmd)
	hookCmd.AddCommand(hookTurnStartCmd)
	hookCmd.AddCommand(hookTurnEndCmd)
	rootCmd.AddCommand(hookCmd)
}

// safeRunHook wraps a hook handler with panic recovery, debug logging,
// and guaranteed {} output. Hooks always exit 0 — they are non-critical.
func safeRunHook(name string, fn func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) (retErr error) {
		debug.Init()
		debug.CaptureStderr()
		defer debug.Close()
		start := time.Now()
		debug.Log("[%s] started", name)

		defer func() {
			if r := recover(); r != nil {
				debug.Log("[%s] PANIC: %v\n%s", name, r, debug.Stack())
			}
			debug.Log("[%s] finished in %s", name, time.Since(start))
			fmt.Println("{}")
		}()

		if err := fn(cmd, args); err != nil {
			debug.Log("[%s] error: %v", name, err)
		}
		return nil
	}
}

// safeExtract wraps symbols.Extract with panic recovery for CGO safety.
func safeExtract(filePath string, content []byte) (syms []symbols.Symbol) {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("[hook-post] symbols.Extract panic for %s: %v", filePath, r)
			syms = nil
		}
	}()
	return symbols.Extract(filePath, content)
}

func runHookPre(cmd *cobra.Command, args []string) error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	debug.Log("[hook-pre] stdin: %s", string(input))

	var hookInput hook.HookInput
	if err := json.Unmarshal(input, &hookInput); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	paths := hook.ExtractPaths(hookInput.ToolInput)
	sharedHash := hook.ReadPending()
	turnID := hook.ReadTurnID()
	debug.Log("[hook-pre] paths=%v hash=%s turnID=%s", paths, sharedHash, turnID)

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
				debug.Log("[hook-pre] fallback reasoning hash=%s", hash)
			} else {
				debug.Log("[hook-pre] store.Put failed: %v", err)
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
			debug.Log("[hook-pre] abs(%s) failed: %v", filePath, err)
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
		if err := state.Save(key); err != nil {
			debug.Log("[hook-pre] state.Save(%s) failed: %v", key, err)
		}
	}

	return nil
}

func runHookPost(cmd *cobra.Command, args []string) error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}
	debug.Log("[hook-post] stdin: %s", string(input))

	var hookInput hook.HookInput
	if err := json.Unmarshal(input, &hookInput); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	paths := hook.ExtractPaths(hookInput.ToolInput)
	refs := store.NewRefs(".why")

	for _, filePath := range paths {
		fileStart := time.Now()
		filePath = relPath(filePath)
		if shouldSkip(filePath) {
			continue
		}

		absPath, err := filepath.Abs(filePath)
		if err != nil {
			debug.Log("[hook-post] abs(%s) failed: %v", filePath, err)
			continue
		}
		key := hook.FileKey(absPath)

		// Load pre-hook state
		state, err := hook.LoadState(key)
		if err != nil {
			debug.Log("[hook-post] LoadState(%s) failed: %v", key, err)
			continue
		}

		// Fallback: try reading pending hash if pre-hook missed it
		if state.ReasoningHash == "" {
			state.ReasoningHash = hook.ReadPending()
		}

		// Read current file
		newContent, err := os.ReadFile(filePath)
		if err != nil {
			debug.Log("[hook-post] ReadFile(%s) failed: %v", filePath, err)
			continue
		}

		oldLines := splitLines(state.Snapshot)
		newLines := splitLines(string(newContent))

		// Read existing refs
		oldHashes, _ := refs.Read(filePath)

		// Rebuild refs
		newHashes := refs.Rebuild(oldLines, newLines, oldHashes, state.ReasoningHash)

		// Write refs
		if err := refs.Write(filePath, newHashes); err != nil {
			debug.Log("[hook-post] refs.Write(%s) failed: %v", filePath, err)
		}

		// Update symbol-level reasoning
		if state.ReasoningHash != "" {
			syms := safeExtract(filePath, newContent)
			if len(syms) > 0 {
				symbolRefs := store.NewSymbolRefs(".why")
				seen := map[string]bool{}
				for i, h := range newHashes {
					if h != state.ReasoningHash {
						continue
					}
					sym := symbols.FindAt(syms, i+1)
					symName := "_module"
					if sym != nil {
						symName = sym.Name
					}
					if !seen[symName] {
						if err := symbolRefs.Append(filePath, symName, state.ReasoningHash, time.Now().Format("2006-01-02 15:04")); err != nil {
							debug.Log("[hook-post] symbolRefs.Append(%s, %s) failed: %v", filePath, symName, err)
						}
						seen[symName] = true
					}
				}
			}
		}
		debug.Log("[hook-post] processed %s in %s", filePath, time.Since(fileStart))
	}

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
		return fmt.Errorf("read stdin: %w", err)
	}

	var hookInput struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(input, &hookInput); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	turnID := fmt.Sprintf("%s:%d", hookInput.SessionID, time.Now().UnixMilli())
	hook.WriteTurnID(turnID)
	debug.Log("[hook-turn-start] turnID=%s", turnID)

	return nil
}

func runHookTurnEnd(cmd *cobra.Command, args []string) error {
	io.ReadAll(os.Stdin)
	hook.ClearTurnID()
	return nil
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}
