-- Migration 009: Fix token_usage deduplication
--
-- Problem: token_usage table had no UNIQUE constraint on message_id.
-- The primary key is AUTOINCREMENT, so INSERT OR REPLACE never detected
-- conflicts — every re-import created duplicate rows, inflating token
-- counts and cost estimates.
--
-- This migration:
-- 1. Removes duplicate rows (keeps the highest id per message_id)
-- 2. Recreates the table with a UNIQUE constraint on message_id
--
-- Note: This migration is also applied automatically on startup via
-- applySchemaUpdates() in database.go. This file is provided for
-- users who prefer to run migrations manually.

-- Step 1: Remove duplicate rows
DELETE FROM token_usage
WHERE id NOT IN (
    SELECT MAX(id)
    FROM token_usage
    GROUP BY message_id
);

-- Step 2: Recreate table with UNIQUE constraint
ALTER TABLE token_usage RENAME TO token_usage_old;

CREATE TABLE token_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL UNIQUE,
    session_id TEXT NOT NULL,
    input_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    cache_creation_input_tokens INTEGER DEFAULT 0,
    cache_read_input_tokens INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    service_tier TEXT,
    estimated_cost REAL DEFAULT 0.0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

INSERT INTO token_usage SELECT * FROM token_usage_old;

DROP TABLE token_usage_old;

CREATE INDEX IF NOT EXISTS idx_token_usage_session_id ON token_usage(session_id);
CREATE INDEX IF NOT EXISTS idx_token_usage_message_id ON token_usage(message_id);
