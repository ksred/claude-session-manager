package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var (
		dbPath    = flag.String("db", "", "Path to SQLite database file")
		migration = flag.String("migration", "", "Path to migration SQL file")
		dataDir   = flag.String("data-dir", "", "Claude data directory (alternative to -db)")
	)
	flag.Parse()

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

	// Check if database exists
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		log.Fatalf("Database file not found: %s", dbFile)
	}

	// Open database
	db, err := sql.Open("sqlite3", dbFile+"?_journal_mode=WAL")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Read migration file
	var migrationSQL string
	if *migration != "" {
		content, err := os.ReadFile(*migration)
		if err != nil {
			log.Fatal("Failed to read migration file:", err)
		}
		migrationSQL = string(content)
	} else {
		// Use the embedded migration
		migrationSQL = `
-- Fix total_tokens values in token_usage table
UPDATE token_usage
SET total_tokens = input_tokens + output_tokens + cache_creation_input_tokens + cache_read_input_tokens
WHERE total_tokens = 0 
AND (input_tokens > 0 OR output_tokens > 0 OR cache_creation_input_tokens > 0 OR cache_read_input_tokens > 0);
`
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		log.Fatal("Failed to begin transaction:", err)
	}
	defer tx.Rollback()

	// Execute migration
	result, err := tx.Exec(migrationSQL)
	if err != nil {
		log.Fatal("Failed to execute migration:", err)
	}

	// Get rows affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal("Failed to get rows affected:", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Fatal("Failed to commit transaction:", err)
	}

	fmt.Printf("Migration completed successfully!\n")
	fmt.Printf("Database: %s\n", dbFile)
	fmt.Printf("Rows updated: %d\n", rowsAffected)
}