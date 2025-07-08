package main

import (
	"fmt"
	"time"

	"github.com/ksred/claude-session-manager/internal/claude"
)

// printFallback prints a simple text-based version when TUI fails
func printFallback() {
	fmt.Println("Claude Session Manager - Text Mode")
	fmt.Println("==================================")
	fmt.Println()
	
	// Try to discover sessions
	sessions, err := claude.DiscoverSessions()
	if err != nil {
		fmt.Printf("Error discovering sessions: %v\n", err)
		fmt.Println("Creating demo data...")
		
		// Create mock session
		mockSession := claude.Session{
			ID:          "demo-session",
			ProjectName: "claude-session-manager",
			GitBranch:   "main",
			Status:      claude.StatusWorking,
			StartTime:   time.Now().Add(-2 * time.Hour),
			LastActivity: time.Now().Add(-30 * time.Second),
			CurrentTask: "Building terminal dashboard...",
			TokensUsed: claude.TokenUsage{
				InputTokens:   25000,
				OutputTokens:  17000,
				TotalTokens:   42000,
				EstimatedCost: 0.84,
			},
			FilesModified: []string{
				"internal/ui/app.go",
				"cmd/main.go",
				"README.md",
			},
		}
		sessions = []claude.Session{mockSession}
	}
	
	if len(sessions) == 0 {
		fmt.Println("No active Claude sessions found.")
		fmt.Println("Start a Claude Code session and run this tool again.")
		return
	}
	
	fmt.Printf("Found %d active session(s):\n\n", len(sessions))
	
	for i, session := range sessions {
		fmt.Printf("Session %d:\n", i+1)
		fmt.Printf("  Project: %s\n", session.ProjectName)
		fmt.Printf("  Branch:  %s\n", session.GitBranch)
		fmt.Printf("  Status:  %s\n", session.Status.String())
		fmt.Printf("  Task:    %s\n", session.CurrentTask)
		fmt.Printf("  Tokens:  %d (Cost: $%.2f)\n", session.TokensUsed.TotalTokens, session.TokensUsed.EstimatedCost)
		fmt.Printf("  Files:   %d modified\n", len(session.FilesModified))
		
		duration := session.Duration()
		lastActivity := session.TimeSinceLastActivity()
		fmt.Printf("  Time:    %v total, %v since last activity\n", duration.Round(time.Minute), lastActivity.Round(time.Second))
		fmt.Println()
	}
	
	fmt.Println("For the full interactive dashboard, ensure you're running in a supported terminal.")
}