package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/ksred/claude-session-manager/internal/database"
	"github.com/sirupsen/logrus"
)

func main() {
	var (
		dbPath   = flag.String("db", "", "Path to SQLite database file")
		filePath = flag.String("file", "", "Path to JSONL file to import")
		dataDir  = flag.String("data-dir", "", "Claude data directory (alternative to -db)")
	)
	flag.Parse()

	if *filePath == "" {
		log.Fatal("File path is required")
	}

	// Determine database path
	var dbFile string
	if *dbPath != "" {
		dbFile = *dbPath
	} else if *dataDir != "" {
		dbFile = filepath.Join(*dataDir, "sessions.db")
	} else {
		// Try default location
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Failed to get home directory:", err)
		}
		dbFile = filepath.Join(homeDir, ".claude", "sessions.db")
	}

	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Open database
	config := database.Config{
		DatabasePath: dbFile,
		Logger:       logger,
	}
	db, err := database.NewDatabase(config)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Create repository
	repo := database.NewSessionRepository(db, logger)

	// Parse project info from file path
	projectDir := filepath.Base(filepath.Dir(*filePath))
	
	projectInfo := database.ProjectInfo{
		ProjectPath: projectDir,
		ProjectName: "claude-session-manager-debug-second-calls-failing",
		FilePath:    *filePath,
	}

	// Create batch importer
	importer := database.NewBatchImporter(repo, logger)

	// Import the file
	logger.WithField("file", *filePath).Info("Starting file import")
	sessions, messages, err := importer.ImportJSONLFileOptimized(*filePath, projectInfo)
	if err != nil {
		logger.WithError(err).Fatal("Failed to import file")
	}

	logger.WithFields(logrus.Fields{
		"database_file":     dbFile,
		"sessions_imported": sessions,
		"messages_imported": messages,
	}).Info("Import completed successfully")
	
	// Verify the data was actually persisted
	logger.Debug("Verifying data persistence")
	var messageCount int
	err = repo.GetDB().Get(&messageCount, "SELECT COUNT(*) FROM messages WHERE session_id = ?", "bd16b52b-ab7d-4a22-b09b-8b1bd2c77a94")
	if err != nil {
		logger.WithError(err).Error("Error checking message count")
	} else {
		logger.WithField("count", messageCount).Info("Messages verified in database")
	}
}