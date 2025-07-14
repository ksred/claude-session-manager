package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type ClaudeResponse struct {
	Type         string  `json:"type"`
	Result       string  `json:"result"`
	SessionID    string  `json:"session_id"`
}

func main() {
	fmt.Println("=== Testing Claude CLI with specific project path ===")
	
	// Use the same project path from the logs
	projectPath := "/Users/ksred/Documents/GitHub/pprofio-server"
	
	// Check if the directory exists
	if info, err := os.Stat(projectPath); err != nil {
		fmt.Printf("ERROR: Project path doesn't exist: %v\n", err)
		return
	} else if !info.IsDir() {
		fmt.Printf("ERROR: Project path is not a directory\n")
		return
	}
	
	fmt.Printf("Project path exists: %s\n", projectPath)
	
	// Test with the exact command from the logs
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "claude", "--print", "--output-format", "json", "hello")
	cmd.Dir = projectPath
	
	fmt.Printf("Command: claude --print --output-format json hello\n")
	fmt.Printf("Working directory: %s\n", projectPath)
	
	// Test 1: CombinedOutput (the method that's hanging)
	fmt.Println("\nTest 1: CombinedOutput")
	done := make(chan struct{})
	var output []byte
	var err error
	
	go func() {
		start := time.Now()
		fmt.Println("Starting CombinedOutput...")
		output, err = cmd.CombinedOutput()
		fmt.Printf("CombinedOutput completed in %v\n", time.Since(start))
		close(done)
	}()
	
	select {
	case <-done:
		fmt.Printf("Success! Output length: %d bytes\n", len(output))
		fmt.Printf("Error: %v\n", err)
		if len(output) > 0 {
			fmt.Printf("Output: %s\n", string(output))
			
			var resp ClaudeResponse
			if err := json.Unmarshal(output, &resp); err == nil {
				fmt.Printf("\nParsed response:\n")
				fmt.Printf("  Session ID: %s\n", resp.SessionID)
				fmt.Printf("  Result preview: %.100s...\n", resp.Result)
			}
		}
	case <-time.After(10 * time.Second):
		fmt.Println("ERROR: CombinedOutput timed out after 10 seconds (hanging)")
		// Try to kill the process
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}
	
	// Test 2: Check environment
	fmt.Println("\n=== Environment Check ===")
	fmt.Printf("Current user: %s\n", os.Getenv("USER"))
	fmt.Printf("Shell: %s\n", os.Getenv("SHELL"))
	
	// Check if there's a .claude directory in the project
	claudeDir := projectPath + "/.claude"
	if _, err := os.Stat(claudeDir); err == nil {
		fmt.Printf(".claude directory exists in project\n")
	} else {
		fmt.Printf(".claude directory does not exist in project\n")
	}
	
	// Check for any CLAUDE.md files
	claudeMd := projectPath + "/CLAUDE.md"
	if _, err := os.Stat(claudeMd); err == nil {
		fmt.Printf("CLAUDE.md exists in project\n")
	} else {
		fmt.Printf("CLAUDE.md does not exist in project\n")
	}
}