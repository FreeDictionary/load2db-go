package importcmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/FreeDictionary/load2db-go/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WordData represents the structure of a wiktionary word entry
// This mirrors the schema from wiktionary-schema-go/en/schema.go
type WordData struct {
	Word     string          `json:"word"`
	Lang     string          `json:"lang"`
	LangCode string          `json:"lang_code"`
	Pos      string          `json:"pos"`
	Data     json.RawMessage `json:"-"` // Store the original JSON for re-serialization
}

// Importer handles the import of wiktionary JSONL data into PostgreSQL
type Importer struct {
	DB           *db.PostgresDB
	BatchSize    int
	ShowProgress bool
}

// ImporterConfig holds the configuration for the importer
type ImporterConfig struct {
	DB           *db.PostgresDB
	BatchSize    int
	ShowProgress bool
}

// NewImporter creates a new Importer instance
func NewImporter(cfg ImporterConfig) *Importer {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 1000
	}
	return &Importer{
		DB:           cfg.DB,
		BatchSize:    cfg.BatchSize,
		ShowProgress: cfg.ShowProgress,
	}
}

// ImportFile imports a single JSONL file into the database
func (imp *Importer) ImportFile(ctx context.Context, filePath string) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max line size

	var (
		batch        []db.WordEntry
		lineCount    int64
		successCount int64
		errorCount   int64
		lastProgress time.Time
	)

	if imp.ShowProgress {
		log.Printf("Starting import of %s", filePath)
	}

	for scanner.Scan() {
		lineCount++
		line := scanner.Bytes()

		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Parse and normalize the JSON
		entry, err := imp.normalizeJSON(line)
		if err != nil {
			errorCount++
			if imp.ShowProgress {
				log.Printf("Warning: failed to parse line %d: %v", lineCount, err)
			}
			continue
		}

		batch = append(batch, entry)

		// Flush batch when it reaches the batch size
		if len(batch) >= imp.BatchSize {
			if err := imp.DB.BatchInsertWords(ctx, batch); err != nil {
				return successCount, fmt.Errorf("failed to insert batch at line %d: %w", lineCount, err)
			}
			successCount += int64(len(batch))
			batch = batch[:0]

			// Show progress periodically
			if imp.ShowProgress && time.Since(lastProgress) > 5*time.Second {
				lastProgress = time.Now()
				log.Printf("Processed %d lines, inserted %d entries", lineCount, successCount)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return successCount, fmt.Errorf("error reading file: %w", err)
	}

	// Flush remaining batch
	if len(batch) > 0 {
		if err := imp.DB.BatchInsertWords(ctx, batch); err != nil {
			return successCount, fmt.Errorf("failed to insert final batch: %w", err)
		}
		successCount += int64(len(batch))
	}

	if imp.ShowProgress {
		log.Printf("Finished importing %s: %d lines processed, %d entries inserted, %d errors",
			filePath, lineCount, successCount, errorCount)
	}

	return successCount, nil
}

// normalizeJSON parses and re-serializes JSON to normalize it
func (imp *Importer) normalizeJSON(line []byte) (db.WordEntry, error) {
	// First, parse into a generic map to validate and normalize
	var rawData map[string]interface{}
	if err := json.Unmarshal(line, &rawData); err != nil {
		return db.WordEntry{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Extract required fields
	word, _ := rawData["word"].(string)
	lang, _ := rawData["lang"].(string)
	langCode, _ := rawData["lang_code"].(string)

	// Validate required fields
	if word == "" {
		return db.WordEntry{}, fmt.Errorf("missing required field: word")
	}
	if langCode == "" {
		return db.WordEntry{}, fmt.Errorf("missing required field: lang_code")
	}

	// Re-serialize to normalize JSON (consistent key order, formatting, etc.)
	normalizedData, err := json.Marshal(rawData)
	if err != nil {
		return db.WordEntry{}, fmt.Errorf("failed to marshal normalized JSON: %w", err)
	}

	return db.WordEntry{
		Word:     word,
		Lang:     lang,
		LangCode: langCode,
		Data:     normalizedData,
	}, nil
}

// ImportDirectory imports all JSONL files in a directory
func (imp *Importer) ImportDirectory(ctx context.Context, dirPath string) (int64, error) {
	var totalInserted int64

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .jsonl or .json files
		ext := filepath.Ext(path)
		if ext != ".jsonl" && ext != ".json" {
			return nil
		}

		count, err := imp.ImportFile(ctx, path)
		if err != nil {
			return fmt.Errorf("failed to import %s: %w", path, err)
		}

		atomic.AddInt64(&totalInserted, count)
		return nil
	})

	if err != nil {
		return totalInserted, fmt.Errorf("error walking directory: %w", err)
	}

	return totalInserted, nil
}

// SetupDatabase initializes the database schema and indexes
func SetupDatabase(ctx context.Context, db *db.PostgresDB) error {
	log.Println("Creating database tables...")
	if err := db.CreateTables(ctx); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	log.Println("Creating indexes...")
	if err := db.CreateIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// QuickCount performs a quick count of records in the database
func QuickCount(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	var count int64
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM wiktionary_words").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count records: %w", err)
	}
	return count, nil
}
