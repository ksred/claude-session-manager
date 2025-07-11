package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// BatchOperations provides optimized batch database operations
type BatchOperations struct {
	db *Database
}

// NewBatchOperations creates a new batch operations handler
func NewBatchOperations(db *Database) *BatchOperations {
	return &BatchOperations{db: db}
}

// BatchImportData imports multiple sessions, messages, token usage, and tool results in a single transaction
func (bo *BatchOperations) BatchImportData(sessions []Session, messages []Message, tokenUsages []TokenUsage, toolResults []ToolResult) error {
	tx, err := bo.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Batch insert sessions
	if len(sessions) > 0 {
		if err := bo.batchUpsertSessions(tx, sessions); err != nil {
			return fmt.Errorf("failed to batch upsert sessions: %w", err)
		}
	}

	// Batch insert messages
	if len(messages) > 0 {
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

	return tx.Commit()
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
	
	for i, session := range sessions {
		placeholders := fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			i*13+1, i*13+2, i*13+3, i*13+4, i*13+5, i*13+6, i*13+7, i*13+8, i*13+9, i*13+10, i*13+11, i*13+12, i*13+13)
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

	query := `
		INSERT OR REPLACE INTO messages (id, session_id, role, content, timestamp, parent_uuid) 
		VALUES `
	
	var values []string
	var args []interface{}
	
	for i, msg := range messages {
		placeholders := fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)",
			i*6+1, i*6+2, i*6+3, i*6+4, i*6+5, i*6+6)
		values = append(values, placeholders)
		
		var parentID interface{} = sql.NullString{}
		if msg.ParentUUID != nil {
			parentID = *msg.ParentUUID
		}
		
		args = append(args, msg.ID, msg.SessionID, msg.Role, msg.Content,
			msg.Timestamp, parentID)
	}
	
	query += strings.Join(values, ", ")

	_, err := tx.Exec(query, args...)
	return err
}

func (bo *BatchOperations) batchUpsertTokenUsages(tx *sqlx.Tx, tokenUsages []TokenUsage) error {
	if len(tokenUsages) == 0 {
		return nil
	}

	query := `
		INSERT OR REPLACE INTO token_usage (message_id, session_id, input_tokens, output_tokens, 
			cache_creation_input_tokens, cache_read_input_tokens, estimated_cost) 
		VALUES `
	
	var values []string
	var args []interface{}
	
	for i, tu := range tokenUsages {
		placeholders := fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			i*7+1, i*7+2, i*7+3, i*7+4, i*7+5, i*7+6, i*7+7)
		values = append(values, placeholders)
		args = append(args, tu.MessageID, tu.SessionID, tu.InputTokens, tu.OutputTokens,
			tu.CacheCreationInputTokens, tu.CacheReadInputTokens, tu.EstimatedCost)
	}
	
	query += strings.Join(values, ", ")

	_, err := tx.Exec(query, args...)
	return err
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
	
	for i, tr := range toolResults {
		placeholders := fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)",
			i*6+1, i*6+2, i*6+3, i*6+4, i*6+5, i*6+6)
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