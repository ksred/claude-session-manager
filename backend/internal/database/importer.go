package database

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Importer handles importing JSONL files into the database
type Importer struct {
	repo   *SessionRepository
	logger *logrus.Logger
	ctx    context.Context
}

// NewImporter creates a new importer
func NewImporter(repo *SessionRepository, logger *logrus.Logger) *Importer {
	return &Importer{
		repo:   repo,
		logger: logger,
		ctx:    context.Background(),
	}
}

// NewImporterWithContext creates a new importer with context
func NewImporterWithContext(ctx context.Context, repo *SessionRepository, logger *logrus.Logger) *Importer {
	return &Importer{
		repo:   repo,
		logger: logger,
		ctx:    ctx,
	}
}

// ImportClaudeDirectory imports all JSONL files from the Claude directory
func (i *Importer) ImportClaudeDirectory(claudeDir string) error {
	projectsDir := filepath.Join(claudeDir, "projects")
	
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return fmt.Errorf("failed to read projects directory: %w", err)
	}

	// Count total files first for progress tracking
	totalProjectDirs := 0
	totalJSONLFiles := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		totalProjectDirs++
		
		projectDir := filepath.Join(projectsDir, entry.Name())
		sessionFiles, err := os.ReadDir(projectDir)
		if err != nil {
			continue
		}
		
		for _, sessionFile := range sessionFiles {
			if strings.HasSuffix(sessionFile.Name(), ".jsonl") {
				totalJSONLFiles++
			}
		}
	}

	i.logger.WithFields(logrus.Fields{
		"total_project_dirs": totalProjectDirs,
		"total_jsonl_files":  totalJSONLFiles,
	}).Info("Starting import of Claude directory")

	processedFiles := 0
	totalSessions := 0
	totalMessages := 0
	startTime := time.Now()

	for projectIdx, entry := range entries {
		// Check for cancellation
		select {
		case <-i.ctx.Done():
			i.logger.Info("Import cancelled by context")
			return i.ctx.Err()
		default:
		}
		
		if !entry.IsDir() {
			continue
		}

		projectPath := entry.Name()
		projectDir := filepath.Join(projectsDir, projectPath)
		
		// Parse project info
		projectInfo := i.parseProjectPath(projectPath)
		
		i.logger.WithFields(logrus.Fields{
			"project":     projectInfo.ProjectName,
			"progress":    fmt.Sprintf("%d/%d", projectIdx+1, totalProjectDirs),
		}).Debug("Processing project directory")
		
		sessionFiles, err := os.ReadDir(projectDir)
		if err != nil {
			i.logger.WithError(err).WithField("project", projectPath).Warn("Failed to read project directory")
			continue
		}

		projectSessions := 0
		projectMessages := 0

		for _, sessionFile := range sessionFiles {
			if !strings.HasSuffix(sessionFile.Name(), ".jsonl") {
				continue
			}

			sessionPath := filepath.Join(projectDir, sessionFile.Name())
			
			// Get file size for progress info
			fileInfo, _ := sessionFile.Info()
			fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)
			
			i.logger.WithFields(logrus.Fields{
				"file":       sessionFile.Name(),
				"project":    projectInfo.ProjectName,
				"size_mb":    fmt.Sprintf("%.1f", fileSizeMB),
				"progress":   fmt.Sprintf("%d/%d files", processedFiles+1, totalJSONLFiles),
			}).Debug("Processing JSONL file")
			
			sessions, messages, err := i.ImportJSONLFile(sessionPath, projectInfo)
			if err != nil {
				i.logger.WithError(err).WithFields(logrus.Fields{
					"file":    sessionPath,
					"project": projectInfo.ProjectName,
				}).Error("Failed to import JSONL file")
				continue
			}

			processedFiles++
			projectSessions += sessions
			projectMessages += messages
			totalSessions += sessions
			totalMessages += messages
			
			// Log every 10 files or large files
			if processedFiles % 10 == 0 || fileSizeMB > 5 {
				elapsed := time.Since(startTime)
				remaining := time.Duration(float64(elapsed) * float64(totalJSONLFiles-processedFiles) / float64(processedFiles))
				
				i.logger.WithFields(logrus.Fields{
					"processed_files":   processedFiles,
					"total_files":       totalJSONLFiles,
					"progress_pct":      fmt.Sprintf("%.1f%%", float64(processedFiles)*100/float64(totalJSONLFiles)),
					"sessions_so_far":   totalSessions,
					"messages_so_far":   totalMessages,
					"elapsed":           elapsed.Round(time.Second),
					"estimated_remaining": remaining.Round(time.Second),
				}).Info("Import progress")
			}
		}
		
		if projectSessions > 0 {
			i.logger.WithFields(logrus.Fields{
				"project":  projectInfo.ProjectName,
				"sessions": projectSessions,
				"messages": projectMessages,
			}).Info("Completed project import")
		}
	}

	duration := time.Since(startTime)
	i.logger.WithFields(logrus.Fields{
		"files":      processedFiles,
		"sessions":   totalSessions,
		"messages":   totalMessages,
		"duration":   duration.Round(time.Second),
		"files_per_sec": fmt.Sprintf("%.1f", float64(processedFiles)/duration.Seconds()),
	}).Info("Import completed successfully")

	return nil
}

// ProjectInfo contains parsed project information
type ProjectInfo struct {
	ProjectPath string
	ProjectName string
	FilePath    string
}

// parseProjectPath extracts project information from the encoded directory name
func (i *Importer) parseProjectPath(encodedPath string) ProjectInfo {
	// Remove leading hyphen
	decodedPath := encodedPath
	if strings.HasPrefix(decodedPath, "-") {
		decodedPath = strings.TrimPrefix(decodedPath, "-")
	}
	
	// Extract project name using the same logic as the original session repository
	parts := strings.Split(decodedPath, "-")
	var projectName string
	
	if len(parts) >= 4 {
		if strings.Contains(decodedPath, "Documents-GitHub") {
			// Standard path: Users-username-Documents-GitHub-project-name
			githubIndex := -1
			for idx, part := range parts {
				if part == "GitHub" {
					githubIndex = idx
					break
				}
			}
			if githubIndex >= 0 && githubIndex < len(parts)-1 {
				projectName = strings.Join(parts[githubIndex+1:], "-")
			} else {
				projectName = parts[len(parts)-1]
			}
		} else if strings.Contains(decodedPath, "ccswitch-worktrees") {
			// Worktree path: Users-username-.ccswitch-worktrees-project-name
			worktreeIndex := -1
			for idx, part := range parts {
				if part == "worktrees" {
					worktreeIndex = idx
					break
				}
			}
			if worktreeIndex >= 0 && worktreeIndex < len(parts)-1 {
				projectName = strings.Join(parts[worktreeIndex+1:], "-")
			} else {
				projectName = parts[len(parts)-1]
			}
		} else {
			// Fallback - take last part
			projectName = parts[len(parts)-1]
		}
	} else {
		projectName = parts[len(parts)-1]
	}

	// Convert back to actual path format
	actualPath := strings.ReplaceAll(decodedPath, "-", "/")

	return ProjectInfo{
		ProjectPath: actualPath,
		ProjectName: projectName,
		FilePath:    encodedPath,
	}
}

// JSONLMessage represents a message from the JSONL file
type JSONLMessage struct {
	ParentUUID    *string         `json:"parentUuid"`
	IsSidechain   bool            `json:"isSidechain"`
	UserType      string          `json:"userType"`
	CWD           string          `json:"cwd"`
	SessionID     string          `json:"sessionId"`
	Version       string          `json:"version"`
	Type          string          `json:"type"`
	Message       MessageContent  `json:"message"`
	UUID          string          `json:"uuid"`
	Timestamp     time.Time       `json:"timestamp"`
	RequestID     *string         `json:"requestId,omitempty"`
	ToolUseResult *FlexibleResult `json:"toolUseResult,omitempty"`
}

// FlexibleResult handles toolUseResult that can be either a string or a map
type FlexibleResult struct {
	Value map[string]interface{}
}

// UnmarshalJSON implements custom unmarshaling for FlexibleResult
func (f *FlexibleResult) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as map first
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err == nil {
		f.Value = m
		return nil
	}
	
	// If that fails, try as string and wrap it
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		f.Value = map[string]interface{}{
			"result": s,
		}
		return nil
	}
	
	// If both fail, return the original error
	return fmt.Errorf("toolUseResult must be either a map or a string")
}

// MessageContent represents the content of a message
type MessageContent struct {
	Role    string                 `json:"role"`
	Content interface{}            `json:"content"`
	ID      *string                `json:"id,omitempty"`
	Model   *string                `json:"model,omitempty"`
	Usage   *JSONLTokenUsage       `json:"usage,omitempty"`
}

// JSONLTokenUsage represents token usage from JSONL files
type JSONLTokenUsage struct {
	InputTokens              int    `json:"input_tokens"`
	OutputTokens             int    `json:"output_tokens"`
	CacheCreationInputTokens int    `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int    `json:"cache_read_input_tokens"`
	ServiceTier              string `json:"service_tier"`
}

// ImportJSONLFile imports a single JSONL file and returns counts
func (i *Importer) ImportJSONLFile(filePath string, projectInfo ProjectInfo) (int, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	// Parse all messages first to group by session
	sessionMessages := make(map[string][]JSONLMessage)
	
	// Create scanner with larger buffer to handle long lines
	scanner := bufio.NewScanner(file)
	// Set max token size to 10MB (default is 64KB)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB max token size
	
	messageCount := 0
	lineNum := 0
	lastLogTime := time.Now()
	
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		var msg JSONLMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			i.logger.WithError(err).WithFields(logrus.Fields{
				"file": filePath,
				"line": lineNum,
			}).Debug("Failed to parse message, skipping")
			continue
		}
		
		sessionMessages[msg.SessionID] = append(sessionMessages[msg.SessionID], msg)
		messageCount++
		
		// Log progress for large files every 5 seconds or every 1000 messages
		if messageCount%1000 == 0 || time.Since(lastLogTime) > 5*time.Second {
			i.logger.WithFields(logrus.Fields{
				"file":     filepath.Base(filePath),
				"lines":    lineNum,
				"messages": messageCount,
				"sessions": len(sessionMessages),
			}).Debug("File parsing progress")
			lastLogTime = time.Now()
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("error reading file: %w", err)
	}

	// Process each session
	sessionCount := 0
	for sessionID, messages := range sessionMessages {
		if err := i.importSession(sessionID, messages, projectInfo, filePath); err != nil {
			i.logger.WithError(err).WithField("session_id", sessionID).Error("Failed to import session")
			continue
		}
		sessionCount++
	}

	return sessionCount, messageCount, nil
}

// importSession imports a single session with its messages
func (i *Importer) importSession(sessionID string, messages []JSONLMessage, projectInfo ProjectInfo, filePath string) error {
	if len(messages) == 0 {
		return nil
	}

	// Calculate session metadata
	startTime := messages[0].Timestamp
	lastActivity := messages[0].Timestamp
	var model string
	var actualProjectPath string
	var actualProjectName string
	
	for _, msg := range messages {
		if msg.Timestamp.Before(startTime) {
			startTime = msg.Timestamp
		}
		if msg.Timestamp.After(lastActivity) {
			lastActivity = msg.Timestamp
		}
		if msg.Message.Model != nil {
			model = *msg.Message.Model
		}
		// Extract the actual project path from CWD field in messages
		if msg.CWD != "" && actualProjectPath == "" {
			actualProjectPath = msg.CWD
			actualProjectName = filepath.Base(actualProjectPath)
		}
	}
	
	// Fallback to parsed project info if no CWD found in messages
	if actualProjectPath == "" {
		actualProjectPath = projectInfo.ProjectPath
		actualProjectName = projectInfo.ProjectName
	}

	// Determine if session is active (activity within last 2 minutes)
	isActive := time.Since(lastActivity) < 2*time.Minute
	duration := lastActivity.Sub(startTime)

	// Create session
	session := &Session{
		ID:              sessionID,
		ProjectPath:     actualProjectPath,
		ProjectName:     actualProjectName,
		FilePath:        filePath,
		StartTime:       startTime,
		LastActivity:    lastActivity,
		IsActive:        isActive,
		Status:          "completed",
		Model:           model,
		MessageCount:    len(messages),
		DurationSeconds: int64(duration.Seconds()),
	}

	if isActive {
		session.Status = "active"
	}

	// Insert session
	if err := i.repo.UpsertSession(session); err != nil {
		return fmt.Errorf("failed to upsert session: %w", err)
	}

	// Insert messages and related data
	for _, msg := range messages {
		// Convert content to JSON string
		contentBytes, err := json.Marshal(msg.Message.Content)
		if err != nil {
			i.logger.WithError(err).Warn("Failed to marshal message content")
			contentBytes = []byte("{}")
		}

		// Create message
		dbMessage := &Message{
			ID:          msg.UUID,
			SessionID:   sessionID,
			ParentUUID:  msg.ParentUUID,
			IsSidechain: msg.IsSidechain,
			UserType:    msg.UserType,
			CWD:         msg.CWD,
			Version:     msg.Version,
			Type:        msg.Type,
			Role:        msg.Message.Role,
			Content:     string(contentBytes),
			RequestID:   msg.RequestID,
			Timestamp:   msg.Timestamp,
		}

		if err := i.repo.UpsertMessage(dbMessage); err != nil {
			return fmt.Errorf("failed to upsert message: %w", err)
		}

		// Handle token usage
		if msg.Message.Usage != nil {
			usage := &TokenUsage{
				MessageID:                msg.UUID,
				SessionID:                sessionID,
				InputTokens:              msg.Message.Usage.InputTokens,
				OutputTokens:             msg.Message.Usage.OutputTokens,
				CacheCreationInputTokens: msg.Message.Usage.CacheCreationInputTokens,
				CacheReadInputTokens:     msg.Message.Usage.CacheReadInputTokens,
				ServiceTier:              msg.Message.Usage.ServiceTier,
			}
			
			// Calculate totals and cost
			usage.TotalTokens = usage.InputTokens + usage.OutputTokens + 
				usage.CacheCreationInputTokens + usage.CacheReadInputTokens
			usage.EstimatedCost = i.calculateTokenCost(usage, model)

			if err := i.repo.UpsertTokenUsage(usage); err != nil {
				return fmt.Errorf("failed to upsert token usage: %w", err)
			}
		}

		// Extract tool calls from message content (for assistant messages)
		if msg.Message.Role == "assistant" && msg.Message.Content != nil {
			// Convert content to string for parsing
			contentStr := ""
			switch v := msg.Message.Content.(type) {
			case string:
				contentStr = v
			case []interface{}:
				// Handle array content by converting to JSON
				if bytes, err := json.Marshal(v); err == nil {
					contentStr = string(bytes)
				}
			default:
				// Try to marshal any other type
				if bytes, err := json.Marshal(v); err == nil {
					contentStr = string(bytes)
				}
			}

			// Extract tool calls from the content
			toolCalls := ExtractToolCallsFromMessage(contentStr, msg.Timestamp)
			for _, toolCall := range toolCalls {
				// Only save file-modifying tools
				if !isFileModifyingTool(toolCall.ToolName) {
					continue
				}

				// Create tool result entry
				resultData := map[string]interface{}{
					"tool_name": toolCall.ToolName,
					"parameters": toolCall.Parameters,
				}
				resultBytes, _ := json.Marshal(resultData)

				var filePath *string
				if toolCall.FilePath != "" {
					filePath = &toolCall.FilePath
				}

				toolResult := &ToolResult{
					MessageID:  msg.UUID,
					SessionID:  sessionID,
					ToolName:   toolCall.ToolName,
					FilePath:   filePath,
					ResultData: string(resultBytes),
					Timestamp:  toolCall.Timestamp,
				}

				if err := i.repo.UpsertToolResult(toolResult); err != nil {
					i.logger.WithError(err).Warn("Failed to upsert tool result from content parsing")
				}
			}
		}

		// Also handle legacy tool use results if present
		if msg.ToolUseResult != nil && msg.ToolUseResult.Value != nil {
			resultBytes, err := json.Marshal(msg.ToolUseResult.Value)
			if err != nil {
				i.logger.WithError(err).Warn("Failed to marshal tool result")
				continue
			}

			// Extract file path if available
			var filePath *string
			if fp, ok := msg.ToolUseResult.Value["file_path"].(string); ok {
				filePath = &fp
			}

			// Extract tool name if available
			toolName := "unknown"
			if tn, ok := msg.ToolUseResult.Value["tool_name"].(string); ok {
				toolName = tn
			}

			toolResult := &ToolResult{
				MessageID:  msg.UUID,
				SessionID:  sessionID,
				ToolName:   toolName,
				FilePath:   filePath,
				ResultData: string(resultBytes),
				Timestamp:  msg.Timestamp,
			}

			if err := i.repo.UpsertToolResult(toolResult); err != nil {
				return fmt.Errorf("failed to upsert tool result: %w", err)
			}
		}
	}

	// Don't log import activity - it clutters the activity timeline
	// Only log real user activities like messages and file modifications

	return nil
}

// calculateTokenCost estimates the cost based on token usage and model
func (i *Importer) calculateTokenCost(usage *TokenUsage, model string) float64 {
	// Pricing per million tokens based on model
	var inputCostPer1M, outputCostPer1M, cacheReadCostPer1M, cacheWriteCostPer1M float64
	
	switch {
	case strings.Contains(model, "claude-3-opus"):
		inputCostPer1M = 15.0
		outputCostPer1M = 75.0
		cacheReadCostPer1M = 1.50
		cacheWriteCostPer1M = 18.75
	case strings.Contains(model, "claude-opus-4"):
		inputCostPer1M = 15.0
		outputCostPer1M = 75.0
		cacheReadCostPer1M = 1.50
		cacheWriteCostPer1M = 18.75
	case strings.Contains(model, "claude-3-5-sonnet"), strings.Contains(model, "claude-3.5-sonnet"):
		inputCostPer1M = 3.0
		outputCostPer1M = 15.0
		cacheReadCostPer1M = 0.30
		cacheWriteCostPer1M = 3.75
	case strings.Contains(model, "claude-3-sonnet"):
		inputCostPer1M = 3.0
		outputCostPer1M = 15.0
		cacheReadCostPer1M = 0.30
		cacheWriteCostPer1M = 3.75
	case strings.Contains(model, "claude-3-5-haiku"), strings.Contains(model, "claude-3.5-haiku"):
		inputCostPer1M = 0.80
		outputCostPer1M = 4.0
		cacheReadCostPer1M = 0.08
		cacheWriteCostPer1M = 1.0
	case strings.Contains(model, "claude-3-haiku"):
		inputCostPer1M = 0.25
		outputCostPer1M = 1.25
		cacheReadCostPer1M = 0.03
		cacheWriteCostPer1M = 0.30
	default:
		// Default to Sonnet pricing if model unknown
		inputCostPer1M = 3.0
		outputCostPer1M = 15.0
		cacheReadCostPer1M = 0.30
		cacheWriteCostPer1M = 3.75
	}

	cost := float64(usage.InputTokens) * inputCostPer1M / 1000000
	cost += float64(usage.OutputTokens) * outputCostPer1M / 1000000
	cost += float64(usage.CacheReadInputTokens) * cacheReadCostPer1M / 1000000
	cost += float64(usage.CacheCreationInputTokens) * cacheWriteCostPer1M / 1000000
	
	return cost
}