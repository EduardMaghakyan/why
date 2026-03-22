package cmd

import (
	"fmt"

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

	names := symbolRefs.ListSymbols(filePath)
	fmt.Println(filePath)
	fmt.Println()

	for _, name := range names {
		entries := refs[name]
		fmt.Printf("%s (%d edit", name, len(entries))
		if len(entries) != 1 {
			fmt.Print("s")
		}
		fmt.Println(")")

		for _, e := range entries {
			obj, err := whyStore.Get(e.Hash)
			if err != nil {
				fmt.Printf("  %s | (missing)\n", e.Timestamp)
				continue
			}
			summary := truncate(obj.Reasoning, 70)
			fmt.Printf("  %s | %s\n", e.Timestamp, summary)
		}
		fmt.Println()
	}

	return nil
}
