package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Get database path from environment or use default
	dbPath := os.Getenv("CLAUDE_DB_PATH")
	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Failed to get home directory:", err)
		}
		dbPath = filepath.Join(homeDir, ".claude", "sessions.db")
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Read migration SQL
	migrationSQL := `
-- Migration: Recalculate token costs with correct pricing
-- This migration updates the estimated_cost column in token_usage table with accurate pricing

-- Create a temporary table to store recalculated costs
CREATE TEMPORARY TABLE token_cost_updates (
    message_id TEXT PRIMARY KEY,
    new_cost REAL
);

-- Insert recalculated costs based on correct pricing model
INSERT INTO token_cost_updates (message_id, new_cost)
SELECT 
    tu.message_id,
    CASE 
        -- Claude Opus 3 and Opus 4
        WHEN s.model LIKE '%claude-3-opus%' OR s.model LIKE '%claude-opus-4%' THEN
            (tu.input_tokens * 15.0 / 1000000) +
            (tu.output_tokens * 75.0 / 1000000) +
            (tu.cache_read_input_tokens * 1.50 / 1000000) +
            (tu.cache_creation_input_tokens * 18.75 / 1000000)
            
        -- Claude Sonnet 3.5, 3.7, 4
        WHEN s.model LIKE '%claude-3-5-sonnet%' OR s.model LIKE '%claude-3.5-sonnet%' 
             OR s.model LIKE '%claude-3-sonnet%' OR s.model LIKE '%claude-sonnet%' THEN
            (tu.input_tokens * 3.0 / 1000000) +
            (tu.output_tokens * 15.0 / 1000000) +
            (tu.cache_read_input_tokens * 0.30 / 1000000) +
            (tu.cache_creation_input_tokens * 3.75 / 1000000)
            
        -- Claude Haiku 3.5
        WHEN s.model LIKE '%claude-3-5-haiku%' OR s.model LIKE '%claude-3.5-haiku%' THEN
            (tu.input_tokens * 0.80 / 1000000) +
            (tu.output_tokens * 4.0 / 1000000) +
            (tu.cache_read_input_tokens * 0.08 / 1000000) +
            (tu.cache_creation_input_tokens * 1.0 / 1000000)
            
        -- Claude Haiku 3
        WHEN s.model LIKE '%claude-3-haiku%' THEN
            (tu.input_tokens * 0.25 / 1000000) +
            (tu.output_tokens * 1.25 / 1000000) +
            (tu.cache_read_input_tokens * 0.03 / 1000000) +
            (tu.cache_creation_input_tokens * 0.30 / 1000000)
            
        -- Default to Sonnet pricing for unknown models
        ELSE
            (tu.input_tokens * 3.0 / 1000000) +
            (tu.output_tokens * 15.0 / 1000000) +
            (tu.cache_read_input_tokens * 0.30 / 1000000) +
            (tu.cache_creation_input_tokens * 3.75 / 1000000)
    END as new_cost
FROM token_usage tu
JOIN sessions s ON tu.session_id = s.id;

-- Update the token_usage table with recalculated costs
UPDATE token_usage
SET estimated_cost = (
    SELECT new_cost 
    FROM token_cost_updates 
    WHERE token_cost_updates.message_id = token_usage.message_id
)
WHERE EXISTS (
    SELECT 1 
    FROM token_cost_updates 
    WHERE token_cost_updates.message_id = token_usage.message_id
);

-- Clean up temporary table
DROP TABLE token_cost_updates;`

	// Count tokens before update
	var beforeCount int
	var beforeTotalCost float64
	err = db.QueryRow("SELECT COUNT(*), COALESCE(SUM(estimated_cost), 0) FROM token_usage").Scan(&beforeCount, &beforeTotalCost)
	if err != nil {
		log.Fatal("Failed to get before statistics:", err)
	}

	fmt.Printf("Before migration:\n")
	fmt.Printf("  Total token usage records: %d\n", beforeCount)
	fmt.Printf("  Total estimated cost: $%.4f\n", beforeTotalCost)

	// Run migration
	fmt.Println("\nRunning migration...")
	result, err := db.Exec(migrationSQL)
	if err != nil {
		log.Fatal("Failed to run migration:", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Migration completed. Rows updated: %d\n", rowsAffected)

	// Count tokens after update
	var afterTotalCost float64
	err = db.QueryRow("SELECT COALESCE(SUM(estimated_cost), 0) FROM token_usage").Scan(&afterTotalCost)
	if err != nil {
		log.Fatal("Failed to get after statistics:", err)
	}

	fmt.Printf("\nAfter migration:\n")
	fmt.Printf("  Total estimated cost: $%.4f\n", afterTotalCost)
	fmt.Printf("  Cost difference: $%.4f\n", afterTotalCost-beforeTotalCost)

	// Show sample of updated costs by model
	fmt.Println("\nSample costs by model:")
	rows, err := db.Query(`
		SELECT s.model, COUNT(*) as count, 
		       SUM(tu.estimated_cost) as total_cost,
		       AVG(tu.estimated_cost) as avg_cost
		FROM token_usage tu
		JOIN sessions s ON tu.session_id = s.id
		GROUP BY s.model
		ORDER BY total_cost DESC
		LIMIT 10
	`)
	if err != nil {
		log.Fatal("Failed to get model statistics:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var model string
		var count int
		var totalCost, avgCost float64
		rows.Scan(&model, &count, &totalCost, &avgCost)
		fmt.Printf("  %-40s: %6d messages, total: $%8.4f, avg: $%6.4f\n", model, count, totalCost, avgCost)
	}

	fmt.Println("\nMigration completed successfully!")
}