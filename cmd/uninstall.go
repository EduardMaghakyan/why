package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eduardmaghakyan/why/internal/hook"
	"github.com/spf13/cobra"
)

var uninstallProject bool

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove why-tracking (from global config by default)",
	RunE:  runUninstall,
}

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallProject, "project", false,
		"Remove from project config only instead of global")
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	if uninstallProject {
		fmt.Printf("Removing why-tracking from: %s\n\n", projectDir)

		// Remove instruction file
		os.Remove(filepath.Join(projectDir, ".claude", "why-tracking.md"))
		fmt.Println("  Removed .claude/why-tracking.md")

		// Remove settings.json if it only has our config
		settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
		if data, err := os.ReadFile(settingsPath); err == nil {
			if strings.Contains(string(data), "why hook pre") {
				// TODO: could be smarter about partial removal
				fmt.Println("  NOTE: Remove why hook entries from .claude/settings.json manually")
			}
		}

		// Remove .mcp.json if it only has our server
		mcpPath := filepath.Join(projectDir, ".mcp.json")
		if data, err := os.ReadFile(mcpPath); err == nil {
			if strings.Contains(string(data), "why-tracker") {
				if !strings.Contains(string(data), `"mcpServers"`) || strings.Count(string(data), `"command"`) <= 1 {
					os.Remove(mcpPath)
					fmt.Println("  Removed .mcp.json")
				} else {
					fmt.Println("  NOTE: Remove why-tracker entry from .mcp.json manually")
				}
			}
		}

		// Remove @include from CLAUDE.md
		claudeMD := filepath.Join(projectDir, "CLAUDE.md")
		if data, err := os.ReadFile(claudeMD); err == nil {
			cleaned := strings.Replace(string(data), "\n@.claude/why-tracking.md\n", "\n", 1)
			cleaned = strings.Replace(cleaned, "@.claude/why-tracking.md\n", "", 1)
			os.WriteFile(claudeMD, []byte(cleaned), 0644)
			fmt.Println("  Cleaned CLAUDE.md")
		}
	} else {
		fmt.Println("Removing why-tracking from global config")
		removeMCPGlobal()
		removeSettingsGlobal()
		removeInstructionsGlobal()
	}

	// Clean temp files
	pendingDir, hookStateDir := hook.TempDirs()
	os.RemoveAll(pendingDir)
	os.RemoveAll(hookStateDir)
	fmt.Println("  Cleaned temp files")

	fmt.Println("\nDone. The .why/ directory is preserved.")
	fmt.Println("To delete reasoning data: rm -rf .why/")
	return nil
}

func removeMCPGlobal() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("  WARNING: could not determine home directory: %v\n", err)
		return
	}
	mcpPath := filepath.Join(homeDir, ".claude.json")
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		fmt.Println("  ~/.claude.json not found, nothing to clean up")
		return
	}
	if !strings.Contains(string(data), "why-tracker") {
		fmt.Println("  ~/.claude.json does not contain why-tracker")
		return
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Println("  NOTE: Remove why-tracker entry from ~/.claude.json manually")
		return
	}
	if servers, ok := cfg["mcpServers"].(map[string]interface{}); ok {
		delete(servers, "why-tracker")
		if len(servers) == 0 {
			delete(cfg, "mcpServers")
		}
	}

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fmt.Println("  NOTE: Remove why-tracker entry from ~/.claude.json manually")
		return
	}
	os.WriteFile(mcpPath, append(out, '\n'), 0644)
	fmt.Println("  Removed why-tracker from ~/.claude.json")
}

func removeSettingsGlobal() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("  WARNING: could not determine home directory: %v\n", err)
		return
	}
	settingsPath := filepath.Join(homeDir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		fmt.Println("  ~/.claude/settings.json not found, nothing to clean up")
		return
	}
	if !strings.Contains(string(data), "why hook pre") {
		fmt.Println("  ~/.claude/settings.json does not contain why-tracking hooks")
		return
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Println("  NOTE: Remove why hook entries from ~/.claude/settings.json manually")
		return
	}

	// Remove permission
	if perms, ok := cfg["permissions"].(map[string]interface{}); ok {
		if allow, ok := perms["allow"].([]interface{}); ok {
			var filtered []interface{}
			for _, item := range allow {
				if s, ok := item.(string); ok && strings.HasPrefix(s, "mcp__why-tracker__") {
					continue
				}
				filtered = append(filtered, item)
			}
			perms["allow"] = filtered
		}
	}

	// Remove hook entries that reference "why hook"
	if hooks, ok := cfg["hooks"].(map[string]interface{}); ok {
		for key, val := range hooks {
			arr, ok := val.([]interface{})
			if !ok {
				continue
			}
			var filtered []interface{}
			for _, entry := range arr {
				entryMap, ok := entry.(map[string]interface{})
				if !ok {
					filtered = append(filtered, entry)
					continue
				}
				hooksList, _ := entryMap["hooks"].([]interface{})
				isWhyHook := false
				for _, h := range hooksList {
					hMap, ok := h.(map[string]interface{})
					if !ok {
						continue
					}
					if cmd, _ := hMap["command"].(string); strings.Contains(cmd, "why hook") {
						isWhyHook = true
						break
					}
				}
				if !isWhyHook {
					filtered = append(filtered, entry)
				}
			}
			hooks[key] = filtered
		}
	}

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fmt.Println("  NOTE: Remove why hook entries from ~/.claude/settings.json manually")
		return
	}
	os.WriteFile(settingsPath, append(out, '\n'), 0644)
	fmt.Println("  Removed why-tracking hooks from ~/.claude/settings.json")
}

func removeInstructionsGlobal() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("  WARNING: could not determine home directory: %v\n", err)
		return
	}
	destPath := filepath.Join(homeDir, ".claude", "CLAUDE.md")
	data, err := os.ReadFile(destPath)
	if err != nil {
		fmt.Println("  ~/.claude/CLAUDE.md not found, nothing to clean up")
		return
	}

	content := string(data)
	if !strings.Contains(content, "Before every edit, call `record_why`") {
		fmt.Println("  ~/.claude/CLAUDE.md does not contain why-tracking instructions")
		return
	}

	// Remove the why-tracking section (from "# Why Tracking" to end of section or file)
	startIdx := strings.Index(content, "# Why Tracking")
	if startIdx < 0 {
		fmt.Println("  NOTE: Remove why-tracking instructions from ~/.claude/CLAUDE.md manually")
		return
	}

	// Find the next top-level heading after our section, or end of file
	rest := content[startIdx+len("# Why Tracking"):]
	endOffset := strings.Index(rest, "\n# ")
	var cleaned string
	if endOffset >= 0 {
		cleaned = content[:startIdx] + rest[endOffset+1:]
	} else {
		cleaned = strings.TrimRight(content[:startIdx], "\n")
	}

	if strings.TrimSpace(cleaned) == "" {
		os.Remove(destPath)
		fmt.Println("  Removed ~/.claude/CLAUDE.md (was only why-tracking)")
		return
	}

	os.WriteFile(destPath, []byte(cleaned), 0644)
	fmt.Println("  Removed why-tracking instructions from ~/.claude/CLAUDE.md")
}
