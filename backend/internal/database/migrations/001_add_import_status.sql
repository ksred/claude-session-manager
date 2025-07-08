-- Migration: Add import_status column to file_watchers table
-- This migration adds the import_status column that was missing from earlier database versions

-- Check if the column already exists before adding it
-- SQLite doesn't support IF NOT EXISTS for ALTER TABLE, so we need to handle this carefully

-- Add the import_status column if it doesn't exist
-- Default value is 'pending' to match the schema
ALTER TABLE file_watchers ADD COLUMN import_status TEXT DEFAULT 'pending';

-- Update any existing rows that might have NULL values
UPDATE file_watchers SET import_status = 'pending' WHERE import_status IS NULL;