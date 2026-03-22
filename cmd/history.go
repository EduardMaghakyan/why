package cmd

import (
	"fmt"
	"sort"

	"github.com/eduardmaghakyan/why/internal/store"
	"github.com/spf13/cobra"
)

var historyRelated bool

var historyCmd = &cobra.Command{
	Use:   "history <file>",
	Short: "Show edit history with reasoning for a file",
	Args:  cobra.ExactArgs(1),
	RunE:  runHistory,
}

func init() {
	historyCmd.Flags().BoolVar(&historyRelated, "related", false, "Show files changed together")
	rootCmd.AddCommand(historyCmd)
}

func runHistory(cmd *cobra.Command, args []string) error {
	filePath := relPath(args[0])

	whyStore := store.New(".why")
	refs := store.NewRefs(".why")
	hashes, _ := refs.Read(filePath)

	// Deduplicate, preserving order of first appearance
	seen := map[string]bool{}
	var unique []string
	for _, h := range hashes {
		if h != "" && !seen[h] {
			seen[h] = true
			unique = append(unique, h)
		}
	}

	type entry struct {
		hash string
		obj  *store.Object
	}
	var entries []entry
	for _, h := range unique {
		obj, err := whyStore.Get(h)
		if err != nil {
			continue
		}
		entries = append(entries, entry{h, obj})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].obj.Timestamp < entries[j].obj.Timestamp
	})

	// Deduplicate by reasoning text (keep earliest)
	seenReasoning := map[string]bool{}
	var deduped []entry
	for _, e := range entries {
		if !seenReasoning[e.obj.Reasoning] {
			seenReasoning[e.obj.Reasoning] = true
			deduped = append(deduped, e)
		}
	}
	entries = deduped

	for _, e := range entries {
		fmt.Printf("## %s | %s\n\n%s\n", e.obj.Timestamp, e.obj.Commit, e.obj.Reasoning)

		if historyRelated {
			related := refs.FindRelated(filePath, whyStore)
			if len(related) > 0 {
				fmt.Printf("\n  Also changed:\n")
				for _, r := range related {
					fmt.Printf("    %s\n", r)
				}
			}
		}

		fmt.Printf("\n---\n\n")
	}

	return nil
}
