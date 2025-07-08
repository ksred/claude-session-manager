package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ksred/claude-session-manager/internal/config"
)

func TestTokenUsage(t *testing.T) {
	tu := TokenUsage{
		InputTokens:  1500,
		OutputTokens: 2500,
	}
	
	tu.UpdateTotals()
	
	if tu.TotalTokens != 4000 {
		t.Errorf("Expected TotalTokens to be 4000, got %d", tu.TotalTokens)
	}
	
	expectedCost := (1.5 * InputTokenPricePerK) + (2.5 * OutputTokenPricePerK)
	if tu.EstimatedCost != expectedCost {
		t.Errorf("Expected EstimatedCost to be %f, got %f", expectedCost, tu.EstimatedCost)
	}
}

func TestSessionStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   SessionStatus
		expected string
	}{
		{"Working", StatusWorking, "Working"},
		{"Idle", StatusIdle, "Idle"},
		{"Complete", StatusComplete, "Complete"},
		{"Error", StatusError, "Error"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("SessionStatus.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSessionDuration(t *testing.T) {
	session := Session{
		StartTime:    time.Now().Add(-2 * time.Hour),
		LastActivity: time.Now().Add(-30 * time.Minute),
	}
	
	duration := session.Duration()
	// Should be approximately 1.5 hours
	if duration < 85*time.Minute || duration > 95*time.Minute {
		t.Errorf("Unexpected duration: %v", duration)
	}
}

func TestExtractProjectPath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "URL encoded path",
			filePath: "/home/user/.claude/projects/%2Fhome%2Fuser%2Fprojects%2Fmyapp/session.jsonl",
			expected: "/home/user/projects/myapp",
		},
		{
			name:     "Path with spaces",
			filePath: "/home/user/.claude/projects/%2Fhome%2Fuser%2FMy%20Projects%2Fapp/session.jsonl",
			expected: "/home/user/My Projects/app",
		},
		{
			name:     "No projects directory",
			filePath: "/home/user/.claude/session.jsonl",
			expected: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractProjectPath(tt.filePath)
			if got != tt.expected {
				t.Errorf("extractProjectPath() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestExtractProjectName(t *testing.T) {
	tests := []struct {
		name        string
		projectPath string
		expected    string
	}{
		{"Normal path", "/home/user/projects/myapp", "myapp"},
		{"Empty path", "", "Unknown Project"},
		{"Root path", "/", "/"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractProjectName(tt.projectPath)
			if got != tt.expected {
				t.Errorf("extractProjectName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestDiscoverSessions tests the session discovery functionality
func TestDiscoverSessions(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "claude-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test directory structure
	projectsDir := filepath.Join(tempDir, "projects")
	projectPath := filepath.Join(projectsDir, "%2Fhome%2Fuser%2Fproject1")
	err = os.MkdirAll(projectPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Create a test session file
	sessionFile := filepath.Join(projectPath, "session1.jsonl")
	testMessages := []Message{
		{
			ID:        "msg1",
			Type:      "message",
			Role:      "user",
			Content:   "Hello Claude",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Usage: TokenUsage{
				InputTokens:  100,
				OutputTokens: 0,
			},
		},
		{
			ID:        "msg2",
			Type:      "message",
			Role:      "assistant",
			Content:   "Hello! How can I help you?",
			Timestamp: time.Now().Add(-50 * time.Minute),
			Usage: TokenUsage{
				InputTokens:  0,
				OutputTokens: 150,
			},
		},
	}

	file, err := os.Create(sessionFile)
	if err != nil {
		t.Fatalf("Failed to create session file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, msg := range testMessages {
		err = encoder.Encode(msg)
		if err != nil {
			t.Fatalf("Failed to write message: %v", err)
		}
	}

	// Create a summary file that should be ignored
	summaryFile := filepath.Join(projectPath, "summary.jsonl")
	err = os.WriteFile(summaryFile, []byte("summary data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create summary file: %v", err)
	}

	// Test discovery with custom config
	cfg := &config.Config{
		Claude: config.ClaudeConfig{
			ProjectsPath: projectsDir,
		},
	}

	sessions, err := DiscoverSessionsWithConfig(cfg)
	if err != nil {
		t.Fatalf("DiscoverSessionsWithConfig failed: %v", err)
	}

	// Verify results
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	if len(sessions) > 0 {
		session := sessions[0]
		if session.ProjectName != "project1" {
			t.Errorf("Expected project name 'project1', got '%s'", session.ProjectName)
		}
		if session.ProjectPath != "/home/user/project1" {
			t.Errorf("Expected project path '/home/user/project1', got '%s'", session.ProjectPath)
		}
		if len(session.Messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(session.Messages))
		}
		if session.TokensUsed.InputTokens != 100 {
			t.Errorf("Expected 100 input tokens, got %d", session.TokensUsed.InputTokens)
		}
		if session.TokensUsed.OutputTokens != 150 {
			t.Errorf("Expected 150 output tokens, got %d", session.TokensUsed.OutputTokens)
		}
	}
}

// TestDiscoverSessionsEmptyDir tests behavior with empty directory
func TestDiscoverSessionsEmptyDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "claude-test-empty")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		Claude: config.ClaudeConfig{
			ProjectsPath: tempDir,
		},
	}

	sessions, err := DiscoverSessionsWithConfig(cfg)
	if err != nil {
		t.Fatalf("DiscoverSessionsWithConfig failed: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(sessions))
	}
}

// TestDiscoverSessionsNonExistentDir tests behavior with non-existent directory
func TestDiscoverSessionsNonExistentDir(t *testing.T) {
	cfg := &config.Config{
		Claude: config.ClaudeConfig{
			ProjectsPath: "/non/existent/path",
		},
	}

	sessions, err := DiscoverSessionsWithConfig(cfg)
	if err != nil {
		t.Fatalf("DiscoverSessionsWithConfig should not fail for non-existent dir: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions for non-existent dir, got %d", len(sessions))
	}
}

// TestParseSessionFile tests parsing of various session file formats
func TestParseSessionFile(t *testing.T) {
	tests := []struct {
		name            string
		messages        []interface{}
		expectedMsgCount int
		expectedError   bool
	}{
		{
			name: "Valid messages",
			messages: []interface{}{
				Message{
					ID:        "msg1",
					Role:      "user",
					Content:   "Test message",
					Timestamp: time.Now(),
				},
				Message{
					ID:        "msg2",
					Role:      "assistant",
					Content:   "Response",
					Timestamp: time.Now(),
				},
			},
			expectedMsgCount: 2,
			expectedError:   false,
		},
		{
			name: "Messages with meta timestamps",
			messages: []interface{}{
				map[string]interface{}{
					"id":      "msg1",
					"role":    "user",
					"content": "Test",
					"meta": map[string]interface{}{
						"timestamp": time.Now().Format(time.RFC3339),
					},
				},
			},
			expectedMsgCount: 1,
			expectedError:   false,
		},
		{
			name: "Mixed valid and malformed messages",
			messages: []interface{}{
				Message{ID: "msg1", Role: "user", Content: "Valid"},
				"malformed", // This will be skipped
				Message{ID: "msg2", Role: "assistant", Content: "Also valid"},
			},
			expectedMsgCount: 2,
			expectedError:   false,
		},
		{
			name:             "Empty file",
			messages:        []interface{}{},
			expectedMsgCount: 0,
			expectedError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tempFile, err := os.CreateTemp("", "session-*.jsonl")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())
			defer tempFile.Close()

			// Write test data
			encoder := json.NewEncoder(tempFile)
			for _, msg := range tt.messages {
				if err := encoder.Encode(msg); err != nil {
					t.Fatalf("Failed to write message: %v", err)
				}
			}
			tempFile.Close()

			// Parse the file
			session, err := ParseSessionFile(tempFile.Name())
			if tt.expectedError && err == nil {
				t.Errorf("Expected error but got none")
			} else if !tt.expectedError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(session.Messages) != tt.expectedMsgCount {
				t.Errorf("Expected %d messages, got %d", tt.expectedMsgCount, len(session.Messages))
			}
		})
	}
}

// TestParseSessionFileErrors tests error handling
func TestParseSessionFileErrors(t *testing.T) {
	// Test non-existent file
	_, err := ParseSessionFile("/non/existent/file.jsonl")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

// TestUpdateSessionMetrics tests metric calculation from messages
func TestUpdateSessionMetrics(t *testing.T) {
	session := &Session{}

	tests := []struct {
		name               string
		message            Message
		expectedInputTokens int
		expectedOutputTokens int
		expectedFiles      []string
	}{
		{
			name: "Message with token usage",
			message: Message{
				Usage: TokenUsage{
					InputTokens:  100,
					OutputTokens: 200,
				},
			},
			expectedInputTokens: 100,
			expectedOutputTokens: 200,
			expectedFiles:      []string{},
		},
		{
			name: "Message with file edit tool",
			message: Message{
				Usage: TokenUsage{
					InputTokens:  50,
					OutputTokens: 50,
				},
				Meta: map[string]interface{}{
					"tools": []interface{}{
						map[string]interface{}{
							"type": "edit",
							"parameters": map[string]interface{}{
								"file_path": "/test/file.go",
							},
						},
					},
				},
			},
			expectedInputTokens: 150,
			expectedOutputTokens: 250,
			expectedFiles:      []string{"/test/file.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateSessionMetrics(session, tt.message)

			if session.TokensUsed.InputTokens != tt.expectedInputTokens {
				t.Errorf("Expected %d input tokens, got %d", tt.expectedInputTokens, session.TokensUsed.InputTokens)
			}
			if session.TokensUsed.OutputTokens != tt.expectedOutputTokens {
				t.Errorf("Expected %d output tokens, got %d", tt.expectedOutputTokens, session.TokensUsed.OutputTokens)
			}
			if len(session.FilesModified) != len(tt.expectedFiles) {
				t.Errorf("Expected %d files modified, got %d", len(tt.expectedFiles), len(session.FilesModified))
			}
		})
	}
}

// TestExtractCurrentTask tests task extraction from messages
func TestExtractCurrentTask(t *testing.T) {
	tests := []struct {
		name         string
		messages     []Message
		expectedTask string
	}{
		{
			name:         "No messages",
			messages:     []Message{},
			expectedTask: "No activity",
		},
		{
			name: "Single user message",
			messages: []Message{
				{Role: "user", Content: "Help me write tests"},
			},
			expectedTask: "Help me write tests",
		},
		{
			name: "Long user message truncated",
			messages: []Message{
				{Role: "user", Content: "This is a very long message that should be truncated because it exceeds the maximum length limit set in the function"},
			},
			expectedTask: "This is a very long message that should be truncated because it exceeds the m...",
		},
		{
			name: "Multiple messages, last user message used",
			messages: []Message{
				{Role: "user", Content: "First task"},
				{Role: "assistant", Content: "Response"},
				{Role: "user", Content: "Second task"},
				{Role: "assistant", Content: "Another response"},
			},
			expectedTask: "Second task",
		},
		{
			name: "No recent user messages",
			messages: []Message{
				{Role: "system", Content: "System message"},
				{Role: "assistant", Content: "Assistant message"},
			},
			expectedTask: "Session active",
		},
		{
			name: "User message with newlines",
			messages: []Message{
				{Role: "user", Content: "Multi\nline\nmessage"},
			},
			expectedTask: "Multi line message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := extractCurrentTask(tt.messages)
			if task != tt.expectedTask {
				t.Errorf("Expected task '%s', got '%s'", tt.expectedTask, task)
			}
		})
	}
}

// TestExtractSessionID tests session ID extraction from file paths
func TestExtractSessionID(t *testing.T) {
	tests := []struct {
		name       string
		filePath   string
		expectedID string
	}{
		{
			name:       "Standard session file",
			filePath:   "/home/user/.claude/projects/project/session123.jsonl",
			expectedID: "session123",
		},
		{
			name:       "UUID session ID",
			filePath:   "/path/to/550e8400-e29b-41d4-a716-446655440000.jsonl",
			expectedID: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:       "File with path separators",
			filePath:   "/Users/test/.claude/session.jsonl",
			expectedID: "session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := extractSessionID(tt.filePath)
			if id != tt.expectedID {
				t.Errorf("Expected ID '%s', got '%s'", tt.expectedID, id)
			}
		})
	}
}

// TestDecodePath tests URL decoding functionality
func TestDecodePath(t *testing.T) {
	tests := []struct {
		name        string
		encodedPath string
		expectedPath string
	}{
		{
			name:        "Simple URL encoded path",
			encodedPath: "%2Fhome%2Fuser%2Fproject",
			expectedPath: "/home/user/project",
		},
		{
			name:        "Path with spaces",
			encodedPath: "%2Fhome%2Fuser%2FMy%20Documents%2Fproject",
			expectedPath: "/home/user/My Documents/project",
		},
		{
			name:        "Path with special characters",
			encodedPath: "%2Fhome%2Fuser%2Fproject%2B%2B%21%40%23",
			expectedPath: "/home/user/project++!@#",
		},
		{
			name:        "Plain path without encoding",
			encodedPath: "project",
			expectedPath: "project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test through extractProjectPath which uses URL decoding
			fullPath := fmt.Sprintf("/home/.claude/projects/%s/session.jsonl", tt.encodedPath)
			decodedPath := extractProjectPath(fullPath)
			if decodedPath != tt.expectedPath {
				t.Errorf("Expected decoded path '%s', got '%s'", tt.expectedPath, decodedPath)
			}
		})
	}
}