# load2db-go

Load Wiktionary JSONL data into PostgreSQL.

## Overview

This tool reads Wiktionary raw data in JSONL format and imports it into a PostgreSQL database. Each JSON line is deserialized and re-serialized to normalize the JSON format before storage. The tool also extracts `word`, `lang`, and `lang_code` columns with indexes for efficient searching.

## Features

- **JSONL Import**: Reads JSONL files (one JSON object per line) efficiently
- **JSON Normalization**: Deserializes and re-serializes JSON to ensure consistent format
- **PostgreSQL Storage**: Stores data in JSONB format for efficient querying
- **Search Indexes**: Creates indexes on `word`, `lang`, `lang_code`, and JSONB data
- **Batch Insertion**: Uses batch inserts for high performance
- **Progress Reporting**: Shows import progress for large files
- **Directory Support**: Can import all JSONL files from a directory

## Installation

```bash
go build -o load2db-go .
```

## Usage

### Basic Import

```bash
./load2db-go import \
  --raw-data /path/to/wiktionary.jsonl \
  --db-host localhost \
  --db-port 5432 \
  --db-name wiktionary \
  --db-user postgres \
  --db-password your_password
```

### Import from Directory

```bash
./load2db-go import \
  --raw-data /path/to/wiktionary-data/ \
  --db-host localhost \
  --db-port 5432 \
  --db-name wiktionary \
  --db-user postgres \
  --db-password your_password
```

### Advanced Options

```bash
./load2db-go import \
  --raw-data /path/to/wiktionary.jsonl \
  --db-host localhost \
  --db-port 5432 \
  --db-name wiktionary \
  --db-user postgres \
  --db-password your_password \
  --batch-size 2000 \
  --progress \
  --truncate \
  --skip-indexes
```

## Command Line Flags

### Database Connection

| Flag            | Default      | Description              |
| --------------- | ------------ | ------------------------ |
| `--db-host`     | `localhost`  | PostgreSQL host          |
| `--db-port`     | `5432`       | PostgreSQL port          |
| `--db-name`     | `wiktionary` | PostgreSQL database name |
| `--db-user`     | `postgres`   | PostgreSQL user          |
| `--db-password` | (required)   | PostgreSQL password      |

### Import Options

| Flag             | Default    | Description                                     |
| ---------------- | ---------- | ----------------------------------------------- |
| `--raw-data`     | (required) | Path to JSONL file or directory                 |
| `--batch-size`   | `1000`     | Batch size for bulk inserts                     |
| `--progress`     | `true`     | Show import progress                            |
| `--truncate`     | `false`    | Truncate table before import                    |
| `--skip-indexes` | `false`    | Skip creating indexes (useful for initial load) |

## Database Schema

**Note:** The table name is fixed to `wiktionary_words` and cannot be configured via command line flags.

The tool creates a table `wiktionary_words` with the following structure:

```sql
CREATE TABLE wiktionary_words (
    id BIGSERIAL PRIMARY KEY,
    word TEXT NOT NULL,
    lang TEXT NOT NULL,
    lang_code TEXT NOT NULL,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Indexes

The following indexes are created for efficient searching:

- `idx_wiktionary_words_word` - Index on `word` column
- `idx_wiktionary_words_lang` - Index on `lang` column
- `idx_wiktionary_words_lang_code` - Index on `lang_code` column
- `idx_wiktionary_words_word_lang_code` - Composite index on `(word, lang_code)`
- `idx_wiktionary_words_data_gin` - GIN index on `data` JSONB column

## Example Queries

### Search by word

```sql
SELECT word, lang, lang_code, data
FROM wiktionary_words
WHERE word = 'hello' AND lang_code = 'en';
```

### Search by pattern

```sql
SELECT word, lang, lang_code, data
FROM wiktionary_words
WHERE word ILIKE '%hello%' AND lang_code = 'en';
```

### Query JSONB data

```sql
-- Find words with a specific part of speech
SELECT word, lang, data->>'pos' as pos
FROM wiktionary_words
WHERE data->>'pos' = 'noun' AND lang_code = 'en';

-- Find words with translations
SELECT word, lang, jsonb_array_length(data->'translations') as translation_count
FROM wiktionary_words
WHERE data ? 'translations' AND lang_code = 'en';
```

## Performance Tips

1. **Use `--skip-indexes` for initial load**: Create indexes after importing all data for faster initial load
2. **Adjust `--batch-size`**: Larger batch sizes can improve performance but use more memory
3. **Use `--truncate`**: When re-importing data, truncate the table first to avoid duplicates
4. **Consider parallel imports**: For very large datasets, consider splitting the data and running multiple imports

## JSON Data Structure

The `data` column stores the full Wiktionary entry in JSONB format. The structure follows the [wiktionary-schema-go](https://github.com/FreeDictionary/wiktionary-schema-go) schema with fields like:

- `word`: The word itself
- `lang`: Language name
- `lang_code`: ISO language code
- `pos`: Part of speech
- `senses`: Array of sense definitions
- `translations`: Array of translations
- `sounds`: Array of pronunciation data
- And many more...

## License

See [LICENSE](LICENSE) file.
