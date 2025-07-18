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

// BatchImporter handles optimized batch imports
type BatchImporter struct {
	repo    *SessionRepository
	batch   *BatchOperations
	logger  *logrus.Logger
	ctx     context.Context
}

// NewBatchImporter creates a new batch importer
func NewBatchImporter(repo *SessionRepository, logger *logrus.Logger) *BatchImporter {
	return &BatchImporter{
		repo:   repo,
		batch:  NewBatchOperations(repo.db, logger),
		logger: logger,
		ctx:    context.Background(),
	}
}

// ImportJSONLFileIncremental imports only new messages from a JSONL file
func (bi *BatchImporter) ImportJSONLFileIncremental(filePath string, projectInfo ProjectInfo) (int, int, error) {
	// First, get existing message IDs for this session to avoid duplicates
	sessionID, err := bi.getSessionIDFromFile(filePath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to determine session ID: %w", err)
	}
	
	existingMessageIDs, err := bi.getExistingMessageIDs(sessionID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get existing message IDs: %w", err)
	}
	
	return bi.importJSONLFileOptimized(filePath, projectInfo, existingMessageIDs, true)
}

// ImportJSONLFileOptimized imports a JSONL file using batch operations
func (bi *BatchImporter) ImportJSONLFileOptimized(filePath string, projectInfo ProjectInfo) (int, int, error) {
	return bi.importJSONLFileOptimized(filePath, projectInfo, make(map[string]bool), false)
}

// getSessionIDFromFile extracts session ID from file path or file content
func (bi *BatchImporter) getSessionIDFromFile(filePath string) (string, error) {
	baseName := filepath.Base(filePath)
	if sessionID := strings.TrimSuffix(baseName, ".jsonl"); sessionID != baseName {
		return sessionID, nil
	}
	
	// If filename doesn't contain session ID, peek at first line
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		var msg JSONLMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err == nil && msg.SessionID != "" {
			return msg.SessionID, nil
		}
	}
	
	return "", fmt.Errorf("could not determine session ID from file")
}

// getExistingMessageIDs gets all message IDs that already exist in the database for a session
func (bi *BatchImporter) getExistingMessageIDs(sessionID string) (map[string]bool, error) {
	rows, err := bi.repo.db.Query("SELECT id FROM messages WHERE session_id = ?", sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	existing := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		existing[id] = true
	}
	
	return existing, nil
}

// importJSONLFileOptimized is the core import logic that can be used for both full and incremental imports
func (bi *BatchImporter) importJSONLFileOptimized(filePath string, projectInfo ProjectInfo, existingMessageIDs map[string]bool, isIncremental bool) (int, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 16*1024*1024), 16*1024*1024) // 16MB buffer

	// Collect all data first
	var sessions []Session
	var messages []Message
	var tokenUsages []TokenUsage
	var toolResults []ToolResult
	sessionMap := make(map[string]*Session)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg JSONLMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			bi.logger.WithError(err).WithField("line", lineNum).Warn("Failed to parse line")
			continue
		}

		sessionID := msg.SessionID
		if sessionID == "" {
			// Extract UUID from filename (e.g., "bd16b52b-ab7d-4a22-b09b-8b1bd2c77a94.jsonl")
			baseName := filepath.Base(filePath)
			sessionID = strings.TrimSuffix(baseName, ".jsonl")
		}

		// Create or update session
		if session, exists := sessionMap[sessionID]; !exists {
			// Extract actual project path from CWD field, fallback to parsed project info
			actualProjectPath := projectInfo.ProjectPath
			actualProjectName := projectInfo.ProjectName
			if msg.CWD != "" {
				actualProjectPath = msg.CWD
				actualProjectName = filepath.Base(actualProjectPath)
			}
			
			session = &Session{
				ID:             sessionID,
				ProjectPath:    actualProjectPath,
				ProjectName:    actualProjectName,
				FilePath:       filePath,
				GitBranch:      "", // Will be populated if available
				GitWorktree:    "", // Will be populated if available
				StartTime:      msg.Timestamp,
				LastActivity:   msg.Timestamp,
				IsActive:       false,
				Status:         "completed",
				Model:          "",
				MessageCount:   1,
				DurationSeconds: 0,
			}
			if msg.Message.Model != nil {
				session.Model = *msg.Message.Model
			}
			sessionMap[sessionID] = session
		} else {
			// Update session metadata
			if msg.Timestamp.Before(session.StartTime) {
				session.StartTime = msg.Timestamp
			}
			if msg.Timestamp.After(session.LastActivity) {
				session.LastActivity = msg.Timestamp
			}
			session.MessageCount++
			if msg.Message.Model != nil {
				session.Model = *msg.Message.Model
			}
		}

		// Skip existing messages in incremental mode
		if isIncremental && existingMessageIDs[msg.UUID] {
			continue
		}

		// Create message
		contentBytes, _ := json.Marshal(msg.Message.Content)
		dbMessage := Message{
			ID:          msg.UUID,
			SessionID:   sessionID,
			Role:        msg.Message.Role,
			Content:     string(contentBytes),
			Timestamp:   msg.Timestamp,
		}
		if msg.ParentUUID != nil && *msg.ParentUUID != "" {
			dbMessage.ParentUUID = msg.ParentUUID
		}
		messages = append(messages, dbMessage)

		// Handle token usage (only if not skipping this message)
		if msg.Message.Usage != nil {
			usage := TokenUsage{
				MessageID:                msg.UUID,
				SessionID:                sessionID,
				InputTokens:              msg.Message.Usage.InputTokens,
				OutputTokens:             msg.Message.Usage.OutputTokens,
				CacheCreationInputTokens: msg.Message.Usage.CacheCreationInputTokens,
				CacheReadInputTokens:     msg.Message.Usage.CacheReadInputTokens,
			}
			
			// Calculate total tokens
			usage.TotalTokens = usage.InputTokens + usage.OutputTokens + 
				usage.CacheCreationInputTokens + usage.CacheReadInputTokens
			
			// Calculate cost
			model := ""
			if s, ok := sessionMap[sessionID]; ok {
				model = s.Model
			}
			if msg.Message.Model != nil {
				model = *msg.Message.Model
			}
			usage.EstimatedCost = bi.calculateTokenCost(&usage, model)
			
			tokenUsages = append(tokenUsages, usage)
		}

		// Extract tool results (only if not skipping this message)
		if msg.Message.Role == "assistant" && msg.Message.Content != nil {
			contentStr := ""
			switch v := msg.Message.Content.(type) {
			case string:
				contentStr = v
			case []interface{}:
				if bytes, err := json.Marshal(v); err == nil {
					contentStr = string(bytes)
				}
			}

			toolCalls := ExtractToolCallsFromMessage(contentStr, msg.Timestamp)
			for _, toolCall := range toolCalls {
				if !isFileModifyingTool(toolCall.ToolName) {
					continue
				}

				var filePath *string
				if toolCall.FilePath != "" {
					filePath = &toolCall.FilePath
				}

				toolResult := ToolResult{
					ID:        0, // Will be auto-generated by database
					MessageID: msg.UUID,
					SessionID: sessionID,
					ToolName:  toolCall.ToolName,
					FilePath:  filePath,
					Timestamp: toolCall.Timestamp,
				}
				
				// Create result data
				resultData := map[string]interface{}{
					"tool_name": toolCall.ToolName,
					"parameters": toolCall.Parameters,
				}
				resultBytes, _ := json.Marshal(resultData)
				toolResult.ResultData = string(resultBytes)
				
				toolResults = append(toolResults, toolResult)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("failed to read file: %w", err)
	}

	// Convert session map to slice and finalize session data
	for _, session := range sessionMap {
		duration := session.LastActivity.Sub(session.StartTime)
		session.DurationSeconds = int64(duration.Seconds())
		session.IsActive = time.Since(session.LastActivity) < 2*time.Minute
		if session.IsActive {
			session.Status = "active"
		} else {
			session.Status = "completed"
		}
		sessions = append(sessions, *session)
	}

	// Perform batch import in a single transaction
	if isIncremental {
		// For incremental imports, use INSERT OR IGNORE to avoid overwriting existing data
		if err := bi.batch.BatchImportDataIncremental(sessions, messages, tokenUsages, toolResults); err != nil {
			return 0, 0, fmt.Errorf("incremental batch import failed: %w", err)
		}
	} else {
		// For full imports, use INSERT OR REPLACE to overwrite existing data
		if err := bi.batch.BatchImportData(sessions, messages, tokenUsages, toolResults); err != nil {
			return 0, 0, fmt.Errorf("batch import failed: %w", err)
		}
	}

	return len(sessions), len(messages), nil
}

// calculateTokenCost estimates the cost based on token usage and model
func (bi *BatchImporter) calculateTokenCost(usage *TokenUsage, model string) float64 {
	var inputCostPer1M, outputCostPer1M, cacheReadCostPer1M, cacheWriteCostPer1M float64
	
	switch {
	case strings.Contains(model, "claude-3-opus"), strings.Contains(model, "claude-opus-4"):
		inputCostPer1M = 15.0
		outputCostPer1M = 75.0
		cacheReadCostPer1M = 1.50
		cacheWriteCostPer1M = 18.75
	case strings.Contains(model, "claude-3-5-sonnet"), strings.Contains(model, "claude-3.5-sonnet"):
		inputCostPer1M = 3.0
		outputCostPer1M = 15.0
		cacheReadCostPer1M = 0.30
		cacheWriteCostPer1M = 3.75
	default:
		inputCostPer1M = 10.0
		outputCostPer1M = 30.0
		cacheReadCostPer1M = 1.0
		cacheWriteCostPer1M = 12.5
	}
	
	cost := float64(usage.InputTokens) * inputCostPer1M / 1_000_000
	cost += float64(usage.OutputTokens) * outputCostPer1M / 1_000_000
	cost += float64(usage.CacheReadInputTokens) * cacheReadCostPer1M / 1_000_000
	cost += float64(usage.CacheCreationInputTokens) * cacheWriteCostPer1M / 1_000_000
	
	return cost
}