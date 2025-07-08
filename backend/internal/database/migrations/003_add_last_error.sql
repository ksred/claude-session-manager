-- Migration: Add last_error column to file_watchers table
-- This column stores any error messages that occurred during the last import attempt

ALTER TABLE file_watchers ADD COLUMN last_error TEXT;