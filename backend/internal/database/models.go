package database

import (
	"encoding/json"
	"time"
)

// Session represents a Claude session in the database
type Session struct {
	ID             string    `db:"id" json:"id"`
	ProjectPath    string    `db:"project_path" json:"project_path"`
	ProjectName    string    `db:"project_name" json:"project_name"`
	FilePath       string    `db:"file_path" json:"file_path"`
	GitBranch      string    `db:"git_branch" json:"git_branch"`
	GitWorktree    string    `db:"git_worktree" json:"git_worktree"`
	StartTime      time.Time `db:"start_time" json:"start_time"`
	LastActivity   time.Time `db:"last_activity" json:"last_activity"`
	IsActive       bool      `db:"is_active" json:"is_active"`
	Status         string    `db:"status" json:"status"`
	Model          string    `db:"model" json:"model"`
	MessageCount   int       `db:"message_count" json:"message_count"`
	DurationSeconds int64    `db:"duration_seconds" json:"duration_seconds"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// Message represents a message within a session
type Message struct {
	ID           string     `db:"id" json:"id"`
	SessionID    string     `db:"session_id" json:"session_id"`
	ParentUUID   *string    `db:"parent_uuid" json:"parent_uuid"`
	IsSidechain  bool       `db:"is_sidechain" json:"is_sidechain"`
	UserType     string     `db:"user_type" json:"user_type"`
	CWD          string     `db:"cwd" json:"cwd"`
	Version      string     `db:"version" json:"version"`
	Type         string     `db:"type" json:"type"`
	Role         string     `db:"role" json:"role"`
	Content      string     `db:"content" json:"content"` // JSON string
	RequestID    *string    `db:"request_id" json:"request_id"`
	Timestamp    time.Time  `db:"timestamp" json:"timestamp"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
}

// TokenUsage represents token consumption for a message
type TokenUsage struct {
	ID                       int       `db:"id" json:"id"`
	MessageID                string    `db:"message_id" json:"message_id"`
	SessionID                string    `db:"session_id" json:"session_id"`
	InputTokens              int       `db:"input_tokens" json:"input_tokens"`
	OutputTokens             int       `db:"output_tokens" json:"output_tokens"`
	CacheCreationInputTokens int       `db:"cache_creation_input_tokens" json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int       `db:"cache_read_input_tokens" json:"cache_read_input_tokens"`
	TotalTokens              int       `db:"total_tokens" json:"total_tokens"`
	ServiceTier              string    `db:"service_tier" json:"service_tier"`
	EstimatedCost            float64   `db:"estimated_cost" json:"estimated_cost"`
	CreatedAt                time.Time `db:"created_at" json:"created_at"`
}

// ToolResult represents tool usage results
type ToolResult struct {
	ID         int       `db:"id" json:"id"`
	MessageID  string    `db:"message_id" json:"message_id"`
	SessionID  string    `db:"session_id" json:"session_id"`
	ToolName   string    `db:"tool_name" json:"tool_name"`
	FilePath   *string   `db:"file_path" json:"file_path"`
	ResultData string    `db:"result_data" json:"result_data"` // JSON string
	Timestamp  time.Time `db:"timestamp" json:"timestamp"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

// FileWatcher represents a monitored file with processing status
type FileWatcher struct {
	ID                     int        `db:"id" json:"id"`
	FilePath               string     `db:"file_path" json:"file_path"`
	LastModified           time.Time  `db:"last_modified" json:"last_modified"`
	LastProcessed          *time.Time `db:"last_processed" json:"last_processed"`
	LastProcessedPosition  int64      `db:"last_processed_position" json:"last_processed_position"`
	FileSize               int64      `db:"file_size" json:"file_size"`
	FileHash               *string    `db:"file_hash" json:"file_hash"`
	ImportStatus           string     `db:"import_status" json:"import_status"`
	SessionsImported       int        `db:"sessions_imported" json:"sessions_imported"`
	MessagesImported       int        `db:"messages_imported" json:"messages_imported"`
	LastError              *string    `db:"last_error" json:"last_error"`
	CreatedAt              time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time  `db:"updated_at" json:"updated_at"`
}

// ImportRun represents a single import operation
type ImportRun struct {
	ID               int        `db:"id" json:"id"`
	RunType          string     `db:"run_type" json:"run_type"`
	StartTime        time.Time  `db:"start_time" json:"start_time"`
	EndTime          *time.Time `db:"end_time" json:"end_time"`
	Status           string     `db:"status" json:"status"`
	FilesProcessed   int        `db:"files_processed" json:"files_processed"`
	FilesSkipped     int        `db:"files_skipped" json:"files_skipped"`
	SessionsImported int        `db:"sessions_imported" json:"sessions_imported"`
	MessagesImported int        `db:"messages_imported" json:"messages_imported"`
	ErrorMessage     *string    `db:"error_message" json:"error_message"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
}

// ActivityLogEntry represents an activity log entry
type ActivityLogEntry struct {
	ID           *int      `db:"id" json:"id"`
	SessionID    *string   `db:"session_id" json:"session_id"`
	ActivityType string    `db:"activity_type" json:"activity_type"`
	Details      string    `db:"details" json:"details"`
	Timestamp    time.Time `db:"timestamp" json:"timestamp"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// SessionSummary represents the session summary view
type SessionSummary struct {
	ID                         string    `db:"id" json:"id"`
	ProjectName                string    `db:"project_name" json:"project_name"`
	ProjectPath                string    `db:"project_path" json:"project_path"`
	StartTime                  time.Time `db:"start_time" json:"start_time"`
	LastActivity               time.Time `db:"last_activity" json:"last_activity"`
	IsActive                   bool      `db:"is_active" json:"is_active"`
	Status                     string    `db:"status" json:"status"`
	Model                      string    `db:"model" json:"model"`
	MessageCount               int       `db:"message_count" json:"message_count"`
	DurationSeconds            int64     `db:"duration_seconds" json:"duration_seconds"`
	TotalInputTokens           int       `db:"total_input_tokens" json:"total_input_tokens"`
	TotalOutputTokens          int       `db:"total_output_tokens" json:"total_output_tokens"`
	TotalCacheCreationTokens   int       `db:"total_cache_creation_tokens" json:"total_cache_creation_tokens"`
	TotalCacheReadTokens       int       `db:"total_cache_read_tokens" json:"total_cache_read_tokens"`
	TotalTokens                int       `db:"total_tokens" json:"total_tokens"`
	TotalEstimatedCost         float64   `db:"total_estimated_cost" json:"total_estimated_cost"`
	FilesModified              string    `db:"files_modified" json:"files_modified"` // JSON array as string
}

// GetFilesModifiedList returns the files modified as a slice
func (ss *SessionSummary) GetFilesModifiedList() ([]string, error) {
	var files []string
	if ss.FilesModified == "" || ss.FilesModified == "[]" {
		return files, nil
	}
	err := json.Unmarshal([]byte(ss.FilesModified), &files)
	return files, err
}

// DailyMetric represents daily usage metrics
type DailyMetric struct {
	Date         string `db:"date" json:"date"`
	SessionCount int    `db:"session_count" json:"session_count"`
	MessageCount int    `db:"message_count" json:"message_count"`
	Model        string `db:"model" json:"model"`
	TotalTokens  int    `db:"total_tokens" json:"total_tokens"`
}

// DatabaseStats represents overall database statistics
type DatabaseStats struct {
	TotalSessions        int     `json:"total_sessions"`
	ActiveSessions       int     `json:"active_sessions"`
	TotalMessages        int     `json:"total_messages"`
	TotalTokens          int     `json:"total_tokens"`
	TotalEstimatedCost   float64 `json:"total_estimated_cost"`
	AverageSessionDuration float64 `json:"average_session_duration"`
	MostUsedModel        string  `json:"most_used_model"`
}