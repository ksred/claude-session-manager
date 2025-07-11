package chat

import (
	"time"
)

// ChatSession represents an active chat session with Claude CLI
type ChatSession struct {
	ID           string    `json:"id" db:"id"`
	SessionID    string    `json:"session_id" db:"session_id"`
	ProcessID    int       `json:"process_id" db:"process_id"`
	Status       string    `json:"status" db:"status"`
	StartedAt    time.Time `json:"started_at" db:"started_at"`
	LastActivity time.Time `json:"last_activity" db:"last_activity"`
}

// ChatMessage represents a message in a chat session
type ChatMessage struct {
	ID            string                 `json:"id" db:"id"`
	ChatSessionID string                 `json:"chat_session_id" db:"chat_session_id"`
	Type          string                 `json:"type" db:"type"` // user, claude, system
	Content       string                 `json:"content" db:"content"`
	Timestamp     time.Time              `json:"timestamp" db:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ChatSessionStatus represents the possible states of a chat session
const (
	StatusActive     = "active"
	StatusInactive   = "inactive"
	StatusTerminated = "terminated"
	StatusError      = "error"
)

// MessageType represents the type of chat message
const (
	MessageTypeUser   = "user"
	MessageTypeClaude = "claude"
	MessageTypeSystem = "system"
)

// WebSocketMessage represents a message sent over WebSocket for chat
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	SessionID string                 `json:"session_id"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// WebSocket message types for chat
const (
	WSMsgChatSessionStart = "chat:session:start"
	WSMsgChatSessionEnd   = "chat:session:end"
	WSMsgChatMessageSend  = "chat:message:send"
	WSMsgChatMessageRecv  = "chat:message:receive"
	WSMsgChatTypingStart  = "chat:typing:start"
	WSMsgChatTypingStop   = "chat:typing:stop"
	WSMsgChatError        = "chat:error"
)