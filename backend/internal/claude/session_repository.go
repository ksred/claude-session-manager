package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SessionRepository provides access to session data from JSONL files
type SessionRepository struct {
	claudeDir string
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(claudeDir string) *SessionRepository {
	if claudeDir == "" {
		// Default to ~/.claude
		homeDir, _ := os.UserHomeDir()
		claudeDir = filepath.Join(homeDir, ".claude")
	}
	return &SessionRepository{
		claudeDir: claudeDir,
	}
}

// SessionMessage represents a message in the JSONL session data
type SessionMessage struct {
	ParentUUID    *string                `json:"parentUuid"`
	IsSidechain   bool                   `json:"isSidechain"`
	UserType      string                 `json:"userType"`
	CWD           string                 `json:"cwd"`
	SessionID     string                 `json:"sessionId"`
	Version       string                 `json:"version"`
	Type          string                 `json:"type"`
	Message       MessageContent         `json:"message"`
	UUID          string                 `json:"uuid"`
	Timestamp     time.Time              `json:"timestamp"`
	RequestID     *string                `json:"requestId,omitempty"`
	ToolUseResult map[string]interface{} `json:"toolUseResult,omitempty"`
}

// MessageContent represents the content of a message
type MessageContent struct {
	Role    string                 `json:"role"`
	Content interface{}            `json:"content"` // Can be string or array
	ID      *string                `json:"id,omitempty"`
	Model   *string                `json:"model,omitempty"`
	Usage   *RepositoryTokenUsage  `json:"usage,omitempty"`
	Other   map[string]interface{} `json:"-"` // For additional fields
}

// RepositoryTokenUsage represents detailed token usage from JSONL files
type RepositoryTokenUsage struct {
	InputTokens              int    `json:"input_tokens"`
	OutputTokens             int    `json:"output_tokens"`
	CacheCreationInputTokens int    `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int    `json:"cache_read_input_tokens"`
	ServiceTier              string `json:"service_tier"`
}

// ProjectSession represents aggregated session data for a project
type ProjectSession struct {
	ProjectPath   string
	ProjectName   string
	SessionCount  int
	TotalTokens   RepositoryTokenUsage
	FirstActivity time.Time
	LastActivity  time.Time
	Models        map[string]int
	ActiveSessions int
}

// DailyMetrics represents metrics for a specific day
type DailyMetrics struct {
	Date         string
	SessionCount int
	MessageCount int
	TotalTokens  RepositoryTokenUsage
	Models       map[string]int
}

// GetAllProjectSessions returns session data grouped by project
func (r *SessionRepository) GetAllProjectSessions() (map[string]*ProjectSession, error) {
	projectsDir := filepath.Join(r.claudeDir, "projects")
	
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	projects := make(map[string]*ProjectSession)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectPath := entry.Name()
		// Decode the project path (it's URL-encoded)
		decodedPath := strings.ReplaceAll(projectPath, "-", "/")
		
		// Extract project name from path
		parts := strings.Split(decodedPath, "/")
		projectName := parts[len(parts)-1]

		projectDir := filepath.Join(projectsDir, projectPath)
		sessionFiles, err := os.ReadDir(projectDir)
		if err != nil {
			continue
		}

		project := &ProjectSession{
			ProjectPath:  decodedPath,
			ProjectName:  projectName,
			Models:       make(map[string]int),
			TotalTokens:  RepositoryTokenUsage{},
		}

		for _, sessionFile := range sessionFiles {
			if !strings.HasSuffix(sessionFile.Name(), ".jsonl") {
				continue
			}

			sessionPath := filepath.Join(projectDir, sessionFile.Name())
			sessionData, err := r.readSessionFile(sessionPath)
			if err != nil {
				continue
			}

			project.SessionCount++
			
			// Track activity times
			for _, msg := range sessionData {
				if project.FirstActivity.IsZero() || msg.Timestamp.Before(project.FirstActivity) {
					project.FirstActivity = msg.Timestamp
				}
				if msg.Timestamp.After(project.LastActivity) {
					project.LastActivity = msg.Timestamp
				}

				// Count tokens
				if msg.Message.Usage != nil {
					project.TotalTokens.InputTokens += msg.Message.Usage.InputTokens
					project.TotalTokens.OutputTokens += msg.Message.Usage.OutputTokens
					project.TotalTokens.CacheCreationInputTokens += msg.Message.Usage.CacheCreationInputTokens
					project.TotalTokens.CacheReadInputTokens += msg.Message.Usage.CacheReadInputTokens
				}

				// Count models
				if msg.Message.Model != nil {
					project.Models[*msg.Message.Model]++
				}
			}

			// Check if session is active (activity within last 2 minutes)
			if time.Since(project.LastActivity) < 2*time.Minute {
				project.ActiveSessions++
			}
		}

		projects[projectName] = project
	}

	return projects, nil
}

// GetDailyMetrics returns metrics for the last N days
func (r *SessionRepository) GetDailyMetrics(days int) ([]DailyMetrics, error) {
	dailyMetrics := make([]DailyMetrics, days)
	now := time.Now()

	// Initialize with empty metrics
	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		dailyMetrics[days-1-i] = DailyMetrics{
			Date:   date,
			Models: make(map[string]int),
		}
	}

	// Track unique sessions per day globally (this is the key fix)
	globalSessionsByDay := make(map[string]map[string]bool)
	for i := range dailyMetrics {
		globalSessionsByDay[dailyMetrics[i].Date] = make(map[string]bool)
	}

	// Re-read files to get daily breakdown
	projectsDir := filepath.Join(r.claudeDir, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return dailyMetrics, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectDir := filepath.Join(projectsDir, entry.Name())
		sessionFiles, err := os.ReadDir(projectDir)
		if err != nil {
			continue
		}

		for _, sessionFile := range sessionFiles {
			if !strings.HasSuffix(sessionFile.Name(), ".jsonl") {
				continue
			}

			sessionPath := filepath.Join(projectDir, sessionFile.Name())
			sessionData, err := r.readSessionFile(sessionPath)
			if err != nil {
				continue
			}

			if len(sessionData) == 0 {
				continue
			}

			// Get session ID from the first message
			sessionID := sessionData[0].SessionID
			if sessionID == "" {
				continue // Skip sessions with empty IDs
			}

			sessionCountedDays := make(map[string]bool) // Track which days this session was already counted
			
			for _, msg := range sessionData {
				dayKey := msg.Timestamp.Format("2006-01-02")
				
				// Find the corresponding day in our metrics
				for i := range dailyMetrics {
					if dailyMetrics[i].Date == dayKey {
						// Count unique sessions per day globally (only once per session per day)
						if !sessionCountedDays[dayKey] {
							// Check if this session was already counted globally for this day
							if !globalSessionsByDay[dayKey][sessionID] {
								dailyMetrics[i].SessionCount++
								globalSessionsByDay[dayKey][sessionID] = true
							}
							sessionCountedDays[dayKey] = true
						}
						
						dailyMetrics[i].MessageCount++
						
						if msg.Message.Usage != nil {
							dailyMetrics[i].TotalTokens.InputTokens += msg.Message.Usage.InputTokens
							dailyMetrics[i].TotalTokens.OutputTokens += msg.Message.Usage.OutputTokens
							dailyMetrics[i].TotalTokens.CacheCreationInputTokens += msg.Message.Usage.CacheCreationInputTokens
							dailyMetrics[i].TotalTokens.CacheReadInputTokens += msg.Message.Usage.CacheReadInputTokens
						}
						
						if msg.Message.Model != nil {
							dailyMetrics[i].Models[*msg.Message.Model]++
						}
						break
					}
				}
			}
		}
	}

	return dailyMetrics, nil
}

// GetActiveSessionsCount returns the number of currently active sessions
func (r *SessionRepository) GetActiveSessionsCount() (int, error) {
	projects, err := r.GetAllProjectSessions()
	if err != nil {
		return 0, err
	}

	activeCount := 0
	for _, project := range projects {
		activeCount += project.ActiveSessions
	}

	return activeCount, nil
}

// GetTotalTokensByProject returns token usage by project
func (r *SessionRepository) GetTotalTokensByProject() (map[string]RepositoryTokenUsage, error) {
	projects, err := r.GetAllProjectSessions()
	if err != nil {
		return nil, err
	}

	tokensByProject := make(map[string]RepositoryTokenUsage)
	for projectName, project := range projects {
		tokensByProject[projectName] = project.TotalTokens
	}

	return tokensByProject, nil
}

// GetModelUsage returns overall model usage statistics
func (r *SessionRepository) GetModelUsage() (map[string]int, error) {
	projects, err := r.GetAllProjectSessions()
	if err != nil {
		return nil, err
	}

	modelUsage := make(map[string]int)
	for _, project := range projects {
		for model, count := range project.Models {
			modelUsage[model] += count
		}
	}

	return modelUsage, nil
}

// GetHourlyActivity returns activity patterns by hour
func (r *SessionRepository) GetHourlyActivity() (map[int][]time.Time, error) {
	hourlyActivity := make(map[int][]time.Time)
	
	projectsDir := filepath.Join(r.claudeDir, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return hourlyActivity, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectDir := filepath.Join(projectsDir, entry.Name())
		sessionFiles, err := os.ReadDir(projectDir)
		if err != nil {
			continue
		}

		for _, sessionFile := range sessionFiles {
			if !strings.HasSuffix(sessionFile.Name(), ".jsonl") {
				continue
			}

			sessionPath := filepath.Join(projectDir, sessionFile.Name())
			sessionData, err := r.readSessionFile(sessionPath)
			if err != nil {
				continue
			}

			for _, msg := range sessionData {
				hour := msg.Timestamp.Hour()
				hourlyActivity[hour] = append(hourlyActivity[hour], msg.Timestamp)
			}
		}
	}

	return hourlyActivity, nil
}

// GetTotalSessions returns the total number of sessions
func (r *SessionRepository) GetTotalSessions() (int, error) {
	projects, err := r.GetAllProjectSessions()
	if err != nil {
		return 0, err
	}

	totalSessions := 0
	for _, project := range projects {
		totalSessions += project.SessionCount
	}

	return totalSessions, nil
}

// GetOverallTokenUsage returns total token usage across all projects
func (r *SessionRepository) GetOverallTokenUsage() (RepositoryTokenUsage, error) {
	projects, err := r.GetAllProjectSessions()
	if err != nil {
		return RepositoryTokenUsage{}, err
	}

	var total RepositoryTokenUsage
	for _, project := range projects {
		total.InputTokens += project.TotalTokens.InputTokens
		total.OutputTokens += project.TotalTokens.OutputTokens
		total.CacheCreationInputTokens += project.TotalTokens.CacheCreationInputTokens
		total.CacheReadInputTokens += project.TotalTokens.CacheReadInputTokens
	}

	return total, nil
}

// RepositorySession represents a session with computed metadata
type RepositorySession struct {
	ID            string
	ProjectPath   string
	ProjectName   string
	Messages      []SessionMessage
	StartTime     time.Time
	LastActivity  time.Time
	TotalTokens   RepositoryTokenUsage
	MessageCount  int
	Model         string
	IsActive      bool
	Duration      time.Duration
	FilesModified []string
}

// GetAllSessions returns all sessions with full metadata
func (r *SessionRepository) GetAllSessions() ([]RepositorySession, error) {
	var allSessions []RepositorySession
	
	projectsDir := filepath.Join(r.claudeDir, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectPath := entry.Name()
		decodedPath := strings.ReplaceAll(projectPath, "-", "/")
		parts := strings.Split(decodedPath, "/")
		projectName := parts[len(parts)-1]

		projectDir := filepath.Join(projectsDir, projectPath)
		sessionFiles, err := os.ReadDir(projectDir)
		if err != nil {
			continue
		}

		for _, sessionFile := range sessionFiles {
			if !strings.HasSuffix(sessionFile.Name(), ".jsonl") {
				continue
			}

			sessionPath := filepath.Join(projectDir, sessionFile.Name())
			sessionData, err := r.readSessionFile(sessionPath)
			if err != nil {
				continue
			}

			if len(sessionData) == 0 {
				continue
			}

			session := r.buildSessionFromMessages(sessionData, decodedPath, projectName)
			allSessions = append(allSessions, session)
		}
	}

	return allSessions, nil
}

// GetSessionById returns a specific session by ID
func (r *SessionRepository) GetSessionById(sessionId string) (*RepositorySession, error) {
	sessions, err := r.GetAllSessions()
	if err != nil {
		return nil, err
	}

	for _, session := range sessions {
		if session.ID == sessionId {
			return &session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionId)
}

// GetActiveSessions returns sessions with recent activity
func (r *SessionRepository) GetActiveSessions() ([]RepositorySession, error) {
	sessions, err := r.GetAllSessions()
	if err != nil {
		return nil, err
	}

	var activeSessions []RepositorySession
	for _, session := range sessions {
		if session.IsActive {
			activeSessions = append(activeSessions, session)
		}
	}

	return activeSessions, nil
}

// GetRecentSessions returns the N most recent sessions
func (r *SessionRepository) GetRecentSessions(limit int) ([]RepositorySession, error) {
	sessions, err := r.GetAllSessions()
	if err != nil {
		return nil, err
	}

	// Sort by last activity (most recent first)
	for i := 0; i < len(sessions)-1; i++ {
		for j := i + 1; j < len(sessions); j++ {
			if sessions[j].LastActivity.After(sessions[i].LastActivity) {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			}
		}
	}

	if limit > len(sessions) {
		limit = len(sessions)
	}

	return sessions[:limit], nil
}

// GetTotalMessages returns the total number of messages across all sessions
func (r *SessionRepository) GetTotalMessages() (int, error) {
	sessions, err := r.GetAllSessions()
	if err != nil {
		return 0, err
	}

	totalMessages := 0
	for _, session := range sessions {
		totalMessages += session.MessageCount
	}

	return totalMessages, nil
}

// GetEstimatedCost calculates estimated cost based on token usage and model pricing
func (r *SessionRepository) GetEstimatedCost() (float64, error) {
	sessions, err := r.GetAllSessions()
	if err != nil {
		return 0, err
	}

	// Simple cost estimation (Claude 3 Opus pricing as example)
	// Input: $15/1M tokens, Output: $75/1M tokens, Cache read: $1.50/1M tokens
	const (
		inputCostPer1M              = 15.0
		outputCostPer1M             = 75.0
		cacheReadCostPer1M          = 1.50
		cacheCreationCostPer1M      = 18.75 // 1.25x input cost
	)

	totalCost := 0.0
	for _, session := range sessions {
		tokens := session.TotalTokens
		
		// Calculate costs per 1M tokens
		totalCost += float64(tokens.InputTokens) * inputCostPer1M / 1000000
		totalCost += float64(tokens.OutputTokens) * outputCostPer1M / 1000000
		totalCost += float64(tokens.CacheReadInputTokens) * cacheReadCostPer1M / 1000000
		totalCost += float64(tokens.CacheCreationInputTokens) * cacheCreationCostPer1M / 1000000
	}

	return totalCost, nil
}

// GetAverageSessionDuration returns the average session duration in minutes
func (r *SessionRepository) GetAverageSessionDuration() (float64, error) {
	sessions, err := r.GetAllSessions()
	if err != nil {
		return 0, err
	}

	if len(sessions) == 0 {
		return 0, nil
	}

	totalDuration := time.Duration(0)
	for _, session := range sessions {
		totalDuration += session.Duration
	}

	return totalDuration.Minutes() / float64(len(sessions)), nil
}

// GetMostUsedModel returns the most frequently used model
func (r *SessionRepository) GetMostUsedModel() (string, error) {
	modelUsage, err := r.GetModelUsage()
	if err != nil {
		return "", err
	}

	mostUsedModel := ""
	maxCount := 0
	for model, count := range modelUsage {
		if count > maxCount {
			maxCount = count
			mostUsedModel = model
		}
	}

	if mostUsedModel == "" {
		return "claude-3-opus", nil // Default
	}

	return mostUsedModel, nil
}

// GetPeakHours returns hours with the highest activity
func (r *SessionRepository) GetPeakHours() ([]map[string]interface{}, error) {
	hourlyActivity, err := r.GetHourlyActivity()
	if err != nil {
		return nil, err
	}

	var peakHours []map[string]interface{}
	for hour := 0; hour < 24; hour++ {
		if timestamps, exists := hourlyActivity[hour]; exists && len(timestamps) > 0 {
			// Calculate average sessions per day for this hour
			uniqueDays := make(map[string]bool)
			for _, ts := range timestamps {
				uniqueDays[ts.Format("2006-01-02")] = true
			}
			avg := float64(len(timestamps)) / float64(len(uniqueDays))
			if avg > 1.0 {
				peakHours = append(peakHours, map[string]interface{}{
					"hour":             hour,
					"average_sessions": avg,
				})
			}
		}
	}

	// Sort by average sessions (highest first)
	for i := 0; i < len(peakHours)-1; i++ {
		for j := i + 1; j < len(peakHours); j++ {
			if peakHours[j]["average_sessions"].(float64) > peakHours[i]["average_sessions"].(float64) {
				peakHours[i], peakHours[j] = peakHours[j], peakHours[i]
			}
		}
	}

	// Limit to top 4
	if len(peakHours) > 4 {
		peakHours = peakHours[:4]
	}

	return peakHours, nil
}

// SearchSessions searches sessions by query string
func (r *SessionRepository) SearchSessions(query string) ([]RepositorySession, error) {
	sessions, err := r.GetAllSessions()
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	var results []RepositorySession

	for _, session := range sessions {
		matched := false

		// Search in project name
		if strings.Contains(strings.ToLower(session.ProjectName), queryLower) {
			matched = true
		}

		// Search in message content (last 10 messages)
		if !matched {
			messageCount := len(session.Messages)
			startIdx := messageCount - 10
			if startIdx < 0 {
				startIdx = 0
			}

			for i := startIdx; i < messageCount && !matched; i++ {
				content := ""
				if str, ok := session.Messages[i].Message.Content.(string); ok {
					content = str
				}
				if strings.Contains(strings.ToLower(content), queryLower) {
					matched = true
				}
			}
		}

		// Search in file paths
		if !matched {
			for _, filePath := range session.FilesModified {
				if strings.Contains(strings.ToLower(filePath), queryLower) {
					matched = true
					break
				}
			}
		}

		if matched {
			results = append(results, session)
		}
	}

	return results, nil
}

// buildSessionFromMessages creates a RepositorySession from message data
func (r *SessionRepository) buildSessionFromMessages(messages []SessionMessage, projectPath, projectName string) RepositorySession {
	if len(messages) == 0 {
		return RepositorySession{}
	}

	session := RepositorySession{
		ID:          messages[0].SessionID,
		ProjectPath: projectPath,
		ProjectName: projectName,
		Messages:    messages,
		StartTime:   messages[0].Timestamp,
		LastActivity: messages[0].Timestamp,
		MessageCount: len(messages),
		FilesModified: make([]string, 0),
	}

	// Aggregate data from messages
	filesModified := make(map[string]bool)
	for _, msg := range messages {
		// Update activity times
		if msg.Timestamp.Before(session.StartTime) {
			session.StartTime = msg.Timestamp
		}
		if msg.Timestamp.After(session.LastActivity) {
			session.LastActivity = msg.Timestamp
		}

		// Aggregate tokens
		if msg.Message.Usage != nil {
			session.TotalTokens.InputTokens += msg.Message.Usage.InputTokens
			session.TotalTokens.OutputTokens += msg.Message.Usage.OutputTokens
			session.TotalTokens.CacheCreationInputTokens += msg.Message.Usage.CacheCreationInputTokens
			session.TotalTokens.CacheReadInputTokens += msg.Message.Usage.CacheReadInputTokens
		}

		// Get model (latest one)
		if msg.Message.Model != nil {
			session.Model = *msg.Message.Model
		}

		// Extract file modifications from tool use results
		if msg.ToolUseResult != nil {
			if filePath, ok := msg.ToolUseResult["file_path"].(string); ok {
				filesModified[filePath] = true
			}
		}
	}

	// Convert files map to slice
	for file := range filesModified {
		session.FilesModified = append(session.FilesModified, file)
	}

	// Calculate duration and active status
	session.Duration = session.LastActivity.Sub(session.StartTime)
	session.IsActive = time.Since(session.LastActivity) < 2*time.Minute

	if session.Model == "" {
		session.Model = "claude-3-opus" // Default
	}

	return session
}

// ActivityEntry represents an activity item for the timeline
type ActivityEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
	SessionID   string    `json:"session_id"`
	SessionName string    `json:"session_name"`
	Details     string    `json:"details"`
}

// GetRecentActivity returns recent activity timeline
func (r *SessionRepository) GetRecentActivity(limit int) ([]ActivityEntry, error) {
	sessions, err := r.GetAllSessions()
	if err != nil {
		return nil, err
	}

	var activities []ActivityEntry

	for _, session := range sessions {
		// Add session start activity
		if !session.StartTime.IsZero() {
			activities = append(activities, ActivityEntry{
				Timestamp:   session.StartTime,
				Type:        "session_created",
				SessionID:   session.ID,
				SessionName: session.ProjectName,
				Details:     "Session started in " + session.ProjectName,
			})
		}

		// Add recent message activities (last 3 messages per session)
		messageCount := len(session.Messages)
		startIdx := messageCount - 3
		if startIdx < 0 {
			startIdx = 0
		}

		for i := startIdx; i < messageCount; i++ {
			msg := session.Messages[i]
			activityType := "message_sent"
			details := "User sent a message"
			
			if msg.Type == "assistant" {
				details = "Assistant responded"
			}

			activities = append(activities, ActivityEntry{
				Timestamp:   msg.Timestamp,
				Type:        activityType,
				SessionID:   session.ID,
				SessionName: session.ProjectName,
				Details:     details,
			})
		}

		// Add session update for recent activity
		if time.Since(session.LastActivity) < 15*time.Minute {
			activities = append(activities, ActivityEntry{
				Timestamp:   session.LastActivity,
				Type:        "session_updated",
				SessionID:   session.ID,
				SessionName: session.ProjectName,
				Details:     "Session activity updated",
			})
		}
	}

	// Sort by timestamp (most recent first)
	for i := 0; i < len(activities)-1; i++ {
		for j := i + 1; j < len(activities); j++ {
			if activities[j].Timestamp.After(activities[i].Timestamp) {
				activities[i], activities[j] = activities[j], activities[i]
			}
		}
	}

	// Apply limit
	if limit < len(activities) {
		activities = activities[:limit]
	}

	return activities, nil
}

// readSessionFile reads and parses a JSONL session file
func (r *SessionRepository) readSessionFile(filePath string) ([]SessionMessage, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var messages []SessionMessage
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		var msg SessionMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			// Skip invalid lines but continue processing
			continue
		}
		messages = append(messages, msg)
	}

	return messages, scanner.Err()
}