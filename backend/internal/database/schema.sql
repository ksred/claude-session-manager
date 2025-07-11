-- SQLite Schema for Claude Session Manager
-- This schema supports the JSONL file structure and API requirements

-- Sessions table - core session information
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    project_path TEXT NOT NULL,
    project_name TEXT NOT NULL,
    file_path TEXT NOT NULL, -- Original JSONL file path
    git_branch TEXT,
    git_worktree TEXT,
    start_time DATETIME NOT NULL,
    last_activity DATETIME NOT NULL,
    is_active BOOLEAN DEFAULT FALSE,
    status TEXT DEFAULT 'completed', -- active, idle, completed, error
    model TEXT,
    message_count INTEGER DEFAULT 0,
    duration_seconds INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Messages table - individual messages within sessions
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY, -- uuid from JSONL
    session_id TEXT NOT NULL,
    parent_uuid TEXT,
    is_sidechain BOOLEAN DEFAULT FALSE,
    user_type TEXT, -- external, internal
    cwd TEXT,
    version TEXT,
    type TEXT, -- user, assistant
    role TEXT, -- user, assistant
    content TEXT, -- JSON string of message content
    request_id TEXT,
    timestamp DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- Token usage table - tracks token consumption per message
CREATE TABLE IF NOT EXISTS token_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL,
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

-- Tool use results table - tracks file modifications and tool interactions
CREATE TABLE IF NOT EXISTS tool_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    tool_name TEXT,
    file_path TEXT,
    result_data TEXT, -- JSON string of full tool result
    timestamp DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- File watchers table - tracks which files we're monitoring and their processing status
CREATE TABLE IF NOT EXISTS file_watchers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path TEXT UNIQUE NOT NULL,
    last_modified DATETIME NOT NULL,
    last_processed DATETIME,
    file_size INTEGER DEFAULT 0,
    file_hash TEXT, -- Simple hash to detect content changes
    import_status TEXT DEFAULT 'pending', -- pending, completed, failed, skipped
    sessions_imported INTEGER DEFAULT 0,
    messages_imported INTEGER DEFAULT 0,
    last_error TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Import runs table - tracks when we've run imports
CREATE TABLE IF NOT EXISTS import_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_type TEXT NOT NULL, -- 'initial', 'incremental', 'manual'
    start_time DATETIME NOT NULL,
    end_time DATETIME,
    status TEXT DEFAULT 'running', -- running, completed, failed, cancelled
    files_processed INTEGER DEFAULT 0,
    files_skipped INTEGER DEFAULT 0,
    sessions_imported INTEGER DEFAULT 0,
    messages_imported INTEGER DEFAULT 0,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Activity log table - for timeline and audit purposes
CREATE TABLE IF NOT EXISTS activity_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT,
    activity_type TEXT NOT NULL, -- session_created, message_sent, session_updated, file_modified
    details TEXT,
    timestamp DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE SET NULL
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_sessions_project_name ON sessions(project_name);
CREATE INDEX IF NOT EXISTS idx_sessions_last_activity ON sessions(last_activity DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_is_active ON sessions(is_active);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_model ON sessions(model);

CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages(session_id);
CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_messages_type ON messages(type);
CREATE INDEX IF NOT EXISTS idx_messages_role ON messages(role);

CREATE INDEX IF NOT EXISTS idx_token_usage_session_id ON token_usage(session_id);
CREATE INDEX IF NOT EXISTS idx_token_usage_message_id ON token_usage(message_id);

CREATE INDEX IF NOT EXISTS idx_tool_results_session_id ON tool_results(session_id);
CREATE INDEX IF NOT EXISTS idx_tool_results_file_path ON tool_results(file_path);

CREATE INDEX IF NOT EXISTS idx_file_watchers_last_modified ON file_watchers(last_modified);

CREATE INDEX IF NOT EXISTS idx_activity_log_session_id ON activity_log(session_id);
CREATE INDEX IF NOT EXISTS idx_activity_log_timestamp ON activity_log(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_activity_log_type ON activity_log(activity_type);

-- Views for common queries
CREATE VIEW IF NOT EXISTS session_summary AS
SELECT 
    s.id,
    s.project_name,
    s.project_path,
    s.start_time,
    s.last_activity,
    s.is_active,
    s.status,
    s.model,
    s.message_count,
    s.duration_seconds,
    COALESCE(tu.total_input_tokens, 0) as total_input_tokens,
    COALESCE(tu.total_output_tokens, 0) as total_output_tokens,
    COALESCE(tu.total_cache_creation_tokens, 0) as total_cache_creation_tokens,
    COALESCE(tu.total_cache_read_tokens, 0) as total_cache_read_tokens,
    COALESCE(tu.total_tokens, 0) as total_tokens,
    COALESCE(tu.total_cost, 0.0) as total_estimated_cost,
    COALESCE(fr.modified_files, '[]') as files_modified
FROM sessions s
LEFT JOIN (
    SELECT 
        session_id,
        SUM(input_tokens) as total_input_tokens,
        SUM(output_tokens) as total_output_tokens,
        SUM(cache_creation_input_tokens) as total_cache_creation_tokens,
        SUM(cache_read_input_tokens) as total_cache_read_tokens,
        SUM(total_tokens) as total_tokens,
        SUM(estimated_cost) as total_cost
    FROM token_usage 
    GROUP BY session_id
) tu ON s.id = tu.session_id
LEFT JOIN (
    SELECT 
        session_id,
        JSON_GROUP_ARRAY(DISTINCT file_path) as modified_files
    FROM tool_results 
    WHERE file_path IS NOT NULL
    GROUP BY session_id
) fr ON s.id = fr.session_id;

-- Chat sessions table - tracks active chat sessions with Claude CLI
CREATE TABLE IF NOT EXISTS chat_sessions (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    process_id INTEGER,
    status TEXT NOT NULL DEFAULT 'active', -- active, inactive, terminated, error
    started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_activity DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

-- Chat messages table - stores messages in chat sessions
CREATE TABLE IF NOT EXISTS chat_messages (
    id TEXT PRIMARY KEY,
    chat_session_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('user', 'claude', 'system')),
    content TEXT NOT NULL,
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT, -- JSON metadata
    FOREIGN KEY (chat_session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE
);

-- Indexes for chat tables
CREATE INDEX IF NOT EXISTS idx_chat_sessions_session_id ON chat_sessions(session_id);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_status ON chat_sessions(status);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_last_activity ON chat_sessions(last_activity DESC);

CREATE INDEX IF NOT EXISTS idx_chat_messages_chat_session_id ON chat_messages(chat_session_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_timestamp ON chat_messages(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_chat_messages_type ON chat_messages(type);

-- Daily metrics view
CREATE VIEW IF NOT EXISTS daily_metrics AS
SELECT 
    DATE(s.start_time) as date,
    COUNT(DISTINCT s.id) as session_count,
    COUNT(m.id) as message_count,
    s.model,
    SUM(COALESCE(tu.total_tokens, 0)) as total_tokens
FROM sessions s
LEFT JOIN messages m ON s.id = m.session_id
LEFT JOIN token_usage tu ON m.id = tu.message_id
GROUP BY DATE(s.start_time), s.model
ORDER BY date DESC;