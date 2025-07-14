# Database Migrations

This directory contains SQL migration files for the Claude Session Manager database.

## Migration Files

- `001_add_import_status.sql` - Adds import_status column to file_watchers table
- `002_add_import_counters.sql` - Adds session and message counter columns
- `003_add_last_error.sql` - Adds last_error column for tracking import failures
- `004_recalculate_token_costs.sql` - Recalculates token costs based on pricing
- `005_fix_total_tokens.sql` - Fixes total_tokens calculation in token_usage table
- `006_update_session_project_paths.sql` - Updates session project paths from message CWD values

## Running Migrations

### Method 1: Using the migration script (Recommended)

For specific migrations like updating project paths:

```bash
# From the backend directory
./scripts/migrate-project-paths.sh
```

This script will:
1. Show a dry run of what will be changed
2. Ask for confirmation
3. Apply the migration if confirmed

### Method 2: Using the run-migration tool

```bash
# Run a specific migration file
./run-migration -migration internal/database/migrations/006_update_session_project_paths.sql

# Or specify a custom database path
./run-migration -db /path/to/sessions.db -migration internal/database/migrations/006_update_session_project_paths.sql
```

### Method 3: Manual SQL execution

You can also run migrations manually using SQLite:

```bash
# Connect to the database
sqlite3 ~/.claude/sessions.db

# Run the migration
.read internal/database/migrations/006_update_session_project_paths.sql

# Exit
.quit
```

## Creating New Migrations

When creating a new migration:

1. Use the next sequential number (e.g., `007_`)
2. Use a descriptive name that explains what the migration does
3. Include comments in the SQL file explaining the purpose
4. Include a SELECT statement at the end to report results
5. Test the migration on a backup database first

Example migration structure:

```sql
-- Migration: Brief description
-- Detailed explanation of what this migration does and why

-- The actual SQL statements
UPDATE table_name SET ...

-- Report the results
SELECT 'Updated ' || changes() || ' records' as migration_result;
```

## Important Notes

- Always backup your database before running migrations
- Migrations are not automatically applied by the server
- Some migrations may take time on large databases
- Use dry-run mode when available to preview changes