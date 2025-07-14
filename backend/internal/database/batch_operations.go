package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// BatchOperations provides optimized batch database operations
type BatchOperations struct {
	db     *Database
	logger *logrus.Logger
}

// NewBatchOperations creates a new batch operations handler
func NewBatchOperations(db *Database, logger *logrus.Logger) *BatchOperations {
	return &BatchOperations{
		db:     db,
		logger: logger,
	}
}

// BatchImportData imports multiple sessions, messages, token usage, and tool results in a single transaction
func (bo *BatchOperations) BatchImportData(sessions []Session, messages []Message, tokenUsages []TokenUsage, toolResults []ToolResult) error {
	return bo.db.WriteOperation(func(tx *sqlx.Tx) error {
		// Batch insert sessions
		if len(sessions) > 0 {
			bo.logger.WithField("count", len(sessions)).Debug("Inserting sessions")
			if err := bo.batchUpsertSessions(tx, sessions); err != nil {
				return fmt.Errorf("failed to batch upsert sessions: %w", err)
			}
		}

		// Batch insert messages
		if len(messages) > 0 {
			bo.logger.WithField("count", len(messages)).Debug("Inserting messages")
			if err := bo.batchUpsertMessages(tx, messages); err != nil {
				return fmt.Errorf("failed to batch upsert messages: %w", err)
			}
		}

		// Batch insert token usage
		if len(tokenUsages) > 0 {
			if err := bo.batchUpsertTokenUsages(tx, tokenUsages); err != nil {
				return fmt.Errorf("failed to batch upsert token usages: %w", err)
			}
		}

		// Batch insert tool results
		if len(toolResults) > 0 {
			if err := bo.batchUpsertToolResults(tx, toolResults); err != nil {
				return fmt.Errorf("failed to batch upsert tool results: %w", err)
			}
		}

		bo.logger.Debug("Batch transaction committed successfully")
		return nil
	})
}

func (bo *BatchOperations) batchUpsertSessions(tx *sqlx.Tx, sessions []Session) error {
	if len(sessions) == 0 {
		return nil
	}

	// Build batch insert with ON CONFLICT UPDATE
	query := `
		INSERT OR REPLACE INTO sessions (id, project_name, project_path, file_path, git_branch, 
			git_worktree, start_time, last_activity, is_active, status, model, 
			message_count, duration_seconds) 
		VALUES `
	
	var values []string
	var args []interface{}
	
	for _, session := range sessions {
		placeholders := "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
		values = append(values, placeholders)
		args = append(args, session.ID, session.ProjectName, session.ProjectPath, 
			session.FilePath, session.GitBranch, session.GitWorktree, session.StartTime,
			session.LastActivity, session.IsActive, session.Status, session.Model,
			session.MessageCount, session.DurationSeconds)
	}
	
	query += strings.Join(values, ", ")

	_, err := tx.Exec(query, args...)
	return err
}

func (bo *BatchOperations) batchUpsertMessages(tx *sqlx.Tx, messages []Message) error {
	if len(messages) == 0 {
		return nil
	}

	// SQLite has a limit of 999 parameters, so batch the inserts
	const batchSize = 100 // 100 messages × 6 params = 600 params (safe under 999 limit)
	
	for i := 0; i < len(messages); i += batchSize {
		end := i + batchSize
		if end > len(messages) {
			end = len(messages)
		}
		batch := messages[i:end]
		
		query := `
			INSERT OR REPLACE INTO messages (id, session_id, role, content, timestamp, parent_uuid) 
			VALUES `
		
		var values []string
		var args []interface{}
		
		for _, msg := range batch {
			placeholders := "(?, ?, ?, ?, ?, ?)"
			values = append(values, placeholders)
			
			var parentID interface{} = sql.NullString{}
			if msg.ParentUUID != nil {
				parentID = *msg.ParentUUID
			}
			
			args = append(args, msg.ID, msg.SessionID, msg.Role, msg.Content,
				msg.Timestamp, parentID)
		}
		
		query += strings.Join(values, ", ")
		
		if i == 0 {
			bo.logger.WithFields(logrus.Fields{
				"batch_size":    batchSize,
				"total_batches": (len(messages)+batchSize-1)/batchSize,
				"total_messages": len(messages),
			}).Debug("Starting message batch processing")
		}
		
		result, err := tx.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("failed to insert message batch %d: %w", i/batchSize, err)
		}
		
		rowsAffected, _ := result.RowsAffected()
		bo.logger.WithFields(logrus.Fields{
			"batch_num": i/batchSize,
			"rows_affected": rowsAffected,
		}).Debug("Message batch processed")
	}
	
	return nil
}

func (bo *BatchOperations) batchUpsertTokenUsages(tx *sqlx.Tx, tokenUsages []TokenUsage) error {
	if len(tokenUsages) == 0 {
		return nil
	}

	// SQLite has a limit of 999 parameters, so batch the inserts
	const batchSize = 100 // 100 records × 8 params = 800 params (safe under 999 limit)
	
	for i := 0; i < len(tokenUsages); i += batchSize {
		end := i + batchSize
		if end > len(tokenUsages) {
			end = len(tokenUsages)
		}
		batch := tokenUsages[i:end]
		
		query := `
			INSERT OR REPLACE INTO token_usage (message_id, session_id, input_tokens, output_tokens, 
				cache_creation_input_tokens, cache_read_input_tokens, total_tokens, estimated_cost) 
			VALUES `
		
		var values []string
		var args []interface{}
		
		for _, tu := range batch {
			placeholders := "(?, ?, ?, ?, ?, ?, ?, ?)"
			values = append(values, placeholders)
			args = append(args, tu.MessageID, tu.SessionID, tu.InputTokens, tu.OutputTokens,
				tu.CacheCreationInputTokens, tu.CacheReadInputTokens, tu.TotalTokens, tu.EstimatedCost)
		}
		
		query += strings.Join(values, ", ")

		if _, err := tx.Exec(query, args...); err != nil {
			return fmt.Errorf("failed to insert token usage batch %d: %w", i/batchSize, err)
		}
	}
	
	return nil
}

func (bo *BatchOperations) batchUpsertToolResults(tx *sqlx.Tx, toolResults []ToolResult) error {
	if len(toolResults) == 0 {
		return nil
	}

	query := `
		INSERT OR REPLACE INTO tool_results (message_id, session_id, tool_name, result_data, 
			file_path, timestamp) 
		VALUES `
	
	var values []string
	var args []interface{}
	
	for _, tr := range toolResults {
		placeholders := "(?, ?, ?, ?, ?, ?)"
		values = append(values, placeholders)
		
		var filePath interface{} = sql.NullString{}
		if tr.FilePath != nil {
			filePath = *tr.FilePath
		}
		
		args = append(args, tr.MessageID, tr.SessionID, tr.ToolName,
			tr.ResultData, filePath, tr.Timestamp)
	}
	
	query += strings.Join(values, ", ")

	_, err := tx.Exec(query, args...)
	return err
}

// ExecuteInReadTransaction executes a function within a transaction optimized for reads
func (bo *BatchOperations) ExecuteInReadTransaction(fn func(*sqlx.Tx) error) error {
	tx, err := bo.db.Beginx()
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

// BatchImportDataIncremental imports new data using INSERT OR IGNORE to preserve existing data
func (bo *BatchOperations) BatchImportDataIncremental(sessions []Session, messages []Message, tokenUsages []TokenUsage, toolResults []ToolResult) error {
	return bo.db.WriteOperation(func(tx *sqlx.Tx) error {
		// For incremental imports, we need to update session metadata intelligently
		// First, update sessions with new activity data
		if len(sessions) > 0 {
			bo.logger.WithField("count", len(sessions)).Debug("Updating session metadata incrementally")
			if err := bo.batchUpdateSessionsIncremental(tx, sessions); err != nil {
				return fmt.Errorf("failed to update sessions incrementally: %w", err)
			}
		}

		// Insert new messages (ignore duplicates)
		if len(messages) > 0 {
			bo.logger.WithField("count", len(messages)).Debug("Inserting new messages")
			if err := bo.batchInsertMessagesIncremental(tx, messages); err != nil {
				return fmt.Errorf("failed to insert messages incrementally: %w", err)
			}
		}

		// Insert new token usage (ignore duplicates)
		if len(tokenUsages) > 0 {
			if err := bo.batchInsertTokenUsageIncremental(tx, tokenUsages); err != nil {
				return fmt.Errorf("failed to insert token usage incrementally: %w", err)
			}
		}

		// Insert new tool results (ignore duplicates)
		if len(toolResults) > 0 {
			if err := bo.batchInsertToolResultsIncremental(tx, toolResults); err != nil {
				return fmt.Errorf("failed to insert tool results incrementally: %w", err)
			}
		}

		bo.logger.Debug("Incremental batch transaction committed successfully")
		return nil
	})
}

// batchUpdateSessionsIncremental updates session metadata without replacing existing data
func (bo *BatchOperations) batchUpdateSessionsIncremental(tx *sqlx.Tx, sessions []Session) error {
	if len(sessions) == 0 {
		return nil
	}

	// For incremental updates, we update session metadata if last_activity is newer
	for _, session := range sessions {
		_, err := tx.Exec(`
			INSERT INTO sessions (id, project_name, project_path, file_path, git_branch, 
				git_worktree, start_time, last_activity, is_active, status, model, 
				message_count, duration_seconds) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				last_activity = CASE WHEN excluded.last_activity > sessions.last_activity THEN excluded.last_activity ELSE sessions.last_activity END,
				is_active = excluded.is_active,
				status = excluded.status,
				model = COALESCE(excluded.model, sessions.model),
				message_count = sessions.message_count + excluded.message_count,
				duration_seconds = CASE WHEN excluded.last_activity > sessions.last_activity THEN excluded.duration_seconds ELSE sessions.duration_seconds END
		`, session.ID, session.ProjectName, session.ProjectPath,
			session.FilePath, session.GitBranch, session.GitWorktree, session.StartTime,
			session.LastActivity, session.IsActive, session.Status, session.Model,
			session.MessageCount, session.DurationSeconds)
		if err != nil {
			return fmt.Errorf("failed to update session %s: %w", session.ID, err)
		}
	}

	return nil
}

// batchInsertMessagesIncremental inserts new messages using INSERT OR IGNORE
func (bo *BatchOperations) batchInsertMessagesIncremental(tx *sqlx.Tx, messages []Message) error {
	if len(messages) == 0 {
		return nil
	}

	// SQLite has a limit of 999 parameters, so batch the inserts
	const batchSize = 100 // 100 messages × 6 params = 600 params (safe under 999 limit)
	
	for i := 0; i < len(messages); i += batchSize {
		end := i + batchSize
		if end > len(messages) {
			end = len(messages)
		}
		batch := messages[i:end]
		
		query := `
			INSERT OR IGNORE INTO messages (id, session_id, role, content, timestamp, parent_uuid) 
			VALUES `
		
		var values []string
		var args []interface{}
		
		for _, msg := range batch {
			placeholders := "(?, ?, ?, ?, ?, ?)"
			values = append(values, placeholders)
			
			var parentID interface{} = sql.NullString{}
			if msg.ParentUUID != nil {
				parentID = *msg.ParentUUID
			}
			
			args = append(args, msg.ID, msg.SessionID, msg.Role, msg.Content,
				msg.Timestamp, parentID)
		}
		
		query += strings.Join(values, ", ")
		
		result, err := tx.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("failed to insert message batch %d: %w", i/batchSize, err)
		}
		
		rowsAffected, _ := result.RowsAffected()
		bo.logger.WithFields(logrus.Fields{
			"batch_num": i/batchSize,
			"rows_affected": rowsAffected,
		}).Debug("Incremental message batch processed")
	}
	
	return nil
}

// batchInsertTokenUsageIncremental inserts new token usage using INSERT OR IGNORE
func (bo *BatchOperations) batchInsertTokenUsageIncremental(tx *sqlx.Tx, tokenUsages []TokenUsage) error {
	if len(tokenUsages) == 0 {
		return nil
	}

	// SQLite has a limit of 999 parameters, so batch the inserts
	const batchSize = 100 // 100 records × 8 params = 800 params (safe under 999 limit)
	
	for i := 0; i < len(tokenUsages); i += batchSize {
		end := i + batchSize
		if end > len(tokenUsages) {
			end = len(tokenUsages)
		}
		batch := tokenUsages[i:end]
		
		query := `
			INSERT OR IGNORE INTO token_usage (message_id, session_id, input_tokens, output_tokens, 
				cache_creation_input_tokens, cache_read_input_tokens, total_tokens, estimated_cost) 
			VALUES `
		
		var values []string
		var args []interface{}
		
		for _, tu := range batch {
			placeholders := "(?, ?, ?, ?, ?, ?, ?, ?)"
			values = append(values, placeholders)
			args = append(args, tu.MessageID, tu.SessionID, tu.InputTokens, tu.OutputTokens,
				tu.CacheCreationInputTokens, tu.CacheReadInputTokens, tu.TotalTokens, tu.EstimatedCost)
		}
		
		query += strings.Join(values, ", ")

		if _, err := tx.Exec(query, args...); err != nil {
			return fmt.Errorf("failed to insert token usage batch %d: %w", i/batchSize, err)
		}
	}
	
	return nil
}

// batchInsertToolResultsIncremental inserts new tool results using INSERT OR IGNORE
func (bo *BatchOperations) batchInsertToolResultsIncremental(tx *sqlx.Tx, toolResults []ToolResult) error {
	if len(toolResults) == 0 {
		return nil
	}

	query := `
		INSERT OR IGNORE INTO tool_results (message_id, session_id, tool_name, result_data, 
			file_path, timestamp) 
		VALUES `
	
	var values []string
	var args []interface{}
	
	for _, tr := range toolResults {
		placeholders := "(?, ?, ?, ?, ?, ?)"
		values = append(values, placeholders)
		
		var filePath interface{} = sql.NullString{}
		if tr.FilePath != nil {
			filePath = *tr.FilePath
		}
		
		args = append(args, tr.MessageID, tr.SessionID, tr.ToolName,
			tr.ResultData, filePath, tr.Timestamp)
	}
	
	query += strings.Join(values, ", ")

	_, err := tx.Exec(query, args...)
	return err
}