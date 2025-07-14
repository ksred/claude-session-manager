package chat

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/google/uuid"
)

// WriteOperationFunc defines a function that can perform write operations
type WriteOperationFunc func(func(*sqlx.Tx) error) error

// Repository handles database operations for chat sessions and messages
type Repository struct {
	db            *sqlx.DB
	writeOperation WriteOperationFunc
}

// NewRepository creates a new chat repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// NewRepositoryWithWriteOp creates a new chat repository with write operation serialization
func NewRepositoryWithWriteOp(db *sqlx.DB, writeOp WriteOperationFunc) *Repository {
	return &Repository{db: db, writeOperation: writeOp}
}

// CreateChatSession creates a new chat session in the database
func (r *Repository) CreateChatSession(sessionID string, processID int) (*ChatSession, error) {
	chatSession := &ChatSession{
		ID:           uuid.New().String(),
		SessionID:    sessionID,
		ProcessID:    processID,
		Status:       StatusActive,
		StartedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	query := `
		INSERT INTO chat_sessions (id, session_id, process_id, status, started_at, last_activity)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	
	_, err := r.db.Exec(query, chatSession.ID, chatSession.SessionID, chatSession.ProcessID,
		chatSession.Status, chatSession.StartedAt, chatSession.LastActivity)
	if err != nil {
		return nil, err
	}

	return chatSession, nil
}

// GetChatSession retrieves a chat session by ID
func (r *Repository) GetChatSession(id string) (*ChatSession, error) {
	var chatSession ChatSession
	query := `SELECT * FROM chat_sessions WHERE id = ?`
	
	err := r.db.Get(&chatSession, query, id)
	if err != nil {
		return nil, err
	}

	return &chatSession, nil
}

// GetChatSessionBySessionID retrieves a chat session by session ID
func (r *Repository) GetChatSessionBySessionID(sessionID string) (*ChatSession, error) {
	var chatSession ChatSession
	query := `SELECT * FROM chat_sessions WHERE session_id = ? AND status = ?`
	
	err := r.db.Get(&chatSession, query, sessionID, StatusActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &chatSession, nil
}

// UpdateChatSessionStatus updates the status of a chat session
func (r *Repository) UpdateChatSessionStatus(id string, status string) error {
	query := `UPDATE chat_sessions SET status = ?, last_activity = ? WHERE id = ?`
	_, err := r.db.Exec(query, status, time.Now(), id)
	return err
}

// UpdateChatSessionActivity updates the last activity timestamp
func (r *Repository) UpdateChatSessionActivity(id string) error {
	query := `UPDATE chat_sessions SET last_activity = ? WHERE id = ?`
	_, err := r.db.Exec(query, time.Now(), id)
	return err
}

// UpdateChatSessionClaudeID updates the Claude session ID for continuity
func (r *Repository) UpdateChatSessionClaudeID(id string, claudeSessionID string) error {
	query := `UPDATE chat_sessions SET claude_session_id = ?, last_activity = ? WHERE id = ?`
	_, err := r.db.Exec(query, claudeSessionID, time.Now(), id)
	return err
}

// GetActiveChatSessions retrieves all active chat sessions
func (r *Repository) GetActiveChatSessions() ([]*ChatSession, error) {
	var sessions []*ChatSession
	query := `SELECT * FROM chat_sessions WHERE status = ? ORDER BY last_activity DESC`
	
	err := r.db.Select(&sessions, query, StatusActive)
	if err != nil {
		return nil, err
	}

	return sessions, nil
}

// CreateChatMessage creates a new chat message
func (r *Repository) CreateChatMessage(chatSessionID, msgType, content string, metadata map[string]interface{}) (*ChatMessage, error) {
	message := &ChatMessage{
		ID:            uuid.New().String(),
		ChatSessionID: chatSessionID,
		Type:          msgType,
		Content:       content,
		Timestamp:     time.Now(),
		Metadata:      metadata,
	}

	var metadataJSON []byte
	var err error
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return nil, err
		}
	}

	query := `
		INSERT INTO chat_messages (id, chat_session_id, type, content, timestamp, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	
	_, err = r.db.Exec(query, message.ID, message.ChatSessionID, message.Type,
		message.Content, message.Timestamp, metadataJSON)
	if err != nil {
		return nil, err
	}

	return message, nil
}

// GetChatMessages retrieves messages for a chat session
func (r *Repository) GetChatMessages(chatSessionID string, limit int, offset int) ([]*ChatMessage, error) {
	var messages []*ChatMessage
	query := `
		SELECT id, chat_session_id, type, content, timestamp, metadata
		FROM chat_messages 
		WHERE chat_session_id = ? 
		ORDER BY timestamp ASC
		LIMIT ? OFFSET ?
	`
	
	rows, err := r.db.Query(query, chatSessionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var message ChatMessage
		var metadataJSON sql.NullString
		
		err := rows.Scan(&message.ID, &message.ChatSessionID, &message.Type,
			&message.Content, &message.Timestamp, &metadataJSON)
		if err != nil {
			return nil, err
		}

		if metadataJSON.Valid && metadataJSON.String != "" {
			err = json.Unmarshal([]byte(metadataJSON.String), &message.Metadata)
			if err != nil {
				// If unmarshaling fails, leave metadata as nil
				message.Metadata = nil
			}
		}

		messages = append(messages, &message)
	}

	return messages, nil
}

// DeleteChatSession deletes a chat session and all its messages
func (r *Repository) DeleteChatSession(id string) error {
	operation := func(tx *sqlx.Tx) error {
		// Delete messages first due to foreign key constraint
		_, err := tx.Exec("DELETE FROM chat_messages WHERE chat_session_id = ?", id)
		if err != nil {
			return err
		}

		// Delete the session
		_, err = tx.Exec("DELETE FROM chat_sessions WHERE id = ?", id)
		if err != nil {
			return err
		}

		return nil
	}

	// Use serialized write operation if available, otherwise use direct transaction
	if r.writeOperation != nil {
		return r.writeOperation(operation)
	}

	// Fallback to direct transaction (legacy behavior)
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := operation(tx); err != nil {
		return err
	}

	return tx.Commit()
}

// CleanupInactiveSessions removes chat sessions that have been inactive for a specified duration
func (r *Repository) CleanupInactiveSessions(inactiveDuration time.Duration) error {
	cutoffTime := time.Now().Add(-inactiveDuration)
	
	operation := func(tx *sqlx.Tx) error {
		// Get inactive session IDs
		var sessionIDs []string
		query := `SELECT id FROM chat_sessions WHERE last_activity < ? AND status = ?`
		err := tx.Select(&sessionIDs, query, cutoffTime, StatusActive)
		if err != nil {
			return err
		}

		// Delete messages for inactive sessions
		if len(sessionIDs) > 0 {
			query, args, err := sqlx.In("DELETE FROM chat_messages WHERE chat_session_id IN (?)", sessionIDs)
			if err != nil {
				return err
			}
			query = tx.Rebind(query)
			_, err = tx.Exec(query, args...)
			if err != nil {
				return err
			}

			// Update session status to inactive
			query, args, err = sqlx.In("UPDATE chat_sessions SET status = ? WHERE id IN (?)", StatusInactive, sessionIDs)
			if err != nil {
				return err
			}
			query = tx.Rebind(query)
			_, err = tx.Exec(query, args...)
			if err != nil {
				return err
			}
		}

		return nil
	}

	// Use serialized write operation if available, otherwise use direct transaction
	if r.writeOperation != nil {
		return r.writeOperation(operation)
	}

	// Fallback to direct transaction (legacy behavior)
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := operation(tx); err != nil {
		return err
	}

	return tx.Commit()
}