-- Migration: Add sessions_imported and messages_imported columns to file_watchers table
-- These columns track how many sessions and messages were imported from each file

-- Add the sessions_imported column if it doesn't exist
-- Default value is 0 to indicate no sessions imported yet
ALTER TABLE file_watchers ADD COLUMN sessions_imported INTEGER DEFAULT 0;

-- Add the messages_imported column if it doesn't exist
-- Default value is 0 to indicate no messages imported yet
ALTER TABLE file_watchers ADD COLUMN messages_imported INTEGER DEFAULT 0;

-- Update any existing rows that might have NULL values
UPDATE file_watchers SET sessions_imported = 0 WHERE sessions_imported IS NULL;
UPDATE file_watchers SET messages_imported = 0 WHERE messages_imported IS NULL;