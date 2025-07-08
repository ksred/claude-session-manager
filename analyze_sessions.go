package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

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

func main() {
	claudeDir := "/Users/ksred/.claude"
	projectsDir := filepath.Join(claudeDir, "projects")
	
	// Get all sessions and count by day
	sessionsByDay := make(map[string]map[string]bool) // date -> sessionID -> bool
	allSessions := make(map[string][]SessionMessage)  // sessionID -> messages
	
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		fmt.Printf("Error reading projects directory: %v\n", err)
		return
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
			sessionData, err := readSessionFile(sessionPath)
			if err != nil {
				fmt.Printf("Error reading session file %s: %v\n", sessionPath, err)
				continue
			}

			if len(sessionData) == 0 {
				continue
			}

			// Use the sessionID from the first message as the session identifier
			sessionID := sessionData[0].SessionID
			allSessions[sessionID] = sessionData
			
			// Count this session for each day it has activity
			for _, msg := range sessionData {
				dayKey := msg.Timestamp.Format("2006-01-02")
				
				if sessionsByDay[dayKey] == nil {
					sessionsByDay[dayKey] = make(map[string]bool)
				}
				
				sessionsByDay[dayKey][sessionID] = true
			}
		}
	}

	// Print results
	fmt.Printf("ACTUAL SESSION COUNTS BY DAY:\n")
	fmt.Printf("=============================\n")
	
	// Sort days for consistent output
	var days []string
	for day := range sessionsByDay {
		days = append(days, day)
	}
	sort.Strings(days)
	
	for _, day := range days {
		sessions := sessionsByDay[day]
		fmt.Printf("Date: %s, Sessions: %d\n", day, len(sessions))
		
		// Show session IDs for this day
		var sessionIDs []string
		for sessionID := range sessions {
			sessionIDs = append(sessionIDs, sessionID)
		}
		sort.Strings(sessionIDs)
		
		for _, sessionID := range sessionIDs {
			messages := allSessions[sessionID]
			fmt.Printf("  - Session %s: %d messages (first: %s, last: %s)\n", 
				sessionID, len(messages), 
				messages[0].Timestamp.Format("15:04:05"), 
				messages[len(messages)-1].Timestamp.Format("15:04:05"))
		}
		fmt.Println()
	}
	
	fmt.Printf("\nTOTAL UNIQUE SESSIONS: %d\n", len(allSessions))
	
	// Now let's simulate what GetDailyMetrics is doing
	fmt.Printf("\nSIMULATING GetDailyMetrics LOGIC:\n")
	fmt.Printf("==================================\n")
	
	now := time.Now()
	days = []string{}
	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		days = append(days, date)
	}
	
	// Reverse to get chronological order
	for i := len(days)/2 - 1; i >= 0; i-- {
		opp := len(days) - 1 - i
		days[i], days[opp] = days[opp], days[i]
	}
	
	for _, day := range days {
		sessionCount := 0
		messageCount := 0
		
		// Re-read files to simulate GetDailyMetrics
		entries, _ := os.ReadDir(projectsDir)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			projectDir := filepath.Join(projectsDir, entry.Name())
			sessionFiles, _ := os.ReadDir(projectDir)

			for _, sessionFile := range sessionFiles {
				if !strings.HasSuffix(sessionFile.Name(), ".jsonl") {
					continue
				}

				sessionPath := filepath.Join(projectDir, sessionFile.Name())
				sessionData, _ := readSessionFile(sessionPath)

				sessionsByDay := make(map[string]bool)
				
				for _, msg := range sessionData {
					dayKey := msg.Timestamp.Format("2006-01-02")
					
					if dayKey == day {
						// Count unique sessions per day
						if !sessionsByDay[dayKey] {
							sessionCount++
							sessionsByDay[dayKey] = true
						}
						
						messageCount++
					}
				}
			}
		}
		
		fmt.Printf("Date: %s, Sessions: %d, Messages: %d\n", day, sessionCount, messageCount)
	}
}

func readSessionFile(filePath string) ([]SessionMessage, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var messages []SessionMessage
	scanner := bufio.NewScanner(file)
	
	// Increase buffer size for large lines
	const maxCapacity = 1024 * 1024 * 10 // 10MB buffer
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)
	
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