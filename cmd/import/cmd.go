package importcmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FreeDictionary/load2db-go/db"
	"github.com/spf13/cobra"
)

var (
	// Raw data path flag (defined in parent command)
	RawDataPath string

	// Database connection flags
	dbHost     string
	dbPort     uint16
	dbName     string
	dbUser     string
	dbPassword string

	// Import options
	batchSize    int
	showProgress bool
	truncate     bool
	skipIndexes  bool

	// Import command
	Cmd = &cobra.Command{
		Use:   "import",
		Short: "Import wiktionary JSONL data into PostgreSQL",
		Long:  `Import wiktionary JSONL data into PostgreSQL database`,
		RunE:  runImport,
	}
)

func init() {
	// Database connection flags
	Cmd.Flags().StringVar(&dbHost, "db-host", "localhost", "PostgreSQL host")
	Cmd.Flags().Uint16Var(&dbPort, "db-port", 5432, "PostgreSQL port")
	Cmd.Flags().StringVar(&dbName, "db-name", "wiktionary", "PostgreSQL database name")
	Cmd.Flags().StringVar(&dbUser, "db-user", "postgres", "PostgreSQL user")
	Cmd.Flags().StringVar(&dbPassword, "db-password", "", "PostgreSQL password")

	// Import options
	Cmd.Flags().IntVar(&batchSize, "batch-size", 1000, "Batch size for bulk inserts")
	Cmd.Flags().BoolVar(&showProgress, "progress", true, "Show import progress")
	Cmd.Flags().BoolVar(&truncate, "truncate", false, "Truncate table before import")
	Cmd.Flags().BoolVar(&skipIndexes, "skip-indexes", false, "Skip creating indexes (useful for initial load)")

	// Mark required flags
	if err := Cmd.MarkFlagRequired("db-password"); err != nil {
		log.Fatalf("failed to mark flag required: %v", err)
	}
}

func runImport(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received interrupt signal, shutting down...")
		cancel()
	}()

	// Get raw data path from parent command
	if RawDataPath == "" {
		return fmt.Errorf("raw-data flag is required")
	}

	// Create database connection
	log.Println("Connecting to PostgreSQL...")
	dbConfig := db.Config{
		Host:     dbHost,
		Port:     dbPort,
		Database: dbName,
		User:     dbUser,
		Password: dbPassword,
	}

	database, err := db.NewPostgresDB(ctx, dbConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer database.Close()

	// Setup database schema
	if !skipIndexes {
		if err := SetupDatabase(ctx, database); err != nil {
			return fmt.Errorf("failed to setup database: %w", err)
		}
	} else {
		// Just create tables without indexes
		if err := database.CreateTables(ctx); err != nil {
			return fmt.Errorf("failed to create tables: %w", err)
		}
	}

	// Truncate table if requested
	if truncate {
		log.Println("Truncating table...")
		if err := database.TruncateTable(ctx); err != nil {
			return fmt.Errorf("failed to truncate table: %w", err)
		}
	}

	// Create importer
	importer := NewImporter(ImporterConfig{
		DB:           database,
		BatchSize:    batchSize,
		ShowProgress: showProgress,
	})

	// Check if path is a file or directory
	info, err := os.Stat(RawDataPath)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}

	startTime := time.Now()
	var totalInserted int64

	if info.IsDir() {
		log.Printf("Importing all JSONL files from directory: %s", RawDataPath)
		totalInserted, err = importer.ImportDirectory(ctx, RawDataPath)
	} else {
		log.Printf("Importing file: %s", RawDataPath)
		totalInserted, err = importer.ImportFile(ctx, RawDataPath)
	}

	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	// Create indexes if they were skipped during import
	if skipIndexes {
		log.Println("Creating indexes after import...")
		if err := database.CreateIndexes(ctx); err != nil {
			return fmt.Errorf("failed to create indexes: %w", err)
		}
	}

	elapsed := time.Since(startTime)
	count, err := database.GetWordCount(ctx)
	if err != nil {
		log.Printf("Warning: failed to get final count: %v", err)
	}

	log.Printf("Import completed successfully!")
	log.Printf("Total entries inserted: %d", totalInserted)
	log.Printf("Total entries in database: %d", count)
	log.Printf("Time elapsed: %v", elapsed)
	log.Printf("Rate: %.2f entries/second", float64(totalInserted)/elapsed.Seconds())

	return nil
}
