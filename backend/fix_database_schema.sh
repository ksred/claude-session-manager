#!/bin/bash

# Script to fix the missing import_status column in existing databases
# This script can be run manually if the automatic migration fails

DATABASE_PATH="${1:-$HOME/.claude-session-manager/sessions.db}"

echo "Fixing database schema at: $DATABASE_PATH"

if [ ! -f "$DATABASE_PATH" ]; then
    echo "Database file not found at: $DATABASE_PATH"
    echo "Usage: $0 [database_path]"
    exit 1
fi

# Check if the column already exists
COLUMN_EXISTS=$(sqlite3 "$DATABASE_PATH" "SELECT COUNT(*) FROM pragma_table_info('file_watchers') WHERE name = 'import_status';")

if [ "$COLUMN_EXISTS" = "0" ]; then
    echo "Adding import_status column to file_watchers table..."
    
    # Add the column
    sqlite3 "$DATABASE_PATH" "ALTER TABLE file_watchers ADD COLUMN import_status TEXT DEFAULT 'pending';"
    
    # Update existing rows
    sqlite3 "$DATABASE_PATH" "UPDATE file_watchers SET import_status = 'pending' WHERE import_status IS NULL;"
    
    echo "Schema fix applied successfully!"
else
    echo "import_status column already exists. No action needed."
fi

# Show the current schema
echo ""
echo "Current file_watchers table schema:"
sqlite3 "$DATABASE_PATH" ".schema file_watchers"