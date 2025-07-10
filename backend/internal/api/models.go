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
	Type        string    `json:"type" example:"message_sent" description:"Activity type" enums:"message_sent,session_created,session_updated,file_modified,error"`
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

// ModelStats represents statistics for a specific model
// @Description Performance statistics for a Claude model
type ModelStats struct {
	TotalSessions          int     `json:"total_sessions" example:"45" description:"Total number of sessions using this model"`
	TotalTokens            int     `json:"total_tokens" example:"1250000" description:"Total tokens consumed"`
	TotalCost              float64 `json:"total_cost" example:"25.50" description:"Total cost in USD"`
	AvgTokensPerSession    int     `json:"avg_tokens_per_session" example:"27777" description:"Average tokens per session"`
	AvgCostPerSession      float64 `json:"avg_cost_per_session" example:"0.57" description:"Average cost per session"`
	CacheEfficiency        float64 `json:"cache_efficiency" example:"0.45" description:"Ratio of cache tokens to total tokens"`
	AvgSessionDurationSecs int64   `json:"avg_session_duration_seconds" example:"3600" description:"Average session duration in seconds"`
}

// ModelPerformanceEntry represents performance data for a single model
// @Description Performance data for a Claude model with statistics
type ModelPerformanceEntry struct {
	Model       string     `json:"model" example:"claude-3-opus-20240229" description:"Model identifier"`
	DisplayName string     `json:"display_name" example:"Claude 3 Opus" description:"Human-readable model name"`
	Stats       ModelStats `json:"stats" description:"Performance statistics"`
}

// ModelPerformanceResponse represents the response for model performance comparison
// @Description Model performance comparison data
type ModelPerformanceResponse struct {
	Models []ModelPerformanceEntry `json:"models" description:"List of models with their performance data"`
}

// TimeSeriesModelData represents model usage data within a time period
// @Description Model usage data for a specific time period
type TimeSeriesModelData struct {
	Sessions int `json:"sessions" example:"3" description:"Number of sessions using this model"`
	Tokens   int `json:"tokens" example:"75000" description:"Total tokens consumed"`
}

// TimeSeriesEntry represents data for a specific time period
// @Description Analytics data for a specific time period
type TimeSeriesEntry struct {
	Date        string                         `json:"date" example:"2024-01-08" description:"Date in YYYY-MM-DD format"`
	Sessions    int                            `json:"sessions" example:"12" description:"Number of sessions"`
	Messages    int                            `json:"messages" example:"145" description:"Number of messages"`
	TotalTokens int                            `json:"total_tokens" example:"125000" description:"Total tokens consumed"`
	TotalCost   float64                        `json:"total_cost" example:"2.50" description:"Total cost in USD"`
	Models      map[string]TimeSeriesModelData `json:"models" description:"Model usage breakdown"`
}

// TimeSeriesResponse represents the response for time series analytics
// @Description Time series analytics data
type TimeSeriesResponse struct {
	Period string            `json:"period" example:"day" description:"Time period granularity" enums:"day,week,month"`
	Data   []TimeSeriesEntry `json:"data" description:"Time series data entries"`
}

// TokenBreakdown represents detailed token usage
// @Description Detailed token usage breakdown
type TokenBreakdown struct {
	Total  int `json:"total" example:"450000" description:"Total tokens"`
	Cached int `json:"cached" example:"180000" description:"Cached tokens"`
	Fresh  int `json:"fresh" example:"270000" description:"Fresh (non-cached) tokens"`
}

// CostBreakdownEntry represents cost data for a group
// @Description Cost breakdown for a specific group (project, model, or day)
type CostBreakdownEntry struct {
	Name       string         `json:"name" example:"my-app" description:"Name of the group (project, model, or date)"`
	Cost       float64        `json:"cost" example:"45.30" description:"Total cost in USD"`
	Tokens     TokenBreakdown `json:"tokens" description:"Token usage breakdown"`
	Sessions   int            `json:"sessions" example:"23" description:"Number of sessions"`
	Percentage float64        `json:"percentage" example:"0.36" description:"Percentage of total cost"`
}

// CostProjection represents projected costs
// @Description Projected cost estimates
type CostProjection struct {
	DailyAverage    float64 `json:"daily_average" example:"4.18" description:"Average daily cost"`
	MonthlyEstimate float64 `json:"monthly_estimate" example:"125.40" description:"Estimated monthly cost"`
}

// CostAnalyticsResponse represents the response for cost analytics
// @Description Cost analytics data with breakdown and projections
type CostAnalyticsResponse struct {
	TotalCost    float64               `json:"total_cost" example:"125.50" description:"Total cost in USD"`
	CacheSavings float64               `json:"cache_savings" example:"35.20" description:"Estimated savings from cache hits in USD"`
	Breakdown    []CostBreakdownEntry  `json:"breakdown" description:"Cost breakdown by group"`
	Projection   CostProjection        `json:"projection" description:"Cost projections"`
}

// RecentFile represents a recently modified file
// @Description Recently modified file information
type RecentFile struct {
	FilePath     string  `json:"file_path" example:"/src/app.ts" description:"Full path to the modified file"`
	LastModified string  `json:"last_modified" example:"2023-01-01T10:00:00Z" description:"Last modification timestamp"`
	SessionID    string  `json:"session_id" example:"session_123456" description:"ID of the session that modified the file"`
	SessionTitle string  `json:"session_title" example:"Implement user authentication" description:"Title or task of the session"`
	ProjectName  string  `json:"project_name" example:"my-app" description:"Name of the project"`
	ProjectPath  string  `json:"project_path" example:"/Users/ksred/projects/my-app" description:"Full path to the project"`
	ToolName     string  `json:"tool_name" example:"Edit" description:"Name of the tool used to modify the file"`
	Occurrences  int     `json:"occurrences" example:"5" description:"Number of times this file was modified"`
	GitBranch    *string `json:"git_branch,omitempty" example:"feature/auth" description:"Git branch where the file was modified"`
}

// RecentFilesResponse represents the response for recent files endpoint
// @Description Response containing recently modified files
type RecentFilesResponse struct {
	Files  []RecentFile `json:"files" description:"List of recently modified files"`
	Total  int          `json:"total" example:"150" description:"Total number of recent files"`
	Limit  int          `json:"limit" example:"20" description:"Maximum number of files returned"`
	Offset int          `json:"offset" example:"0" description:"Number of files skipped"`
}

// ProjectRecentFile represents a file modified within a specific project
// @Description File modification information for a specific project
type ProjectRecentFile struct {
	FilePath           string                    `json:"file_path" example:"/src/app.ts" description:"Full path to the modified file"`
	LastModified       string                    `json:"last_modified" example:"2023-01-01T10:00:00Z" description:"Last modification timestamp"`
	Sessions           []ProjectRecentFileSession `json:"sessions" description:"Sessions that modified this file"`
	ToolsUsed          []string                  `json:"tools_used" example:"[\"Edit\", \"Write\"]" description:"List of tools used to modify the file"`
	TotalModifications int                       `json:"total_modifications" example:"8" description:"Total number of modifications"`
}

// ProjectRecentFileSession represents session info for a project recent file
// @Description Session information for a file modification
type ProjectRecentFileSession struct {
	SessionID    string  `json:"session_id" example:"session_123456" description:"ID of the session"`
	SessionTitle string  `json:"session_title" example:"Implement user authentication" description:"Title or task of the session"`
	GitBranch    *string `json:"git_branch,omitempty" example:"feature/auth" description:"Git branch of the session"`
}

// ProjectRecentFilesResponse represents the response for project recent files endpoint
// @Description Response containing recently modified files for a specific project
type ProjectRecentFilesResponse struct {
	ProjectName string              `json:"project_name" example:"my-app" description:"Name of the project"`
	Files       []ProjectRecentFile `json:"files" description:"List of recently modified files in the project"`
	Total       int                 `json:"total" example:"45" description:"Total number of recent files in the project"`
}

// TokenTimelineEntry represents a point in the token usage timeline
// @Description Token usage data for a specific time period
type TokenTimelineEntry struct {
	Timestamp           string  `json:"timestamp" example:"2024-01-08T14:00:00Z" description:"Timestamp for this data point"`
	InputTokens         int     `json:"input_tokens" example:"15000" description:"Number of input tokens"`
	OutputTokens        int     `json:"output_tokens" example:"8000" description:"Number of output tokens"`
	CacheCreationTokens int     `json:"cache_creation_tokens" example:"5000" description:"Number of cache creation tokens"`
	CacheReadTokens     int     `json:"cache_read_tokens" example:"3000" description:"Number of cache read tokens"`
	TotalTokens         int     `json:"total_tokens" example:"31000" description:"Total number of tokens"`
	EstimatedCost       float64 `json:"estimated_cost" example:"0.75" description:"Estimated cost in USD"`
	MessageCount        int     `json:"message_count" example:"12" description:"Number of messages in this period"`
}

// TokenTimelineResponse represents the response for token timeline endpoints
// @Description Response containing token usage timeline data
type TokenTimelineResponse struct {
	Timeline     []TokenTimelineEntry `json:"timeline" description:"List of timeline data points"`
	Hours        int                  `json:"hours,omitempty" example:"24" description:"Number of hours included"`
	Granularity  string               `json:"granularity" example:"hour" description:"Time granularity (minute, hour, day)"`
	Total        int                  `json:"total" example:"24" description:"Total number of data points"`
	SessionID    string               `json:"session_id,omitempty" example:"session_123456" description:"Session ID (for session-specific timeline)"`
	ProjectName  string               `json:"project_name,omitempty" example:"my-app" description:"Project name (for project-specific timeline)"`
}