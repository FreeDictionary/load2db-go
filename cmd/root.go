package cmd

import (
	importcmd "github.com/FreeDictionary/load2db-go/cmd/import"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "load2db-go",
	Short:   "A tool to load wiktionary data to PostgreSQL",
	Long:    `A tool to load wiktionary data to PostgreSQL database`,
	Version: "v0.1.0",
	Run: func(cmd *cobra.Command, args []string) {
		// Show help if no subcommand is provided
		cmd.Help()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&importcmd.RawDataPath,
		"raw-data",
		"",
		"Path to JSONL file or directory (required)",
	)
	rootCmd.MarkPersistentFlagRequired("raw-data")

	// Register subcommands
	rootCmd.AddCommand(importcmd.Cmd)
}
