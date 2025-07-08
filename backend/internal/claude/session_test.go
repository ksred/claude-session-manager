package claude

import (
	"testing"
	"time"
)

// TestTokenUsageCalculations tests various token usage calculations
func TestTokenUsageCalculations(t *testing.T) {
	tests := []struct {
		name               string
		usage              TokenUsage
		expectedTotal      int
		expectedCost       float64
		customInputPrice   float64
		customOutputPrice  float64
		expectedCustomCost float64
	}{
		{
			name: "Basic token calculation",
			usage: TokenUsage{
				InputTokens:  1000,
				OutputTokens: 2000,
			},
			expectedTotal:      3000,
			expectedCost:       0.07, // (1 * 0.01) + (2 * 0.03)
			customInputPrice:   0.02,
			customOutputPrice:  0.04,
			expectedCustomCost: 0.10, // (1 * 0.02) + (2 * 0.04)
		},
		{
			name: "Zero tokens",
			usage: TokenUsage{
				InputTokens:  0,
				OutputTokens: 0,
			},
			expectedTotal:      0,
			expectedCost:       0.0,
			customInputPrice:   0.02,
			customOutputPrice:  0.04,
			expectedCustomCost: 0.0,
		},
		{
			name: "Large token counts",
			usage: TokenUsage{
				InputTokens:  100000,
				OutputTokens: 200000,
			},
			expectedTotal:      300000,
			expectedCost:       7.0, // (100 * 0.01) + (200 * 0.03)
			customInputPrice:   0.005,
			customOutputPrice:  0.015,
			expectedCustomCost: 3.5, // (100 * 0.005) + (200 * 0.015)
		},
		{
			name: "Fractional tokens (should handle properly)",
			usage: TokenUsage{
				InputTokens:  1500,
				OutputTokens: 2750,
			},
			expectedTotal:      4250,
			expectedCost:       0.0975, // (1.5 * 0.01) + (2.75 * 0.03)
			customInputPrice:   0.008,
			customOutputPrice:  0.024,
			expectedCustomCost: 0.078, // (1.5 * 0.008) + (2.75 * 0.024)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test deprecated methods
			tt.usage.UpdateTotals()
			if tt.usage.TotalTokens != tt.expectedTotal {
				t.Errorf("UpdateTotals: expected total %d, got %d", tt.expectedTotal, tt.usage.TotalTokens)
			}
			// Use tolerance for floating point comparison
			tolerance := 0.0001
			if diff := tt.usage.EstimatedCost - tt.expectedCost; diff < -tolerance || diff > tolerance {
				t.Errorf("UpdateTotals: expected cost %.4f, got %.4f", tt.expectedCost, tt.usage.EstimatedCost)
			}

			// Test CalculateCost directly
			cost := tt.usage.CalculateCost()
			if diff := cost - tt.expectedCost; diff < -tolerance || diff > tolerance {
				t.Errorf("CalculateCost: expected %.4f, got %.4f", tt.expectedCost, cost)
			}

			// Test custom pricing methods
			tt.usage.UpdateTotalsWithPricing(tt.customInputPrice, tt.customOutputPrice)
			if tt.usage.TotalTokens != tt.expectedTotal {
				t.Errorf("UpdateTotalsWithPricing: expected total %d, got %d", tt.expectedTotal, tt.usage.TotalTokens)
			}
			if diff := tt.usage.EstimatedCost - tt.expectedCustomCost; diff < -tolerance || diff > tolerance {
				t.Errorf("UpdateTotalsWithPricing: expected cost %.4f, got %.4f", tt.expectedCustomCost, tt.usage.EstimatedCost)
			}

			// Test CalculateCostWithPricing directly
			customCost := tt.usage.CalculateCostWithPricing(tt.customInputPrice, tt.customOutputPrice)
			if diff := customCost - tt.expectedCustomCost; diff < -tolerance || diff > tolerance {
				t.Errorf("CalculateCostWithPricing: expected %.4f, got %.4f", tt.expectedCustomCost, customCost)
			}
		})
	}
}

// TestSessionStatusDetermination tests session status logic
func TestSessionStatusDetermination(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		session        Session
		expectedStatus SessionStatus
	}{
		{
			name: "Empty session - Idle",
			session: Session{
				Messages: []Message{},
			},
			expectedStatus: StatusIdle,
		},
		{
			name: "Recent activity - Working",
			session: Session{
				LastActivity: now.Add(-30 * time.Second),
				Messages: []Message{
					{Type: "message", Role: "user", Content: "Test"},
				},
			},
			expectedStatus: StatusWorking,
		},
		{
			name: "Recent but idle - Idle",
			session: Session{
				LastActivity: now.Add(-5 * time.Minute),
				Messages: []Message{
					{Type: "message", Role: "user", Content: "Test"},
				},
			},
			expectedStatus: StatusIdle,
		},
		{
			name: "Old session - Complete",
			session: Session{
				LastActivity: now.Add(-30 * time.Minute),
				Messages: []Message{
					{Type: "message", Role: "user", Content: "Test"},
				},
			},
			expectedStatus: StatusComplete,
		},
		{
			name: "Session with error - Error",
			session: Session{
				LastActivity: now.Add(-1 * time.Minute),
				Messages: []Message{
					{Type: "message", Role: "user", Content: "Test"},
					{Type: "error", Role: "system", Content: "Error occurred"},
				},
			},
			expectedStatus: StatusError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.session.UpdateStatus()
			if tt.session.Status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, tt.session.Status)
			}
		})
	}
}

// TestSessionMessageCounting tests message counting methods
func TestSessionMessageCounting(t *testing.T) {
	session := Session{
		Messages: []Message{
			{Role: "user", Content: "Question 1"},
			{Role: "assistant", Content: "Answer 1"},
			{Role: "user", Content: "Question 2"},
			{Role: "assistant", Content: "Answer 2"},
			{Role: "system", Content: "System message"},
			{Role: "user", Content: "Question 3"},
		},
	}

	tests := []struct {
		name     string
		method   func() int
		expected int
	}{
		{
			name:     "Total message count",
			method:   session.GetMessageCount,
			expected: 6,
		},
		{
			name:     "User message count",
			method:   session.GetUserMessageCount,
			expected: 3,
		},
		{
			name:     "Assistant message count",
			method:   session.GetAssistantMessageCount,
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.method()
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestSessionErrorDetection tests error detection methods
func TestSessionErrorDetection(t *testing.T) {
	tests := []struct {
		name              string
		messages          []Message
		expectedHasErrors bool
		expectedErrorCount int
	}{
		{
			name: "No errors",
			messages: []Message{
				{Type: "message", Role: "user", Content: "Test"},
				{Type: "message", Role: "assistant", Content: "Response"},
			},
			expectedHasErrors:  false,
			expectedErrorCount: 0,
		},
		{
			name: "Single error",
			messages: []Message{
				{Type: "message", Role: "user", Content: "Test"},
				{Type: "error", Role: "system", Content: "Error occurred"},
				{Type: "message", Role: "assistant", Content: "Recovery"},
			},
			expectedHasErrors:  true,
			expectedErrorCount: 1,
		},
		{
			name: "Multiple errors",
			messages: []Message{
				{Type: "error", Role: "system", Content: "Error 1"},
				{Type: "message", Role: "user", Content: "Retry"},
				{Type: "error", Role: "system", Content: "Error 2"},
			},
			expectedHasErrors:  true,
			expectedErrorCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := Session{Messages: tt.messages}
			
			hasErrors := session.HasErrors()
			if hasErrors != tt.expectedHasErrors {
				t.Errorf("HasErrors: expected %v, got %v", tt.expectedHasErrors, hasErrors)
			}

			errorMessages := session.GetErrorMessages()
			if len(errorMessages) != tt.expectedErrorCount {
				t.Errorf("GetErrorMessages: expected %d errors, got %d", tt.expectedErrorCount, len(errorMessages))
			}

			// Verify all returned messages are actually errors
			for _, msg := range errorMessages {
				if msg.Type != "error" {
					t.Errorf("GetErrorMessages returned non-error message: %+v", msg)
				}
			}
		})
	}
}

// TestSessionDurationCalculations tests duration-related methods
func TestSessionDurationCalculations(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name                   string
		session                Session
		expectedDurationRange  [2]time.Duration // min, max
		expectedIsActive       bool
		expectedSinceActivity  time.Duration
	}{
		{
			name: "Active session",
			session: Session{
				StartTime:    now.Add(-1 * time.Hour),
				LastActivity: now.Add(-30 * time.Second),
			},
			expectedDurationRange:  [2]time.Duration{59*time.Minute + 30*time.Second, 60*time.Minute + 30*time.Second},
			expectedIsActive:       true,
			expectedSinceActivity:  30 * time.Second,
		},
		{
			name: "Idle session",
			session: Session{
				StartTime:    now.Add(-2 * time.Hour),
				LastActivity: now.Add(-5 * time.Minute),
			},
			expectedDurationRange:  [2]time.Duration{115 * time.Minute, 116 * time.Minute},
			expectedIsActive:       false,
			expectedSinceActivity:  5 * time.Minute,
		},
		{
			name: "Zero times",
			session: Session{
				StartTime:    time.Time{},
				LastActivity: time.Time{},
			},
			expectedDurationRange:  [2]time.Duration{0, 0},
			expectedIsActive:       false,
			expectedSinceActivity:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := tt.session.Duration()
			if duration < tt.expectedDurationRange[0] || duration > tt.expectedDurationRange[1] {
				t.Errorf("Duration out of expected range: got %v, expected between %v and %v",
					duration, tt.expectedDurationRange[0], tt.expectedDurationRange[1])
			}

			isActive := tt.session.IsActive()
			if isActive != tt.expectedIsActive {
				t.Errorf("IsActive: expected %v, got %v", tt.expectedIsActive, isActive)
			}

			// For non-zero times, verify TimeSinceLastActivity
			if !tt.session.LastActivity.IsZero() {
				sinceActivity := tt.session.TimeSinceLastActivity()
				// Allow 1 second tolerance for test execution time
				tolerance := 1 * time.Second
				if sinceActivity < tt.expectedSinceActivity-tolerance || 
				   sinceActivity > tt.expectedSinceActivity+tolerance {
					t.Errorf("TimeSinceLastActivity: expected ~%v, got %v", 
						tt.expectedSinceActivity, sinceActivity)
				}
			}
		})
	}
}

// TestGetLastUserMessage tests retrieving the last user message
func TestGetLastUserMessage(t *testing.T) {
	tests := []struct {
		name            string
		messages        []Message
		expectedContent string
	}{
		{
			name:            "No messages",
			messages:        []Message{},
			expectedContent: "",
		},
		{
			name: "Only assistant messages",
			messages: []Message{
				{Role: "assistant", Content: "Hello"},
				{Role: "system", Content: "System"},
			},
			expectedContent: "",
		},
		{
			name: "Single user message",
			messages: []Message{
				{Role: "user", Content: "My question"},
			},
			expectedContent: "My question",
		},
		{
			name: "Mixed messages",
			messages: []Message{
				{Role: "user", Content: "First question"},
				{Role: "assistant", Content: "First answer"},
				{Role: "user", Content: "Second question"},
				{Role: "assistant", Content: "Second answer"},
			},
			expectedContent: "Second question",
		},
		{
			name: "User message not last",
			messages: []Message{
				{Role: "user", Content: "Question"},
				{Role: "assistant", Content: "Answer"},
				{Role: "system", Content: "Info"},
			},
			expectedContent: "Question",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := Session{Messages: tt.messages}
			content := session.GetLastUserMessage()
			if content != tt.expectedContent {
				t.Errorf("Expected '%s', got '%s'", tt.expectedContent, content)
			}
		})
	}
}

// TestSessionStatusString tests the String() method of SessionStatus
func TestSessionStatusString(t *testing.T) {
	tests := []struct {
		status   SessionStatus
		expected string
	}{
		{StatusWorking, "Working"},
		{StatusIdle, "Idle"},
		{StatusComplete, "Complete"},
		{StatusError, "Error"},
		{SessionStatus(999), "Unknown"}, // Invalid status
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.status.String()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestSessionWithEdgeCases tests edge cases and boundary conditions
func TestSessionWithEdgeCases(t *testing.T) {
	t.Run("Session with nil values", func(t *testing.T) {
		session := Session{
			Messages:      nil,
			FilesModified: nil,
		}
		
		// Should handle nil slices gracefully
		if session.GetMessageCount() != 0 {
			t.Error("Expected 0 messages for nil slice")
		}
		if session.HasErrors() {
			t.Error("Expected no errors for nil slice")
		}
	})

	t.Run("Session activity boundary", func(t *testing.T) {
		// Test the exact 2-minute boundary for IsActive
		session := Session{
			LastActivity: time.Now().Add(-2 * time.Minute),
		}
		
		// Due to timing, this might be slightly over 2 minutes
		// So we test that it's considered inactive
		if session.IsActive() {
			t.Error("Session at 2-minute boundary should be inactive")
		}
		
		// Just under 2 minutes should be active
		session.LastActivity = time.Now().Add(-119 * time.Second)
		if !session.IsActive() {
			t.Error("Session just under 2 minutes should be active")
		}
	})

	t.Run("Empty message content", func(t *testing.T) {
		session := Session{
			Messages: []Message{
				{Role: "user", Content: ""},
				{Role: "user", Content: "Valid content"},
			},
		}
		
		lastMsg := session.GetLastUserMessage()
		if lastMsg != "Valid content" {
			t.Errorf("Expected 'Valid content', got '%s'", lastMsg)
		}
	})
}