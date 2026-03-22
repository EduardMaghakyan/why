package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eduardmaghakyan/why/internal/store"
	"github.com/eduardmaghakyan/why/internal/symbols"
	"github.com/spf13/cobra"
)

var reindexCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Rebuild symbol index from existing line-level refs",
	RunE:  runReindex,
}

func init() {
	rootCmd.AddCommand(reindexCmd)
}

func runReindex(cmd *cobra.Command, args []string) error {
	whyStore := store.New(".why")
	refs := store.NewRefs(".why")
	symbolRefs := store.NewSymbolRefs(".why")
	refsDir := filepath.Join(".why", "refs")

	var filesProcessed, symbolsIndexed int

	err := filepath.Walk(refsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		filePath, err := filepath.Rel(refsDir, path)
		if err != nil {
			return nil
		}

		// Read source file
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil // file deleted since tracking, skip
		}

		// Extract symbols
		syms := symbols.Extract(filePath, content)
		if len(syms) == 0 {
			return nil // no symbols or unsupported language
		}

		// Read line hashes
		hashes, _ := refs.Read(filePath)
		if len(hashes) == 0 {
			return nil
		}

		// Map each line with a hash to its containing symbol
		seen := map[string]map[string]bool{} // symbol → set of hashes already appended
		count := 0

		for i, hash := range hashes {
			if hash == "" {
				continue
			}

			sym := symbols.FindAt(syms, i+1)
			symName := "_module"
			if sym != nil {
				symName = sym.Name
			}

			if seen[symName] == nil {
				seen[symName] = map[string]bool{}
			}
			if seen[symName][hash] {
				continue
			}
			seen[symName][hash] = true

			// Get timestamp from object
			ts := ""
			if obj, err := whyStore.Get(hash); err == nil {
				ts = obj.Timestamp
			}

			symbolRefs.Append(filePath, symName, hash, ts)
			count++
		}

		if count > 0 {
			fmt.Printf("  %s: %d symbols indexed\n", filePath, count)
			filesProcessed++
			symbolsIndexed += count
		}

		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("\nReindexed %d files, %d symbol entries\n", filesProcessed, symbolsIndexed)
	return nil
}
