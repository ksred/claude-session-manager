package api

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ksred/claude-session-manager/internal/database"
	"github.com/sirupsen/logrus"
)

// SQLiteHandlers contains handlers that use the SQLite database
type SQLiteHandlers struct {
	repo          *database.SessionRepository
	readOptimized *database.ReadOptimizedRepository
	adapter       *database.APIAdapter
	logger        *logrus.Logger
}

// NewSQLiteHandlers creates new SQLite-based handlers
func NewSQLiteHandlers(repo *database.SessionRepository, logger *logrus.Logger) *SQLiteHandlers {
	return &SQLiteHandlers{
		repo:          repo,
		readOptimized: database.NewReadOptimizedRepository(repo.GetDB()),
		adapter:       database.NewAPIAdapter(repo),
		logger:        logger,
	}
}

// GetSessionsHandler returns all sessions
func (h *SQLiteHandlers) GetSessionsHandler(c *gin.Context) {
	sessions, err := h.readOptimized.GetAllSessionsOptimized()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get sessions from database")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve sessions",
		})
		return
	}

	// Convert to API response format
	responses := make([]database.SessionResponse, len(sessions))
	for i, session := range sessions {
		response, err := h.adapter.SessionSummaryToSessionResponse(session)
		if err != nil {
			h.logger.WithError(err).Error("Failed to convert session to response")
			continue
		}
		responses[i] = *response
	}

	// Sort by last activity (most recent first)
	sort.Slice(responses, func(i, j int) bool {
		return responses[i].UpdatedAt.After(responses[j].UpdatedAt)
	})

	c.JSON(http.StatusOK, gin.H{
		"sessions": responses,
		"total":    len(responses),
	})
}

// GetSessionHandler returns a specific session by ID
func (h *SQLiteHandlers) GetSessionHandler(c *gin.Context) {
	sessionID := c.Param("id")

	session, err := h.repo.GetSessionByID(sessionID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get session from database")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Session not found",
		})
		return
	}

	response, err := h.adapter.SessionSummaryToSessionResponse(session)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert session to response")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process session",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetActiveSessionsHandler returns currently active sessions
func (h *SQLiteHandlers) GetActiveSessionsHandler(c *gin.Context) {
	sessions, err := h.readOptimized.GetActiveSessionsOptimized()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get active sessions from database")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve sessions",
		})
		return
	}

	// Convert to API response format
	responses := make([]database.SessionResponse, len(sessions))
	for i, session := range sessions {
		response, err := h.adapter.SessionSummaryToSessionResponse(session)
		if err != nil {
			h.logger.WithError(err).Error("Failed to convert session to response")
			continue
		}
		responses[i] = *response
	}

	// Sort by last activity (most recent first)
	sort.Slice(responses, func(i, j int) bool {
		return responses[i].UpdatedAt.After(responses[j].UpdatedAt)
	})

	c.JSON(http.StatusOK, gin.H{
		"sessions": responses,
		"total":    len(responses),
	})
}

// GetRecentSessionsHandler returns recent sessions
func (h *SQLiteHandlers) GetRecentSessionsHandler(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	sessions, err := h.repo.GetRecentSessions(limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get recent sessions from database")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve sessions",
		})
		return
	}

	// Convert to API response format
	responses := make([]database.SessionResponse, len(sessions))
	for i, session := range sessions {
		response, err := h.adapter.SessionSummaryToSessionResponse(session)
		if err != nil {
			h.logger.WithError(err).Error("Failed to convert session to response")
			continue
		}
		responses[i] = *response
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": responses,
		"limit":    limit,
	})
}

// GetMetricsSummaryHandler returns overall metrics summary
func (h *SQLiteHandlers) GetMetricsSummaryHandler(c *gin.Context) {
	// Get total sessions
	totalSessions, err := h.repo.GetTotalSessions()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get total sessions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	// Get active sessions
	activeSessions, err := h.repo.GetActiveSessionsCount()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get active sessions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	// Get total messages
	totalMessages, err := h.repo.GetTotalMessages()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get total messages")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	// Get overall token usage
	tokenUsage, err := h.repo.GetOverallTokenUsage()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get token usage")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	// Get estimated cost
	totalCost, err := h.repo.GetEstimatedCost()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get estimated cost")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	// Get average session duration
	avgDuration, err := h.repo.GetAverageSessionDuration()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get average duration")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	// Get most used model
	mostUsedModel, err := h.repo.GetMostUsedModel()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get most used model")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	// Get model usage
	modelUsage, err := h.repo.GetModelUsage()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get model usage")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve metrics",
		})
		return
	}

	summary := MetricsSummary{
		TotalSessions:          totalSessions,
		ActiveSessions:         activeSessions,
		TotalMessages:          totalMessages,
		TotalTokensUsed:        tokenUsage.TotalTokens,
		TotalEstimatedCost:     totalCost,
		AverageSessionDuration: avgDuration,
		MostUsedModel:          mostUsedModel,
		ModelUsage:             modelUsage,
	}

	c.JSON(http.StatusOK, summary)
}

// GetActivityHandler returns activity timeline data
func (h *SQLiteHandlers) GetActivityHandler(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	activities, err := h.readOptimized.GetRecentActivityOptimized(limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get recent activity from database")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve activity",
		})
		return
	}

	// Convert to API response format
	apiActivities := make([]database.ActivityEntry, len(activities))
	for i, activity := range activities {
		apiActivities[i] = h.adapter.ActivityLogEntryToAPIActivityEntry(activity)
	}

	c.JSON(http.StatusOK, gin.H{
		"activity": apiActivities,
		"total":    len(apiActivities),
	})
}

// GetSessionActivityHandler returns activity for a specific session
func (h *SQLiteHandlers) GetSessionActivityHandler(c *gin.Context) {
	sessionID := c.Param("id")

	limitStr := c.DefaultQuery("limit", "50")
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	activities, err := h.readOptimized.GetSessionActivityOptimized(sessionID, limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get session activity")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve session activity",
		})
		return
	}

	// Convert to API model
	apiActivities := make([]database.ActivityEntry, len(activities))
	for i, activity := range activities {
		apiActivities[i] = h.adapter.ActivityLogEntryToAPIActivityEntry(activity)
	}

	c.JSON(http.StatusOK, gin.H{
		"activity": apiActivities,
		"total":    len(apiActivities),
	})
}

// GetProjectActivityHandler returns activity for a specific project
func (h *SQLiteHandlers) GetProjectActivityHandler(c *gin.Context) {
	projectName := c.Param("projectName")

	limitStr := c.DefaultQuery("limit", "50")
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	activities, err := h.repo.GetProjectActivity(projectName, limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get project activity")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve project activity",
		})
		return
	}

	// Convert to API model
	apiActivities := make([]database.ActivityEntry, len(activities))
	for i, activity := range activities {
		apiActivities[i] = h.adapter.ActivityLogEntryToAPIActivityEntry(activity)
	}

	c.JSON(http.StatusOK, gin.H{
		"activity": apiActivities,
		"total":    len(apiActivities),
	})
}

// GetUsageStatsHandler returns usage statistics
func (h *SQLiteHandlers) GetUsageStatsHandler(c *gin.Context) {
	// Get daily metrics for the last 7 days
	dailyMetrics, err := h.repo.GetDailyMetrics(7)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get daily metrics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve usage stats",
		})
		return
	}

	// Get model usage
	modelUsage, err := h.repo.GetModelUsage()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get model usage")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve usage stats",
		})
		return
	}

	// Get peak hours
	peakHours, err := h.repo.GetPeakHours()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get peak hours")
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

// SearchHandler handles search queries across sessions
func (h *SQLiteHandlers) SearchHandler(c *gin.Context) {
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

	sessions, err := h.repo.SearchSessions(query)
	if err != nil {
		h.logger.WithError(err).Error("Failed to search sessions in database")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to search sessions",
		})
		return
	}

	// Convert to API response format
	results := make([]database.SessionResponse, len(sessions))
	for i, session := range sessions {
		response, err := h.adapter.SessionSummaryToSessionResponse(session)
		if err != nil {
			h.logger.WithError(err).Error("Failed to convert session to response")
			continue
		}
		results[i] = *response
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

// GetRecentFilesHandler returns recently modified files across all sessions
// @Summary Get recently modified files
// @Description Retrieve a list of files that were recently modified across all Claude sessions
// @Tags Files
// @Accept json
// @Produce json
// @Param limit query int false "Maximum number of files to return (default: 20, max: 100)"
// @Param offset query int false "Number of files to skip for pagination (default: 0)"
// @Success 200 {object} RecentFilesResponse "Successfully retrieved recent files"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /files/recent [get]
func (h *SQLiteHandlers) GetRecentFilesHandler(c *gin.Context) {
	// Parse query parameters
	limit := 20
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Get recent files from repository
	files, total, err := h.repo.GetRecentFiles(limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get recent files")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve recent files",
		})
		return
	}

	// Convert to API response format
	var apiFiles []gin.H
	for _, file := range files {
		apiFile := gin.H{
			"file_path":     file.FilePath,
			"last_modified": file.LastModified,
			"session_id":    file.SessionID,
			"session_title": file.SessionTitle,
			"project_name":  file.ProjectName,
			"project_path":  file.ProjectPath,
			"tool_name":     file.ToolName,
			"occurrences":   file.Occurrences,
		}

		if file.GitBranch != nil {
			apiFile["git_branch"] = *file.GitBranch
		}

		apiFiles = append(apiFiles, apiFile)
	}

	c.JSON(http.StatusOK, gin.H{
		"files":  apiFiles,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetProjectRecentFilesHandler returns recently modified files for a specific project
// @Summary Get project recent files
// @Description Retrieve files that were recently modified within a specific project
// @Tags Projects
// @Accept json
// @Produce json
// @Param projectName path string true "Name of the project"
// @Param limit query int false "Maximum number of files to return (default: 20, max: 100)"
// @Param branch query string false "Filter by git branch name"
// @Success 200 {object} ProjectRecentFilesResponse "Successfully retrieved project recent files"
// @Failure 400 {object} ErrorResponse "Bad request - missing project name"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /projects/{projectName}/files/recent [get]
func (h *SQLiteHandlers) GetProjectRecentFilesHandler(c *gin.Context) {
	projectName := c.Param("projectName")
	if projectName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project name is required",
		})
		return
	}

	// Parse query parameters
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	var branch *string
	if b := c.Query("branch"); b != "" {
		branch = &b
	}

	// Get project recent files from repository
	files, err := h.repo.GetProjectRecentFiles(projectName, limit, branch)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get project recent files")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve project recent files",
		})
		return
	}

	// Convert to API response format
	var apiFiles []gin.H
	for _, file := range files {
		// Parse tools used
		toolsList := []string{}
		if file.ToolsUsed != "" {
			toolsList = strings.Split(file.ToolsUsed, ",")
		}

		apiFile := gin.H{
			"file_path":           file.FilePath,
			"last_modified":       file.LastModified,
			"sessions":            file.Sessions,
			"tools_used":          toolsList,
			"total_modifications": file.TotalModifications,
		}

		apiFiles = append(apiFiles, apiFile)
	}

	c.JSON(http.StatusOK, gin.H{
		"project_name": projectName,
		"files":        apiFiles,
		"total":        len(apiFiles),
	})
}

// GetTokenTimelineHandler returns overall token usage timeline
// @Summary Get token usage timeline
// @Description Retrieve token usage over time with configurable granularity
// @Tags Analytics
// @Accept json
// @Produce json
// @Param hours query int false "Number of hours to look back (default: 24, max: 720)"
// @Param granularity query string false "Time granularity: minute, hour, day (default: hour)"
// @Success 200 {object} TokenTimelineResponse "Successfully retrieved token timeline"
// @Failure 400 {object} ErrorResponse "Invalid query parameters"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /analytics/tokens/timeline [get]
func (h *SQLiteHandlers) GetTokenTimelineHandler(c *gin.Context) {
	// Parse query parameters
	hours := 24
	if hoursStr := c.Query("hours"); hoursStr != "" {
		if parsed, err := strconv.Atoi(hoursStr); err == nil && parsed > 0 && parsed <= 720 {
			hours = parsed
		}
	}

	granularity := c.DefaultQuery("granularity", "hour")
	if granularity != "minute" && granularity != "hour" && granularity != "day" {
		granularity = "hour"
	}

	timeline, err := h.readOptimized.GetTokenTimelineOptimized(hours, granularity)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get token timeline")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve token timeline",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"timeline":    timeline,
		"hours":       hours,
		"granularity": granularity,
		"total":       len(timeline),
	})
}

// GetSessionTokenTimelineHandler returns token usage timeline for a specific session
// @Summary Get session token timeline
// @Description Retrieve token usage over time for a specific session
// @Tags Sessions
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Param hours query int false "Number of hours to look back (default: 168)"
// @Param granularity query string false "Time granularity: minute, hour, day (default: minute)"
// @Success 200 {object} TokenTimelineResponse "Successfully retrieved session token timeline"
// @Failure 400 {object} ErrorResponse "Invalid parameters"
// @Failure 404 {object} ErrorResponse "Session not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /sessions/{id}/tokens/timeline [get]
func (h *SQLiteHandlers) GetSessionTokenTimelineHandler(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Session ID is required",
		})
		return
	}

	// Parse hours parameter with default of 168 (1 week)
	hours := 168
	if hoursStr := c.Query("hours"); hoursStr != "" {
		if parsedHours, err := strconv.Atoi(hoursStr); err == nil && parsedHours > 0 {
			hours = parsedHours
		}
	}

	granularity := c.DefaultQuery("granularity", "minute")
	if granularity != "minute" && granularity != "hour" && granularity != "day" {
		granularity = "minute"
	}

	// Log the request parameters
	h.logger.WithFields(logrus.Fields{
		"session_id":  sessionID,
		"hours":       hours,
		"granularity": granularity,
	}).Debug("Getting session token timeline")

	timeline, err := h.readOptimized.GetSessionTokenTimelineOptimized(sessionID, hours, granularity)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get session token timeline")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve session token timeline",
		})
		return
	}

	// Log the result count
	h.logger.WithFields(logrus.Fields{
		"session_id":     sessionID,
		"timeline_count": len(timeline),
		"hours":          hours,
		"granularity":    granularity,
	}).Debug("Retrieved session token timeline")

	// If no timeline data, check if session exists
	if len(timeline) == 0 {
		// Verify if the session actually exists
		_, err := h.readOptimized.GetSessionByIDOptimized(sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Session not found",
			})
			return
		}

		// Session exists but has no token usage data yet - return empty timeline
		c.JSON(http.StatusOK, gin.H{
			"session_id":  sessionID,
			"timeline":    []interface{}{},
			"granularity": granularity,
			"total":       0,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":  sessionID,
		"timeline":    timeline,
		"granularity": granularity,
		"total":       len(timeline),
	})
}

// GetProjectTokenTimelineHandler returns token usage timeline for a specific project
// @Summary Get project token timeline
// @Description Retrieve token usage over time for a specific project
// @Tags Projects
// @Accept json
// @Produce json
// @Param projectName path string true "Name of the project"
// @Param hours query int false "Number of hours to look back (default: 168/7 days, max: 720)"
// @Param granularity query string false "Time granularity: minute, hour, day (default: hour)"
// @Success 200 {object} TokenTimelineResponse "Successfully retrieved project token timeline"
// @Failure 400 {object} ErrorResponse "Invalid parameters"
// @Failure 404 {object} ErrorResponse "Project not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /projects/{projectName}/tokens/timeline [get]
func (h *SQLiteHandlers) GetProjectTokenTimelineHandler(c *gin.Context) {
	projectName := c.Param("projectName")
	if projectName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Project name is required",
		})
		return
	}

	// Parse query parameters
	hours := 168 // Default to 7 days for project view
	if hoursStr := c.Query("hours"); hoursStr != "" {
		if parsed, err := strconv.Atoi(hoursStr); err == nil && parsed > 0 && parsed <= 720 {
			hours = parsed
		}
	}

	granularity := c.DefaultQuery("granularity", "hour")
	if granularity != "minute" && granularity != "hour" && granularity != "day" {
		granularity = "hour"
	}

	timeline, err := h.repo.GetProjectTokenTimeline(projectName, hours, granularity)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get project token timeline")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve project token timeline",
		})
		return
	}

	if len(timeline) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Project not found or has no token usage",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"project_name": projectName,
		"timeline":     timeline,
		"hours":        hours,
		"granularity":  granularity,
		"total":        len(timeline),
	})
}

// CreateSessionHandler creates a new UI-initiated session
func (h *SQLiteHandlers) CreateSessionHandler(c *gin.Context) {
	var req struct {
		ProjectPath string `json:"project_path" binding:"required"`
		ProjectName string `json:"project_name" binding:"required"`
		Model       string `json:"model"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Set default model if not provided
	if req.Model == "" {
		req.Model = "claude-opus-4-20250514"
	}

	// Create the session
	session, err := h.repo.CreateUISession(req.ProjectPath, req.ProjectName, req.Model)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create UI session")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create session",
		})
		return
	}

	// Convert to API response
	response, err := h.adapter.SessionToSessionResponse(session)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert session to response")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to format session response",
		})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetChatMessagesHandler returns chat messages for a session
func (h *SQLiteHandlers) GetChatMessagesHandler(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Session ID is required",
		})
		return
	}

	// Parse query parameters
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Get chat messages
	messages, err := h.repo.GetChatMessages(sessionID, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get chat messages")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve chat messages",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"messages":   messages,
		"limit":      limit,
		"offset":     offset,
		"total":      len(messages),
	})
}
