package database

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

// ClaudeFileWatcher monitors the Claude directory for changes and updates the database
type ClaudeFileWatcher struct {
	claudeDir           string
	repo                *SessionRepository
	importer            *Importer
	incrementalImporter *IncrementalImporter
	logger              *logrus.Logger
	watcher             *fsnotify.Watcher
	mu                  sync.RWMutex
	stopCh              chan struct{}
	doneCh              chan struct{}
	updateCallback      UpdateCallback
	started             bool
}

// UpdateCallback is called when sessions are updated
type UpdateCallback interface {
	OnSessionUpdate(updateType string, sessionID string, session *Session)
	OnActivityUpdate(activity *ActivityLogEntry)
	OnMetricsUpdate(sessionID string, usage *TokenUsage)
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(claudeDir string, repo *SessionRepository, logger *logrus.Logger) (*ClaudeFileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	importer := NewImporter(repo, logger)
	incrementalImporter := NewIncrementalImporter(context.Background(), repo, repo.db, logger)

	fw := &ClaudeFileWatcher{
		claudeDir:           claudeDir,
		repo:                repo,
		importer:            importer,
		incrementalImporter: incrementalImporter,
		logger:              logger,
		watcher:             watcher,
		stopCh:              make(chan struct{}),
		doneCh:              make(chan struct{}),
	}

	return fw, nil
}

// SetUpdateCallback sets the callback for update notifications
func (fw *ClaudeFileWatcher) SetUpdateCallback(callback UpdateCallback) {
	fw.updateCallback = callback
}

// Start begins monitoring the Claude directory
func (fw *ClaudeFileWatcher) Start(ctx context.Context) error {
	fw.mu.Lock()
	if fw.started {
		fw.mu.Unlock()
		return fmt.Errorf("file watcher already started")
	}
	fw.started = true
	fw.mu.Unlock()

	// Add the projects directory to watch
	projectsDir := filepath.Join(fw.claudeDir, "projects")
	if err := fw.addDirectoryRecursively(projectsDir); err != nil {
		return fmt.Errorf("failed to add directory to watcher: %w", err)
	}

	fw.logger.WithField("directory", projectsDir).Info("Started file watcher")

	// Start the event processing goroutine
	go fw.processEvents(ctx)

	return nil
}

// Stop stops the file watcher
func (fw *ClaudeFileWatcher) Stop() {
	fw.mu.Lock()
	if !fw.started {
		fw.mu.Unlock()
		fw.logger.Info("File watcher was not started, nothing to stop")
		return
	}
	fw.started = false
	fw.mu.Unlock()

	fw.logger.Info("Stopping file watcher...")
	
	// Signal stop
	fw.logger.Debug("Closing stop channel")
	close(fw.stopCh)
	
	// Close the fsnotify watcher
	if fw.watcher != nil {
		fw.logger.Debug("Closing fsnotify watcher")
		fw.watcher.Close()
	}
	
	// Wait for processEvents goroutine to finish
	fw.logger.Debug("Waiting for processEvents goroutine to finish...")
	select {
	case <-fw.doneCh:
		fw.logger.Info("File watcher stopped successfully")
	case <-time.After(5 * time.Second):
		fw.logger.Warn("Timeout waiting for file watcher to stop")
	}
}

// addDirectoryRecursively adds a directory and all its subdirectories to the watcher
func (fw *ClaudeFileWatcher) addDirectoryRecursively(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fw.logger.WithError(err).WithField("path", path).Debug("Error walking directory, skipping")
			return nil // Continue walking
		}

		if info.IsDir() {
			if err := fw.watcher.Add(path); err != nil {
				fw.logger.WithError(err).WithField("path", path).Warn("Failed to add directory to watcher")
				return nil // Continue walking
			}
			fw.logger.WithField("path", path).Debug("Added directory to watcher")
		}

		return nil
	})
}

// processEvents processes file system events
func (fw *ClaudeFileWatcher) processEvents(ctx context.Context) {
	defer close(fw.doneCh)

	// Debounce events to avoid processing the same file multiple times rapidly
	debouncer := make(map[string]time.Time)
	debounceMutex := sync.Mutex{}
	debounceInterval := 2 * time.Second

	for {
		select {
		case <-ctx.Done():
			return
		case <-fw.stopCh:
			return
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Only process JSONL files
			if !strings.HasSuffix(event.Name, ".jsonl") {
				continue
			}

			// Debounce events for the same file
			debounceMutex.Lock()
			lastEvent, exists := debouncer[event.Name]
			if exists && time.Since(lastEvent) < debounceInterval {
				debounceMutex.Unlock()
				continue
			}
			debouncer[event.Name] = time.Now()
			debounceMutex.Unlock()

			fw.logger.WithFields(logrus.Fields{
				"event": event.Op.String(),
				"file":  event.Name,
			}).Debug("Processing file event")

			switch {
			case event.Op&fsnotify.Create == fsnotify.Create:
				fw.handleFileCreate(event.Name)
			case event.Op&fsnotify.Write == fsnotify.Write:
				fw.handleFileWrite(event.Name)
			case event.Op&fsnotify.Remove == fsnotify.Remove:
				fw.handleFileRemove(event.Name)
			case event.Op&fsnotify.Rename == fsnotify.Rename:
				fw.handleFileRemove(event.Name)
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			fw.logger.WithError(err).Error("File watcher error")
		}
	}
}

// handleFileCreate handles file creation events
func (fw *ClaudeFileWatcher) handleFileCreate(filePath string) {
	// For new files, wait a bit to ensure they're fully written
	time.Sleep(100 * time.Millisecond)
	fw.processJSONLFile(filePath)
}

// handleFileWrite handles file modification events
func (fw *ClaudeFileWatcher) handleFileWrite(filePath string) {
	// For modified files, use the incremental importer
	fw.processFileWithIncrementalImporter(filePath)
}

// handleFileRemove handles file removal events
func (fw *ClaudeFileWatcher) handleFileRemove(filePath string) {
	// When a file is removed, we could optionally mark sessions as inactive
	// For now, we'll leave the data in the database
	fw.logger.WithField("file", filePath).Info("JSONL file removed")
}

// processFileWithIncrementalImporter uses the incremental importer for real-time updates
func (fw *ClaudeFileWatcher) processFileWithIncrementalImporter(filePath string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Extract project info from file path
	projectInfo := fw.extractProjectInfo(filePath)
	
	// Create a batch importer
	batchImporter := NewBatchImporter(fw.repo, fw.logger)
	
	// Use incremental import that won't delete existing data
	sessions, messages, err := batchImporter.ImportJSONLFileIncremental(filePath, projectInfo)
	if err != nil {
		fw.logger.WithError(err).WithField("file", filePath).Error("Failed to process JSONL file incrementally")
		return
	}

	fw.logger.WithFields(logrus.Fields{
		"file":         filePath,
		"sessions":     sessions,
		"new_messages": messages,
	}).Info("Processed JSONL file incrementally")
	
	// Get session ID from file for notifications
	sessionID := strings.TrimSuffix(filepath.Base(filePath), ".jsonl")
	
	// Notify about updates if we processed new messages
	if messages > 0 && fw.updateCallback != nil {
		// Get the session data for notification
		if sessionSummary, err := fw.repo.GetSessionByID(sessionID); err == nil {
			// Convert SessionSummary to Session for the callback
			session := &Session{
				ID:              sessionSummary.ID,
				ProjectPath:     sessionSummary.ProjectPath,
				ProjectName:     sessionSummary.ProjectName,
				FilePath:        filePath, // Use the actual file path
				GitBranch:       "",       // SessionSummary doesn't have this field
				GitWorktree:     "",       // SessionSummary doesn't have this field
				StartTime:       sessionSummary.StartTime,
				LastActivity:    sessionSummary.LastActivity,
				IsActive:        sessionSummary.IsActive,
				Status:          sessionSummary.Status,
				Model:           sessionSummary.Model,
				MessageCount:    sessionSummary.MessageCount,
				DurationSeconds: sessionSummary.DurationSeconds,
			}
			fw.updateCallback.OnSessionUpdate("session_update", sessionID, session)
		}
	}
}

// processJSONLFile processes a complete JSONL file
func (fw *ClaudeFileWatcher) processJSONLFile(filePath string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Extract project info from file path
	projectInfo := fw.extractProjectInfo(filePath)
	
	sessions, messages, err := fw.importer.ImportJSONLFile(filePath, projectInfo)
	if err != nil {
		fw.logger.WithError(err).WithField("file", filePath).Error("Failed to process JSONL file")
		return
	}

	fw.logger.WithFields(logrus.Fields{
		"file":     filePath,
		"sessions": sessions,
		"messages": messages,
	}).Debug("Processed JSONL file")
}

// processJSONLFileIncremental processes only new lines in a JSONL file
func (fw *ClaudeFileWatcher) processJSONLFileIncremental(filePath string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Get the last processed position for this file
	lastProcessed, err := fw.getLastProcessedPosition(filePath)
	if err != nil {
		fw.logger.WithError(err).WithField("file", filePath).Debug("Failed to get last processed position, processing full file")
		fw.processJSONLFile(filePath)
		return
	}

	// Open the file and seek to the last processed position
	file, err := os.Open(filePath)
	if err != nil {
		fw.logger.WithError(err).WithField("file", filePath).Error("Failed to open file for incremental processing")
		return
	}
	defer file.Close()

	// Get file size to check if it shrunk (indicating a rewrite)
	fileInfo, err := file.Stat()
	if err != nil {
		fw.logger.WithError(err).WithField("file", filePath).Error("Failed to stat file")
		return
	}

	if fileInfo.Size() < lastProcessed {
		// File was rewritten, process from beginning
		fw.logger.WithField("file", filePath).Debug("File size decreased, reprocessing entire file")
		fw.processJSONLFile(filePath)
		return
	}

	// Seek to last processed position
	if _, err := file.Seek(lastProcessed, 0); err != nil {
		fw.logger.WithError(err).WithField("file", filePath).Error("Failed to seek to last processed position")
		return
	}

	// Process new lines with larger buffer for long lines
	scanner := bufio.NewScanner(file)
	// Set max token size to 10MB (default is 64KB)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB max token size
	
	newMessages := 0
	projectInfo := fw.extractProjectInfo(filePath)
	
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var msg JSONLMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			fw.logger.WithError(err).WithField("file", filePath).Debug("Failed to parse message line, skipping")
			continue
		}

		// Process this single message
		if err := fw.processSingleMessage(msg, projectInfo, filePath); err != nil {
			fw.logger.WithError(err).WithField("message_id", msg.UUID).Error("Failed to process message")
			continue
		}

		newMessages++
	}

	if err := scanner.Err(); err != nil {
		fw.logger.WithError(err).WithField("file", filePath).Error("Error scanning file")
		return
	}

	// Update the last processed position
	currentPos, err := file.Seek(0, 1) // Current position
	if err == nil {
		fw.updateLastProcessedPosition(filePath, currentPos, fileInfo.Size())
	}

	if newMessages > 0 {
		fw.logger.WithFields(logrus.Fields{
			"file":         filePath,
			"new_messages": newMessages,
		}).Debug("Processed new messages incrementally")
	}
}

// processSingleMessage processes a single message and updates the database
func (fw *ClaudeFileWatcher) processSingleMessage(msg JSONLMessage, projectInfo ProjectInfo, filePath string) error {
	// Update or create session
	session := &Session{
		ID:           msg.SessionID,
		ProjectPath:  projectInfo.ProjectPath,
		ProjectName:  projectInfo.ProjectName,
		FilePath:     filePath,
		LastActivity: msg.Timestamp,
		IsActive:     time.Since(msg.Timestamp) < 2*time.Minute,
		Status:       "active",
	}

	if !session.IsActive {
		session.Status = "completed"
	}

	if msg.Message.Model != nil {
		session.Model = *msg.Message.Model
	}

	// Get existing session to preserve start time and message count
	if existing, err := fw.repo.GetSessionByID(msg.SessionID); err == nil {
		session.StartTime = existing.StartTime
		session.MessageCount = existing.MessageCount + 1
		session.DurationSeconds = int64(session.LastActivity.Sub(session.StartTime).Seconds())
	} else {
		// New session
		session.StartTime = msg.Timestamp
		session.MessageCount = 1
		session.DurationSeconds = 0
	}

	if err := fw.repo.UpsertSession(session); err != nil {
		return fmt.Errorf("failed to upsert session: %w", err)
	}

	// Notify about session update
	if fw.updateCallback != nil {
		updateType := "session_update"
		if _, err := fw.repo.GetSessionByID(msg.SessionID); err != nil {
			updateType = "session_created"
		}
		fw.logger.WithFields(logrus.Fields{
			"update_type":  updateType,
			"session_id":   session.ID,
			"project_name": session.ProjectName,
		}).Debug("File watcher notifying callback about session update")
		fw.updateCallback.OnSessionUpdate(updateType, session.ID, session)
	}

	// Process message
	contentBytes, err := json.Marshal(msg.Message.Content)
	if err != nil {
		contentBytes = []byte("{}")
	}

	dbMessage := &Message{
		ID:          msg.UUID,
		SessionID:   msg.SessionID,
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

	if err := fw.repo.UpsertMessage(dbMessage); err != nil {
		return fmt.Errorf("failed to upsert message: %w", err)
	}

	// Log activity for user messages
	if msg.Message.Role == "user" {
		// Extract first 100 chars of message for activity details
		contentStr := ""
		if strContent, ok := msg.Message.Content.(string); ok {
			contentStr = strContent
		} else if contentBytes, err := json.Marshal(msg.Message.Content); err == nil {
			contentStr = string(contentBytes)
		}
		
		if len(contentStr) > 100 {
			contentStr = contentStr[:100] + "..."
		}
		
		activity := &ActivityLogEntry{
			SessionID:    &msg.SessionID,
			ActivityType: "message_sent",
			Details:      fmt.Sprintf("User: %s", contentStr),
			Timestamp:    msg.Timestamp,
		}
		
		if err := fw.repo.LogActivity(activity); err != nil {
			fw.logger.WithError(err).Warn("Failed to log message activity")
		}

		// Notify about activity update
		if fw.updateCallback != nil {
			fw.logger.WithFields(logrus.Fields{
				"activity_type": activity.ActivityType,
				"session_id":    msg.SessionID,
			}).Debug("File watcher notifying callback about activity update")
			fw.updateCallback.OnActivityUpdate(activity)
		}
	}

	// Process token usage if present
	if msg.Message.Usage != nil {
		usage := &TokenUsage{
			MessageID:                msg.UUID,
			SessionID:                msg.SessionID,
			InputTokens:              msg.Message.Usage.InputTokens,
			OutputTokens:             msg.Message.Usage.OutputTokens,
			CacheCreationInputTokens: msg.Message.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     msg.Message.Usage.CacheReadInputTokens,
			ServiceTier:              msg.Message.Usage.ServiceTier,
		}
		
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens + 
			usage.CacheCreationInputTokens + usage.CacheReadInputTokens
		usage.EstimatedCost = fw.importer.calculateTokenCost(usage, session.Model)

		if err := fw.repo.UpsertTokenUsage(usage); err != nil {
			return fmt.Errorf("failed to upsert token usage: %w", err)
		}

		// Notify about metrics update
		if fw.updateCallback != nil {
			fw.logger.WithFields(logrus.Fields{
				"session_id":    msg.SessionID,
				"total_tokens":  usage.TotalTokens,
				"estimated_cost": usage.EstimatedCost,
			}).Debug("File watcher notifying callback about metrics update")
			fw.updateCallback.OnMetricsUpdate(msg.SessionID, usage)
		}
	}

	// Process tool results if present
	if msg.ToolUseResult != nil && msg.ToolUseResult.Value != nil {
		resultBytes, _ := json.Marshal(msg.ToolUseResult.Value)
		
		var filePath *string
		if fp, ok := msg.ToolUseResult.Value["file_path"].(string); ok {
			filePath = &fp
		}

		toolName := "unknown"
		if tn, ok := msg.ToolUseResult.Value["tool_name"].(string); ok {
			toolName = tn
		}

		toolResult := &ToolResult{
			MessageID:  msg.UUID,
			SessionID:  msg.SessionID,
			ToolName:   toolName,
			FilePath:   filePath,
			ResultData: string(resultBytes),
			Timestamp:  msg.Timestamp,
		}

		if err := fw.repo.UpsertToolResult(toolResult); err != nil {
			return fmt.Errorf("failed to upsert tool result: %w", err)
		}

		// Log activity for file modifications
		if filePath != nil && *filePath != "" && (toolName == "Edit" || toolName == "Write" || toolName == "MultiEdit" || 
			toolName == "NotebookEdit" || toolName == "NotebookWrite") {
			activity := &ActivityLogEntry{
				SessionID:    &msg.SessionID,
				ActivityType: "file_modified",
				Details:      fmt.Sprintf("Modified %s using %s", *filePath, toolName),
				Timestamp:    msg.Timestamp,
			}
			
			if err := fw.repo.LogActivity(activity); err != nil {
				fw.logger.WithError(err).Warn("Failed to log file modification activity")
			}

			// Notify about activity update
			if fw.updateCallback != nil {
				fw.logger.WithFields(logrus.Fields{
					"activity_type": activity.ActivityType,
					"session_id":    msg.SessionID,
					"file_path":     *filePath,
					"tool_name":     toolName,
				}).Debug("File watcher notifying callback about file modification activity")
				fw.updateCallback.OnActivityUpdate(activity)
			}
		}
	}

	return nil
}

// extractProjectInfo extracts project information from file path
func (fw *ClaudeFileWatcher) extractProjectInfo(filePath string) ProjectInfo {
	// Extract the project directory name from the file path
	projectDir := filepath.Dir(filePath)
	projectDirName := filepath.Base(projectDir)
	
	return fw.importer.parseProjectPath(projectDirName)
}

// getLastProcessedPosition gets the last processed file position
func (fw *ClaudeFileWatcher) getLastProcessedPosition(filePath string) (int64, error) {
	var lastProcessed sql.NullInt64
	err := fw.repo.db.Get(&lastProcessed, `
		SELECT COALESCE(last_processed_position, 0) as last_processed
		FROM file_watchers 
		WHERE file_path = ?
	`, filePath)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	
	return lastProcessed.Int64, nil
}

// updateLastProcessedPosition updates the last processed file position
func (fw *ClaudeFileWatcher) updateLastProcessedPosition(filePath string, position, fileSize int64) {
	_, err := fw.repo.db.Exec(`
		INSERT OR REPLACE INTO file_watchers (file_path, last_modified, last_processed_position, file_size, updated_at)
		VALUES (?, CURRENT_TIMESTAMP, ?, ?, CURRENT_TIMESTAMP)
	`, filePath, position, fileSize)
	
	if err != nil {
		fw.logger.WithError(err).WithField("file", filePath).Error("Failed to update last processed position")
	}
}

