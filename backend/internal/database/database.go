package database

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

//go:embed schema.sql
var schemaFiles embed.FS

// Database represents the SQLite database connection
type Database struct {
	*sqlx.DB
	logger *logrus.Logger
}

// Config represents database configuration
type Config struct {
	DatabasePath string
	Logger       *logrus.Logger
}

// NewDatabase creates a new database connection and runs migrations
func NewDatabase(config Config) (*Database, error) {
	// Ensure the directory exists
	dir := filepath.Dir(config.DatabasePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open SQLite database with better concurrency settings
	dsn := config.DatabasePath + "?_journal_mode=WAL&_timeout=30000&_foreign_keys=on&_busy_timeout=30000&_synchronous=NORMAL&_cache_size=10000"
	db, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool for better concurrency
	// SQLite with WAL mode can handle multiple readers + 1 writer
	db.SetMaxOpenConns(10)           // Allow multiple concurrent read operations
	db.SetMaxIdleConns(5)            // Keep some connections ready
	db.SetConnMaxLifetime(time.Hour) // Recycle connections hourly

	database := &Database{
		DB:     db,
		logger: config.Logger,
	}

	// Run migrations
	if err := database.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Apply schema updates for existing tables
	if err := database.applySchemaUpdates(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply schema updates: %w", err)
	}

	database.logger.WithField("path", config.DatabasePath).Info("Database initialized successfully")
	return database, nil
}

// migrate runs the database migrations
func (db *Database) migrate() error {
	schemaSQL, err := schemaFiles.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Execute the schema
	if _, err := db.Exec(string(schemaSQL)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	db.logger.Info("Database migrations completed successfully")
	return nil
}

// applySchemaUpdates applies incremental schema updates for existing tables
func (db *Database) applySchemaUpdates() error {
	// Check if file_watchers table exists
	var tableExists bool
	err := db.Get(&tableExists, `
		SELECT COUNT(*) > 0 
		FROM sqlite_master 
		WHERE type='table' AND name='file_watchers'
	`)
	if err != nil {
		return fmt.Errorf("failed to check if file_watchers table exists: %w", err)
	}

	// If table doesn't exist, schema.sql will create it
	if !tableExists {
		return nil
	}

	// List of columns to check and add if missing
	columnsToCheck := []struct {
		name         string
		definition   string
		defaultValue string
	}{
		{
			name:         "import_status",
			definition:   "TEXT DEFAULT 'pending'",
			defaultValue: "'pending'",
		},
		{
			name:         "sessions_imported",
			definition:   "INTEGER DEFAULT 0",
			defaultValue: "0",
		},
		{
			name:         "messages_imported",
			definition:   "INTEGER DEFAULT 0",
			defaultValue: "0",
		},
		{
			name:         "last_error",
			definition:   "TEXT",
			defaultValue: "NULL",
		},
		{
			name:         "last_processed_position",
			definition:   "INTEGER DEFAULT 0",
			defaultValue: "0",
		},
	}

	// Check and add each column if it doesn't exist
	for _, col := range columnsToCheck {
		var columnExists bool
		err = db.Get(&columnExists, `
			SELECT COUNT(*) > 0 
			FROM pragma_table_info('file_watchers') 
			WHERE name = ?
		`, col.name)
		if err != nil {
			return fmt.Errorf("failed to check for %s column: %w", col.name, err)
		}

		// If column doesn't exist, add it
		if !columnExists {
			db.logger.Infof("Adding missing %s column to file_watchers table", col.name)

			_, err = db.Exec(fmt.Sprintf(`
				ALTER TABLE file_watchers ADD COLUMN %s %s
			`, col.name, col.definition))
			if err != nil {
				return fmt.Errorf("failed to add %s column: %w", col.name, err)
			}

			// Update any existing rows
			_, err = db.Exec(fmt.Sprintf(`
				UPDATE file_watchers SET %s = %s WHERE %s IS NULL
			`, col.name, col.defaultValue, col.name))
			if err != nil {
				return fmt.Errorf("failed to update %s values: %w", col.name, err)
			}

			db.logger.Infof("Successfully added %s column to file_watchers table", col.name)
		}
	}

	return nil
}

// CleanupStuckImports marks old running imports as failed
// This should be called during server startup to clean up orphaned imports
func (db *Database) CleanupStuckImports() error {
	db.logger.Info("Cleaning up stuck import processes...")

	// Mark imports that have been running for more than 1 hour as failed
	result, err := db.Exec(`
		UPDATE import_runs 
		SET status = 'failed', 
		    error_message = 'Import process was interrupted - cleaned up on startup',
		    end_time = CURRENT_TIMESTAMP
		WHERE status = 'running' 
		AND start_time < datetime('now', '-1 hour')
	`)
	if err != nil {
		return fmt.Errorf("failed to cleanup stuck imports: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		db.logger.WithError(err).Warn("Could not get rows affected count")
	} else if rowsAffected > 0 {
		db.logger.WithField("count", rowsAffected).Info("Cleaned up stuck import processes")
	} else {
		db.logger.Info("No stuck import processes found")
	}

	return nil
}

// CheckForMissedFiles scans for files modified since the last successful import
// This should be called during startup to catch files modified while the server was down
// Returns the number of missed files found
func (db *Database) CheckForMissedFiles(claudeDir string) (int, error) {
	db.logger.Info("Checking for files modified since last import...")

	// Get the timestamp of the last successful import
	var lastImportTime *time.Time
	err := db.Get(&lastImportTime, `
		SELECT MAX(end_time) 
		FROM import_runs 
		WHERE status = 'completed' 
		AND end_time IS NOT NULL
	`)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to get last import time: %w", err)
	}

	projectsDir := filepath.Join(claudeDir, "projects")

	// If no successful imports, we'll catch everything in the next import
	if lastImportTime == nil {
		db.logger.Info("No previous successful imports found - full import will be triggered")
		return 0, nil
	}

	db.logger.WithField("last_import", lastImportTime.Format(time.RFC3339)).Info("Last successful import time")

	// Walk through all project directories to find modified files
	missedFiles := 0
	updatedFiles := 0

	err = filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Only process .jsonl files
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}

		// Check if file was modified after last import
		if info.ModTime().After(*lastImportTime) {
			missedFiles++

			// Update or insert file watcher entry
			_, err := db.Exec(`
				INSERT INTO file_watchers (file_path, last_modified, import_status)
				VALUES (?, ?, 'pending')
				ON CONFLICT(file_path) DO UPDATE SET
					last_modified = excluded.last_modified,
					import_status = 'pending',
					updated_at = CURRENT_TIMESTAMP
			`, path, info.ModTime())

			if err != nil {
				db.logger.WithError(err).WithField("file", path).Warn("Failed to update file watcher entry")
			} else {
				updatedFiles++
			}
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to scan for missed files: %w", err)
	}

	if missedFiles > 0 {
		db.logger.WithFields(logrus.Fields{
			"missed_files":  missedFiles,
			"updated_files": updatedFiles,
			"since":         lastImportTime.Format(time.RFC3339),
		}).Info("Found files modified since last import - will be processed in next incremental import")
	} else {
		db.logger.Info("No files modified since last import")
	}

	return missedFiles, nil
}

// Close closes the database connection
func (db *Database) Close() error {
	return db.DB.Close()
}

// Transaction executes a function within a database transaction
func (db *Database) Transaction(fn func(*sqlx.Tx) error) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	err = fn(tx)
	return err
}

// Health checks database connectivity
func (db *Database) Health() error {
	return db.Ping()
}

// GetStats returns overall database statistics
func (db *Database) GetStats() (*DatabaseStats, error) {
	stats := &DatabaseStats{}

	// Total sessions
	err := db.Get(&stats.TotalSessions, "SELECT COUNT(*) FROM sessions")
	if err != nil {
		return nil, fmt.Errorf("failed to get total sessions: %w", err)
	}

	// Active sessions
	err = db.Get(&stats.ActiveSessions, "SELECT COUNT(*) FROM sessions WHERE is_active = true")
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}

	// Total messages
	err = db.Get(&stats.TotalMessages, "SELECT COUNT(*) FROM messages")
	if err != nil {
		return nil, fmt.Errorf("failed to get total messages: %w", err)
	}

	// Total tokens and cost
	err = db.Get(&stats.TotalTokens, `
		SELECT COALESCE(SUM(total_tokens), 0) FROM token_usage
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get total tokens: %w", err)
	}

	err = db.Get(&stats.TotalEstimatedCost, `
		SELECT COALESCE(SUM(estimated_cost), 0.0) FROM token_usage
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get total cost: %w", err)
	}

	// Average session duration
	err = db.Get(&stats.AverageSessionDuration, `
		SELECT COALESCE(AVG(duration_seconds / 60.0), 0.0) FROM sessions WHERE duration_seconds > 0
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get average duration: %w", err)
	}

	// Most used model
	err = db.Get(&stats.MostUsedModel, `
		SELECT COALESCE(model, 'unknown') 
		FROM sessions 
		WHERE model IS NOT NULL AND model != ''
		GROUP BY model 
		ORDER BY COUNT(*) DESC 
		LIMIT 1
	`)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get most used model: %w", err)
	}
	if err == sql.ErrNoRows {
		stats.MostUsedModel = "unknown"
	}

	return stats, nil
}

// Vacuum runs SQLite VACUUM to optimize database
func (db *Database) Vacuum() error {
	_, err := db.Exec("VACUUM")
	if err != nil {
		return fmt.Errorf("failed to vacuum database: %w", err)
	}
	db.logger.Info("Database vacuum completed")
	return nil
}

// Analyze runs SQLite ANALYZE to update query planner statistics
func (db *Database) Analyze() error {
	_, err := db.Exec("ANALYZE")
	if err != nil {
		return fmt.Errorf("failed to analyze database: %w", err)
	}
	db.logger.Debug("Database analyze completed")
	return nil
}
