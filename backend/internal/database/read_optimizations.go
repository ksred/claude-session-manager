package database

import (
	"database/sql"
	"fmt"

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