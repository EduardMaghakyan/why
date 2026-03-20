package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eduardmaghakyan/why/internal/config"
	"github.com/spf13/cobra"
)

var setupProject bool

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Install why-tracking (globally by default)",
	RunE:  runSetup,
}

func init() {
	setupCmd.Flags().BoolVar(&setupProject, "project", false,
		"Install per-project only (.claude/settings.json + .mcp.json) instead of globally")
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	if setupProject {
		fmt.Printf("Installing why-tracking into: %s\n\n", projectDir)

		// 1. Create .claude/ directory
		claudeDir := filepath.Join(projectDir, ".claude")
		os.MkdirAll(claudeDir, 0755)

		// 2. Write .claude/settings.json
		if err := writeTemplate(projectDir, ".claude/settings.json", "templates/settings.json"); err != nil {
			return err
		}

		// 3. Write .mcp.json
		if err := writeTemplate(projectDir, ".mcp.json", "templates/mcp.json"); err != nil {
			return err
		}

		// 4. Write .claude/why-tracking.md
		if err := writeTemplate(projectDir, ".claude/why-tracking.md", "templates/why-tracking.md"); err != nil {
			return err
		}

		// 5. Patch CLAUDE.md
		claudeMD := filepath.Join(projectDir, "CLAUDE.md")
		include := "@.claude/why-tracking.md"
		if data, err := os.ReadFile(claudeMD); err != nil {
			os.WriteFile(claudeMD, []byte(include+"\n"), 0644)
			fmt.Println("  Created CLAUDE.md")
		} else if !strings.Contains(string(data), include) {
			f, _ := os.OpenFile(claudeMD, os.O_APPEND|os.O_WRONLY, 0644)
			f.WriteString("\n" + include + "\n")
			f.Close()
			fmt.Println("  Appended to CLAUDE.md")
		} else {
			fmt.Println("  CLAUDE.md already configured")
		}
	} else {
		fmt.Println("Installing why-tracking globally")

		// 1. Settings (hooks + permissions) → ~/.claude/settings.json
		if err := writeSettingsGlobal(); err != nil {
			return err
		}

		// 2. MCP config → ~/.claude.json
		if err := writeMCPGlobal(); err != nil {
			return err
		}

		// 3. Instructions → ~/.claude/CLAUDE.md
		if err := writeInstructionsGlobal(); err != nil {
			return err
		}
	}

	// Always per-project: .gitignore + .why/ directory
	gitignore := filepath.Join(projectDir, ".gitignore")
	addLineToFile(gitignore, ".why/", "  Added .why/ to .gitignore")

	os.MkdirAll(filepath.Join(projectDir, ".why", "objects"), 0755)
	os.MkdirAll(filepath.Join(projectDir, ".why", "refs"), 0755)

	fmt.Println("\nDone. why-tracking installed.")
	fmt.Println("\nUseful commands:")
	fmt.Println("  why blame <file>      # line-by-line reasoning")
	fmt.Println("  why history <file>    # edit history for a file")
	fmt.Println("  why uninstall         # remove why-tracking")
	return nil
}

func writeTemplate(projectDir, destRel, templatePath string) error {
	dest := filepath.Join(projectDir, destRel)

	if _, err := os.Stat(dest); err == nil {
		// File exists — check if it's ours or needs manual merge
		if destRel == ".claude/settings.json" {
			existing, _ := os.ReadFile(dest)
			if strings.Contains(string(existing), "why hook pre") {
				fmt.Printf("  %s already configured\n", destRel)
				return nil
			}
			// Try to merge permissions and hooks
			if err := mergeSettings(dest, templatePath); err != nil {
				fmt.Printf("  WARNING: %s exists. Merge manually.\n", destRel)
				return nil
			}
			fmt.Printf("  Merged into %s\n", destRel)
			return nil
		}
		if destRel == ".mcp.json" {
			existing, _ := os.ReadFile(dest)
			if strings.Contains(string(existing), "why-tracker") {
				fmt.Printf("  %s already configured\n", destRel)
				return nil
			}
			if err := mergeMCPConfig(dest, templatePath); err != nil {
				fmt.Printf("  WARNING: %s exists. Merge manually: %v\n", destRel, err)
				return nil
			}
			fmt.Printf("  Merged into %s\n", destRel)
			return nil
		}
		fmt.Printf("  %s already exists, skipping\n", destRel)
		return nil
	}

	data, err := config.Templates.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("read template %s: %w", templatePath, err)
	}
	os.MkdirAll(filepath.Dir(dest), 0755)
	if err := os.WriteFile(dest, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", destRel, err)
	}
	fmt.Printf("  Created %s\n", destRel)
	return nil
}

func mergeSettings(destPath, templatePath string) error {
	existing, err := os.ReadFile(destPath)
	if err != nil {
		return err
	}
	templateData, err := config.Templates.ReadFile(templatePath)
	if err != nil {
		return err
	}

	var dest, src map[string]interface{}
	if err := json.Unmarshal(existing, &dest); err != nil {
		return err
	}
	if err := json.Unmarshal(templateData, &src); err != nil {
		return err
	}

	// Merge permissions.allow
	if srcPerms, ok := src["permissions"].(map[string]interface{}); ok {
		if srcAllow, ok := srcPerms["allow"].([]interface{}); ok {
			destPerms, _ := dest["permissions"].(map[string]interface{})
			if destPerms == nil {
				destPerms = map[string]interface{}{}
				dest["permissions"] = destPerms
			}
			destAllow, _ := destPerms["allow"].([]interface{})
			for _, item := range srcAllow {
				found := false
				for _, existing := range destAllow {
					if existing == item {
						found = true
						break
					}
				}
				if !found {
					destAllow = append(destAllow, item)
				}
			}
			destPerms["allow"] = destAllow
		}
	}

	// Merge hooks
	if srcHooks, ok := src["hooks"].(map[string]interface{}); ok {
		destHooks, _ := dest["hooks"].(map[string]interface{})
		if destHooks == nil {
			destHooks = map[string]interface{}{}
			dest["hooks"] = destHooks
		}
		for key, val := range srcHooks {
			if _, exists := destHooks[key]; !exists {
				destHooks[key] = val
			} else {
				// Append hook entries, skipping duplicates
				srcArr, _ := val.([]interface{})
				destArr, _ := destHooks[key].([]interface{})
				for _, srcEntry := range srcArr {
					if !hookEntryExists(destArr, srcEntry) {
						destArr = append(destArr, srcEntry)
					}
				}
				destHooks[key] = destArr
			}
		}
	}

	out, err := json.MarshalIndent(dest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(destPath, append(out, '\n'), 0644)
}

// hookEntryExists checks if a hook entry with the same matcher and command already exists.
func hookEntryExists(arr []interface{}, entry interface{}) bool {
	entryMap, ok := entry.(map[string]interface{})
	if !ok {
		return false
	}
	entryMatcher, _ := entryMap["matcher"].(string)
	for _, existing := range arr {
		existingMap, ok := existing.(map[string]interface{})
		if !ok {
			continue
		}
		if existingMatcher, _ := existingMap["matcher"].(string); existingMatcher == entryMatcher {
			return true
		}
	}
	return false
}

// mergeMCPConfig merges the why-tracker server into an existing .mcp.json.
func mergeMCPConfig(destPath, templatePath string) error {
	existing, err := os.ReadFile(destPath)
	if err != nil {
		return err
	}
	templateData, err := config.Templates.ReadFile(templatePath)
	if err != nil {
		return err
	}

	var dest, src map[string]interface{}
	if err := json.Unmarshal(existing, &dest); err != nil {
		return err
	}
	if err := json.Unmarshal(templateData, &src); err != nil {
		return err
	}

	// Merge mcpServers
	srcServers, _ := src["mcpServers"].(map[string]interface{})
	destServers, _ := dest["mcpServers"].(map[string]interface{})
	if destServers == nil {
		destServers = map[string]interface{}{}
		dest["mcpServers"] = destServers
	}
	for name, serverCfg := range srcServers {
		if _, exists := destServers[name]; !exists {
			destServers[name] = serverCfg
		}
	}

	out, err := json.MarshalIndent(dest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(destPath, append(out, '\n'), 0644)
}

func writeSettingsGlobal() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}
	destPath := filepath.Join(homeDir, ".claude", "settings.json")

	if _, err := os.Stat(destPath); err == nil {
		existing, _ := os.ReadFile(destPath)
		if strings.Contains(string(existing), "why hook pre") {
			fmt.Println("  ~/.claude/settings.json already configured")
			return nil
		}
		if err := mergeSettings(destPath, "templates/settings.json"); err != nil {
			fmt.Printf("  WARNING: ~/.claude/settings.json exists. Merge manually: %v\n", err)
			return nil
		}
		fmt.Println("  Merged why-tracking into ~/.claude/settings.json")
		return nil
	}

	data, err := config.Templates.ReadFile("templates/settings.json")
	if err != nil {
		return fmt.Errorf("read settings template: %w", err)
	}
	os.MkdirAll(filepath.Dir(destPath), 0755)
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("write ~/.claude/settings.json: %w", err)
	}
	fmt.Println("  Created ~/.claude/settings.json with why-tracking hooks")
	return nil
}

func writeMCPGlobal() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}
	destPath := filepath.Join(homeDir, ".claude.json")

	if _, err := os.Stat(destPath); err == nil {
		existing, _ := os.ReadFile(destPath)
		if strings.Contains(string(existing), "why-tracker") {
			fmt.Println("  ~/.claude.json already has why-tracker configured")
			return nil
		}

		// Parse existing file, merge only mcpServers
		var dest map[string]interface{}
		if err := json.Unmarshal(existing, &dest); err != nil {
			fmt.Printf("  WARNING: ~/.claude.json exists but could not parse. Add why-tracker manually.\n")
			return nil
		}

		servers, _ := dest["mcpServers"].(map[string]interface{})
		if servers == nil {
			servers = map[string]interface{}{}
			dest["mcpServers"] = servers
		}
		servers["why-tracker"] = map[string]interface{}{
			"command": "why",
			"args":    []interface{}{"serve"},
		}

		out, err := json.MarshalIndent(dest, "", "  ")
		if err != nil {
			fmt.Printf("  WARNING: could not serialize ~/.claude.json: %v\n", err)
			return nil
		}
		if err := os.WriteFile(destPath, append(out, '\n'), 0644); err != nil {
			return fmt.Errorf("write ~/.claude.json: %w", err)
		}
		fmt.Println("  Merged why-tracker into ~/.claude.json")
		return nil
	}

	// File does not exist — create with just mcpServers
	data, err := config.Templates.ReadFile("templates/mcp.json")
	if err != nil {
		return fmt.Errorf("read mcp template: %w", err)
	}
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("write ~/.claude.json: %w", err)
	}
	fmt.Println("  Created ~/.claude.json with why-tracker")
	return nil
}

func writeInstructionsGlobal() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}
	destPath := filepath.Join(homeDir, ".claude", "CLAUDE.md")

	content, err := config.Templates.ReadFile("templates/why-tracking.md")
	if err != nil {
		return fmt.Errorf("read why-tracking template: %w", err)
	}

	if data, err := os.ReadFile(destPath); err == nil {
		if strings.Contains(string(data), "Before every edit, call `record_why`") {
			fmt.Println("  ~/.claude/CLAUDE.md already has why-tracking instructions")
			return nil
		}
		// Append to existing file
		f, err := os.OpenFile(destPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("open ~/.claude/CLAUDE.md: %w", err)
		}
		f.WriteString("\n" + string(content))
		f.Close()
		fmt.Println("  Appended why-tracking instructions to ~/.claude/CLAUDE.md")
		return nil
	}

	// File does not exist — create it
	os.MkdirAll(filepath.Dir(destPath), 0755)
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return fmt.Errorf("write ~/.claude/CLAUDE.md: %w", err)
	}
	fmt.Println("  Created ~/.claude/CLAUDE.md with why-tracking instructions")
	return nil
}

func addLineToFile(path, line, msg string) {
	data, err := os.ReadFile(path)
	if err == nil && strings.Contains(string(data), line) {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(line + "\n")
	fmt.Println(msg)
}
