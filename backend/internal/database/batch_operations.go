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
		INSERT INTO sessions (id, project_name, project_path, git_branch, start_time) 
		VALUES `
	
	var values []string
	var args []interface{}
	
	for i, session := range sessions {
		placeholders := fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)",
			i*5+1, i*5+2, i*5+3, i*5+4, i*5+5)
		values = append(values, placeholders)
		args = append(args, session.ID, session.ProjectName, session.ProjectPath, 
			session.GitBranch, session.StartTime)
	}
	
	query += strings.Join(values, ", ") + `
		ON CONFLICT(id) DO UPDATE SET
			project_name = EXCLUDED.project_name,
			project_path = EXCLUDED.project_path,
			git_branch = EXCLUDED.git_branch,
			start_time = EXCLUDED.start_time`

	_, err := tx.Exec(query, args...)
	return err
}

func (bo *BatchOperations) batchUpsertMessages(tx *sqlx.Tx, messages []Message) error {
	if len(messages) == 0 {
		return nil
	}

	query := `
		INSERT INTO messages (id, session_id, role, content, timestamp, parent_uuid) 
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
	
	query += strings.Join(values, ", ") + `
		ON CONFLICT(id) DO UPDATE SET
			session_id = EXCLUDED.session_id,
			role = EXCLUDED.role,
			content = EXCLUDED.content,
			timestamp = EXCLUDED.timestamp,
			parent_uuid = EXCLUDED.parent_uuid`

	_, err := tx.Exec(query, args...)
	return err
}

func (bo *BatchOperations) batchUpsertTokenUsages(tx *sqlx.Tx, tokenUsages []TokenUsage) error {
	if len(tokenUsages) == 0 {
		return nil
	}

	query := `
		INSERT INTO token_usage (message_id, session_id, input_tokens, output_tokens, 
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
	
	query += strings.Join(values, ", ") + `
		ON CONFLICT(message_id) DO UPDATE SET
			session_id = EXCLUDED.session_id,
			input_tokens = EXCLUDED.input_tokens,
			output_tokens = EXCLUDED.output_tokens,
			cache_creation_input_tokens = EXCLUDED.cache_creation_input_tokens,
			cache_read_input_tokens = EXCLUDED.cache_read_input_tokens,
			estimated_cost = EXCLUDED.estimated_cost`

	_, err := tx.Exec(query, args...)
	return err
}

func (bo *BatchOperations) batchUpsertToolResults(tx *sqlx.Tx, toolResults []ToolResult) error {
	if len(toolResults) == 0 {
		return nil
	}

	query := `
		INSERT INTO tool_results (message_id, session_id, tool_name, result_data, 
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

// ExecuteInReadTransaction executes a function within a read-only transaction
func (bo *BatchOperations) ExecuteInReadTransaction(fn func(*sqlx.Tx) error) error {
	tx, err := bo.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin read transaction: %w", err)
	}
	defer tx.Rollback()

	// Set transaction to read-only mode
	if _, err := tx.Exec("PRAGMA query_only = ON"); err != nil {
		return fmt.Errorf("failed to set read-only mode: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}