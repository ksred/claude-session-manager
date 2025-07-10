package api

import (
	"github.com/gin-gonic/gin"
	"github.com/ksred/claude-session-manager/internal/database"
	"github.com/sirupsen/logrus"
)

// WebSocketUpdateAdapter adapts database updates to WebSocket broadcasts
type WebSocketUpdateAdapter struct {
	wsHub   *WebSocketHub
	repo    *database.SessionRepository
	adapter *database.APIAdapter
	logger  *logrus.Logger
}

// NewWebSocketUpdateAdapter creates a new WebSocket update adapter
func NewWebSocketUpdateAdapter(wsHub *WebSocketHub, sessionRepo *database.SessionRepository, logger *logrus.Logger) *WebSocketUpdateAdapter {
	return &WebSocketUpdateAdapter{
		wsHub:   wsHub,
		repo:    sessionRepo,
		adapter: database.NewAPIAdapter(sessionRepo),
		logger:  logger,
	}
}

// OnSessionUpdate handles session update notifications
func (w *WebSocketUpdateAdapter) OnSessionUpdate(updateType string, sessionID string, session *database.Session) {
	if w.wsHub == nil {
		w.logger.Debug("WebSocket hub is nil, skipping session update broadcast")
		return
	}

	w.logger.WithFields(logrus.Fields{
		"type":         updateType,
		"session_id":   sessionID,
		"project_name": session.ProjectName,
		"is_active":    session.IsActive,
		"model":        session.Model,
	}).Info("WebSocket adapter received session update notification")

	// Get session summary with aggregated token data
	sessionSummary, err := w.repo.GetSessionByID(session.ID)
	if err != nil {
		w.logger.WithError(err).Error("Failed to get session summary for update")
		return
	}

	sessionResponse, err := w.adapter.SessionSummaryToSessionResponse(sessionSummary)
	if err != nil {
		w.logger.WithError(err).Error("Failed to convert session to response format")
		return
	}

	// Broadcast the update
	data := gin.H{
		"session_id": sessionID,
		"session":    sessionResponse,
	}

	w.logger.WithFields(logrus.Fields{
		"type":         updateType,
		"session_id":   sessionID,
		"project_name": sessionResponse.ProjectName,
	}).Info("Sending session update to WebSocket hub for broadcast")

	w.wsHub.BroadcastUpdate(updateType, data)
}

// OnActivityUpdate handles activity update notifications
func (w *WebSocketUpdateAdapter) OnActivityUpdate(activity *database.ActivityLogEntry) {
	if w.wsHub == nil {
		w.logger.Debug("WebSocket hub is nil, skipping activity update broadcast")
		return
	}

	sessionID := ""
	if activity.SessionID != nil {
		sessionID = *activity.SessionID
	}

	w.logger.WithFields(logrus.Fields{
		"type":         "activity_update",
		"activity_type": activity.ActivityType,
		"session_id":   sessionID,
		"details":      activity.Details,
	}).Info("WebSocket adapter received activity update notification")

	// Convert activity to API format
	activityEntry := w.adapter.ActivityLogEntryToAPIActivityEntry(activity)

	// Broadcast the update
	data := gin.H{
		"activity": activityEntry,
	}

	w.logger.WithFields(logrus.Fields{
		"type":         "activity_update",
		"activity_type": activity.ActivityType,
		"session_id":   sessionID,
	}).Info("Sending activity update to WebSocket hub for broadcast")

	w.wsHub.BroadcastUpdate("activity_update", data)
}

// OnMetricsUpdate handles metrics update notifications
func (w *WebSocketUpdateAdapter) OnMetricsUpdate(sessionID string, usage *database.TokenUsage) {
	if w.wsHub == nil {
		w.logger.Debug("WebSocket hub is nil, skipping metrics update broadcast")
		return
	}

	w.logger.WithFields(logrus.Fields{
		"type":           "metrics_update",
		"session_id":     sessionID,
		"input_tokens":   usage.InputTokens,
		"output_tokens":  usage.OutputTokens,
		"total_tokens":   usage.TotalTokens,
		"estimated_cost": usage.EstimatedCost,
	}).Info("WebSocket adapter received metrics update notification")

	// Convert to API format
	tokenUsage := database.TokenUsageAggregate{
		InputTokens:              usage.InputTokens,
		OutputTokens:             usage.OutputTokens,
		CacheCreationInputTokens: usage.CacheCreationInputTokens,
		CacheReadInputTokens:     usage.CacheReadInputTokens,
		TotalTokens:              usage.TotalTokens,
		EstimatedCost:            usage.EstimatedCost,
	}

	// Broadcast the update
	data := gin.H{
		"session_id": sessionID,
		"usage":      w.adapter.TokenUsageAggregateToClaudeTokenUsage(&tokenUsage),
	}

	w.logger.WithFields(logrus.Fields{
		"type":         "metrics_update",
		"session_id":   sessionID,
		"total_tokens": usage.TotalTokens,
	}).Info("Sending metrics update to WebSocket hub for broadcast")

	w.wsHub.BroadcastUpdate("metrics_update", data)
}