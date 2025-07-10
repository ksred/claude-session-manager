package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// SessionRepository provides database operations for sessions
type SessionRepository struct {
	db     *Database
	logger *logrus.Logger
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *Database, logger *logrus.Logger) *SessionRepository {
	return &SessionRepository{
		db:     db,
		logger: logger,
	}
}

// GetAllSessions returns all sessions with summary information
func (r *SessionRepository) GetAllSessions() ([]*SessionSummary, error) {
	var sessions []*SessionSummary
	err := r.db.Select(&sessions, "SELECT * FROM session_summary ORDER BY last_activity DESC")
	if err != nil {
		return nil, fmt.Errorf("failed to get all sessions: %w", err)
	}
	return sessions, nil
}

// GetSessionByID returns a specific session by ID
func (r *SessionRepository) GetSessionByID(sessionID string) (*SessionSummary, error) {
	var session SessionSummary
	err := r.db.Get(&session, "SELECT * FROM session_summary WHERE id = ?", sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found: %s", sessionID)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &session, nil
}

// GetActiveSessions returns currently active sessions
func (r *SessionRepository) GetActiveSessions() ([]*SessionSummary, error) {
	var sessions []*SessionSummary
	err := r.db.Select(&sessions,
		"SELECT * FROM session_summary WHERE is_active = true ORDER BY last_activity DESC")
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}
	return sessions, nil
}

// GetRecentSessions returns the N most recent sessions
func (r *SessionRepository) GetRecentSessions(limit int) ([]*SessionSummary, error) {
	var sessions []*SessionSummary
	err := r.db.Select(&sessions,
		"SELECT * FROM session_summary ORDER BY last_activity DESC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent sessions: %w", err)
	}
	return sessions, nil
}

// SearchSessions searches sessions by query string
func (r *SessionRepository) SearchSessions(query string) ([]*SessionSummary, error) {
	query = strings.ToLower(query)
	var sessions []*SessionSummary

	searchSQL := `
		SELECT DISTINCT s.* FROM session_summary s
		LEFT JOIN messages m ON s.id = m.session_id
		WHERE LOWER(s.project_name) LIKE ? 
		   OR LOWER(m.content) LIKE ?
		   OR LOWER(s.files_modified) LIKE ?
		ORDER BY s.last_activity DESC
	`

	searchPattern := "%" + query + "%"
	err := r.db.Select(&sessions, searchSQL, searchPattern, searchPattern, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search sessions: %w", err)
	}
	return sessions, nil
}

// GetTotalSessions returns the total number of sessions
func (r *SessionRepository) GetTotalSessions() (int, error) {
	var count int
	err := r.db.Get(&count, "SELECT COUNT(*) FROM sessions")
	if err != nil {
		return 0, fmt.Errorf("failed to get total sessions: %w", err)
	}
	return count, nil
}

// GetActiveSessionsCount returns the number of active sessions
func (r *SessionRepository) GetActiveSessionsCount() (int, error) {
	var count int
	err := r.db.Get(&count, "SELECT COUNT(*) FROM sessions WHERE is_active = true")
	if err != nil {
		return 0, fmt.Errorf("failed to get active sessions count: %w", err)
	}
	return count, nil
}

// GetTotalMessages returns the total number of messages
func (r *SessionRepository) GetTotalMessages() (int, error) {
	var count int
	err := r.db.Get(&count, "SELECT COUNT(*) FROM messages")
	if err != nil {
		return 0, fmt.Errorf("failed to get total messages: %w", err)
	}
	return count, nil
}

// GetOverallTokenUsage returns aggregated token usage
func (r *SessionRepository) GetOverallTokenUsage() (*TokenUsageAggregate, error) {
	var usage TokenUsageAggregate
	err := r.db.Get(&usage, `
		SELECT 
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(cache_creation_input_tokens), 0) as cache_creation_input_tokens,
			COALESCE(SUM(cache_read_input_tokens), 0) as cache_read_input_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(estimated_cost), 0.0) as estimated_cost
		FROM token_usage
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get overall token usage: %w", err)
	}
	return &usage, nil
}

// GetEstimatedCost returns total estimated cost
func (r *SessionRepository) GetEstimatedCost() (float64, error) {
	var cost float64
	err := r.db.Get(&cost, "SELECT COALESCE(SUM(estimated_cost), 0.0) FROM token_usage")
	if err != nil {
		return 0, fmt.Errorf("failed to get estimated cost: %w", err)
	}
	return cost, nil
}

// GetAverageSessionDuration returns average session duration in minutes
func (r *SessionRepository) GetAverageSessionDuration() (float64, error) {
	var duration float64
	err := r.db.Get(&duration, `
		SELECT COALESCE(AVG(duration_seconds / 60.0), 0.0) 
		FROM sessions 
		WHERE duration_seconds > 0
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to get average duration: %w", err)
	}
	return duration, nil
}

// GetMostUsedModel returns the most frequently used model
func (r *SessionRepository) GetMostUsedModel() (string, error) {
	var model string
	err := r.db.Get(&model, `
		SELECT COALESCE(model, 'unknown') 
		FROM sessions 
		WHERE model IS NOT NULL AND model != ''
		GROUP BY model 
		ORDER BY COUNT(*) DESC 
		LIMIT 1
	`)
	if err != nil {
		if err == sql.ErrNoRows {
			return "unknown", nil
		}
		return "", fmt.Errorf("failed to get most used model: %w", err)
	}
	return model, nil
}

// GetModelUsage returns usage count by model
func (r *SessionRepository) GetModelUsage() (map[string]int, error) {
	rows, err := r.db.Query(`
		SELECT model, COUNT(*) as count 
		FROM sessions 
		WHERE model IS NOT NULL AND model != ''
		GROUP BY model 
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get model usage: %w", err)
	}
	defer rows.Close()

	usage := make(map[string]int)
	for rows.Next() {
		var model string
		var count int
		if err := rows.Scan(&model, &count); err != nil {
			return nil, fmt.Errorf("failed to scan model usage: %w", err)
		}
		usage[model] = count
	}
	return usage, nil
}

// GetDailyMetrics returns daily metrics for the last N days
func (r *SessionRepository) GetDailyMetrics(days int) ([]*DailyMetric, error) {
	var metrics []*DailyMetric
	err := r.db.Select(&metrics, `
		SELECT 
			date,
			SUM(session_count) as session_count,
			SUM(message_count) as message_count,
			'all' as model,
			SUM(total_tokens) as total_tokens
		FROM daily_metrics 
		WHERE date >= date('now', '-' || ? || ' days')
		GROUP BY date
		ORDER BY date DESC
	`, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily metrics: %w", err)
	}
	return metrics, nil
}

// GetPeakHours returns peak usage hours
func (r *SessionRepository) GetPeakHours() ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`
		SELECT 
			strftime('%H', timestamp) as hour,
			COUNT(*) as message_count,
			COUNT(DISTINCT DATE(timestamp)) as unique_days
		FROM messages 
		WHERE timestamp >= datetime('now', '-30 days')
		GROUP BY strftime('%H', timestamp)
		HAVING message_count > 10
		ORDER BY message_count DESC
		LIMIT 4
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get peak hours: %w", err)
	}
	defer rows.Close()

	var peakHours []map[string]interface{}
	for rows.Next() {
		var hourStr string
		var messageCount int
		var uniqueDays int
		if err := rows.Scan(&hourStr, &messageCount, &uniqueDays); err != nil {
			return nil, fmt.Errorf("failed to scan peak hours: %w", err)
		}

		// Convert hour string to int
		var hour int
		if _, err := fmt.Sscanf(hourStr, "%d", &hour); err != nil {
			continue
		}

		averageSessions := float64(messageCount) / float64(uniqueDays)
		if averageSessions > 1.0 {
			peakHours = append(peakHours, map[string]interface{}{
				"hour":             hour,
				"average_sessions": averageSessions,
			})
		}
	}
	return peakHours, nil
}

// GetRecentActivity returns recent activity timeline
func (r *SessionRepository) GetRecentActivity(limit int) ([]*ActivityLogEntry, error) {
	// Create a combined view of activities from multiple sources
	query := `
		WITH combined_activity AS (
			-- Get recent user messages directly from messages table
			SELECT 
				NULL as id,
				m.session_id,
				'message_sent' as activity_type,
				CASE 
					-- Tool results
					WHEN m.content LIKE '%"type":"tool_result"%' THEN 
						CASE
							WHEN m.content LIKE '%"is_error":true%' THEN 'Tool error response'
							WHEN m.content LIKE '%has been updated%' THEN 'File edited'
							WHEN m.content LIKE '%File created successfully%' THEN 'File created'
							WHEN m.content LIKE '%curl%' OR m.content LIKE '%http%' THEN 'API test result'
							ELSE 'Tool result'
						END
					-- System messages
					WHEN m.content LIKE '%[Request interrupted%' THEN 'Request interrupted by user'
					-- JSON arrays (other tool responses)
					WHEN m.content LIKE '[{%' THEN 'Tool response'
					-- Regular messages
					WHEN LENGTH(m.content) > 100 THEN 'User: ' || SUBSTR(m.content, 1, 100) || '...'
					ELSE 'User: ' || m.content
				END as details,
				m.timestamp,
				m.timestamp as created_at
			FROM messages m
			JOIN sessions s ON m.session_id = s.id
			WHERE m.role = 'user'
			
			UNION ALL
			
			-- Get file modifications from tool_results
			SELECT 
				NULL as id,
				tr.session_id,
				'file_modified' as activity_type,
				'Modified ' || tr.file_path || ' using ' || tr.tool_name as details,
				tr.timestamp,
				tr.timestamp as created_at
			FROM tool_results tr
			WHERE tr.file_path IS NOT NULL
			
			UNION ALL
			
			-- Get non-import activities from activity_log
			SELECT 
				id,
				session_id,
				activity_type,
				details,
				timestamp,
				created_at
			FROM activity_log
			WHERE activity_type NOT IN ('session_imported', 'import_started', 'import_completed')
		)
		SELECT 
			COALESCE(id, ROW_NUMBER() OVER (ORDER BY timestamp DESC)) as id,
			session_id,
			activity_type,
			details,
			timestamp,
			created_at
		FROM combined_activity
		ORDER BY timestamp DESC
		LIMIT ?
	`
	
	type tempActivity struct {
		ID           int       `db:"id"`
		SessionID    *string   `db:"session_id"`
		ActivityType string    `db:"activity_type"`
		Details      string    `db:"details"`
		Timestamp    time.Time `db:"timestamp"`
		CreatedAt    time.Time `db:"created_at"`
	}
	
	var tempActivities []tempActivity
	err := r.db.Select(&tempActivities, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent activity: %w", err)
	}
	
	// Convert to ActivityLogEntry
	activities := make([]*ActivityLogEntry, len(tempActivities))
	for i, ta := range tempActivities {
		activities[i] = &ActivityLogEntry{
			ID:           ta.ID,
			SessionID:    ta.SessionID,
			ActivityType: ta.ActivityType,
			Details:      ta.Details,
			Timestamp:    ta.Timestamp,
			CreatedAt:    ta.CreatedAt,
		}
	}
	
	return activities, nil
}

// GetSessionActivity returns recent activity for a specific session
func (r *SessionRepository) GetSessionActivity(sessionID string, limit int) ([]*ActivityLogEntry, error) {
	query := `
		WITH combined_activity AS (
			-- Get recent user messages for this session
			SELECT 
				NULL as id,
				m.session_id,
				'message_sent' as activity_type,
				CASE 
					-- Tool results
					WHEN m.content LIKE '%"type":"tool_result"%' THEN 
						CASE
							WHEN m.content LIKE '%"is_error":true%' THEN 'Tool error response'
							WHEN m.content LIKE '%has been updated%' THEN 'File edited'
							WHEN m.content LIKE '%File created successfully%' THEN 'File created'
							WHEN m.content LIKE '%curl%' OR m.content LIKE '%http%' THEN 'API test result'
							ELSE 'Tool result'
						END
					-- System messages
					WHEN m.content LIKE '%[Request interrupted%' THEN 'Request interrupted by user'
					-- JSON arrays (other tool responses)
					WHEN m.content LIKE '[{%' THEN 'Tool response'
					-- Regular messages
					WHEN LENGTH(m.content) > 100 THEN 'User: ' || SUBSTR(m.content, 1, 100) || '...'
					ELSE 'User: ' || m.content
				END as details,
				m.timestamp,
				m.timestamp as created_at
			FROM messages m
			WHERE m.role = 'user' 
			AND m.session_id = ?
			
			UNION ALL
			
			-- Get assistant responses (using first 100 chars as preview)
			SELECT 
				NULL as id,
				m.session_id,
				'message_received' as activity_type,
				CASE
					WHEN m.content LIKE '%' || CHAR(96) || CHAR(96) || CHAR(96) || '%' THEN 'Assistant provided code'
					WHEN LENGTH(m.content) > 100 THEN 'Assistant: ' || SUBSTR(m.content, 1, 100) || '...'
					ELSE 'Assistant: ' || m.content
				END as details,
				m.timestamp,
				m.timestamp as created_at
			FROM messages m
			WHERE m.role = 'assistant' 
			AND m.session_id = ?
			
			UNION ALL
			
			-- Get tool uses for this session
			SELECT 
				NULL as id,
				tr.session_id,
				'tool_used' as activity_type,
				tr.tool_name || 
				CASE 
					WHEN tr.file_path IS NOT NULL AND tr.file_path != '' THEN 
						': ' || tr.file_path
					ELSE ''
				END as details,
				tr.timestamp,
				tr.timestamp as created_at
			FROM tool_results tr
			WHERE tr.session_id = ?
			
			UNION ALL
			
			-- Get activity log entries for this session
			SELECT 
				al.id,
				al.session_id,
				al.activity_type,
				al.details,
				al.timestamp,
				al.created_at
			FROM activity_log al
			WHERE al.session_id = ?
		)
		SELECT DISTINCT * FROM combined_activity
		ORDER BY timestamp DESC
		LIMIT ?
	`

	var activities []*ActivityLogEntry
	err := r.db.Select(&activities, query, sessionID, sessionID, sessionID, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get session activity: %w", err)
	}

	return activities, nil
}

// GetProjectActivity returns recent activity for all sessions in a project
func (r *SessionRepository) GetProjectActivity(projectName string, limit int) ([]*ActivityLogEntry, error) {
	query := `
		WITH project_sessions AS (
			SELECT id FROM sessions WHERE project_name = ?
		),
		combined_activity AS (
			-- Get recent user messages for project sessions
			SELECT 
				NULL as id,
				m.session_id,
				'message_sent' as activity_type,
				CASE 
					-- Tool results
					WHEN m.content LIKE '%"type":"tool_result"%' THEN 
						CASE
							WHEN m.content LIKE '%"is_error":true%' THEN 'Tool error response'
							WHEN m.content LIKE '%has been updated%' THEN 'File edited'
							WHEN m.content LIKE '%File created successfully%' THEN 'File created'
							WHEN m.content LIKE '%curl%' OR m.content LIKE '%http%' THEN 'API test result'
							ELSE 'Tool result'
						END
					-- System messages
					WHEN m.content LIKE '%[Request interrupted%' THEN 'Request interrupted by user'
					-- JSON arrays (other tool responses)
					WHEN m.content LIKE '[{%' THEN 'Tool response'
					-- Regular messages
					WHEN LENGTH(m.content) > 100 THEN 'User: ' || SUBSTR(m.content, 1, 100) || '...'
					ELSE 'User: ' || m.content
				END as details,
				m.timestamp,
				m.timestamp as created_at
			FROM messages m
			WHERE m.role = 'user' 
			AND m.session_id IN (SELECT id FROM project_sessions)
			
			UNION ALL
			
			-- Get assistant responses
			SELECT 
				NULL as id,
				m.session_id,
				'message_received' as activity_type,
				CASE
					WHEN m.content LIKE '%' || CHAR(96) || CHAR(96) || CHAR(96) || '%' THEN 'Assistant provided code'
					WHEN LENGTH(m.content) > 100 THEN 'Assistant: ' || SUBSTR(m.content, 1, 100) || '...'
					ELSE 'Assistant: ' || m.content
				END as details,
				m.timestamp,
				m.timestamp as created_at
			FROM messages m
			WHERE m.role = 'assistant' 
			AND m.session_id IN (SELECT id FROM project_sessions)
			
			UNION ALL
			
			-- Get tool uses for project sessions
			SELECT 
				NULL as id,
				tr.session_id,
				'tool_used' as activity_type,
				tr.tool_name || 
				CASE 
					WHEN tr.file_path IS NOT NULL AND tr.file_path != '' THEN 
						': ' || tr.file_path
					ELSE ''
				END as details,
				tr.timestamp,
				tr.timestamp as created_at
			FROM tool_results tr
			WHERE tr.session_id IN (SELECT id FROM project_sessions)
			
			UNION ALL
			
			-- Get activity log entries for project sessions
			SELECT 
				al.id,
				al.session_id,
				al.activity_type,
				al.details,
				al.timestamp,
				al.created_at
			FROM activity_log al
			WHERE al.session_id IN (SELECT id FROM project_sessions)
		)
		SELECT DISTINCT * FROM combined_activity
		ORDER BY timestamp DESC
		LIMIT ?
	`

	var activities []*ActivityLogEntry
	err := r.db.Select(&activities, query, projectName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get project activity: %w", err)
	}

	return activities, nil
}

// TokenUsageAggregate represents aggregated token usage
type TokenUsageAggregate struct {
	InputTokens              int     `db:"input_tokens" json:"input_tokens"`
	OutputTokens             int     `db:"output_tokens" json:"output_tokens"`
	CacheCreationInputTokens int     `db:"cache_creation_input_tokens" json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int     `db:"cache_read_input_tokens" json:"cache_read_input_tokens"`
	TotalTokens              int     `db:"total_tokens" json:"total_tokens"`
	EstimatedCost            float64 `db:"estimated_cost" json:"estimated_cost"`
}

// UpsertSession creates or updates a session
func (r *SessionRepository) UpsertSession(session *Session) error {
	return r.db.Transaction(func(tx *sqlx.Tx) error {
		_, err := tx.NamedExec(`
			INSERT OR REPLACE INTO sessions (
				id, project_path, project_name, file_path, git_branch, git_worktree,
				start_time, last_activity, is_active, status, model, message_count,
				duration_seconds, updated_at
			) VALUES (
				:id, :project_path, :project_name, :file_path, :git_branch, :git_worktree,
				:start_time, :last_activity, :is_active, :status, :model, :message_count,
				:duration_seconds, CURRENT_TIMESTAMP
			)
		`, session)
		return err
	})
}

// UpsertMessage creates or updates a message
func (r *SessionRepository) UpsertMessage(message *Message) error {
	return r.db.Transaction(func(tx *sqlx.Tx) error {
		_, err := tx.NamedExec(`
			INSERT OR REPLACE INTO messages (
				id, session_id, parent_uuid, is_sidechain, user_type, cwd, version,
				type, role, content, request_id, timestamp
			) VALUES (
				:id, :session_id, :parent_uuid, :is_sidechain, :user_type, :cwd, :version,
				:type, :role, :content, :request_id, :timestamp
			)
		`, message)
		return err
	})
}

// UpsertTokenUsage creates or updates token usage
func (r *SessionRepository) UpsertTokenUsage(usage *TokenUsage) error {
	return r.db.Transaction(func(tx *sqlx.Tx) error {
		_, err := tx.NamedExec(`
			INSERT OR REPLACE INTO token_usage (
				message_id, session_id, input_tokens, output_tokens,
				cache_creation_input_tokens, cache_read_input_tokens, total_tokens,
				service_tier, estimated_cost
			) VALUES (
				:message_id, :session_id, :input_tokens, :output_tokens,
				:cache_creation_input_tokens, :cache_read_input_tokens, :total_tokens,
				:service_tier, :estimated_cost
			)
		`, usage)
		return err
	})
}

// UpsertToolResult creates or updates a tool result
func (r *SessionRepository) UpsertToolResult(result *ToolResult) error {
	return r.db.Transaction(func(tx *sqlx.Tx) error {
		_, err := tx.NamedExec(`
			INSERT OR REPLACE INTO tool_results (
				message_id, session_id, tool_name, file_path, result_data, timestamp
			) VALUES (
				:message_id, :session_id, :tool_name, :file_path, :result_data, :timestamp
			)
		`, result)
		return err
	})
}

// LogActivity logs an activity entry
func (r *SessionRepository) LogActivity(entry *ActivityLogEntry) error {
	_, err := r.db.NamedExec(`
		INSERT INTO activity_log (session_id, activity_type, details, timestamp)
		VALUES (:session_id, :activity_type, :details, :timestamp)
	`, entry)
	return err
}

// RecentFile represents a recently modified file
type RecentFile struct {
	FilePath     string  `db:"file_path" json:"file_path"`
	LastModified string  `db:"last_modified" json:"last_modified"`
	SessionID    string  `db:"session_id" json:"session_id"`
	SessionTitle string  `db:"session_title" json:"session_title"`
	ProjectName  string  `db:"project_name" json:"project_name"`
	ProjectPath  string  `db:"project_path" json:"project_path"`
	ToolName     string  `db:"tool_name" json:"tool_name"`
	Occurrences  int     `db:"occurrences" json:"occurrences"`
	GitBranch    *string `db:"git_branch" json:"git_branch,omitempty"`
}

// ProjectRecentFile represents a file modified within a specific project
type ProjectRecentFile struct {
	FilePath           string              `db:"file_path" json:"file_path"`
	LastModified       string              `db:"last_modified" json:"last_modified"`
	TotalModifications int                 `db:"total_modifications" json:"total_modifications"`
	ToolsUsed          string              `db:"tools_used" json:"tools_used"` // Comma-separated list
	Sessions           []RecentFileSession `json:"sessions"`
}

// RecentFileSession represents session info for a recently modified file
type RecentFileSession struct {
	SessionID    string  `json:"session_id"`
	SessionTitle string  `json:"session_title"`
	GitBranch    *string `json:"git_branch,omitempty"`
}

// GetRecentFiles returns recently modified files across all sessions
func (r *SessionRepository) GetRecentFiles(limit, offset int) ([]RecentFile, int, error) {
	// Count total recent files
	var total int
	err := r.db.Get(&total, `
		SELECT COUNT(DISTINCT file_path) 
		FROM tool_results 
		WHERE file_path IS NOT NULL
	`)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count recent files: %w", err)
	}

	// Get recent files with session details
	var files []RecentFile
	err = r.db.Select(&files, `
		WITH recent_files AS (
			SELECT 
				tr.file_path,
				MAX(tr.timestamp) as last_modified,
				tr.session_id,
				tr.tool_name,
				COUNT(*) as occurrences,
				s.project_name,
				s.project_path,
				s.git_branch,
				COALESCE(s.project_name || ' - ' || s.git_branch, s.project_name) as session_title
			FROM tool_results tr
			JOIN sessions s ON tr.session_id = s.id
			WHERE tr.file_path IS NOT NULL
			GROUP BY tr.file_path, tr.session_id
		)
		SELECT 
			file_path,
			last_modified,
			session_id,
			session_title,
			project_name,
			project_path,
			tool_name,
			occurrences,
			git_branch
		FROM recent_files
		ORDER BY last_modified DESC
		LIMIT ? OFFSET ?
	`, limit, offset)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to get recent files: %w", err)
	}

	return files, total, nil
}

// TokenTimelineEntry represents a single point in the token usage timeline
type TokenTimelineEntry struct {
	Timestamp           string  `db:"timestamp" json:"timestamp"`
	InputTokens         int     `db:"input_tokens" json:"input_tokens"`
	OutputTokens        int     `db:"output_tokens" json:"output_tokens"`
	CacheCreationTokens int     `db:"cache_creation_tokens" json:"cache_creation_tokens"`
	CacheReadTokens     int     `db:"cache_read_tokens" json:"cache_read_tokens"`
	TotalTokens         int     `db:"total_tokens" json:"total_tokens"`
	EstimatedCost       float64 `db:"estimated_cost" json:"estimated_cost"`
	MessageCount        int     `db:"message_count" json:"message_count"`
}

// GetTokenTimeline returns overall token usage over time with configurable granularity
func (r *SessionRepository) GetTokenTimeline(hours int, granularity string) ([]TokenTimelineEntry, error) {
	// Determine the time format based on granularity
	var timeFormat string
	switch granularity {
	case "minute":
		timeFormat = "%Y-%m-%d %H:%M:00"
	case "hour":
		timeFormat = "%Y-%m-%d %H:00:00"
	case "day":
		timeFormat = "%Y-%m-%d 00:00:00"
	default:
		timeFormat = "%Y-%m-%d %H:00:00" // Default to hourly
	}

	query := `
		SELECT 
			strftime(?, m.timestamp) as timestamp,
			SUM(tu.input_tokens) as input_tokens,
			SUM(tu.output_tokens) as output_tokens,
			SUM(tu.cache_creation_input_tokens) as cache_creation_tokens,
			SUM(tu.cache_read_input_tokens) as cache_read_tokens,
			SUM(tu.input_tokens + tu.output_tokens + tu.cache_creation_input_tokens + tu.cache_read_input_tokens) as total_tokens,
			SUM(tu.estimated_cost) as estimated_cost,
			COUNT(DISTINCT m.id) as message_count
		FROM messages m
		JOIN token_usage tu ON m.id = tu.message_id
		WHERE m.timestamp >= datetime('now', '-' || ? || ' hours')
		GROUP BY strftime(?, m.timestamp)
		ORDER BY timestamp ASC
	`

	var entries []TokenTimelineEntry
	err := r.db.Select(&entries, query, timeFormat, hours, timeFormat)
	return entries, err
}

// GetSessionTokenTimeline returns token usage over time for a specific session
func (r *SessionRepository) GetSessionTokenTimeline(sessionID string, granularity string) ([]TokenTimelineEntry, error) {
	// Determine the time format based on granularity
	var timeFormat string
	switch granularity {
	case "minute":
		timeFormat = "%Y-%m-%d %H:%M:00"
	case "hour":
		timeFormat = "%Y-%m-%d %H:00:00"
	case "day":
		timeFormat = "%Y-%m-%d 00:00:00"
	default:
		timeFormat = "%Y-%m-%d %H:%M:00" // Default to minute for session view
	}

	query := `
		SELECT 
			strftime(?, m.timestamp) as timestamp,
			SUM(tu.input_tokens) as input_tokens,
			SUM(tu.output_tokens) as output_tokens,
			SUM(tu.cache_creation_input_tokens) as cache_creation_tokens,
			SUM(tu.cache_read_input_tokens) as cache_read_tokens,
			SUM(tu.input_tokens + tu.output_tokens + tu.cache_creation_input_tokens + tu.cache_read_input_tokens) as total_tokens,
			SUM(tu.estimated_cost) as estimated_cost,
			COUNT(DISTINCT m.id) as message_count
		FROM messages m
		JOIN token_usage tu ON m.id = tu.message_id
		WHERE m.session_id = ?
		GROUP BY strftime(?, m.timestamp)
		ORDER BY timestamp ASC
	`

	var entries []TokenTimelineEntry
	err := r.db.Select(&entries, query, timeFormat, sessionID, timeFormat)
	return entries, err
}

// GetProjectTokenTimeline returns token usage over time for a specific project
func (r *SessionRepository) GetProjectTokenTimeline(projectName string, hours int, granularity string) ([]TokenTimelineEntry, error) {
	// Determine the time format based on granularity
	var timeFormat string
	switch granularity {
	case "minute":
		timeFormat = "%Y-%m-%d %H:%M:00"
	case "hour":
		timeFormat = "%Y-%m-%d %H:00:00"
	case "day":
		timeFormat = "%Y-%m-%d 00:00:00"
	default:
		timeFormat = "%Y-%m-%d %H:00:00" // Default to hourly
	}

	query := `
		SELECT 
			strftime(?, m.timestamp) as timestamp,
			SUM(tu.input_tokens) as input_tokens,
			SUM(tu.output_tokens) as output_tokens,
			SUM(tu.cache_creation_input_tokens) as cache_creation_tokens,
			SUM(tu.cache_read_input_tokens) as cache_read_tokens,
			SUM(tu.input_tokens + tu.output_tokens + tu.cache_creation_input_tokens + tu.cache_read_input_tokens) as total_tokens,
			SUM(tu.estimated_cost) as estimated_cost,
			COUNT(DISTINCT m.id) as message_count
		FROM messages m
		JOIN token_usage tu ON m.id = tu.message_id
		JOIN sessions s ON m.session_id = s.id
		WHERE s.project_name = ? AND m.timestamp >= datetime('now', '-' || ? || ' hours')
		GROUP BY strftime(?, m.timestamp)
		ORDER BY timestamp ASC
	`

	var entries []TokenTimelineEntry
	err := r.db.Select(&entries, query, timeFormat, projectName, hours, timeFormat)
	return entries, err
}

// GetProjectRecentFiles returns recently modified files for a specific project
func (r *SessionRepository) GetProjectRecentFiles(projectName string, limit int, branch *string) ([]ProjectRecentFile, error) {
	// Base query
	query := `
		WITH project_files AS (
			SELECT 
				tr.file_path,
				MAX(tr.timestamp) as last_modified,
				COUNT(*) as total_modifications,
				GROUP_CONCAT(DISTINCT tr.tool_name) as tools_used,
				GROUP_CONCAT(DISTINCT tr.session_id || '|' || COALESCE(s.project_name || ' - ' || s.git_branch, s.project_name) || '|' || COALESCE(s.git_branch, '')) as sessions_info
			FROM tool_results tr
			JOIN sessions s ON tr.session_id = s.id
			WHERE tr.file_path IS NOT NULL
			AND s.project_name = ?
	`

	args := []interface{}{projectName}

	// Add branch filter if specified
	if branch != nil && *branch != "" {
		query += " AND s.git_branch = ?"
		args = append(args, *branch)
	}

	query += `
			GROUP BY tr.file_path
		)
		SELECT 
			file_path,
			last_modified,
			total_modifications,
			tools_used,
			sessions_info
		FROM project_files
		ORDER BY last_modified DESC
		LIMIT ?
	`

	args = append(args, limit)

	// Execute query
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get project recent files: %w", err)
	}
	defer rows.Close()

	var files []ProjectRecentFile
	for rows.Next() {
		var file ProjectRecentFile
		var sessionsInfo string

		err := rows.Scan(
			&file.FilePath,
			&file.LastModified,
			&file.TotalModifications,
			&file.ToolsUsed,
			&sessionsInfo,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Parse sessions info
		if sessionsInfo != "" {
			sessionParts := strings.Split(sessionsInfo, ",")
			seen := make(map[string]bool)

			for _, part := range sessionParts {
				info := strings.Split(part, "|")
				if len(info) >= 3 && !seen[info[0]] {
					seen[info[0]] = true
					session := RecentFileSession{
						SessionID:    info[0],
						SessionTitle: info[1],
					}
					if info[2] != "" {
						session.GitBranch = &info[2]
					}
					file.Sessions = append(file.Sessions, session)
				}
			}
		}

		files = append(files, file)
	}

	return files, nil
}
