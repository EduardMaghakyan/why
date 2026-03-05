package cmd

import (
	"github.com/eduardmaghakyan/why/internal/mcp"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run MCP server over stdio",
	RunE: func(cmd *cobra.Command, args []string) error {
		server := mcp.NewServer(".why", Version)
		return server.Run()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
