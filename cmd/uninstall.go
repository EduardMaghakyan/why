package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eduardmaghakyan/why/internal/hook"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove why-tracking from the current project",
	RunE:  runUninstall,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

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
			// Check if it's the only server
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

	// Clean temp files
	pendingDir, hookStateDir := hook.TempDirs()
	os.RemoveAll(pendingDir)
	os.RemoveAll(hookStateDir)
	fmt.Println("  Cleaned temp files")

	fmt.Println("\nDone. The .why/ directory is preserved.")
	fmt.Println("To delete reasoning data: rm -rf .why/")
	return nil
}
