package database

import (
	"encoding/json"
	"time"

	"github.com/ksred/claude-session-manager/internal/claude"
)

// APIAdapter converts database models to API response models
type APIAdapter struct {
	repo *SessionRepository
}

// NewAPIAdapter creates a new API adapter
func NewAPIAdapter(repo *SessionRepository) *APIAdapter {
	return &APIAdapter{
		repo: repo,
	}
}

// SessionSummaryToSessionResponse converts a SessionSummary to API SessionResponse
func (a *APIAdapter) SessionSummaryToSessionResponse(summary *SessionSummary) (*SessionResponse, error) {
	// Parse files modified
	filesModified := []string{}
	if summary.FilesModified != "" {
		if err := json.Unmarshal([]byte(summary.FilesModified), &filesModified); err != nil {
			filesModified = []string{}
		}
	}

	// Convert token usage
	tokenUsage := claude.TokenUsage{
		InputTokens:              summary.TotalInputTokens,
		OutputTokens:             summary.TotalOutputTokens,
		CacheCreationInputTokens: summary.TotalCacheCreationTokens,
		CacheReadInputTokens:     summary.TotalCacheReadTokens,
		TotalTokens:              summary.TotalTokens,
		EstimatedCost:            summary.TotalEstimatedCost,
	}

	return &SessionResponse{
		ID:            summary.ID,
		Title:         summary.ProjectName,
		ProjectPath:   summary.ProjectPath,
		ProjectName:   summary.ProjectName,
		GitBranch:     "", // Not stored in database yet
		GitWorktree:   "", // Not stored in database yet
		Status:        summary.Status,
		CreatedAt:     summary.StartTime,
		UpdatedAt:     summary.LastActivity,
		MessageCount:  summary.MessageCount,
		CurrentTask:   summary.ProjectName,
		TokensUsed:    tokenUsage,
		FilesModified: filesModified,
		Duration:      summary.DurationSeconds,
		IsActive:      summary.IsActive,
		Model:         summary.Model,
	}, nil
}

// TokenUsageAggregateToClaudeTokenUsage converts database aggregate to claude.TokenUsage
func (a *APIAdapter) TokenUsageAggregateToClaudeTokenUsage(agg *TokenUsageAggregate) claude.TokenUsage {
	return claude.TokenUsage{
		InputTokens:              agg.InputTokens,
		OutputTokens:             agg.OutputTokens,
		CacheCreationInputTokens: agg.CacheCreationInputTokens,
		CacheReadInputTokens:     agg.CacheReadInputTokens,
		TotalTokens:              agg.TotalTokens,
		EstimatedCost:            agg.EstimatedCost,
	}
}

// ActivityLogEntryToAPIActivityEntry converts database activity to API format
func (a *APIAdapter) ActivityLogEntryToAPIActivityEntry(entry *ActivityLogEntry) ActivityEntry {
	sessionID := ""
	sessionName := ""
	if entry.SessionID != nil {
		sessionID = *entry.SessionID
		// Try to get the project name from the session
		session, err := a.repo.GetSessionByID(sessionID)
		if err == nil && session != nil {
			sessionName = session.ProjectName
		} else {
			sessionName = sessionID
		}
	}

	return ActivityEntry{
		Timestamp:   entry.Timestamp,
		Type:        entry.ActivityType,
		SessionID:   sessionID,
		SessionName: sessionName,
		Details:     entry.Details,
	}
}

// SessionResponse represents the API response format (matching existing API)
type SessionResponse struct {
	ID            string            `json:"id"`
	Title         string            `json:"title"`
	ProjectPath   string            `json:"project_path"`
	ProjectName   string            `json:"project_name"`
	GitBranch     string            `json:"git_branch,omitempty"`
	GitWorktree   string            `json:"git_worktree,omitempty"`
	Status        string            `json:"status"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	MessageCount  int               `json:"message_count"`
	CurrentTask   string            `json:"current_task"`
	TokensUsed    claude.TokenUsage `json:"tokens_used"`
	FilesModified []string          `json:"files_modified"`
	Duration      int64             `json:"duration_seconds"`
	IsActive      bool              `json:"is_active"`
	Model         string            `json:"model"`
}

// ActivityEntry represents an activity entry for the API
type ActivityEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	SessionID   string    `json:"session_id"`
	SessionName string    `json:"session_name"`
	Details     string    `json:"details"`
}