package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

type ClaudeResponse struct {
	Type         string  `json:"type"`
	Subtype      string  `json:"subtype"`
	IsError      bool    `json:"is_error"`
	Result       string  `json:"result"`
	SessionID    string  `json:"session_id"`
	Error        string  `json:"error,omitempty"`
	DurationMs   int     `json:"duration_ms,omitempty"`
	NumTurns     int     `json:"num_turns,omitempty"`
	TotalCostUSD float64 `json:"total_cost_usd,omitempty"`
}

func main() {
	fmt.Println("=== Claude CLI Debug Test ===")
	fmt.Println()

	// Test different command variations
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "Test 1: With --print and --output-format json",
			args: []string{"--print", "--output-format", "json", "hello"},
		},
		{
			name: "Test 2: With --output-format json only",
			args: []string{"--output-format", "json", "hello"},
		},
		{
			name: "Test 3: With --print only",
			args: []string{"--print", "hello"},
		},
		{
			name: "Test 4: No flags",
			args: []string{"hello"},
		},
	}

	for _, test := range tests {
		fmt.Printf("\n%s\n", test.name)
		fmt.Println(strings.Repeat("-", len(test.name)))
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Try to find claude
		claudePath := "claude"
		if _, err := exec.LookPath(claudePath); err != nil {
			fmt.Printf("Claude not found in PATH, checking common locations...\n")
			homeDir, _ := os.UserHomeDir()
			possiblePaths := []string{
				"/usr/local/bin/claude",
				"/opt/homebrew/bin/claude",
				homeDir + "/.npm-global/bin/claude",
				homeDir + "/.local/bin/claude",
			}
			
			found := false
			for _, path := range possiblePaths {
				if _, err := os.Stat(path); err == nil {
					claudePath = path
					found = true
					fmt.Printf("Found claude at: %s\n", path)
					break
				}
			}
			
			if !found {
				fmt.Println("ERROR: Could not find claude CLI")
				continue
			}
		}

		cmd := exec.CommandContext(ctx, claudePath, test.args...)
		cmd.Dir = "/tmp" // Use a safe directory

		fmt.Printf("Command: %s %s\n", claudePath, strings.Join(test.args, " "))
		
		// Method 1: Using StdoutPipe
		fmt.Println("\nMethod 1: StdoutPipe + Start/Wait")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Printf("ERROR getting stdout pipe: %v\n", err)
			continue
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			fmt.Printf("ERROR getting stderr pipe: %v\n", err)
			continue
		}

		start := time.Now()
		if err := cmd.Start(); err != nil {
			fmt.Printf("ERROR starting command: %v\n", err)
			continue
		}

		// Read stdout
		output, err := io.ReadAll(stdout)
		if err != nil {
			fmt.Printf("ERROR reading stdout: %v\n", err)
		}

		// Read stderr
		errOutput, err := io.ReadAll(stderr)
		if err != nil {
			fmt.Printf("ERROR reading stderr: %v\n", err)
		}

		// Wait for completion
		waitErr := cmd.Wait()
		duration := time.Since(start)
		
		fmt.Printf("Duration: %v\n", duration)
		fmt.Printf("Exit error: %v\n", waitErr)
		fmt.Printf("Stdout length: %d bytes\n", len(output))
		fmt.Printf("Stderr length: %d bytes\n", len(errOutput))
		
		if len(output) > 0 {
			fmt.Printf("Stdout preview: %s\n", truncate(string(output), 200))
			
			// Try to parse as JSON
			var resp ClaudeResponse
			if err := json.Unmarshal(output, &resp); err == nil {
				fmt.Printf("Parsed JSON response:\n")
				fmt.Printf("  Session ID: %s\n", resp.SessionID)
				fmt.Printf("  Result: %s\n", truncate(resp.Result, 100))
				fmt.Printf("  Is Error: %v\n", resp.IsError)
			}
		}
		
		if len(errOutput) > 0 {
			fmt.Printf("Stderr: %s\n", string(errOutput))
		}

		// Method 2: Using CombinedOutput (for comparison)
		fmt.Println("\nMethod 2: CombinedOutput (with timeout)")
		cmd2 := exec.CommandContext(ctx, claudePath, test.args...)
		cmd2.Dir = "/tmp"
		
		done := make(chan struct{})
		var output2 []byte
		var err2 error
		
		go func() {
			start := time.Now()
			output2, err2 = cmd2.CombinedOutput()
			fmt.Printf("CombinedOutput completed in %v\n", time.Since(start))
			close(done)
		}()
		
		select {
		case <-done:
			fmt.Printf("CombinedOutput error: %v\n", err2)
			fmt.Printf("CombinedOutput length: %d bytes\n", len(output2))
			if len(output2) > 0 {
				fmt.Printf("CombinedOutput preview: %s\n", truncate(string(output2), 200))
			}
		case <-time.After(5 * time.Second):
			fmt.Println("CombinedOutput timed out after 5 seconds")
			cmd2.Process.Kill()
		}
	}

	// Test environment variables
	fmt.Println("\n=== Environment Check ===")
	envVars := []string{"CLAUDE_API_KEY", "ANTHROPIC_API_KEY", "HOME", "PATH"}
	for _, env := range envVars {
		if val := os.Getenv(env); val != "" {
			if env == "CLAUDE_API_KEY" || env == "ANTHROPIC_API_KEY" {
				fmt.Printf("%s: [SET]\n", env)
			} else {
				fmt.Printf("%s: %s\n", env, val)
			}
		} else {
			fmt.Printf("%s: [NOT SET]\n", env)
		}
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}