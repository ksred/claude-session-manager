# Database Migrations

This directory contains SQL migration scripts for updating the database schema.

## Migration Files

### 001_add_import_status.sql
- Adds the `import_status` column to the `file_watchers` table
- This column was missing in early versions of the database schema
- The column tracks the import status of each file (pending, processing, completed, failed)

### 002_add_import_counters.sql
- Adds the `sessions_imported` and `messages_imported` columns to the `file_watchers` table
- These columns track how many sessions and messages were imported from each file
- Both columns default to 0 for existing rows

### 003_add_last_error.sql
- Adds the `last_error` column to the `file_watchers` table
- This column stores any error messages that occurred during the last import attempt
- The column can be NULL if no error occurred

## How Migrations Work

The application automatically handles schema updates in two ways:

1. **Initial Schema Creation**: When creating a new database, the `schema.sql` file is executed to create all tables with the latest schema.

2. **Schema Updates**: The `applySchemaUpdates()` method in `database.go` checks for missing columns and applies necessary updates to existing tables.

## Adding New Migrations

When you need to update the database schema:

1. Update the `schema.sql` file with your changes
2. Add logic to `applySchemaUpdates()` in `database.go` to handle the update for existing databases
3. Create a migration SQL file here for documentation purposes

## Manual Migration

If you need to manually apply migrations:

### For import_status column:
```sql
-- Add the import_status column if it doesn't exist
ALTER TABLE file_watchers ADD COLUMN import_status TEXT DEFAULT 'pending';

-- Update any existing rows that might have NULL values
UPDATE file_watchers SET import_status = 'pending' WHERE import_status IS NULL;
```

### For import counters:
```sql
-- Add the sessions_imported column if it doesn't exist
ALTER TABLE file_watchers ADD COLUMN sessions_imported INTEGER DEFAULT 0;

-- Add the messages_imported column if it doesn't exist
ALTER TABLE file_watchers ADD COLUMN messages_imported INTEGER DEFAULT 0;

-- Update any existing rows that might have NULL values
UPDATE file_watchers SET sessions_imported = 0 WHERE sessions_imported IS NULL;
UPDATE file_watchers SET messages_imported = 0 WHERE messages_imported IS NULL;
```

### For last_error column:
```sql
-- Add the last_error column if it doesn't exist
ALTER TABLE file_watchers ADD COLUMN last_error TEXT;
```