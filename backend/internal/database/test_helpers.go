package database

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func init() {
	// Set up test logger
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
}

// setupTestDB creates a temporary SQLite database for testing
func setupTestDB(t *testing.T) (*Database, func()) {
	// Create temporary database file
	tmpFile, err := os.CreateTemp("", "test-claude-session-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	// Initialize database
	config := Config{
		DatabasePath: tmpFile.Name(),
		Logger:       logger,
	}
	db, err := NewDatabase(config)
	if err != nil {
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to create database: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		db.Close()
		os.Remove(tmpFile.Name())
	}

	return db, cleanup
}