package store

import (
	"database/sql"
	"fmt"
)

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	Up      string
}

// migrations contains all database migrations in order
var migrations = []Migration{
	{
		Version: 1,
		Name:    "initial_schema",
		Up: `
-- Queue items table
CREATE TABLE IF NOT EXISTS queue_items (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    artist TEXT,
    album TEXT,
    status TEXT NOT NULL,
    progress INTEGER DEFAULT 0,
    download_url TEXT,
    output_path TEXT,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    metadata_json TEXT,
    parent_id TEXT,
    total_tracks INTEGER DEFAULT 0,
    completed_tracks INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_queue_status ON queue_items(status);
CREATE INDEX IF NOT EXISTS idx_queue_created ON queue_items(created_at);
CREATE INDEX IF NOT EXISTS idx_queue_parent ON queue_items(parent_id);

-- Download history table
CREATE TABLE IF NOT EXISTS download_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    track_id TEXT NOT NULL,
    title TEXT NOT NULL,
    artist TEXT,
    album TEXT,
    file_path TEXT,
    file_size INTEGER,
    quality TEXT,
    downloaded_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_history_track ON download_history(track_id);
CREATE INDEX IF NOT EXISTS idx_history_date ON download_history(downloaded_at);

-- Configuration cache table
CREATE TABLE IF NOT EXISTS config_cache (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Migration tracking table
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`,
	},
	{
		Version: 2,
		Name:    "add_resume_support",
		Up: `
-- Add columns for download resume capability
ALTER TABLE queue_items ADD COLUMN partial_file_path TEXT;
ALTER TABLE queue_items ADD COLUMN bytes_downloaded INTEGER DEFAULT 0;
ALTER TABLE queue_items ADD COLUMN total_bytes INTEGER DEFAULT 0;

-- Create index for resumable downloads
CREATE INDEX IF NOT EXISTS idx_queue_resumable ON queue_items(status, bytes_downloaded);
`,
	},
	{
		Version: 3,
		Name:    "optimize_large_queues",
		Up: `
-- Add composite indexes for efficient pagination and filtering
CREATE INDEX IF NOT EXISTS idx_queue_status_created ON queue_items(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_queue_updated ON queue_items(updated_at DESC);

-- Add index for common query patterns
CREATE INDEX IF NOT EXISTS idx_queue_status_progress ON queue_items(status, progress);

-- Optimize history queries
CREATE INDEX IF NOT EXISTS idx_history_composite ON download_history(downloaded_at DESC, track_id);
`,
	},
	{
		Version: 4,
		Name:    "add_failed_tracks_table",
		Up: `
-- Failed tracks table to track which tracks failed and why
CREATE TABLE IF NOT EXISTS failed_tracks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_id TEXT NOT NULL,
    track_id TEXT NOT NULL,
    track_title TEXT NOT NULL,
    track_artist TEXT,
    error_message TEXT NOT NULL,
    retry_count INTEGER DEFAULT 0,
    failed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_id) REFERENCES queue_items(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_failed_tracks_parent ON failed_tracks(parent_id);
CREATE INDEX IF NOT EXISTS idx_failed_tracks_date ON failed_tracks(failed_at DESC);
`,
	},
}

// RunMigrations executes all pending migrations
func RunMigrations(db *sql.DB) error {
	// Create migrations table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	currentVersion, err := getCurrentVersion(db)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	// Apply pending migrations
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		// Begin transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Execute migration
		if _, err := tx.Exec(migration.Up); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration %d (%s): %w", migration.Version, migration.Name, err)
		}

		// Record migration
		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
			migration.Version,
			migration.Name,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}
	}

	return nil
}

// getCurrentVersion returns the current schema version
func getCurrentVersion(db *sql.DB) (int, error) {
	var version int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}
