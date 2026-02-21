package cmd

import (
	"log"

	importcmd "github.com/FreeDictionary/load2db-go/cmd/import"
	"github.com/spf13/cobra"
	"golang.org/x/text/language"
)

var (
	RawDataPath string

	// SqlBackend    SQLBackend
	// sqlBackendRaw string

	langsRaw []string
	Langs    []language.Tag
	AllLangs bool
)

// type SQLBackend int

// const (
// 	POSTGRESQL SQLBackend = iota
// 	SQLITE
// )

// func (b SQLBackend) String() string {
// 	switch b {
// 	case POSTGRESQL:
// 		return "PostgreSQL"
// 	case SQLITE:
// 		return "SQLite"
// 	default:
// 		return "UNKNOWN"
// 	}
// }

var rootCmd = &cobra.Command{
	Use:     "load2db-go",
	Short:   "A tool to load wiktionary data to SQL database",
	Long:    `A tool to load wiktionary data to SQL database`,
	Version: "v0.1.0",
	Run: func(cmd *cobra.Command, args []string) {
		// Show help if no subcommand is provided
		cmd.Help()
	},
}

// func parseSQLBackend() {
// 	switch strings.ToUpper(sqlBackendRaw) {
// 	case strings.ToUpper(POSTGRESQL.String()):
// 		SqlBackend = POSTGRESQL
// 	case strings.ToUpper(SQLITE.String()):
// 		SqlBackend = SQLITE
// 	default:
// 		log.Fatalf("invalid SQL backend: %s", sqlBackendRaw)
// 	}
// }

func parseLangs() {
	// if len(langsRaw) == 0 {
	// 	Langs = []language.Tag{language.English}
	// }
	for _, lang := range langsRaw {
		tag, err := language.Parse(lang)
		if err != nil {
			log.Fatalf("failed to parse language tag: %v", err)
		}
		Langs = append(Langs, tag)
	}
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return err
	}
	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVar(&RawDataPath, "raw-data", "", "raw data file path")
	rootCmd.PersistentFlags().StringSliceVar(
		&langsRaw,
		"langs",
		[]string{"en"},
		"Languages to load",
	)
	rootCmd.PersistentFlags().BoolVar(
		&AllLangs,
		"all-langs",
		false,
		"Load all languages",
	)

	rootCmd.MarkFlagsMutuallyExclusive(
		"langs", "all-langs",
	)

	// Register subcommands
	rootCmd.AddCommand(importcmd.Cmd)
}
