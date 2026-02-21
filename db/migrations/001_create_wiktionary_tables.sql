-- Migration: 001_create_wiktionary_tables.sql
-- Creates the wiktionary_words table and indexes

-- Create the main table for storing wiktionary entries
CREATE TABLE IF NOT EXISTS wiktionary_words (
    id BIGSERIAL PRIMARY KEY,
    word TEXT NOT NULL,
    lang TEXT NOT NULL,
    lang_code TEXT NOT NULL,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Add comments to table and columns
COMMENT ON TABLE wiktionary_words IS 'Stores Wiktionary word entries from JSONL data';
COMMENT ON COLUMN wiktionary_words.word IS 'The word itself';
COMMENT ON COLUMN wiktionary_words.lang IS 'Full language name (e.g., English, Spanish)';
COMMENT ON COLUMN wiktionary_words.lang_code IS 'ISO language code (e.g., en, es)';
COMMENT ON COLUMN wiktionary_words.data IS 'Full Wiktionary entry in JSONB format';

-- Create indexes for efficient searching
-- Note: CONCURRENTLY cannot be used inside transactions, so run these separately if needed

-- Index on word column for exact and pattern matching
CREATE INDEX IF NOT EXISTS idx_wiktionary_words_word ON wiktionary_words(word);

-- Index on lang column for language filtering
CREATE INDEX IF NOT EXISTS idx_wiktionary_words_lang ON wiktionary_words(lang);

-- Index on lang_code column for language code filtering
CREATE INDEX IF NOT EXISTS idx_wiktionary_words_lang_code ON wiktionary_words(lang_code);

-- Composite index for word + lang_code queries (most common search pattern)
CREATE INDEX IF NOT EXISTS idx_wiktionary_words_word_lang_code ON wiktionary_words(word, lang_code);

-- GIN index on JSONB data for querying nested fields
CREATE INDEX IF NOT EXISTS idx_wiktionary_words_data_gin ON wiktionary_words USING GIN(data);

-- Create a unique constraint to prevent duplicate entries
-- This ensures each word+lang+lang_code combination is unique
CREATE UNIQUE INDEX IF NOT EXISTS idx_wiktionary_words_unique ON wiktionary_words(word, lang, lang_code);

-- Optional: Create a trigger to automatically update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_wiktionary_words_updated_at
    BEFORE UPDATE ON wiktionary_words
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
