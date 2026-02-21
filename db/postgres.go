package db

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresDB wraps the connection pool and provides database operations
type PostgresDB struct {
	Pool *pgxpool.Pool
}

// Config holds the database connection configuration
type Config struct {
	Host     string
	Port     uint16
	Database string
	User     string
	Password string
}

// NewPostgresDB creates a new database connection pool
func NewPostgresDB(ctx context.Context, cfg Config) (*PostgresDB, error) {
	connString := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s",
		cfg.Host, cfg.Port, cfg.Database, cfg.User, cfg.Password,
	)

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return &PostgresDB{Pool: pool}, nil
}

// Close closes the database connection pool
func (db *PostgresDB) Close() {
	db.Pool.Close()
}

// CreateTables creates the necessary tables for wiktionary data
func (db *PostgresDB) CreateTables(ctx context.Context) error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS wiktionary_words (
		id BIGSERIAL PRIMARY KEY,
		word TEXT NOT NULL,
		lang TEXT NOT NULL,
		lang_code TEXT NOT NULL,
		data JSONB NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);
	`

	_, err := db.Pool.Exec(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	log.Println("Table 'wiktionary_words' created or already exists")
	return nil
}

// CreateIndexes creates indexes on the search columns
func (db *PostgresDB) CreateIndexes(ctx context.Context) error {
	indexes := []struct {
		name string
		sql  string
	}{
		{
			name: "idx_wiktionary_words_word",
			sql:  `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_wiktionary_words_word ON wiktionary_words(word)`,
		},
		{
			name: "idx_wiktionary_words_lang",
			sql:  `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_wiktionary_words_lang ON wiktionary_words(lang)`,
		},
		{
			name: "idx_wiktionary_words_lang_code",
			sql:  `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_wiktionary_words_lang_code ON wiktionary_words(lang_code)`,
		},
		{
			name: "idx_wiktionary_words_word_lang_code",
			sql:  `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_wiktionary_words_word_lang_code ON wiktionary_words(word, lang_code)`,
		},
		{
			name: "idx_wiktionary_words_data_gin",
			sql:  `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_wiktionary_words_data_gin ON wiktionary_words USING GIN(data)`,
		},
	}

	for _, idx := range indexes {
		_, err := db.Pool.Exec(ctx, idx.sql)
		if err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}
		log.Printf("Index '%s' created or already exists", idx.name)
	}

	return nil
}

// InsertWord inserts a single word entry into the database
func (db *PostgresDB) InsertWord(ctx context.Context, word, lang, langCode string, jsonData []byte) error {
	insertSQL := `
	INSERT INTO wiktionary_words (word, lang, lang_code, data)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (word, lang, lang_code) DO UPDATE SET
		data = EXCLUDED.data,
		updated_at = NOW();
	`

	_, err := db.Pool.Exec(ctx, insertSQL, word, lang, langCode, jsonData)
	if err != nil {
		return fmt.Errorf("failed to insert word: %w", err)
	}

	return nil
}

// BatchInsertWords inserts multiple word entries in a single transaction
func (db *PostgresDB) BatchInsertWords(ctx context.Context, words []WordEntry) error {
	if len(words) == 0 {
		return nil
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	batch := &pgx.Batch{}
	insertSQL := `
	INSERT INTO wiktionary_words (word, lang, lang_code, data)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (word, lang, lang_code) DO UPDATE SET
		data = EXCLUDED.data,
		updated_at = NOW();
	`

	for _, w := range words {
		batch.Queue(insertSQL, w.Word, w.Lang, w.LangCode, w.Data)
	}

	br := tx.SendBatch(ctx, batch)
	for i := 0; i < batch.Len(); i++ {
		_, err := br.Exec()
		if err != nil {
			br.Close()
			return fmt.Errorf("failed to execute batch item %d: %w", i, err)
		}
	}
	br.Close()

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WordEntry represents a single word entry for batch insertion
type WordEntry struct {
	Word     string
	Lang     string
	LangCode string
	Data     []byte
}

// TruncateTable removes all data from the wiktionary_words table
func (db *PostgresDB) TruncateTable(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, "TRUNCATE TABLE wiktionary_words RESTART IDENTITY;")
	if err != nil {
		return fmt.Errorf("failed to truncate table: %w", err)
	}

	log.Println("Table 'wiktionary_words' truncated")
	return nil
}

// GetWordCount returns the total number of entries in the table
func (db *PostgresDB) GetWordCount(ctx context.Context) (int64, error) {
	var count int64
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM wiktionary_words").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get word count: %w", err)
	}
	return count, nil
}

// SearchWords searches for words matching the given query
func (db *PostgresDB) SearchWords(ctx context.Context, word string, langCode string, limit int) ([]WordEntry, error) {
	query := `
	SELECT word, lang, lang_code, data
	FROM wiktionary_words
	WHERE word ILIKE $1 AND lang_code = $2
	LIMIT $3;
	`

	rows, err := db.Pool.Query(ctx, query, "%"+word+"%", langCode, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search words: %w", err)
	}
	defer rows.Close()

	var results []WordEntry
	for rows.Next() {
		var entry WordEntry
		if err := rows.Scan(&entry.Word, &entry.Lang, &entry.LangCode, &entry.Data); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, entry)
	}

	return results, nil
}
