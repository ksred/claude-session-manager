package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ksred/claude-session-manager/internal/database"
	"github.com/sirupsen/logrus"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var (
		dbPath  string
		dryRun  bool
		verbose bool
		limit   int
	)

	flag.StringVar(&dbPath, "db", "", "Path to SQLite database (defaults to ~/.claude/sessions.db)")
	flag.BoolVar(&dryRun, "dry-run", false, "Show what would be updated without making changes")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.IntVar(&limit, "limit", 0, "Limit number of sessions to process (0 = all)")
	flag.Parse()

	// Setup logging
	logger := logrus.New()
	if verbose {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}

	// Default database path
	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			logger.Fatal("Failed to get home directory:", err)
		}
		dbPath = filepath.Join(homeDir, ".claude", "sessions.db")
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	logger.WithField("database", dbPath).Info("Starting tool extraction migration")

	// Count existing tool results
	var existingCount int
	err = db.QueryRow("SELECT COUNT(*) FROM tool_results").Scan(&existingCount)
	if err != nil {
		logger.Fatal("Failed to count existing tool results:", err)
	}
	logger.WithField("count", existingCount).Info("Existing tool results")

	// Get messages that might contain tool calls
	query := `
		SELECT m.id, m.session_id, m.content, m.timestamp 
		FROM messages m 
		WHERE m.role = 'assistant' 
		AND m.content LIKE '%"type":"tool_use"%'
		AND m.content LIKE '%"file_path":%'
		AND (
			m.content LIKE '%"name":"Edit"%' 
			OR m.content LIKE '%"name":"Write"%'
			OR m.content LIKE '%"name":"MultiEdit"%'
			OR m.content LIKE '%"name":"NotebookEdit"%'
			OR m.content LIKE '%"name":"NotebookWrite"%'
		)
		ORDER BY m.timestamp DESC
	`
	
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(query)
	if err != nil {
		logger.Fatal("Failed to query messages:", err)
	}
	defer rows.Close()

	// Track statistics
	var (
		messagesProcessed int
		toolsExtracted    int
		filesFound        int
	)

	// Process messages
	for rows.Next() {
		var (
			messageID string
			sessionID string
			content   string
			timestamp time.Time
		)

		if err := rows.Scan(&messageID, &sessionID, &content, &timestamp); err != nil {
			logger.WithError(err).Error("Failed to scan row")
			continue
		}

		messagesProcessed++

		// Try to parse as JSON array first (new format)
		var toolCalls []database.ToolCall
		var jsonArray []map[string]interface{}
		if err := json.Unmarshal([]byte(content), &jsonArray); err == nil {
			// Parse JSON format
			for _, item := range jsonArray {
				if toolType, ok := item["type"].(string); ok && toolType == "tool_use" {
					if name, ok := item["name"].(string); ok {
						if input, ok := item["input"].(map[string]interface{}); ok {
							if filePath, ok := input["file_path"].(string); ok {
								toolCall := database.ToolCall{
									ToolName:   name,
									FilePath:   filePath,
									Parameters: input,
									Timestamp:  timestamp,
								}
								toolCalls = append(toolCalls, toolCall)
							}
						}
					}
				}
			}
		} else {
			// Fall back to existing parser for XML format
			toolCalls = database.ExtractToolCallsFromMessage(content, timestamp)
		}
		
		for _, toolCall := range toolCalls {
			// Only process file-modifying tools
			if !isFileModifyingTool(toolCall.ToolName) {
				continue
			}

			if toolCall.FilePath == "" {
				logger.WithFields(logrus.Fields{
					"tool":    toolCall.ToolName,
					"message": messageID,
				}).Debug("Tool call has no file path")
				continue
			}

			toolsExtracted++
			filesFound++

			if dryRun {
				logger.WithFields(logrus.Fields{
					"message_id": messageID,
					"session_id": sessionID,
					"tool":       toolCall.ToolName,
					"file":       toolCall.FilePath,
				}).Info("Would insert tool result")
				continue
			}

			// Insert or update tool result
			resultData := map[string]interface{}{
				"tool_name":   toolCall.ToolName,
				"parameters":  toolCall.Parameters,
				"extracted":   true,
				"migration":   "migrate-tools",
			}
			resultBytes, _ := json.Marshal(resultData)

			_, err = db.Exec(`
				INSERT OR REPLACE INTO tool_results (message_id, session_id, tool_name, file_path, result_data, timestamp)
				VALUES (?, ?, ?, ?, ?, ?)
			`, messageID, sessionID, toolCall.ToolName, toolCall.FilePath, string(resultBytes), timestamp)

			if err != nil {
				logger.WithError(err).WithFields(logrus.Fields{
					"message_id": messageID,
					"tool":       toolCall.ToolName,
					"file":       toolCall.FilePath,
				}).Error("Failed to insert tool result")
				continue
			}

			if verbose {
				logger.WithFields(logrus.Fields{
					"message_id": messageID,
					"session_id": sessionID,
					"tool":       toolCall.ToolName,
					"file":       toolCall.FilePath,
				}).Debug("Inserted tool result")
			}
		}

		// Log progress every 100 messages
		if messagesProcessed%100 == 0 {
			logger.WithFields(logrus.Fields{
				"messages_processed": messagesProcessed,
				"tools_extracted":    toolsExtracted,
				"files_found":        filesFound,
			}).Info("Progress update")
		}
	}

	// Delete old "unknown" tool results if not in dry-run mode
	if !dryRun {
		result, err := db.Exec("DELETE FROM tool_results WHERE tool_name = 'unknown'")
		if err != nil {
			logger.WithError(err).Error("Failed to delete unknown tool results")
		} else {
			deleted, _ := result.RowsAffected()
			logger.WithField("deleted", deleted).Info("Deleted unknown tool results")
		}
	}

	// Final statistics
	logger.WithFields(logrus.Fields{
		"messages_processed": messagesProcessed,
		"tools_extracted":    toolsExtracted,
		"files_found":        filesFound,
		"dry_run":            dryRun,
	}).Info("Migration completed")

	// Show sample of results
	if !dryRun {
		var sampleCount int
		err = db.QueryRow(`
			SELECT COUNT(DISTINCT file_path) 
			FROM tool_results 
			WHERE file_path IS NOT NULL
		`).Scan(&sampleCount)
		if err == nil {
			logger.WithField("unique_files", sampleCount).Info("Total unique files modified")
		}

		// Show top modified files
		rows, err := db.Query(`
			SELECT file_path, COUNT(*) as count 
			FROM tool_results 
			WHERE file_path IS NOT NULL 
			GROUP BY file_path 
			ORDER BY count DESC 
			LIMIT 5
		`)
		if err == nil {
			defer rows.Close()
			logger.Info("Top 5 most modified files:")
			for rows.Next() {
				var filePath string
				var count int
				if err := rows.Scan(&filePath, &count); err == nil {
					logger.WithFields(logrus.Fields{
						"file":  filePath,
						"count": count,
					}).Info("  File modifications")
				}
			}
		}
	}
}

func isFileModifyingTool(toolName string) bool {
	switch toolName {
	case "Edit", "Write", "MultiEdit", "NotebookEdit", "NotebookWrite":
		return true
	default:
		return false
	}
}