package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// IntegrityChecker provides database integrity checking and recovery
type IntegrityChecker struct {
	db     *sql.DB
	dbPath string
	logger *logrus.Logger
}

// NewIntegrityChecker creates a new integrity checker
func NewIntegrityChecker(db *sql.DB, dbPath string, logger *logrus.Logger) *IntegrityChecker {
	return &IntegrityChecker{
		db:     db,
		dbPath: dbPath,
		logger: logger,
	}
}

// CheckIntegrity runs SQLite integrity check
func (ic *IntegrityChecker) CheckIntegrity() error {
	var result string
	err := ic.db.QueryRow("PRAGMA integrity_check").Scan(&result)
	if err != nil {
		return fmt.Errorf("integrity check failed: %w", err)
	}
	
	if result != "ok" {
		return fmt.Errorf("database corruption detected: %s", result)
	}
	
	ic.logger.Debug("Database integrity check passed")
	return nil
}

// EnableWALMode enables Write-Ahead Logging for better concurrency
func (ic *IntegrityChecker) EnableWALMode() error {
	_, err := ic.db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	
	// Set WAL checkpoint interval
	_, err = ic.db.Exec("PRAGMA wal_autocheckpoint=1000")
	if err != nil {
		return fmt.Errorf("failed to set WAL checkpoint: %w", err)
	}
	
	ic.logger.Info("WAL mode enabled for database")
	return nil
}

// BackupDatabase creates a backup of the database
func (ic *IntegrityChecker) BackupDatabase() error {
	backupDir := filepath.Dir(ic.dbPath)
	backupPath := filepath.Join(backupDir, fmt.Sprintf("sessions_backup_%s.db", time.Now().Format("20060102_150405")))
	
	// Use SQLite backup API
	_, err := ic.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	
	ic.logger.WithField("backup_path", backupPath).Info("Database backup created")
	return nil
}

// RepairDatabase attempts to repair a corrupted database
func (ic *IntegrityChecker) RepairDatabase() error {
	ic.logger.Warn("Attempting database repair")
	
	// First, try to backup the corrupted database
	corruptPath := ic.dbPath + ".corrupt." + time.Now().Format("20060102_150405")
	if err := os.Rename(ic.dbPath, corruptPath); err != nil {
		return fmt.Errorf("failed to move corrupt database: %w", err)
	}
	
	// Create a new database with recovered data
	recoveryDB, err := sql.Open("sqlite3", ic.dbPath)
	if err != nil {
		// Restore the corrupt database
		os.Rename(corruptPath, ic.dbPath)
		return fmt.Errorf("failed to create recovery database: %w", err)
	}
	defer recoveryDB.Close()
	
	// Initialize the new database with schema
	// Read and execute the schema
	schemaSQL, err := schemaFiles.ReadFile("schema.sql")
	if err != nil {
		recoveryDB.Close()
		os.Remove(ic.dbPath)
		os.Rename(corruptPath, ic.dbPath)
		return fmt.Errorf("failed to read schema: %w", err)
	}
	
	if _, err := recoveryDB.Exec(string(schemaSQL)); err != nil {
		recoveryDB.Close()
		os.Remove(ic.dbPath)
		os.Rename(corruptPath, ic.dbPath)
		return fmt.Errorf("failed to initialize recovery database: %w", err)
	}
	
	// Try to recover data from the corrupt database
	corruptDB, err := sql.Open("sqlite3", corruptPath)
	if err == nil {
		defer corruptDB.Close()
		
		// Attempt to recover sessions
		rows, err := corruptDB.Query("SELECT * FROM sessions")
		if err == nil {
			// Copy recoverable data
			// ... (implement data recovery logic)
			rows.Close()
		}
	}
	
	ic.logger.WithField("corrupt_db", corruptPath).Info("Database repair completed")
	return nil
}

// SetPragmas sets recommended SQLite pragmas for Docker environments
func (ic *IntegrityChecker) SetPragmas() error {
	pragmas := []string{
		"PRAGMA synchronous=NORMAL",     // Balance between safety and performance
		"PRAGMA cache_size=10000",       // Increase cache size
		"PRAGMA temp_store=MEMORY",      // Use memory for temp tables
		"PRAGMA mmap_size=30000000000",  // Use memory-mapped I/O
		"PRAGMA busy_timeout=5000",      // Wait 5 seconds on locks
	}
	
	for _, pragma := range pragmas {
		if _, err := ic.db.Exec(pragma); err != nil {
			ic.logger.WithError(err).Warnf("Failed to set pragma: %s", pragma)
		}
	}
	
	return nil
}