package api

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ksred/claude-session-manager/internal/claude"
)

// This file contains HTTP handlers for the Claude Session Manager API.
// All handlers include proper error handling, input validation, and logging.
// Session data is retrieved from a cached copy that is automatically updated
// when files change on disk.


// Helper function to convert claude.Session to SessionResponse
func sessionToResponse(session claude.Session) SessionResponse {
	return SessionResponse{
		ID:            session.ID,
		Title:         session.CurrentTask,
		ProjectPath:   session.ProjectPath,
		ProjectName:   session.ProjectName,
		GitBranch:     session.GitBranch,
		GitWorktree:   session.GitWorktree,
		Status:        session.Status.String(),
		CreatedAt:     session.StartTime,
		UpdatedAt:     session.LastActivity,
		MessageCount:  session.GetMessageCount(),
		CurrentTask:   session.CurrentTask,
		TokensUsed:    session.TokensUsed,
		FilesModified: session.FilesModified,
		Duration:      int64(session.Duration().Seconds()),
		IsActive:      session.IsActive(),
		Model:         inferModelFromSession(session),
	}
}

// Helper function to convert repository session to API response format
func repoSessionToResponse(session claude.RepositorySession) SessionResponse {
	// Convert token usage to expected format
	tokenUsage := claude.TokenUsage{
		InputTokens:              session.TotalTokens.InputTokens,
		OutputTokens:             session.TotalTokens.OutputTokens,
		CacheCreationInputTokens: session.TotalTokens.CacheCreationInputTokens,
		CacheReadInputTokens:     session.TotalTokens.CacheReadInputTokens,
		TotalTokens:              session.TotalTokens.InputTokens + session.TotalTokens.OutputTokens + session.TotalTokens.CacheCreationInputTokens + session.TotalTokens.CacheReadInputTokens,
		EstimatedCost:            calculateCostFromTokens(session.TotalTokens),
	}

	return SessionResponse{
		ID:            session.ID,
		Title:         session.ProjectName, // Use project name as title
		ProjectPath:   session.ProjectPath,
		ProjectName:   session.ProjectName,
		GitBranch:     "", // Not available from JSONL data
		GitWorktree:   "", // Not available from JSONL data
		Status:        determineSessionStatus(session),
		CreatedAt:     session.StartTime,
		UpdatedAt:     session.LastActivity,
		MessageCount:  session.MessageCount,
		CurrentTask:   session.ProjectName, // Use project name as current task
		TokensUsed:    tokenUsage,
		FilesModified: session.FilesModified,
		Duration:      int64(session.Duration.Seconds()),
		IsActive:      session.IsActive,
		Model:         session.Model,
	}
}

// Helper to calculate cost from detailed token usage
func calculateCostFromTokens(tokens claude.RepositoryTokenUsage) float64 {
	const (
		inputCostPer1M              = 15.0
		outputCostPer1M             = 75.0
		cacheReadCostPer1M          = 1.50
		cacheCreationCostPer1M      = 18.75
	)

	cost := float64(tokens.InputTokens) * inputCostPer1M / 1000000
	cost += float64(tokens.OutputTokens) * outputCostPer1M / 1000000
	cost += float64(tokens.CacheReadInputTokens) * cacheReadCostPer1M / 1000000
	cost += float64(tokens.CacheCreationInputTokens) * cacheCreationCostPer1M / 1000000
	
	return cost
}

// Helper to determine session status from repository data
func determineSessionStatus(session claude.RepositorySession) string {
	if session.IsActive {
		return "working"
	}
	if len(session.Messages) > 0 {
		return "completed"
	}
	return "idle"
}

// Helper function to infer model from session messages
func inferModelFromSession(session claude.Session) string {
	// Look for model information in messages metadata - check multiple locations
	for i := len(session.Messages) - 1; i >= 0; i-- {
		msg := session.Messages[i]
		
		// Check if model is directly in Meta
		if model, ok := msg.Meta["model"].(string); ok {
			return model
		}
		
		// Check if model is in nested message object within Meta
		if msgData, ok := msg.Meta["message"].(map[string]interface{}); ok {
			if model, ok := msgData["model"].(string); ok {
				return model
			}
		}
		
		// Also check any other nested structures that might contain model info
		for _, value := range msg.Meta {
			if valueMap, ok := value.(map[string]interface{}); ok {
				if model, ok := valueMap["model"].(string); ok {
					return model
				}
			}
		}
	}
	return "claude-3-opus" // Default assumption
}

// Helper function to sort sessions by last activity (most recent first)
func sortSessionsByActivity(sessions []claude.Session) {
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastActivity.After(sessions[j].LastActivity)
	})
}

// Helper function to filter sessions by status
func filterSessionsByStatus(sessions []claude.Session, statuses ...claude.SessionStatus) []claude.Session {
	var filtered []claude.Session
	statusMap := make(map[claude.SessionStatus]bool)
	for _, status := range statuses {
		statusMap[status] = true
	}
	
	for _, session := range sessions {
		if statusMap[session.Status] {
			filtered = append(filtered, session)
		}
	}
	return filtered
}

// getSessionsHandler returns all sessions
// @Summary Get all sessions
// @Description Retrieve all Claude sessions with their metadata and statistics
// @Tags Sessions
// @Accept json
// @Produce json
// @Success 200 {object} SessionsResponse "Successfully retrieved sessions"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /sessions [get]
func (s *Server) getSessionsHandler(c *gin.Context) {
	sessions, err := s.sessionRepo.GetAllSessions()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get sessions from repository")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve sessions",
		})
		return
	}

	// Convert to response format
	responses := make([]SessionResponse, len(sessions))
	for i, session := range sessions {
		responses[i] = repoSessionToResponse(session)
	}

	// Sort by last activity
	sort.Slice(responses, func(i, j int) bool {
		return responses[i].UpdatedAt.After(responses[j].UpdatedAt)
	})

	c.JSON(http.StatusOK, gin.H{
		"sessions": responses,
		"total":    len(responses),
	})
}

// getSessionHandler returns a specific session by ID
// @Summary Get session by ID
// @Description Retrieve a specific Claude session by its unique identifier
// @Tags Sessions
// @Accept json
// @Produce json
// @Param id path string true "Session ID" example("session_123456")
// @Success 200 {object} SessionResponse "Session found"
// @Failure 404 {object} ErrorResponse "Session not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /sessions/{id} [get]
func (s *Server) getSessionHandler(c *gin.Context) {
	sessionID := c.Param("id")

	session, err := s.sessionRepo.GetSessionById(sessionID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get session from repository")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Session not found",
		})
		return
	}

	c.JSON(http.StatusOK, repoSessionToResponse(*session))
}

// getActiveSessionsHandler returns currently active sessions
// @Summary Get active sessions
// @Description Retrieve all currently active Claude sessions (working or idle status)
// @Tags Sessions
// @Accept json
// @Produce json
// @Success 200 {object} SessionsResponse "Successfully retrieved active sessions"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /sessions/active [get]
func (s *Server) getActiveSessionsHandler(c *gin.Context) {
	activeSessions, err := s.sessionRepo.GetActiveSessions()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get active sessions from repository")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve sessions",
		})
		return
	}

	// Convert to response format
	responses := make([]SessionResponse, len(activeSessions))
	for i, session := range activeSessions {
		responses[i] = repoSessionToResponse(session)
	}

	// Sort by last activity
	sort.Slice(responses, func(i, j int) bool {
		return responses[i].UpdatedAt.After(responses[j].UpdatedAt)
	})

	c.JSON(http.StatusOK, gin.H{
		"sessions": responses,
		"total":    len(responses),
	})
}

// getRecentSessionsHandler returns recent sessions
// @Summary Get recent sessions
// @Description Retrieve the most recent Claude sessions with optional limit
// @Tags Sessions
// @Accept json
// @Produce json
// @Param limit query int false "Maximum number of sessions to return" default(10) minimum(1) maximum(100)
// @Success 200 {object} SessionsLimitResponse "Successfully retrieved recent sessions"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /sessions/recent [get]
func (s *Server) getRecentSessionsHandler(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100 // Cap at 100 to prevent excessive response sizes
	}

	sessions, err := s.sessionRepo.GetRecentSessions(limit)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get recent sessions from repository")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve sessions",
		})
		return
	}

	// Convert to response format
	responses := make([]SessionResponse, len(sessions))
	for i, session := range sessions {
		responses[i] = repoSessionToResponse(session)
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": responses,
		"limit":    limit,
	})
}

// getMetricsSummaryHandler returns overall metrics summary
// @Summary Get metrics summary
// @Description Retrieve overall system metrics including session counts, token usage, and cost estimates
// @Tags Metrics
// @Accept json
// @Produce json
// @Success 200 {object} MetricsSummary "Successfully retrieved metrics summary"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /metrics/summary [get]
func (s *Server) getMetricsSummaryHandler(c *gin.Context) {
	// Get total sessions
	totalSessions, err := s.sessionRepo.GetTotalSessions()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get total sessions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	// Get active sessions
	activeSessions, err := s.sessionRepo.GetActiveSessionsCount()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get active sessions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	// Get total messages
	totalMessages, err := s.sessionRepo.GetTotalMessages()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get total messages")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	// Get overall token usage
	tokenUsage, err := s.sessionRepo.GetOverallTokenUsage()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get token usage")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	// Get estimated cost
	totalCost, err := s.sessionRepo.GetEstimatedCost()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get estimated cost")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	// Get average session duration
	avgDuration, err := s.sessionRepo.GetAverageSessionDuration()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get average duration")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	// Get most used model
	mostUsedModel, err := s.sessionRepo.GetMostUsedModel()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get most used model")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	// Get model usage
	modelUsage, err := s.sessionRepo.GetModelUsage()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get model usage")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	totalTokensUsed := tokenUsage.InputTokens + tokenUsage.OutputTokens + 
		tokenUsage.CacheCreationInputTokens + tokenUsage.CacheReadInputTokens

	summary := MetricsSummary{
		TotalSessions:          totalSessions,
		ActiveSessions:         activeSessions,
		TotalMessages:          totalMessages,
		TotalTokensUsed:        totalTokensUsed,
		TotalEstimatedCost:     totalCost,
		AverageSessionDuration: avgDuration,
		MostUsedModel:          mostUsedModel,
		ModelUsage:             modelUsage,
	}

	c.JSON(http.StatusOK, summary)
}

// getActivityHandler returns activity timeline data
// @Summary Get activity timeline
// @Description Retrieve recent activity timeline including session events and message activity
// @Tags Metrics
// @Accept json
// @Produce json
// @Param limit query int false "Maximum number of activities to return" default(50) minimum(1) maximum(500)
// @Success 200 {object} ActivityResponse "Successfully retrieved activity timeline"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /metrics/activity [get]
func (s *Server) getActivityHandler(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500 // Cap at 500 to prevent excessive response sizes
	}

	activities, err := s.sessionRepo.GetRecentActivity(limit)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get recent activity from repository")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve activity",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activity": activities,
		"total":    len(activities),
	})
}

// getUsageStatsHandler returns usage statistics
// @Summary Get usage statistics
// @Description Retrieve detailed usage statistics including daily sessions, model usage, and peak hours
// @Tags Metrics
// @Accept json
// @Produce json
// @Success 200 {object} UsageStats "Successfully retrieved usage statistics"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /metrics/usage [get]
func (s *Server) getUsageStatsHandler(c *gin.Context) {
	// Get daily metrics for the last 7 days
	dailyMetrics, err := s.sessionRepo.GetDailyMetrics(7)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get daily metrics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve usage stats",
		})
		return
	}

	// Get model usage
	modelUsage, err := s.sessionRepo.GetModelUsage()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get model usage")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve usage stats",
		})
		return
	}

	// Get peak hours
	peakHours, err := s.sessionRepo.GetPeakHours()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get peak hours")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve usage stats",
		})
		return
	}

	// Format daily sessions
	var dailySessionsList []gin.H
	for _, daily := range dailyMetrics {
		dailySessionsList = append(dailySessionsList, gin.H{
			"date":  daily.Date,
			"count": daily.SessionCount,
		})
	}

	stats := gin.H{
		"daily_sessions": dailySessionsList,
		"model_usage":    modelUsage,
		"peak_hours":     peakHours,
	}

	c.JSON(http.StatusOK, stats)
}

// searchHandler handles search queries across sessions
// @Summary Search sessions
// @Description Search across sessions by project name, task description, message content, or file paths
// @Tags Search
// @Accept json
// @Produce json
// @Param q query string true "Search query" minlength(1) maxlength(100) example("authentication")
// @Success 200 {object} SearchResponse "Successfully retrieved search results"
// @Failure 400 {object} ErrorResponse "Invalid search query"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /search [get]
func (s *Server) searchHandler(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Query parameter 'q' is required",
		})
		return
	}

	// Validate query length
	if len(query) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Query too long (max 100 characters)",
		})
		return
	}

	sessions, err := s.sessionRepo.SearchSessions(query)
	if err != nil {
		s.logger.WithError(err).Error("Failed to search sessions in repository")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to search sessions",
		})
		return
	}

	// Convert to response format
	results := make([]SessionResponse, len(sessions))
	for i, session := range sessions {
		results[i] = repoSessionToResponse(session)
	}

	// Sort results by relevance (most recent first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].UpdatedAt.After(results[j].UpdatedAt)
	})

	c.JSON(http.StatusOK, gin.H{
		"query":   query,
		"results": results,
		"total":   len(results),
	})
}

// getModelPerformanceHandler returns model performance comparison data
// @Summary Get model performance comparison
// @Description Retrieve performance statistics for each Claude model including token usage, costs, and cache efficiency
// @Tags Analytics
// @Accept json
// @Produce json
// @Success 200 {object} ModelPerformanceResponse "Successfully retrieved model performance data"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /analytics/models [get]
func (s *Server) getModelPerformanceHandler(c *gin.Context) {
	modelStats, err := s.sessionRepo.GetModelPerformanceStats()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get model performance stats")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve model performance data",
		})
		return
	}

	// Map model names to display names
	modelDisplayNames := map[string]string{
		"claude-3-opus-20240229":    "Claude 3 Opus",
		"claude-3-sonnet-20240229":  "Claude 3 Sonnet",
		"claude-3-haiku-20240307":   "Claude 3 Haiku",
		"claude-3-5-sonnet-20241022": "Claude 3.5 Sonnet",
		"claude-3-5-haiku-20241022":  "Claude 3.5 Haiku",
	}

	var models []ModelPerformanceEntry
	for model, stats := range modelStats {
		displayName := modelDisplayNames[model]
		if displayName == "" {
			displayName = model // Fallback to model ID if no display name
		}

		// Calculate total tokens and cost
		totalTokens := stats.TotalTokens.InputTokens + stats.TotalTokens.OutputTokens +
			stats.TotalTokens.CacheCreationInputTokens + stats.TotalTokens.CacheReadInputTokens
		totalCost := s.calculateCostFromTokens(stats.TotalTokens)

		// Calculate averages
		avgCost := 0.0
		if stats.TotalSessions > 0 {
			avgCost = totalCost / float64(stats.TotalSessions)
		}

		entry := ModelPerformanceEntry{
			Model:       model,
			DisplayName: displayName,
			Stats: ModelStats{
				TotalSessions:          stats.TotalSessions,
				TotalTokens:            totalTokens,
				TotalCost:              totalCost,
				AvgTokensPerSession:    stats.AvgTokensPerSession,
				AvgCostPerSession:      avgCost,
				CacheEfficiency:        stats.CacheEfficiency,
				AvgSessionDurationSecs: stats.AvgDurationSeconds,
			},
		}
		models = append(models, entry)
	}

	// Sort by total sessions (most used first)
	sort.Slice(models, func(i, j int) bool {
		return models[i].Stats.TotalSessions > models[j].Stats.TotalSessions
	})

	c.JSON(http.StatusOK, ModelPerformanceResponse{
		Models: models,
	})
}

// getTimeSeriesHandler returns time series analytics data
// @Summary Get time series analytics
// @Description Retrieve time series data showing sessions, messages, tokens, and costs over time
// @Tags Analytics
// @Accept json
// @Produce json
// @Param period query string false "Time period granularity" Enums(day, week, month) Default(day)
// @Param days query int false "Number of days to look back" Default(30)
// @Success 200 {object} TimeSeriesResponse "Successfully retrieved time series data"
// @Failure 400 {object} ErrorResponse "Invalid query parameters"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /analytics/timeseries [get]
func (s *Server) getTimeSeriesHandler(c *gin.Context) {
	// Parse query parameters
	period := c.DefaultQuery("period", "day")
	daysStr := c.DefaultQuery("days", "30")

	// Validate period
	if period != "day" && period != "week" && period != "month" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid period. Must be 'day', 'week', or 'month'",
		})
		return
	}

	// Parse days
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 || days > 365 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid days parameter. Must be between 1 and 365",
		})
		return
	}

	// Get time series data
	timeSeriesData, err := s.sessionRepo.GetTimeSeriesData(period, days)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get time series data")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve time series data",
		})
		return
	}

	// Convert to API response format
	var entries []TimeSeriesEntry
	for _, data := range timeSeriesData {
		// Calculate total cost for this period
		totalCost := 0.0
		models := make(map[string]TimeSeriesModelData)

		for model, usage := range data.Models {
			// Estimate cost based on token usage (simplified calculation)
			modelCost := float64(usage.Tokens) * 15.0 / 1000000 // Rough average cost
			totalCost += modelCost

			models[model] = TimeSeriesModelData{
				Sessions: usage.Sessions,
				Tokens:   usage.Tokens,
			}
		}

		entry := TimeSeriesEntry{
			Date:        data.Date,
			Sessions:    data.Sessions,
			Messages:    data.Messages,
			TotalTokens: data.TotalTokens,
			TotalCost:   totalCost,
			Models:      models,
		}
		entries = append(entries, entry)
	}

	c.JSON(http.StatusOK, TimeSeriesResponse{
		Period: period,
		Data:   entries,
	})
}

// getCostAnalyticsHandler returns cost analytics data
// @Summary Get cost analytics
// @Description Retrieve cost breakdown by project, model, or day with projections and cache savings
// @Tags Analytics
// @Accept json
// @Produce json
// @Param group_by query string false "Group costs by" Enums(project, model, day) Default(project)
// @Param days query int false "Number of days to analyze" Default(30)
// @Success 200 {object} CostAnalyticsResponse "Successfully retrieved cost analytics"
// @Failure 400 {object} ErrorResponse "Invalid query parameters"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /analytics/costs [get]
func (s *Server) getCostAnalyticsHandler(c *gin.Context) {
	// Parse query parameters
	groupBy := c.DefaultQuery("group_by", "project")
	daysStr := c.DefaultQuery("days", "30")

	// Validate groupBy
	if groupBy != "project" && groupBy != "model" && groupBy != "day" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid group_by parameter. Must be 'project', 'model', or 'day'",
		})
		return
	}

	// Parse days
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 || days > 365 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid days parameter. Must be between 1 and 365",
		})
		return
	}

	// Get cost analytics data
	costData, err := s.sessionRepo.GetCostAnalytics(groupBy, days)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get cost analytics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve cost analytics",
		})
		return
	}

	// Convert to API response format
	var breakdown []CostBreakdownEntry
	for _, data := range costData.Breakdown {
		entry := CostBreakdownEntry{
			Name:       data.Name,
			Cost:       data.Cost,
			Tokens:     TokenBreakdown{
				Total:  data.TotalTokens,
				Cached: data.CachedTokens,
				Fresh:  data.FreshTokens,
			},
			Sessions:   data.Sessions,
			Percentage: data.Percentage,
		}
		breakdown = append(breakdown, entry)
	}

	response := CostAnalyticsResponse{
		TotalCost:    costData.TotalCost,
		CacheSavings: costData.CacheSavings,
		Breakdown:    breakdown,
		Projection: CostProjection{
			DailyAverage:    costData.DailyAverage,
			MonthlyEstimate: costData.MonthlyEstimate,
		},
	}

	c.JSON(http.StatusOK, response)
}

// Helper method to calculate cost from detailed token usage
func (s *Server) calculateCostFromTokens(tokens claude.RepositoryTokenUsage) float64 {
	const (
		inputCostPer1M         = 15.0
		outputCostPer1M        = 75.0
		cacheReadCostPer1M     = 1.50
		cacheCreationCostPer1M = 18.75
	)

	cost := float64(tokens.InputTokens) * inputCostPer1M / 1000000
	cost += float64(tokens.OutputTokens) * outputCostPer1M / 1000000
	cost += float64(tokens.CacheReadInputTokens) * cacheReadCostPer1M / 1000000
	cost += float64(tokens.CacheCreationInputTokens) * cacheCreationCostPer1M / 1000000

	return cost
}