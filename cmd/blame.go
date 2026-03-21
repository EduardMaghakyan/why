package cmd

import (
	"fmt"
	"os"
	"strings"

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
	sourceLines := strings.Split(strings.TrimSuffix(string(sourceBytes), "\n"), "\n")

	whyStore := store.New(".why")
	refs := store.NewRefs(".why")
	hashes, _ := refs.Read(filePath)

	fmt.Println(filePath)
	fmt.Println()

	cache := map[string]*store.Object{}
	prevHash := ""

	for i, line := range sourceLines {
		hash := ""
		if i < len(hashes) {
			hash = hashes[i]
		}

		// Print header when entering a new reasoning group
		if hash != prevHash && hash != "" {
			obj, ok := cache[hash]
			if !ok {
				obj, err = whyStore.Get(hash)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%4d │ %s\n", i+1, line)
					prevHash = hash
					continue
				}
				cache[hash] = obj
			}
			summary := truncate(obj.Reasoning, 70)
			fmt.Printf("── %s: %s ──\n", obj.Commit, summary)
		}

		fmt.Printf("%4d │ %s\n", i+1, line)
		prevHash = hash
	}

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
