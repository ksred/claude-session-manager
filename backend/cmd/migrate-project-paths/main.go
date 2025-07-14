package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var (
		dbPath  = flag.String("db", "", "Path to SQLite database file")
		dataDir = flag.String("data-dir", "", "Claude data directory (alternative to -db)")
		dryRun  = flag.Bool("dry-run", false, "Show what would be updated without making changes")
	)
	flag.Parse()

	// Determine database path
	databasePath := ""
	if *dbPath != "" {
		databasePath = *dbPath
	} else if *dataDir != "" {
		databasePath = filepath.Join(*dataDir, "sessions.db")
	} else {
		// Default to ~/.claude/sessions.db
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Failed to get home directory:", err)
		}
		databasePath = filepath.Join(homeDir, ".claude", "sessions.db")
	}

	// Open database
	db, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	fmt.Printf("Connected to database: %s\n", databasePath)

	if *dryRun {
		fmt.Println("\n=== DRY RUN MODE - No changes will be made ===\n")
		
		// Show sessions that would be updated
		rows, err := db.Query(`
			SELECT s.id, s.project_path, s.project_name, m.cwd
			FROM sessions s
			INNER JOIN (
				SELECT session_id, cwd, 
					   ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY timestamp ASC) as rn
				FROM messages
				WHERE cwd IS NOT NULL AND cwd != ''
			) m ON s.id = m.session_id AND m.rn = 1
			WHERE s.project_path != m.cwd
			ORDER BY s.last_activity DESC
		`)
		if err != nil {
			log.Fatal("Failed to query sessions:", err)
		}
		defer rows.Close()

		count := 0
		fmt.Println("Sessions that would be updated:")
		fmt.Println("=====================================")
		for rows.Next() {
			var sessionID, currentPath, currentName, newPath string
			if err := rows.Scan(&sessionID, &currentPath, &currentName, &newPath); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			// Calculate new project name from path
			newName := filepath.Base(newPath)
			
			count++
			fmt.Printf("\nSession ID: %s\n", sessionID)
			fmt.Printf("  Current path: %s\n", currentPath)
			fmt.Printf("  Current name: %s\n", currentName)
			fmt.Printf("  New path:     %s\n", newPath)
			fmt.Printf("  New name:     %s\n", newName)
		}
		fmt.Printf("\nTotal sessions to update: %d\n", count)
		return
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		log.Fatal("Failed to begin transaction:", err)
	}
	defer tx.Rollback()

	// First update project paths
	fmt.Println("\n=== Updating session project paths ===")
	
	pathResult, err := tx.Exec(`
		UPDATE sessions
		SET project_path = (
			SELECT m.cwd
			FROM messages m
			WHERE m.session_id = sessions.id
			  AND m.cwd IS NOT NULL
			  AND m.cwd != ''
			ORDER BY m.timestamp ASC
			LIMIT 1
		),
		updated_at = CURRENT_TIMESTAMP
		WHERE EXISTS (
			SELECT 1
			FROM messages m
			WHERE m.session_id = sessions.id
			  AND m.cwd IS NOT NULL
			  AND m.cwd != ''
			  AND m.cwd != sessions.project_path
		)
	`)
	if err != nil {
		log.Fatal("Failed to update project paths:", err)
	}

	pathRowsAffected, _ := pathResult.RowsAffected()
	fmt.Printf("Updated project paths for %d sessions\n", pathRowsAffected)

	// Now update project names by fetching the paths and calculating names in Go
	fmt.Println("\n=== Updating project names ===")
	
	// Get all sessions with their paths
	rows, err := tx.Query(`
		SELECT id, project_path 
		FROM sessions 
		WHERE project_path IS NOT NULL AND project_path != ''
	`)
	if err != nil {
		log.Fatal("Failed to query sessions for name update:", err)
	}
	
	type sessionUpdate struct {
		ID   string
		Name string
	}
	
	var updates []sessionUpdate
	for rows.Next() {
		var id, path string
		if err := rows.Scan(&id, &path); err != nil {
			continue
		}
		// Calculate project name from path
		name := filepath.Base(path)
		// Special handling for paths ending with slash
		if name == "" || name == "." {
			parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
			if len(parts) > 0 {
				name = parts[len(parts)-1]
			}
		}
		updates = append(updates, sessionUpdate{ID: id, Name: name})
	}
	rows.Close()
	
	// Update names
	stmt, err := tx.Prepare("UPDATE sessions SET project_name = ? WHERE id = ?")
	if err != nil {
		log.Fatal("Failed to prepare update statement:", err)
	}
	defer stmt.Close()
	
	nameCount := 0
	for _, update := range updates {
		if _, err := stmt.Exec(update.Name, update.ID); err != nil {
			log.Printf("Failed to update name for session %s: %v", update.ID, err)
		} else {
			nameCount++
		}
	}
	
	fmt.Printf("Updated project names for %d sessions\n", nameCount)

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Fatal("Failed to commit transaction:", err)
	}

	fmt.Println("\n✅ Migration completed successfully!")
	fmt.Printf("   - Updated %d session paths from CWD values\n", pathRowsAffected)
	fmt.Printf("   - Updated %d project names\n", nameCount)
}