-- Add source column to sessions table to distinguish UI-initiated sessions
ALTER TABLE sessions ADD COLUMN source VARCHAR(20) DEFAULT 'import' CHECK (source IN ('import', 'ui'));

-- Add claude_session_id to chat_sessions for conversation continuity
ALTER TABLE chat_sessions ADD COLUMN claude_session_id VARCHAR(255);

-- Create index for faster lookups
CREATE INDEX idx_sessions_source ON sessions(source);
CREATE INDEX idx_chat_sessions_claude_id ON chat_sessions(claude_session_id);

-- Update existing sessions to have 'import' source
UPDATE sessions SET source = 'import' WHERE source IS NULL;