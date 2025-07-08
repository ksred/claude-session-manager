package database

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

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
	db.SetMaxOpenConns(1)  // SQLite works better with single connection for writes
	db.SetMaxIdleConns(1)

	database := &Database{
		DB:     db,
		logger: config.Logger,
	}

	// Run migrations
	if err := database.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
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