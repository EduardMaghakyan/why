package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/eduardmaghakyan/why/internal/store"
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query <question>",
	Short: "Ask a question about code reasoning using Claude",
	Args:  cobra.ExactArgs(1),
	RunE:  runQuery,
}

func init() {
	rootCmd.AddCommand(queryCmd)
}

func runQuery(cmd *cobra.Command, args []string) error {
	question := args[0]

	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("claude CLI not found in PATH. Install it from https://claude.ai/download")
	}

	whyStore := store.New(".why")
	entries, err := whyStore.ListAll()
	if err != nil {
		return fmt.Errorf("read reasoning store: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no reasoning entries found in .why/objects/")
	}

	var context strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&context, "--- %s | %s | %s ---\n%s\n\n",
			e.Object.Timestamp, e.Object.Commit, e.Hash[:12], e.Object.Reasoning)
	}

	prompt := fmt.Sprintf(`You have access to a decision journal that records the reasoning behind code changes.

<reasoning-journal>
%s</reasoning-journal>

Question: %s

Answer based on the reasoning journal above. Be concise.`, context.String(), question)

	claude := exec.Command("claude", "-p", prompt)
	claude.Stdout = os.Stdout
	claude.Stderr = os.Stderr
	return claude.Run()
}
