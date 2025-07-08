package api

import (
	"time"

	"github.com/ksred/claude-session-manager/internal/claude"
)

// SessionResponse represents the API response for a session
// @Description Session information with metadata and statistics
type SessionResponse struct {
	ID            string                `json:"id" example:"session_123456" description:"Unique session identifier"`
	Title         string                `json:"title" example:"Implement user authentication" description:"Session title or current task"`
	ProjectPath   string                `json:"project_path" example:"/home/user/my-project" description:"Absolute path to the project directory"`
	ProjectName   string                `json:"project_name" example:"my-awesome-project" description:"Name of the project"`
	GitBranch     string                `json:"git_branch,omitempty" example:"feature/auth" description:"Current Git branch"`
	GitWorktree   string                `json:"git_worktree,omitempty" example:"main" description:"Git worktree information"`
	Status        string                `json:"status" example:"active" description:"Session status" enums:"active,idle,completed,error"`
	CreatedAt     time.Time             `json:"created_at" example:"2023-01-01T10:00:00Z" description:"Session creation timestamp"`
	UpdatedAt     time.Time             `json:"updated_at" example:"2023-01-01T11:30:00Z" description:"Last activity timestamp"`
	MessageCount  int                   `json:"message_count" example:"25" description:"Total number of messages in session"`
	CurrentTask   string                `json:"current_task" example:"Implement user authentication" description:"Current task description"`
	TokensUsed    claude.TokenUsage     `json:"tokens_used" description:"Token usage statistics"`
	FilesModified []string              `json:"files_modified" example:"[\"/src/auth.go\",\"/src/user.go\"]" description:"List of modified files"`
	Duration      int64                 `json:"duration_seconds" example:"5400" description:"Session duration in seconds"`
	IsActive      bool                  `json:"is_active" example:"true" description:"Whether session is currently active"`
	Model         string                `json:"model" example:"claude-3-opus" description:"Claude model used"`
}

// SessionsResponse represents the response for sessions list
// @Description List of sessions with pagination info
type SessionsResponse struct {
	Sessions []SessionResponse `json:"sessions" description:"List of sessions"`
	Total    int               `json:"total" example:"42" description:"Total number of sessions"`
}

// SessionsLimitResponse represents the response for sessions with limit
// @Description List of sessions with limit info
type SessionsLimitResponse struct {
	Sessions []SessionResponse `json:"sessions" description:"List of sessions"`
	Limit    int               `json:"limit" example:"10" description:"Applied limit"`
}

// MetricsSummary represents overall metrics
// @Description Overall system metrics and statistics
type MetricsSummary struct {
	TotalSessions          int            `json:"total_sessions" example:"150" description:"Total number of sessions"`
	ActiveSessions         int            `json:"active_sessions" example:"5" description:"Currently active sessions"`
	TotalMessages          int            `json:"total_messages" example:"2500" description:"Total messages across all sessions"`
	TotalTokensUsed        int            `json:"total_tokens_used" example:"125000" description:"Total tokens consumed"`
	TotalEstimatedCost     float64        `json:"total_estimated_cost" example:"15.75" description:"Estimated total cost in USD"`
	AverageSessionDuration float64        `json:"average_session_duration_minutes" example:"45.2" description:"Average session duration in minutes"`
	MostUsedModel          string         `json:"most_used_model" example:"claude-3-opus" description:"Most frequently used model"`
	ModelUsage             map[string]int `json:"model_usage" description:"Usage count by model"`
}

// ActivityEntry represents a single activity in the timeline
// @Description A single activity entry in the system timeline
type ActivityEntry struct {
	Timestamp   time.Time `json:"timestamp" example:"2023-01-01T10:00:00Z" description:"Activity timestamp"`
	Type        string    `json:"type" example:"message_sent" description:"Activity type" enums:"message_sent,session_created,session_updated,error"`
	SessionID   string    `json:"session_id" example:"session_123456" description:"Related session ID"`
	SessionName string    `json:"session_name" example:"my-project" description:"Session/project name"`
	Details     string    `json:"details" example:"User sent a message" description:"Activity details"`
}

// ActivityResponse represents the response for activity timeline
// @Description Activity timeline with entries
type ActivityResponse struct {
	Activity []ActivityEntry `json:"activity" description:"List of activity entries"`
	Total    int             `json:"total" example:"100" description:"Total number of activities"`
}

// UsageStats represents usage statistics
// @Description Usage statistics and analytics
type UsageStats struct {
	DailySessions []DailySessionCount `json:"daily_sessions" description:"Daily session counts"`
	ModelUsage    map[string]int      `json:"model_usage" description:"Usage count by model"`
	PeakHours     []PeakHourInfo      `json:"peak_hours" description:"Peak usage hours"`
}

// DailySessionCount represents daily session count
// @Description Daily session count information
type DailySessionCount struct {
	Date  string `json:"date" example:"2023-01-01" description:"Date in YYYY-MM-DD format"`
	Count int    `json:"count" example:"15" description:"Number of sessions on this date"`
}

// PeakHourInfo represents peak hour information
// @Description Peak hour usage information
type PeakHourInfo struct {
	Hour            int     `json:"hour" example:"14" description:"Hour of day (0-23)"`
	AverageSessions float64 `json:"average_sessions" example:"5.2" description:"Average sessions during this hour"`
}

// SearchResponse represents search results
// @Description Search results with query information
type SearchResponse struct {
	Query   string            `json:"query" example:"authentication" description:"Search query"`
	Results []SessionResponse `json:"results" description:"Matching sessions"`
	Total   int               `json:"total" example:"5" description:"Total number of results"`
}

// ErrorResponse represents an error response
// @Description Error response structure
type ErrorResponse struct {
	Error   string `json:"error" example:"Session not found" description:"Error message"`
	Code    int    `json:"code,omitempty" example:"404" description:"Error code"`
	Details string `json:"details,omitempty" example:"Session with ID 'invalid' does not exist" description:"Additional error details"`
}

// HealthResponse represents health check response
// @Description Health check response
type HealthResponse struct {
	Status    string    `json:"status" example:"ok" description:"Health status"`
	Timestamp time.Time `json:"timestamp" example:"2023-01-01T10:00:00Z" description:"Health check timestamp"`
	Version   string    `json:"version,omitempty" example:"1.0.0" description:"Application version"`
	Uptime    string    `json:"uptime,omitempty" example:"2h30m15s" description:"Application uptime"`
}

// WebSocketMessage represents a WebSocket message
// @Description WebSocket message structure for real-time updates
type WebSocketMessage struct {
	Type      string      `json:"type" example:"session_update" description:"Message type"`
	SessionID string      `json:"session_id,omitempty" example:"session_123456" description:"Related session ID"`
	Data      interface{} `json:"data,omitempty" description:"Message data"`
	Timestamp time.Time   `json:"timestamp" example:"2023-01-01T10:00:00Z" description:"Message timestamp"`
}