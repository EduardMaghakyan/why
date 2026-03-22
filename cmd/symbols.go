package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/eduardmaghakyan/why/internal/store"
	"github.com/spf13/cobra"
)

var symbolsCmd = &cobra.Command{
	Use:   "symbols <file>",
	Short: "Show reasoning grouped by symbol (function, class, method)",
	Args:  cobra.ExactArgs(1),
	RunE:  runSymbols,
}

func init() {
	rootCmd.AddCommand(symbolsCmd)
}

func runSymbols(cmd *cobra.Command, args []string) error {
	filePath := relPath(args[0])

	whyStore := store.New(".why")
	symbolRefs := store.NewSymbolRefs(".why")

	refs := symbolRefs.Read(filePath)
	if refs == nil || len(refs) == 0 {
		fmt.Printf("No symbol reasoning found for %s\n", filePath)
		return nil
	}

	// Collect unique hashes and map hash → symbols
	type entry struct {
		hash    string
		ts      string
		symbols []string
	}
	hashToSymbols := map[string]*entry{}
	var ordered []string

	for symName, entries := range refs {
		for _, e := range entries {
			if existing, ok := hashToSymbols[e.Hash]; ok {
				existing.symbols = append(existing.symbols, symName)
			} else {
				hashToSymbols[e.Hash] = &entry{
					hash:    e.Hash,
					ts:      e.Timestamp,
					symbols: []string{symName},
				}
				ordered = append(ordered, e.Hash)
			}
		}
	}

	// Sort by timestamp
	sort.Slice(ordered, func(i, j int) bool {
		return hashToSymbols[ordered[i]].ts < hashToSymbols[ordered[j]].ts
	})

	fmt.Println(filePath)
	fmt.Println()

	for _, hash := range ordered {
		e := hashToSymbols[hash]
		sort.Strings(e.symbols)

		obj, err := whyStore.Get(hash)
		if err != nil {
			fmt.Printf("── %s | (missing) ──\n\n", e.ts)
			continue
		}

		fmt.Printf("── %s | %s ──\n%s\n\n  Symbols: %s\n\n", e.ts, obj.Commit, obj.Reasoning, strings.Join(e.symbols, ", "))
	}

	return nil
}
