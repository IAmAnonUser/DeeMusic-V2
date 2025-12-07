package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// QueueItem represents a download queue item
type QueueItem struct {
	ID              string     `json:"id"`
	Type            string     `json:"type"` // track, album, playlist
	Title           string     `json:"title"`
	Artist          string     `json:"artist"`
	Album           string     `json:"album"`
	Status          string     `json:"status"` // pending, downloading, completed, failed
	Progress        int        `json:"progress"`
	DownloadURL     string     `json:"-"`
	OutputPath      string     `json:"output_path"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	RetryCount      int        `json:"retry_count"`
	MetadataJSON    string     `json:"-"`
	PartialFilePath string     `json:"-"`                       // Path to partial download file
	BytesDownloaded int64      `json:"bytes_downloaded"`        // Bytes downloaded so far
	TotalBytes      int64      `json:"total_bytes"`             // Total file size
	ParentID        string     `json:"parent_id,omitempty"`     // For tracks: the album/playlist ID
	TotalTracks     int        `json:"total_tracks,omitempty"`  // For albums: total number of tracks
	CompletedTracks int        `json:"completed_tracks"`        // For albums: number of completed tracks
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	AddedAt         time.Time  `json:"added_at"`                // When item was added to queue
	PlaylistID      string     `json:"playlist_id,omitempty"`   // For playlists
	IsCustom        bool       `json:"is_custom"`               // True for custom/imported playlists
	CustomTracks    []string   `json:"custom_tracks,omitempty"` // Track IDs for custom playlists
}

// QueueStats represents queue statistics
type QueueStats struct {
	Total       int `json:"total"`
	Pending     int `json:"pending"`
	Downloading int `json:"downloading"`
	Completed   int `json:"completed"`
	Failed      int `json:"failed"`
}

// QueueStore manages queue items in the database
type QueueStore struct {
	db      *sql.DB
	batchMu sync.Mutex // Mutex to serialize batch operations
}

// NewQueueStore creates a new QueueStore
func NewQueueStore(db *sql.DB) *QueueStore {
	return &QueueStore{db: db}
}

// Add adds a new item to the queue
func (qs *QueueStore) Add(item *QueueItem) error {
	query := `
		INSERT INTO queue_items (
			id, type, title, artist, album, status, progress,
			download_url, output_path, error_message, retry_count,
			metadata_json, parent_id, total_tracks, completed_tracks,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err := qs.db.Exec(
		query,
		item.ID,
		item.Type,
		item.Title,
		item.Artist,
		item.Album,
		item.Status,
		item.Progress,
		item.DownloadURL,
		item.OutputPath,
		item.ErrorMessage,
		item.RetryCount,
		item.MetadataJSON,
		item.ParentID,
		item.TotalTracks,
		item.CompletedTracks,
		item.CreatedAt,
		item.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to add queue item: %w", err)
	}

	return nil
}

// AddBatch adds multiple items to the queue in a single transaction
func (qs *QueueStore) AddBatch(items []*QueueItem) error {
	if len(items) == 0 {
		return nil
	}

	// Serialize batch operations to avoid database lock contention
	qs.batchMu.Lock()
	defer qs.batchMu.Unlock()

	tx, err := qs.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT OR IGNORE INTO queue_items (
			id, type, title, artist, album, status, progress,
			download_url, output_path, error_message, retry_count,
			metadata_json, parent_id, total_tracks, completed_tracks,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, item := range items {
		item.CreatedAt = now
		item.UpdatedAt = now

		_, err := stmt.Exec(
			item.ID,
			item.Type,
			item.Title,
			item.Artist,
			item.Album,
			item.Status,
			item.Progress,
			item.DownloadURL,
			item.OutputPath,
			item.ErrorMessage,
			item.RetryCount,
			item.MetadataJSON,
			item.ParentID,
			item.TotalTracks,
			item.CompletedTracks,
			item.CreatedAt,
			item.UpdatedAt,
		)

		if err != nil {
			return fmt.Errorf("failed to add queue item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Update updates an existing queue item
func (qs *QueueStore) Update(item *QueueItem) error {
	// VALIDATION: Prevent albums/playlists from being marked as completed if not all tracks are finished
	// A track is "finished" if it's completed OR permanently failed (status='failed')
	if (item.Type == "album" || item.Type == "playlist") && item.Status == "completed" {
		if item.TotalTracks > 0 {
			// Count how many tracks are finished (completed + failed)
			finishedCount := qs.CountFinishedChildren(item.ID, 3)
			
			// Only allow completion if all tracks are finished
			if finishedCount < item.TotalTracks {
				// Log the validation failure
				if logFile, logErr := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); logErr == nil {
					fmt.Fprintf(logFile, "[%s] VALIDATION FAILED: Preventing %s %s from being marked completed - only %d/%d tracks finished (completed=%d)\n", 
						time.Now().Format("2006-01-02 15:04:05"), item.Type, item.ID, finishedCount, item.TotalTracks, item.CompletedTracks)
					logFile.Close()
				}
				// Force status back to downloading
				item.Status = "downloading"
				item.CompletedAt = nil
			} else {
				// All tracks are finished - log success
				if logFile, logErr := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); logErr == nil {
					fmt.Fprintf(logFile, "[%s] VALIDATION PASSED: Allowing %s %s to complete - %d/%d tracks finished (completed=%d, failed=%d)\n", 
						time.Now().Format("2006-01-02 15:04:05"), item.Type, item.ID, finishedCount, item.TotalTracks, item.CompletedTracks, finishedCount-item.CompletedTracks)
					logFile.Close()
				}
			}
		}
	}

	query := `
		UPDATE queue_items
		SET type = ?, title = ?, artist = ?, album = ?, status = ?,
		    progress = ?, download_url = ?, output_path = ?,
		    error_message = ?, retry_count = ?, metadata_json = ?,
		    parent_id = ?, total_tracks = ?, completed_tracks = ?,
		    updated_at = ?, completed_at = ?
		WHERE id = ?
	`

	item.UpdatedAt = time.Now()

	result, err := qs.db.Exec(
		query,
		item.Type,
		item.Title,
		item.Artist,
		item.Album,
		item.Status,
		item.Progress,
		item.DownloadURL,
		item.OutputPath,
		item.ErrorMessage,
		item.RetryCount,
		item.MetadataJSON,
		item.ParentID,
		item.TotalTracks,
		item.CompletedTracks,
		item.UpdatedAt,
		item.CompletedAt,
		item.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update queue item: %w", err)
	}

	// Verify the update actually happened
	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return fmt.Errorf("update affected 0 rows for item %s", item.ID)
	}

	// Log successful update for debugging
	if logFile, logErr := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); logErr == nil {
		fmt.Fprintf(logFile, "[%s] DB UPDATE: ID=%s, Status=%s, Progress=%d, RowsAffected=%d\n", 
			time.Now().Format("2006-01-02 15:04:05"), item.ID, item.Status, item.Progress, rowsAffected)
		logFile.Close()
	}

	// Force aggressive WAL checkpoint for album status updates
	if item.Type == "album" && item.Status == "completed" {
		_, err := qs.db.Exec("PRAGMA wal_checkpoint(RESTART)")
		if logFile, logErr := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); logErr == nil {
			if err != nil {
				fmt.Fprintf(logFile, "[%s] WAL CHECKPOINT FAILED for %s: %v\n", time.Now().Format("2006-01-02 15:04:05"), item.ID, err)
			} else {
				fmt.Fprintf(logFile, "[%s] WAL CHECKPOINT SUCCESS for %s\n", time.Now().Format("2006-01-02 15:04:05"), item.ID)
			}
			logFile.Close()
		}
	}

	return nil
}

// Delete removes an item from the queue
func (qs *QueueStore) Delete(id string) error {
	query := "DELETE FROM queue_items WHERE id = ?"

	result, err := qs.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete queue item: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("queue item not found: %s", id)
	}

	return nil
}

// GetByID retrieves a queue item by ID
func (qs *QueueStore) GetByID(id string) (*QueueItem, error) {
	query := `
		SELECT id, type, title, artist, album, status, progress,
		       download_url, output_path, error_message, retry_count,
		       metadata_json, parent_id, total_tracks, completed_tracks,
		       created_at, updated_at, completed_at
		FROM queue_items
		WHERE id = ?
	`

	item := &QueueItem{}
	var completedAt sql.NullTime
	var parentID sql.NullString

	err := qs.db.QueryRow(query, id).Scan(
		&item.ID,
		&item.Type,
		&item.Title,
		&item.Artist,
		&item.Album,
		&item.Status,
		&item.Progress,
		&item.DownloadURL,
		&item.OutputPath,
		&item.ErrorMessage,
		&item.RetryCount,
		&item.MetadataJSON,
		&parentID,
		&item.TotalTracks,
		&item.CompletedTracks,
		&item.CreatedAt,
		&item.UpdatedAt,
		&completedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("queue item not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get queue item: %w", err)
	}

	if completedAt.Valid {
		item.CompletedAt = &completedAt.Time
	}
	if parentID.Valid {
		item.ParentID = parentID.String
	}

	return item, nil
}

// GetPending retrieves pending queue items
func (qs *QueueStore) GetPending(limit int) ([]*QueueItem, error) {
	if qs == nil {
		return nil, fmt.Errorf("queue store is nil")
	}
	if qs.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	
	query := `
		SELECT id, type, title, artist, album, status, progress,
		       download_url, output_path, error_message, retry_count,
		       metadata_json, parent_id, total_tracks, completed_tracks,
		       created_at, updated_at, completed_at
		FROM queue_items
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT ?
	`

	rows, err := qs.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending items: %w", err)
	}
	defer rows.Close()

	return qs.scanItems(rows)
}

// GetAll retrieves all queue items with pagination
func (qs *QueueStore) GetAll(offset, limit int) ([]*QueueItem, error) {
	// Enforce maximum limit to prevent memory issues
	if limit > 1000 {
		limit = 1000
	}
	
	query := `
		SELECT id, type, title, artist, album, status, progress,
		       download_url, output_path, error_message, retry_count,
		       metadata_json, parent_id, total_tracks, completed_tracks,
		       created_at, updated_at, completed_at
		FROM queue_items
		WHERE type IN ('album', 'playlist')
		ORDER BY created_at ASC
		LIMIT ? OFFSET ?
	`

	rows, err := qs.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get all items: %w", err)
	}
	defer rows.Close()

	return qs.scanItems(rows)
}

// GetByStatus retrieves queue items filtered by status with pagination
// Only returns albums and playlists (parent items), not individual tracks
func (qs *QueueStore) GetByStatus(status string, offset, limit int) ([]*QueueItem, error) {
	// Enforce maximum limit to prevent memory issues
	if limit > 1000 {
		limit = 1000
	}
	
	query := `
		SELECT id, type, title, artist, album, status, progress,
		       download_url, output_path, error_message, retry_count,
		       metadata_json, parent_id, total_tracks, completed_tracks,
		       created_at, updated_at, completed_at
		FROM queue_items
		WHERE status = ? AND type IN ('album', 'playlist')
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := qs.db.Query(query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get items by status: %w", err)
	}
	defer rows.Close()

	return qs.scanItems(rows)
}

// GetCount returns the total count of queue items
func (qs *QueueStore) GetCount() (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM queue_items WHERE type IN ('album', 'playlist')"
	err := qs.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue count: %w", err)
	}
	// Debug log to verify the query is working
	fmt.Printf("[DEBUG] GetCount query: %s, result: %d\n", query, count)
	return count, nil
}

// GetCountByStatus returns the count of queue items for a specific status
// Only counts albums and playlists (parent items), not individual tracks
func (qs *QueueStore) GetCountByStatus(status string) (int, error) {
	var count int
	err := qs.db.QueryRow("SELECT COUNT(*) FROM queue_items WHERE status = ? AND type IN ('album', 'playlist')", status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue count by status: %w", err)
	}
	return count, nil
}

// GetStats retrieves queue statistics
// Only counts albums and playlists (parent items), not individual tracks
func (qs *QueueStore) GetStats() (*QueueStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) as pending,
			COALESCE(SUM(CASE WHEN status = 'downloading' THEN 1 ELSE 0 END), 0) as downloading,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) as completed,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) as failed
		FROM queue_items
		WHERE type IN ('album', 'playlist')
	`

	stats := &QueueStats{}
	err := qs.db.QueryRow(query).Scan(
		&stats.Total,
		&stats.Pending,
		&stats.Downloading,
		&stats.Completed,
		&stats.Failed,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}

	return stats, nil
}

// ClearCompleted removes all completed items from the queue
func (qs *QueueStore) ClearCompleted() error {
	// Start transaction
	tx, err := qs.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete completed track items, but ONLY if their parent album is also completed
	// This prevents deleting tracks from albums that are still downloading
	_, err = tx.Exec(`
		DELETE FROM queue_items 
		WHERE status = 'completed' 
		AND type = 'track'
		AND (
			parent_id IS NULL 
			OR parent_id = ''
			OR parent_id IN (
				SELECT id FROM queue_items 
				WHERE type IN ('album', 'playlist') 
				AND status = 'completed'
			)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to clear completed items: %w", err)
	}

	// Delete album items where all tracks are completed or deleted
	// (albums with no remaining pending/downloading/failed tracks)
	// BUT exclude albums with partial failures (completed_tracks < total_tracks)
	_, err = tx.Exec(`
		DELETE FROM queue_items 
		WHERE type = 'album' 
		AND status = 'completed'
		AND completed_tracks = total_tracks
		AND id NOT IN (
			SELECT DISTINCT parent_id 
			FROM queue_items 
			WHERE type = 'track' 
			AND parent_id IS NOT NULL 
			AND parent_id != ''
			AND parent_id LIKE 'album_%'
			AND status IN ('pending', 'downloading', 'failed')
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to clear completed albums: %w", err)
	}

	// Delete playlist items where all tracks are completed or deleted
	// BUT exclude playlists with partial failures (completed_tracks < total_tracks)
	_, err = tx.Exec(`
		DELETE FROM queue_items 
		WHERE type = 'playlist' 
		AND status = 'completed'
		AND completed_tracks = total_tracks
		AND id NOT IN (
			SELECT DISTINCT parent_id 
			FROM queue_items 
			WHERE type = 'track' 
			AND parent_id IS NOT NULL 
			AND parent_id != ''
			AND parent_id LIKE 'playlist_%'
			AND status IN ('pending', 'downloading', 'failed')
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to clear completed playlists: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// FixIncompleteAlbums fixes albums/playlists that were incorrectly marked as completed
// when they actually have 0 tracks downloaded. Returns the number of items fixed.
func (qs *QueueStore) FixIncompleteAlbums() (int, error) {
	// Find albums/playlists marked as completed but with completed_tracks < total_tracks
	query := `
		UPDATE queue_items
		SET status = 'pending', progress = 0, completed_at = NULL
		WHERE (type = 'album' OR type = 'playlist')
		AND status = 'completed'
		AND completed_tracks < total_tracks
		AND total_tracks > 0
	`
	
	result, err := qs.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to fix incomplete albums: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	// Log to debug file
	if rowsAffected > 0 {
		if logFile, logErr := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); logErr == nil {
			fmt.Fprintf(logFile, "[%s] DATABASE CLEANUP: Fixed %d incomplete albums/playlists\n", 
				time.Now().Format("2006-01-02 15:04:05"), rowsAffected)
			logFile.Close()
		}
	}
	
	return int(rowsAffected), nil
}

// FixStuckAlbums fixes albums/playlists stuck in "downloading" status where all tracks are finished
// (either completed or permanently failed). Returns the number of items fixed.
func (qs *QueueStore) FixStuckAlbums() (int, error) {
	// Get all albums/playlists in downloading status
	query := `
		SELECT id, total_tracks, completed_tracks, updated_at
		FROM queue_items
		WHERE (type = 'album' OR type = 'playlist')
		AND status = 'downloading'
		AND total_tracks > 0
	`
	
	rows, err := qs.db.Query(query)
	if err != nil {
		return 0, fmt.Errorf("failed to query stuck albums: %w", err)
	}
	defer rows.Close()
	
	fixedCount := 0
	now := time.Now()
	
	for rows.Next() {
		var id string
		var totalTracks, completedTracks int
		var updatedAt time.Time
		
		if err := rows.Scan(&id, &totalTracks, &completedTracks, &updatedAt); err != nil {
			continue
		}
		
		// Count finished tracks (completed + failed) in database
		finishedCount := qs.CountFinishedChildren(id, 3)
		
		// Count total tracks that exist in database
		var tracksInDB int
		countQuery := `SELECT COUNT(*) FROM queue_items WHERE parent_id = ?`
		if err := qs.db.QueryRow(countQuery, id).Scan(&tracksInDB); err != nil {
			continue
		}
		
		shouldComplete := false
		reason := ""
		
		// Case 1: All tracks in database are finished
		if finishedCount >= totalTracks {
			shouldComplete = true
			reason = fmt.Sprintf("all %d/%d tracks finished", finishedCount, totalTracks)
		}
		
		// Case 2: Album hasn't been updated in 5+ minutes and has very few tracks in DB
		// This handles cases where album download job failed to add all tracks
		timeSinceUpdate := now.Sub(updatedAt)
		if !shouldComplete && tracksInDB > 0 && tracksInDB < totalTracks && timeSinceUpdate > 5*time.Minute {
			// If all tracks that DO exist are finished, mark album as completed
			if finishedCount == tracksInDB {
				shouldComplete = true
				reason = fmt.Sprintf("stale album (updated %v ago) with only %d/%d tracks in DB, all finished", 
					timeSinceUpdate.Round(time.Second), tracksInDB, totalTracks)
			}
		}
		
		if shouldComplete {
			updateQuery := `
				UPDATE queue_items
				SET status = 'completed', completed_at = ?, progress = 100
				WHERE id = ?
			`
			
			_, err := qs.db.Exec(updateQuery, now, id)
			if err == nil {
				fixedCount++
				
				if logFile, logErr := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); logErr == nil {
					fmt.Fprintf(logFile, "[%s] DATABASE CLEANUP: Fixed stuck album %s - %s (completed=%d, failed=%d)\n", 
						time.Now().Format("2006-01-02 15:04:05"), id, reason, completedTracks, finishedCount-completedTracks)
					logFile.Close()
				}
			}
		}
	}
	
	return fixedCount, nil
}

// scanItems scans multiple queue items from rows
func (qs *QueueStore) scanItems(rows *sql.Rows) ([]*QueueItem, error) {
	items := []*QueueItem{}

	for rows.Next() {
		item := &QueueItem{}
		var completedAt sql.NullTime
		var parentID sql.NullString

		err := rows.Scan(
			&item.ID,
			&item.Type,
			&item.Title,
			&item.Artist,
			&item.Album,
			&item.Status,
			&item.Progress,
			&item.DownloadURL,
			&item.OutputPath,
			&item.ErrorMessage,
			&item.RetryCount,
			&item.MetadataJSON,
			&parentID,
			&item.TotalTracks,
			&item.CompletedTracks,
			&item.CreatedAt,
			&item.UpdatedAt,
			&completedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan queue item: %w", err)
		}

		if completedAt.Valid {
			item.CompletedAt = &completedAt.Time
		}
		if parentID.Valid {
			item.ParentID = parentID.String
		}

		// For albums/playlists, dynamically calculate completed tracks count
		// This ensures we always have accurate data even if the app was closed during downloads
		// BUT: Only recalculate for non-terminal states to avoid race conditions during completion
		if item.Type == "album" || item.Type == "playlist" {
			// Only recalculate if album is still downloading or pending
			// For completed/failed albums, trust the stored value to avoid visual glitches
			if item.Status == "downloading" || item.Status == "pending" {
				actualCompletedCount := qs.CountCompletedChildren(item.ID)
				if actualCompletedCount != item.CompletedTracks {
					if logFile, logErr := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); logErr == nil {
						fmt.Fprintf(logFile, "[%s] DB READ: Correcting completed count for %s: DB says %d, actual is %d\n", 
							time.Now().Format("2006-01-02 15:04:05"), item.ID, item.CompletedTracks, actualCompletedCount)
						logFile.Close()
					}
					item.CompletedTracks = actualCompletedCount
				}
			}
		}
		
		// Log what we read from database for albums
		if item.Type == "album" && (item.Status == "completed" || item.CompletedTracks >= item.TotalTracks) {
			if logFile, logErr := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); logErr == nil {
				fmt.Fprintf(logFile, "[%s] DB READ: ID=%s, Status=%s, Progress=%d, Completed=%d/%d\n", 
					time.Now().Format("2006-01-02 15:04:05"), item.ID, item.Status, item.Progress, item.CompletedTracks, item.TotalTracks)
				logFile.Close()
			}
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return items, nil
}

// AddToHistory adds a completed download to history
func (qs *QueueStore) AddToHistory(trackID, title, artist, album, filePath, quality string, fileSize int64) error {
	query := `
		INSERT INTO download_history (
			track_id, title, artist, album, file_path, file_size, quality
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := qs.db.Exec(query, trackID, title, artist, album, filePath, fileSize, quality)
	if err != nil {
		return fmt.Errorf("failed to add to history: %w", err)
	}

	return nil
}

// GetHistory retrieves download history with pagination
func (qs *QueueStore) GetHistory(offset, limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT id, track_id, title, artist, album, file_path, file_size, quality, downloaded_at
		FROM download_history
		ORDER BY downloaded_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := qs.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}
	defer rows.Close()

	history := []map[string]interface{}{}

	for rows.Next() {
		var id int
		var trackID, title, artist, album, filePath, quality string
		var fileSize int64
		var downloadedAt time.Time

		err := rows.Scan(&id, &trackID, &title, &artist, &album, &filePath, &fileSize, &quality, &downloadedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan history row: %w", err)
		}

		history = append(history, map[string]interface{}{
			"id":            id,
			"track_id":      trackID,
			"title":         title,
			"artist":        artist,
			"album":         album,
			"file_path":     filePath,
			"file_size":     fileSize,
			"quality":       quality,
			"downloaded_at": downloadedAt,
		})
	}

	return history, nil
}

// SetConfigCache sets a configuration cache value
func (qs *QueueStore) SetConfigCache(key, value string) error {
	query := `
		INSERT INTO config_cache (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = ?
	`

	now := time.Now()
	_, err := qs.db.Exec(query, key, value, now, value, now)
	if err != nil {
		return fmt.Errorf("failed to set config cache: %w", err)
	}

	return nil
}

// GetConfigCache retrieves a configuration cache value
func (qs *QueueStore) GetConfigCache(key string) (string, error) {
	query := "SELECT value FROM config_cache WHERE key = ?"

	var value string
	err := qs.db.QueryRow(query, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("config cache key not found: %s", key)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get config cache: %w", err)
	}

	return value, nil
}

// SetMetadata sets metadata as JSON for a queue item
func (item *QueueItem) SetMetadata(metadata interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	item.MetadataJSON = string(data)
	return nil
}

// GetMetadata retrieves metadata from JSON
func (item *QueueItem) GetMetadata(target interface{}) error {
	if item.MetadataJSON == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(item.MetadataJSON), target); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}
	return nil
}

// IsResumable checks if a download can be resumed
func (item *QueueItem) IsResumable() bool {
	return item.PartialFilePath != "" && item.BytesDownloaded > 0 && item.TotalBytes > 0
}

// GetResumableDownloads retrieves downloads that can be resumed
func (qs *QueueStore) GetResumableDownloads(limit int) ([]*QueueItem, error) {
	query := `
		SELECT id, type, title, artist, album, status, progress,
		       download_url, output_path, error_message, retry_count,
		       metadata_json, partial_file_path, bytes_downloaded, total_bytes,
		       created_at, updated_at, completed_at
		FROM queue_items
		WHERE status IN ('pending', 'failed') 
		  AND partial_file_path IS NOT NULL 
		  AND bytes_downloaded > 0
		  AND total_bytes > 0
		ORDER BY updated_at DESC
		LIMIT ?
	`

	rows, err := qs.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get resumable downloads: %w", err)
	}
	defer rows.Close()

	return qs.scanItems(rows)
}

// GetDB returns the underlying database connection
func (qs *QueueStore) GetDB() *sql.DB {
	return qs.db
}

// ClearAll removes all items from the queue
func (qs *QueueStore) ClearAll() error {
	query := "DELETE FROM queue_items"
	
	_, err := qs.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clear all items: %w", err)
	}
	
	return nil
}

// CountCompletedChildren counts how many child tracks of a parent are completed
func (qs *QueueStore) CountCompletedChildren(parentID string) int {
	query := `
		SELECT COUNT(*) 
		FROM queue_items 
		WHERE parent_id = ? AND status = 'completed'
	`
	
	var count int
	err := qs.db.QueryRow(query, parentID).Scan(&count)
	if err != nil {
		return 0
	}
	
	return count
}

// CountFinishedChildren counts how many child tracks are finished (completed or failed)
// Any track with status 'completed' or 'failed' is considered finished
// This allows albums to complete even when tracks fail without exhausting all retries
func (qs *QueueStore) CountFinishedChildren(parentID string, maxRetries int) int {
	query := `
		SELECT COUNT(*) 
		FROM queue_items 
		WHERE parent_id = ? 
		AND status IN ('completed', 'failed')
	`
	
	var count int
	err := qs.db.QueryRow(query, parentID).Scan(&count)
	if err != nil {
		return 0
	}
	
	return count
}

// FailedTrack represents a failed track with error details
type FailedTrack struct {
	ID           int       `json:"id"`
	ParentID     string    `json:"parent_id"`
	TrackID      string    `json:"track_id"`
	TrackTitle   string    `json:"track_title"`
	TrackArtist  string    `json:"track_artist"`
	ErrorMessage string    `json:"error_message"`
	RetryCount   int       `json:"retry_count"`
	FailedAt     time.Time `json:"failed_at"`
}

// AddFailedTrack records a failed track
func (qs *QueueStore) AddFailedTrack(parentID, trackID, title, artist, errorMsg string, retryCount int) error {
	query := `
		INSERT INTO failed_tracks (parent_id, track_id, track_title, track_artist, error_message, retry_count)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	
	_, err := qs.db.Exec(query, parentID, trackID, title, artist, errorMsg, retryCount)
	if err != nil {
		return fmt.Errorf("failed to add failed track: %w", err)
	}
	
	return nil
}

// GetFailedTracks retrieves all failed tracks for a parent (album/playlist)
func (qs *QueueStore) GetFailedTracks(parentID string) ([]*FailedTrack, error) {
	query := `
		SELECT id, parent_id, track_id, track_title, track_artist, error_message, retry_count, failed_at
		FROM failed_tracks
		WHERE parent_id = ?
		ORDER BY failed_at DESC
	`
	
	rows, err := qs.db.Query(query, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed tracks: %w", err)
	}
	defer rows.Close()
	
	var tracks []*FailedTrack
	for rows.Next() {
		track := &FailedTrack{}
		err := rows.Scan(
			&track.ID,
			&track.ParentID,
			&track.TrackID,
			&track.TrackTitle,
			&track.TrackArtist,
			&track.ErrorMessage,
			&track.RetryCount,
			&track.FailedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan failed track: %w", err)
		}
		tracks = append(tracks, track)
	}
	
	return tracks, nil
}

// ClearFailedTracks removes all failed track records for a parent
func (qs *QueueStore) ClearFailedTracks(parentID string) error {
	query := "DELETE FROM failed_tracks WHERE parent_id = ?"
	_, err := qs.db.Exec(query, parentID)
	if err != nil {
		return fmt.Errorf("failed to clear failed tracks: %w", err)
	}
	return nil
}
