# Database migrations

## Overview

This directory holds the database migration scripts for E-SP-Line2.

## Usage

### Automatic migration

The schema is created automatically on startup by GORM `AutoMigrate`.

### Manual migration

```bash
# Run all migrations
cd backend
go run main.go migrate

# Run one specific version
go run main.go migrate --version 001

# Roll back to the previous version
go run main.go migrate rollback
```

> The `migrate` subcommand is documented here but not yet implemented in `main.go`; until it is, apply the SQL files by hand.

## Migration file naming

```
XXX_description.sql
```

- `XXX`: three-digit version number (001, 002, …)
- `description`: what the migration does, underscore-separated

## Current migrations

| Version | File | Description |
|------|------|------|
| 001 | 001_initial_schema.sql | Initial database schema |

## Writing a migration

### Basic structure

```sql
-- E-SP-Line2 Database Migration
-- Version: XXX
-- Description: Your description

-- Your SQL here
```

### Rules

1. Use `IF NOT EXISTS` so re-runs are safe
2. Use `ON CONFLICT` for duplicates
3. Add the indexes the queries need
4. Provide a rollback script where practical

## Database support

- SQLite (development)
- PostgreSQL (production)

## Version history

- v001 (2024-01-01): initial schema
