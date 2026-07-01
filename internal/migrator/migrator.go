package migrator

import (
	"database/sql"
	"fmt"
	"log"
	"os"
)

// Run scans the migrationsDir for all .up.sql files, checks which ones have
// already been applied against the schema_migrations tracking table, and
// executes any pending migrations in order.
//
// Call this after db.Ping() succeeds and before any repository initialization.
// If any migration fails, Run returns an error — the caller should abort startup.
func Run(db *sql.DB, migrationsDir string) error {
	// 1. Ensure the tracking table exists
	if err := ensureTrackingTable(db); err != nil {
		return fmt.Errorf("migrator: ensure tracking table: %w", err)
	}

	// 2. Discover migration files on disk
	migrations, err := discoverMigrations(migrationsDir)
	if err != nil {
		return fmt.Errorf("migrator: discover migrations: %w", err)
	}
	if len(migrations) == 0 {
		log.Println("migrator: no migration files found in", migrationsDir)
		return nil
	}

	// 3. Query already-applied migrations
	applied, err := fetchApplied(db)
	if err != nil {
		return fmt.Errorf("migrator: fetch applied migrations: %w", err)
	}

	// 4. Apply pending migrations in order
	appliedCount := 0
	for _, m := range migrations {
		if applied[m.Filename] {
			continue
		}

		// Read file content lazily (only when needed)
		content, err := os.ReadFile(m.FilePath)
		if err != nil {
			return fmt.Errorf("migrator: read %s: %w", m.Filename, err)
		}
		m.Content = string(content)

		log.Printf("migrator: applying %s", m.Filename)
		if err := applyMigration(db, &m); err != nil {
			return fmt.Errorf("migrator: apply %s: %w", m.Filename, err)
		}
		appliedCount++
	}

	if appliedCount > 0 {
		log.Printf("migrator: applied %d migration(s), %d total", appliedCount, len(migrations))
	} else {
		log.Printf("migrator: all %d migration(s) already applied", len(migrations))
	}

	return nil
}

// ensureTrackingTable creates the schema_migrations table if it doesn't exist.
func ensureTrackingTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename    VARCHAR(255) PRIMARY KEY,
		applied_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}
	return nil
}

// fetchApplied returns the set of already-applied migration filenames.
func fetchApplied(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT filename FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	applied := make(map[string]bool)
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return nil, fmt.Errorf("scan schema_migrations row: %w", err)
		}
		applied[filename] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema_migrations: %w", err)
	}
	return applied, nil
}

// applyMigration executes a single migration in a transaction: split the SQL
// content into individual statements, execute each, then record the migration
// as applied.
func applyMigration(db *sql.DB, m *Migration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op if already committed

	stmts := splitStatements(m.Content)
	for i, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			preview := truncate(stmt, 80)
			return fmt.Errorf("execute statement %d/%d %q: %w", i+1, len(stmts), preview, err)
		}
	}

	// Record the migration as applied
	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (filename) VALUES (?)",
		m.Filename,
	); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// truncate truncates s to maxLen characters, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
