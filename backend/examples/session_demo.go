package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/ksred/claude-session-manager/internal/claude"
)

func main() {
	fmt.Println("Claude Session Manager - Demo")
	fmt.Println("=============================")

	// Discover all sessions
	sessions, err := claude.DiscoverSessions()
	if err != nil {
		log.Printf("Error discovering sessions: %v", err)
	}

	fmt.Printf("\nFound %d sessions\n\n", len(sessions))

	// Display each session
	for i, session := range sessions {
		fmt.Printf("Session %d:\n", i+1)
		fmt.Printf("  ID: %s\n", session.ID)
		fmt.Printf("  Project: %s (%s)\n", session.ProjectName, session.ProjectPath)
		fmt.Printf("  Status: %s\n", session.Status)
		fmt.Printf("  Git Branch: %s\n", session.GitBranch)
		if session.GitWorktree != "" {
			fmt.Printf("  Git Worktree: %s\n", session.GitWorktree)
		}
		fmt.Printf("  Current Task: %s\n", session.CurrentTask)
		fmt.Printf("  Duration: %s\n", session.Duration())
		fmt.Printf("  Messages: %d (User: %d, Assistant: %d)\n",
			session.GetMessageCount(),
			session.GetUserMessageCount(),
			session.GetAssistantMessageCount())
		fmt.Printf("  Tokens: %d input, %d output (Cost: $%.4f)\n",
			session.TokensUsed.InputTokens,
			session.TokensUsed.OutputTokens,
			session.TokensUsed.EstimatedCost)
		fmt.Printf("  Files Modified: %d\n", len(session.FilesModified))
		if len(session.FilesModified) > 0 && len(session.FilesModified) <= 5 {
			for _, file := range session.FilesModified {
				fmt.Printf("    - %s\n", file)
			}
		}
		fmt.Println()
	}

	// Example: Watch for changes
	fmt.Println("\nSetting up file watcher...")
	watcher, err := claude.NewSessionWatcher(func(sessions []claude.Session) {
		fmt.Printf("\n[Update] Sessions changed. Total: %d\n", len(sessions))
	})
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}

	// Set up event callback for individual events
	watcher.SetEventCallback(func(event claude.WatchEvent) {
		eventJSON, _ := json.MarshalIndent(event, "", "  ")
		fmt.Printf("\n[Event] %s\n%s\n", event.Type, eventJSON)
	})

	if err := watcher.Start(); err != nil {
		log.Fatalf("Failed to start watcher: %v", err)
	}

	fmt.Println("Watching for session changes. Press Ctrl+C to exit.")
	select {} // Block forever
}