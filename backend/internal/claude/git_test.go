package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractGitBranch(t *testing.T) {
	// Test with non-existent directory
	branch := extractGitBranch("/non/existent/path")
	if branch != "" {
		t.Errorf("Expected empty string for non-existent path, got %s", branch)
	}
	
	// Test with temp directory (not a git repo)
	tempDir, err := os.MkdirTemp("", "claude-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	branch = extractGitBranch(tempDir)
	if branch != "" {
		t.Errorf("Expected empty string for non-git directory, got %s", branch)
	}
}

func TestExtractGitWorktree(t *testing.T) {
	// Test with non-existent directory
	worktree := extractGitWorktree("/non/existent/path")
	if worktree != "" {
		t.Errorf("Expected empty string for non-existent path, got %s", worktree)
	}
	
	// Test with regular directory
	tempDir, err := os.MkdirTemp("", "claude-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	worktree = extractGitWorktree(tempDir)
	if worktree != "" {
		t.Errorf("Expected empty string for non-git directory, got %s", worktree)
	}
	
	// Test with mock git directory (not a worktree)
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatal(err)
	}
	
	worktree = extractGitWorktree(tempDir)
	if worktree != "" {
		t.Errorf("Expected empty string for regular git repo, got %s", worktree)
	}
	
	// Test with mock worktree
	os.RemoveAll(gitDir)
	gitFile := filepath.Join(tempDir, ".git")
	gitContent := "gitdir: /path/to/repo/.git/worktrees/feature-branch\n"
	if err := os.WriteFile(gitFile, []byte(gitContent), 0644); err != nil {
		t.Fatal(err)
	}
	
	worktree = extractGitWorktree(tempDir)
	if worktree != "feature-branch" {
		t.Errorf("Expected 'feature-branch', got %s", worktree)
	}
}