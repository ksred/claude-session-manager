package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ksred/claude-session-manager/internal/config"
)

// DiscoverSessions scans the Claude projects directory for active sessions
func DiscoverSessions() ([]Session, error) {
	// Use default configuration
	cfg := config.DefaultConfig()
	return DiscoverSessionsWithConfig(cfg)
}

// DiscoverSessionsWithConfig scans the Claude projects directory for active sessions using provided config
func DiscoverSessionsWithConfig(cfg *config.Config) ([]Session, error) {
	claudeDir := cfg.Claude.ProjectsPath
	
	// Check if the Claude directory exists
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		return []Session{}, nil // Return empty list if directory doesn't exist
	}
	
	var sessions []Session
	
	err := filepath.Walk(claudeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't read
		}
		
		// Look for .jsonl files that are not summary files
		if strings.HasSuffix(path, ".jsonl") && !strings.Contains(path, "summary") {
			session, err := ParseSessionFile(path)
			if err == nil {
				sessions = append(sessions, session)
			}
		}
		return nil
	})
	
	return sessions, err
}

// ParseSessionFile reads and parses a Claude session .jsonl file
func ParseSessionFile(filePath string) (Session, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Session{}, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()
	
	var session Session
	session.FilePath = filePath
	session.ID = extractSessionID(filePath)
	session.ProjectPath = extractProjectPath(filePath)
	session.ProjectName = extractProjectName(filePath)
	
	scanner := bufio.NewScanner(file)
	lineCount := 0
	
	for scanner.Scan() {
		lineCount++
		line := scanner.Bytes()
		
		// First parse as a generic map to handle the actual Claude session format
		var rawData map[string]interface{}
		if err := json.Unmarshal(line, &rawData); err != nil {
			// Skip malformed lines but continue parsing
			continue
		}
		
		// Skip summary entries
		if msgType, ok := rawData["type"].(string); ok && msgType == "summary" {
			continue
		}
		
		var message Message
		
		// Extract timestamp from top level
		if tsStr, ok := rawData["timestamp"].(string); ok {
			timeFormats := []string{
				time.RFC3339,
				time.RFC3339Nano,
				"2006-01-02T15:04:05.999999999Z07:00",
				"2006-01-02T15:04:05Z",
			}
			
			for _, format := range timeFormats {
				if ts, err := time.Parse(format, tsStr); err == nil {
					message.Timestamp = ts
					break
				}
			}
		}
		
		// Extract message data from nested message object if it exists
		if msgData, ok := rawData["message"].(map[string]interface{}); ok {
			if role, ok := msgData["role"].(string); ok {
				message.Role = role
			}
			if content, ok := msgData["content"].(string); ok {
				message.Content = content
			}
			if id, ok := msgData["id"].(string); ok {
				message.ID = id
			}
			// Extract usage data if present
			if usage, ok := msgData["usage"].(map[string]interface{}); ok {
				if inputTokens, ok := usage["input_tokens"].(float64); ok {
					message.Usage.InputTokens = int(inputTokens)
				}
				if outputTokens, ok := usage["output_tokens"].(float64); ok {
					message.Usage.OutputTokens = int(outputTokens)
				}
				if cacheCreationTokens, ok := usage["cache_creation_input_tokens"].(float64); ok {
					message.Usage.CacheCreationInputTokens = int(cacheCreationTokens)
				}
				if cacheReadTokens, ok := usage["cache_read_input_tokens"].(float64); ok {
					message.Usage.CacheReadInputTokens = int(cacheReadTokens)
				}
				message.Usage.UpdateTotals()
			}
		}
		
		// Extract type from top level
		if msgType, ok := rawData["type"].(string); ok {
			message.Type = msgType
		}
		
		// If we still don't have a timestamp, use current time as last resort
		if message.Timestamp.IsZero() {
			message.Timestamp = time.Now()
		}
		
		// Store all other data in Meta for debugging
		message.Meta = rawData
		
		session.Messages = append(session.Messages, message)
		updateSessionMetrics(&session, message)
	}
	
	// Set session timing information
	if len(session.Messages) > 0 {
		session.StartTime = session.Messages[0].Timestamp
		session.LastActivity = session.Messages[len(session.Messages)-1].Timestamp
	}
	
	// Extract git information
	if session.ProjectPath != "" {
		session.GitBranch = extractGitBranch(session.ProjectPath)
		session.GitWorktree = extractGitWorktree(session.ProjectPath)
	}
	
	// Determine current task from recent messages
	session.CurrentTask = extractCurrentTask(session.Messages)
	
	// Update session status based on activity and content
	session.UpdateStatus()
	
	// Recalculate costs using model-specific pricing
	if len(session.Messages) > 0 {
		// Infer model from session messages  
		model := inferModelFromSessionMessages(session.Messages)
		
		// Recalculate total cost with model-specific pricing
		session.TokensUsed.EstimatedCost = session.TokensUsed.CalculateCostWithModel(model)
	}
	
	return session, scanner.Err()
}

// inferModelFromSessionMessages infers the model used from session messages
func inferModelFromSessionMessages(messages []Message) string {
	// Look for model information in messages metadata - check multiple locations
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		
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
	return "claude-3.5-sonnet" // Default to Sonnet 3.5 pricing if unknown
}

// extractSessionID extracts the session UUID from the file path
func extractSessionID(filePath string) string {
	fileName := filepath.Base(filePath)
	return strings.TrimSuffix(fileName, ".jsonl")
}

// extractProjectPath decodes the project path from the directory structure
func extractProjectPath(filePath string) string {
	dir := filepath.Dir(filePath)
	parts := strings.Split(dir, string(filepath.Separator))
	
	// Find the "projects" directory and take the next part
	for i, part := range parts {
		if part == "projects" && i+1 < len(parts) {
			encodedPath := parts[i+1]
			
			// Claude Code uses hyphens instead of slashes for directory names
			// Convert from Claude's format: -Users-ksred-Documents-GitHub-project
			// To actual path: /Users/ksred/Documents/GitHub/project
			if strings.HasPrefix(encodedPath, "-") {
				// Remove leading hyphen and replace remaining hyphens with slashes
				decodedPath := encodedPath[1:] // Remove leading hyphen
				decodedPath = strings.ReplaceAll(decodedPath, "-", "/")
				return "/" + decodedPath
			}
			
			// Fallback: try URL decoding for other formats
			decodedPath, err := url.QueryUnescape(encodedPath)
			if err != nil {
				// Final fallback to simple replacement
				decodedPath = strings.ReplaceAll(encodedPath, "%2F", "/")
				decodedPath = strings.ReplaceAll(decodedPath, "%20", " ")
			}
			return decodedPath
		}
	}
	return ""
}

// extractProjectName gets a simple display name from the directory name
func extractProjectName(filePath string) string {
	// Extract from the ~/.claude/projects directory structure  
	dir := filepath.Dir(filePath)
	parts := strings.Split(dir, string(filepath.Separator))
	
	// Find the "projects" directory and take the next part
	for i, part := range parts {
		if part == "projects" && i+1 < len(parts) {
			dirName := parts[i+1]
			
			// Remove leading hyphen if present
			if strings.HasPrefix(dirName, "-") {
				dirName = dirName[1:]
			}
			
			// Split by hyphens and take the last meaningful part
			nameParts := strings.Split(dirName, "-")
			if len(nameParts) > 0 {
				return nameParts[len(nameParts)-1]
			}
		}
	}
	
	return "Unknown Project"
}

// extractCurrentTask attempts to determine what the session is currently working on
func extractCurrentTask(messages []Message) string {
	if len(messages) == 0 {
		return "No activity"
	}
	
	// First, look for recent meaningful user messages (last 20 messages)
	searchLimit := 20
	if len(messages) < searchLimit {
		searchLimit = len(messages)
	}
	
	for i := len(messages) - 1; i >= len(messages)-searchLimit; i-- {
		msg := messages[i]
		if msg.Role == "user" && len(msg.Content) > 10 {
			content := strings.TrimSpace(msg.Content)
			
			// Skip tool results and other system-generated content
			if strings.Contains(content, `"tool_use_id"`) ||
			   strings.Contains(content, `"type":"tool_result"`) ||
			   strings.HasPrefix(content, `[{"tool_use_id"`) ||
			   strings.HasPrefix(content, `[{"type":"tool_result"`) {
				continue
			}
			
			// Skip command outputs and system messages
			if strings.HasPrefix(content, "Caveat:") ||
			   strings.Contains(content, "<command-name>") ||
			   strings.Contains(content, "<local-command-stdout>") {
				continue
			}
			
			// Clean up the task description
			task := strings.ReplaceAll(content, "\n", " ")
			task = strings.Join(strings.Fields(task), " ")
			
			// Truncate long messages
			if len(task) > 80 {
				task = task[:77] + "..."
			}
			return task
		}
	}
	
	// If no recent meaningful message found, look at first few user messages
	// for the initial task description (for long sessions)
	earlySearchLimit := 10
	if len(messages) < earlySearchLimit {
		earlySearchLimit = len(messages)
	}
	
	for i := 0; i < earlySearchLimit; i++ {
		msg := messages[i]
		if msg.Role == "user" && len(msg.Content) > 20 {
			content := strings.TrimSpace(msg.Content)
			
			// Skip tool results and system messages  
			if strings.Contains(content, `"tool_use_id"`) ||
			   strings.Contains(content, `"type":"tool_result"`) ||
			   strings.HasPrefix(content, `[{"tool_use_id"`) ||
			   strings.HasPrefix(content, `[{"type":"tool_result"`) ||
			   strings.HasPrefix(content, "Caveat:") ||
			   strings.Contains(content, "<command-name>") {
				continue
			}
			
			// Clean up the task description
			task := strings.ReplaceAll(content, "\n", " ")
			task = strings.Join(strings.Fields(task), " ")
			
			// Truncate long messages
			if len(task) > 80 {
				task = task[:77] + "..."
			}
			return task
		}
	}
	
	return "Session active"
}

// updateSessionMetrics updates token usage and other metrics from a message
func updateSessionMetrics(session *Session, message Message) {
	// Accumulate token usage including cache tokens
	session.TokensUsed.InputTokens += message.Usage.InputTokens
	session.TokensUsed.OutputTokens += message.Usage.OutputTokens
	session.TokensUsed.CacheCreationInputTokens += message.Usage.CacheCreationInputTokens
	session.TokensUsed.CacheReadInputTokens += message.Usage.CacheReadInputTokens
	
	// Update totals and recalculate cost
	session.TokensUsed.UpdateTotals()
	
	// Extract file modifications from tool uses
	if tools, ok := message.Meta["tools"].([]interface{}); ok {
		for _, tool := range tools {
			if toolMap, ok := tool.(map[string]interface{}); ok {
				if toolType, ok := toolMap["type"].(string); ok {
					if toolType == "edit" || toolType == "write" || toolType == "multiedit" {
						if params, ok := toolMap["parameters"].(map[string]interface{}); ok {
							if filePath, ok := params["file_path"].(string); ok {
								// Add to files modified if not already present
								found := false
								for _, existing := range session.FilesModified {
									if existing == filePath {
										found = true
										break
									}
								}
								if !found {
									session.FilesModified = append(session.FilesModified, filePath)
								}
							}
						}
					}
				}
			}
		}
	}
}

// extractGitBranch gets the current git branch for the project
func extractGitBranch(projectPath string) string {
	// First check if the directory exists and is accessible
	if _, err := os.Stat(projectPath); err != nil {
		return ""
	}
	
	// Check if it's a git repository
	gitDir := filepath.Join(projectPath, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return ""
	}
	
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		// Try alternative command for older git versions
		cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		cmd.Dir = projectPath
		output, err = cmd.Output()
		if err != nil {
			return ""
		}
	}
	return strings.TrimSpace(string(output))
}

// extractGitWorktree determines if the project is a git worktree
func extractGitWorktree(projectPath string) string {
	// Check if this is a worktree by looking for .git file (not directory)
	gitPath := filepath.Join(projectPath, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return ""
	}
	
	// If .git is a file, it's likely a worktree
	if !info.IsDir() {
		// Read the .git file to get the actual git directory
		content, err := os.ReadFile(gitPath)
		if err != nil {
			return ""
		}
		
		// Parse the gitdir line
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "gitdir:") {
				gitDir := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
				// Extract the worktree name from the path
				if strings.Contains(gitDir, ".git/worktrees/") {
					parts := strings.Split(gitDir, "/")
					for i, part := range parts {
						if part == "worktrees" && i+1 < len(parts) {
							return parts[i+1]
						}
					}
				}
			}
		}
	}
	
	return ""
}