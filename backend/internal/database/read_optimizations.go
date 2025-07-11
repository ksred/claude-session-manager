package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// ReadOptimizedRepository provides read-optimized database operations
type ReadOptimizedRepository struct {
	db *Database
}

// NewReadOptimizedRepository creates a new read-optimized repository
func NewReadOptimizedRepository(db *Database) *ReadOptimizedRepository {
	return &ReadOptimizedRepository{db: db}
}

// GetSessionTokenTimelineOptimized returns token usage over time for a specific session using read-only transaction
func (r *ReadOptimizedRepository) GetSessionTokenTimelineOptimized(sessionID string, hours int, granularity string) ([]TokenTimelineEntry, error) {
	var entries []TokenTimelineEntry
	
	// Execute in read-only transaction
	err := r.executeInReadTransaction(func(tx *sqlx.Tx) error {
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
			timeFormat = "%Y-%m-%d %H:%M:00"
		}

		query := `
			SELECT 
				strftime(?, m.timestamp) as timestamp,
				COALESCE(SUM(tu.input_tokens), 0) as input_tokens,
				COALESCE(SUM(tu.output_tokens), 0) as output_tokens,
				COALESCE(SUM(tu.cache_creation_input_tokens), 0) as cache_creation_tokens,
				COALESCE(SUM(tu.cache_read_input_tokens), 0) as cache_read_tokens,
				COALESCE(SUM(tu.input_tokens + tu.output_tokens + tu.cache_creation_input_tokens + tu.cache_read_input_tokens), 0) as total_tokens,
				COALESCE(SUM(tu.estimated_cost), 0.0) as estimated_cost,
				COUNT(DISTINCT m.id) as message_count
			FROM messages m
			LEFT JOIN token_usage tu ON m.id = tu.message_id
			WHERE m.session_id = ?
			AND m.timestamp >= datetime('now', '-' || ? || ' hours')
			GROUP BY strftime(?, m.timestamp)
			ORDER BY timestamp ASC
		`

		return tx.Select(&entries, query, timeFormat, sessionID, hours, timeFormat)
	})
	
	return entries, err
}

// GetTokenTimelineOptimized returns overall token usage timeline using read-only transaction
func (r *ReadOptimizedRepository) GetTokenTimelineOptimized(hours int, granularity string) ([]TokenTimelineEntry, error) {
	var entries []TokenTimelineEntry
	
	err := r.executeInReadTransaction(func(tx *sqlx.Tx) error {
		var timeFormat string
		switch granularity {
		case "minute":
			timeFormat = "%Y-%m-%d %H:%M:00"
		case "hour":
			timeFormat = "%Y-%m-%d %H:00:00"
		case "day":
			timeFormat = "%Y-%m-%d 00:00:00"
		default:
			timeFormat = "%Y-%m-%d %H:00:00"
		}

		query := `
			SELECT 
				strftime(?, m.timestamp) as timestamp,
				COALESCE(SUM(tu.input_tokens), 0) as input_tokens,
				COALESCE(SUM(tu.output_tokens), 0) as output_tokens,
				COALESCE(SUM(tu.cache_creation_input_tokens), 0) as cache_creation_tokens,
				COALESCE(SUM(tu.cache_read_input_tokens), 0) as cache_read_tokens,
				COALESCE(SUM(tu.input_tokens + tu.output_tokens + tu.cache_creation_input_tokens + tu.cache_read_input_tokens), 0) as total_tokens,
				COALESCE(SUM(tu.estimated_cost), 0.0) as estimated_cost,
				COUNT(DISTINCT m.id) as message_count
			FROM messages m
			LEFT JOIN token_usage tu ON m.id = tu.message_id
			WHERE m.timestamp >= datetime('now', '-' || ? || ' hours')
			GROUP BY strftime(?, m.timestamp)
			ORDER BY timestamp ASC
		`

		return tx.Select(&entries, query, timeFormat, hours, timeFormat)
	})
	
	return entries, err
}

// GetAllSessionsOptimized returns all sessions with summary information using read-only transaction
func (r *ReadOptimizedRepository) GetAllSessionsOptimized() ([]*SessionSummary, error) {
	var sessions []*SessionSummary
	
	err := r.executeInReadTransaction(func(tx *sqlx.Tx) error {
		return tx.Select(&sessions, "SELECT * FROM session_summary ORDER BY last_activity DESC")
	})
	
	return sessions, err
}

// GetActiveSessionsOptimized returns currently active sessions using read-only transaction
func (r *ReadOptimizedRepository) GetActiveSessionsOptimized() ([]*SessionSummary, error) {
	var sessions []*SessionSummary
	
	err := r.executeInReadTransaction(func(tx *sqlx.Tx) error {
		return tx.Select(&sessions, `
			SELECT * FROM session_summary 
			WHERE is_active = 1 
			ORDER BY last_activity DESC
		`)
	})
	
	return sessions, err
}

// executeInReadTransaction executes a function within a transaction optimized for reads
func (r *ReadOptimizedRepository) executeInReadTransaction(fn func(*sqlx.Tx) error) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin read transaction: %w", err)
	}
	defer tx.Rollback()

	// Note: We don't use PRAGMA query_only because it affects the entire connection,
	// not just the transaction, which can cause "readonly database" errors elsewhere

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}

// GetSessionByIDOptimized returns a specific session by ID using read-only transaction
func (r *ReadOptimizedRepository) GetSessionByIDOptimized(sessionID string) (*SessionSummary, error) {
	var session SessionSummary
	
	err := r.executeInReadTransaction(func(tx *sqlx.Tx) error {
		err := tx.Get(&session, "SELECT * FROM session_summary WHERE id = ?", sessionID)
		if err == sql.ErrNoRows {
			return fmt.Errorf("session not found: %s", sessionID)
		}
		return err
	})
	
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetSessionActivityOptimized returns recent activity for a specific session using read-only transaction
func (r *ReadOptimizedRepository) GetSessionActivityOptimized(sessionID string, limit int) ([]*ActivityLogEntry, error) {
	var activities []*ActivityLogEntry
	
	err := r.executeInReadTransaction(func(tx *sqlx.Tx) error {
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

		return tx.Select(&activities, query, sessionID, sessionID, sessionID, sessionID, limit)
	})
	
	return activities, err
}

// GetRecentActivityOptimized returns recent activity timeline using read-only transaction
func (r *ReadOptimizedRepository) GetRecentActivityOptimized(limit int) ([]*ActivityLogEntry, error) {
	var activities []*ActivityLogEntry
	
	err := r.executeInReadTransaction(func(tx *sqlx.Tx) error {
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
		err := tx.Select(&tempActivities, query, limit)
		if err != nil {
			return err
		}
		
		// Convert to ActivityLogEntry
		activities = make([]*ActivityLogEntry, len(tempActivities))
		for i, ta := range tempActivities {
			activities[i] = &ActivityLogEntry{
				ID:           &ta.ID,
				SessionID:    ta.SessionID,
				ActivityType: ta.ActivityType,
				Details:      ta.Details,
				Timestamp:    ta.Timestamp,
				CreatedAt:    ta.CreatedAt,
			}
		}
		
		return nil
	})
	
	return activities, err
}