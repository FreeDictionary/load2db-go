# Quick Start Guide

This guide will help you quickly set up and import Wiktionary data into PostgreSQL.

## Prerequisites

1. **Go 1.25+** - Install from [go.dev](https://go.dev/dl/)
2. **PostgreSQL 12+** - Install from [postgresql.org](https://www.postgresql.org/download/)
3. **Wiktionary JSONL data** - Download from [Wiktionary Data Downloads](https://kaikki.org/dictionary/rawdata.html)

## Step 1: Build the Tool

```bash
cd /home/jimmy/projects/load2db-go
go build -o load2db-go .
```

## Step 2: Create PostgreSQL Database

```bash
# Connect to PostgreSQL
psql -U postgres

# Create the database
CREATE DATABASE wiktionary;

# Exit psql
\q
```

## Step 3: Import Data

### Option A: Import a Single File

```bash
./load2db-go import \
  --raw-data /path/to/wiktionary.jsonl \
  --db-host localhost \
  --db-port 5432 \
  --db-name wiktionary \
  --db-user postgres \
  --db-password your_password \
  --progress
```

### Option B: Import a Directory of Files

```bash
./load2db-go import \
  --raw-data /path/to/wiktionary-data/ \
  --db-host localhost \
  --db-port 5432 \
  --db-name wiktionary \
  --db-user postgres \
  --db-password your_password \
  --progress
```

### Option C: Test with Sample Data

Use the included sample data to test the tool:

```bash
./load2db-go import \
  --raw-data examples/sample.jsonl \
  --db-host localhost \
  --db-port 5432 \
  --db-name wiktionary \
  --db-user postgres \
  --db-password your_password \
  --progress
```

## Step 4: Verify Import

Connect to PostgreSQL and query the data:

```bash
psql -U postgres -d wiktionary
```

```sql
-- Count total entries
SELECT COUNT(*) FROM wiktionary_words;

-- Search for a specific word
SELECT word, lang, lang_code, data->>'pos' as pos
FROM wiktionary_words
WHERE word = 'hello' AND lang_code = 'en';

-- View the full JSON data
SELECT jsonb_pretty(data)
FROM wiktionary_words
WHERE word = 'hello' AND lang_code = 'en'
LIMIT 1;

-- Find all English nouns
SELECT word, data->>'pos' as pos
FROM wiktionary_words
WHERE lang_code = 'en' AND data->>'pos' = 'noun'
LIMIT 10;
```

## Performance Optimization

For large datasets (millions of entries), use these optimizations:

### 1. Skip Indexes During Initial Load

```bash
./load2db-go import \
  --raw-data /path/to/large-dataset/ \
  --db-host localhost \
  --db-port 5432 \
  --db-name wiktionary \
  --db-user postgres \
  --db-password your_password \
  --skip-indexes \
  --batch-size 5000
```

Then create indexes separately:

```bash
./load2db-go import \
  --raw-data /path/to/large-dataset/ \
  --db-host localhost \
  --db-port 5432 \
  --db-name wiktionary \
  --db-user postgres \
  --db-password your_password \
  --skip-indexes
```

### 2. Increase PostgreSQL Memory Settings

Edit `postgresql.conf`:

```conf
shared_buffers = 4GB
work_mem = 256MB
maintenance_work_mem = 2GB
effective_cache_size = 12GB
```

### 3. Use Parallel Imports (Advanced)

Split your data into multiple files and run multiple import processes:

```bash
# Split large file into chunks
split -l 100000 wiktionary.jsonl chunk_

# Import each chunk in parallel (in separate terminals)
./load2db-go import --raw-data chunk_aa --db-password xxx --batch-size 5000
./load2db-go import --raw-data chunk_ab --db-password xxx --batch-size 5000
./load2db-go import --raw-data chunk_ac --db-password xxx --batch-size 5000
```

## Troubleshooting

### Connection Refused

```
Error: failed to connect to database: unable to ping database
```

**Solution:** Ensure PostgreSQL is running and the connection parameters are correct.

```bash
# Check if PostgreSQL is running
sudo systemctl status postgresql

# Start PostgreSQL if needed
sudo systemctl start postgresql
```

### Permission Denied

```
Error: failed to open file: permission denied
```

**Solution:** Check file permissions.

```bash
chmod 644 /path/to/wiktionary.jsonl
```

### Out of Memory

```
Error: runtime: out of memory
```

**Solution:** Reduce batch size.

```bash
./load2db-go import --raw-data data.jsonl --batch-size 500 --db-password xxx
```

### Duplicate Key Error

```
Error: failed to insert batch: duplicate key value violates unique constraint
```

**Solution:** Use `--truncate` flag to clear existing data before import.

```bash
./load2db-go import --raw-data data.jsonl --truncate --db-password xxx
```

## Next Steps

- Read the full [README.md](README.md) for advanced usage
- Explore example queries in [README.md](README.md#example-queries)
- Check the database schema in [db/migrations/001_create_wiktionary_tables.sql](db/migrations/001_create_wiktionary_tables.sql)

## Support

For issues or questions, please open an issue on the project repository.
