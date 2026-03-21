package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/eduardmaghakyan/why/internal/hook"
	"github.com/eduardmaghakyan/why/internal/store"
	"github.com/spf13/cobra"
)

var recordCmd = &cobra.Command{
	Use:   "record <file> <reasoning>",
	Short: "Record reasoning for an upcoming edit",
	Args:  cobra.ExactArgs(2),
	RunE:  runRecord,
}

func init() {
	rootCmd.AddCommand(recordCmd)
}

func runRecord(cmd *cobra.Command, args []string) error {
	filePath := args[0]
	reasoning := args[1]

	whyStore := store.New(".why")
	obj := &store.Object{
		Timestamp: time.Now().Format("2006-01-02 15:04"),
		Commit:    gitCommit(),
		TurnID:    hook.ReadTurnID(),
		Reasoning: reasoning,
	}

	hash, err := whyStore.Put(obj)
	if err != nil {
		return fmt.Errorf("store object: %w", err)
	}

	if err := hook.WritePending(hash); err != nil {
		return fmt.Errorf("write pending: %w", err)
	}

	fmt.Printf("Reasoning recorded for %s. Proceed with your edit.\n", filePath)
	return nil
}

func gitCommit() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "no-git"
	}
	return strings.TrimSpace(string(out))
}
