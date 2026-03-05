package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/eduardmaghakyan/why/internal/store"
	"github.com/spf13/cobra"
)

var blameCmd = &cobra.Command{
	Use:   "blame <file>",
	Short: "Show line-by-line reasoning for a file",
	Args:  cobra.ExactArgs(1),
	RunE:  runBlame,
}

func init() {
	rootCmd.AddCommand(blameCmd)
}

func runBlame(cmd *cobra.Command, args []string) error {
	filePath := relPath(args[0])

	sourceBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	sourceLines := strings.Split(string(sourceBytes), "\n")

	whyStore := store.New(".why")
	refs := store.NewRefs(".why")
	hashes, _ := refs.Read(filePath)

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	for i, line := range sourceLines {
		hash := ""
		if i < len(hashes) {
			hash = hashes[i]
		}

		if hash == "" {
			fmt.Fprintf(w, "%4d\t\t\t%s\n", i+1, line)
			continue
		}

		obj, err := whyStore.Get(hash)
		if err != nil {
			fmt.Fprintf(w, "%4d\t%s\t(missing)\t%s\n", i+1, hash[:8], line)
			continue
		}

		summary := truncate(obj.Reasoning, 60)
		fmt.Fprintf(w, "%4d\t%s\t%s\t%s\n", i+1, obj.Commit, summary, line)
	}
	w.Flush()
	return nil
}

func truncate(s string, max int) string {
	// Use first line only, truncate if needed
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}
