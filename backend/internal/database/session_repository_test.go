package database

import (
	"testing"
	"time"
)

func TestSessionRepository_GetRecentFiles(t *testing.T) {
	// Setup test database
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewSessionRepository(db, logger)

	// Create test data
	sessionID := "test-session-123"
	session := &Session{
		ID:          sessionID,
		ProjectPath: "/test/project",
		ProjectName: "test-project",
		GitBranch:   "main",
		StartTime:   time.Now().Add(-1 * time.Hour),
		Status:      "active",
	}
	
	if err := repo.UpsertSession(session); err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Create test messages
	messageID := "test-msg-123"
	message := &Message{
		ID:        messageID,
		SessionID: sessionID,
		Role:      "assistant",
		Content:   `{"content": "Modified file"}`,
		Timestamp: time.Now(),
	}
	
	if err := repo.UpsertMessage(message); err != nil {
		t.Fatalf("Failed to create test message: %v", err)
	}

	// Create tool results
	toolResults := []*ToolResult{
		{
			MessageID:  messageID,
			SessionID:  sessionID,
			ToolName:   "Edit",
			FilePath:   stringPtr("/test/project/src/app.ts"),
			ResultData: `{"status": "success"}`,
			Timestamp:  time.Now().Add(-30 * time.Minute),
		},
		{
			MessageID:  messageID,
			SessionID:  sessionID,
			ToolName:   "Write",
			FilePath:   stringPtr("/test/project/src/config.json"),
			ResultData: `{"status": "success"}`,
			Timestamp:  time.Now().Add(-20 * time.Minute),
		},
		{
			MessageID:  messageID,
			SessionID:  sessionID,
			ToolName:   "Edit",
			FilePath:   stringPtr("/test/project/src/app.ts"),
			ResultData: `{"status": "success"}`,
			Timestamp:  time.Now().Add(-10 * time.Minute),
		},
	}

	for _, tr := range toolResults {
		if err := repo.UpsertToolResult(tr); err != nil {
			t.Fatalf("Failed to create tool result: %v", err)
		}
	}

	// Test GetRecentFiles
	t.Run("GetRecentFiles", func(t *testing.T) {
		files, total, err := repo.GetRecentFiles(10, 0)
		if err != nil {
			t.Fatalf("GetRecentFiles failed: %v", err)
		}

		if total != 2 {
			t.Errorf("Expected 2 unique files, got %d", total)
		}

		if len(files) != 2 {
			t.Errorf("Expected 2 files in result, got %d", len(files))
		}

		// Verify file details
		for _, file := range files {
			if file.SessionID != sessionID {
				t.Errorf("Expected session ID %s, got %s", sessionID, file.SessionID)
			}
			if file.ProjectName != "test-project" {
				t.Errorf("Expected project name 'test-project', got %s", file.ProjectName)
			}
			if file.SessionTitle != "test-project - main" {
				t.Errorf("Expected session title 'test-project - main', got %s", file.SessionTitle)
			}
		}
	})

	// Test pagination
	t.Run("GetRecentFiles_Pagination", func(t *testing.T) {
		files, _, err := repo.GetRecentFiles(1, 1)
		if err != nil {
			t.Fatalf("GetRecentFiles with pagination failed: %v", err)
		}

		if len(files) != 1 {
			t.Errorf("Expected 1 file with limit=1, got %d", len(files))
		}
	})
}

func TestSessionRepository_GetProjectRecentFiles(t *testing.T) {
	// Setup test database
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewSessionRepository(db, logger)

	// Create test sessions for different projects
	sessions := []*Session{
		{
			ID:          "session-1",
			ProjectPath: "/test/project1",
			ProjectName: "project1",
			GitBranch:   "main",
			StartTime:   time.Now().Add(-2 * time.Hour),
			Status:      "active",
		},
		{
			ID:          "session-2",
			ProjectPath: "/test/project1",
			ProjectName: "project1",
			GitBranch:   "feature",
			StartTime:   time.Now().Add(-1 * time.Hour),
			Status:      "active",
		},
		{
			ID:          "session-3",
			ProjectPath: "/test/project2",
			ProjectName: "project2",
			GitBranch:   "main",
			StartTime:   time.Now().Add(-30 * time.Minute),
			Status:      "active",
		},
	}

	for _, session := range sessions {
		if err := repo.UpsertSession(session); err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Create message for each session
		message := &Message{
			ID:        "msg-" + session.ID,
			SessionID: session.ID,
			Role:      "assistant",
			Content:   `{"content": "Modified file"}`,
			Timestamp: time.Now(),
		}
		
		if err := repo.UpsertMessage(message); err != nil {
			t.Fatalf("Failed to create message: %v", err)
		}

		// Create tool results
		toolResult := &ToolResult{
			MessageID:  message.ID,
			SessionID:  session.ID,
			ToolName:   "Edit",
			FilePath:   stringPtr("/test/file.ts"),
			ResultData: `{"status": "success"}`,
			Timestamp:  time.Now(),
		}
		
		if err := repo.UpsertToolResult(toolResult); err != nil {
			t.Fatalf("Failed to create tool result: %v", err)
		}
	}

	// Test GetProjectRecentFiles
	t.Run("GetProjectRecentFiles", func(t *testing.T) {
		files, err := repo.GetProjectRecentFiles("project1", 10, nil)
		if err != nil {
			t.Fatalf("GetProjectRecentFiles failed: %v", err)
		}

		if len(files) != 1 {
			t.Errorf("Expected 1 file for project1, got %d", len(files))
		}

		if len(files) > 0 {
			file := files[0]
			if len(file.Sessions) != 2 {
				t.Errorf("Expected 2 sessions for the file, got %d", len(file.Sessions))
			}
			if file.TotalModifications != 2 {
				t.Errorf("Expected 2 total modifications, got %d", file.TotalModifications)
			}
		}
	})

	// Test branch filtering
	t.Run("GetProjectRecentFiles_BranchFilter", func(t *testing.T) {
		branch := "feature"
		files, err := repo.GetProjectRecentFiles("project1", 10, &branch)
		if err != nil {
			t.Fatalf("GetProjectRecentFiles with branch filter failed: %v", err)
		}

		if len(files) != 1 {
			t.Errorf("Expected 1 file for branch 'feature', got %d", len(files))
		}

		if len(files) > 0 && len(files[0].Sessions) != 1 {
			t.Errorf("Expected 1 session for branch 'feature', got %d", len(files[0].Sessions))
		}
	})
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}