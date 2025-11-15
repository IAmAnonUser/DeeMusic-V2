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
		WHERE status = ?
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
	err := qs.db.QueryRow("SELECT COUNT(*) FROM queue_items").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue count: %w", err)
	}
	return count, nil
}

// GetCountByStatus returns the count of queue items for a specific status
func (qs *QueueStore) GetCountByStatus(status string) (int, error) {
	var count int
	err := qs.db.QueryRow("SELECT COUNT(*) FROM queue_items WHERE status = ?", status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue count by status: %w", err)
	}
	return count, nil
}

// GetStats retrieves queue statistics
func (qs *QueueStore) GetStats() (*QueueStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) as pending,
			COALESCE(SUM(CASE WHEN status = 'downloading' THEN 1 ELSE 0 END), 0) as downloading,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) as completed,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) as failed
		FROM queue_items
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
	_, err = tx.Exec(`
		DELETE FROM queue_items 
		WHERE type = 'album' 
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
	_, err = tx.Exec(`
		DELETE FROM queue_items 
		WHERE type = 'playlist' 
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
		if item.Type == "album" || item.Type == "playlist" {
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

// CountFinishedChildren counts how many child tracks are finished (completed or permanently failed)
// A track is considered permanently failed if it has failed and reached the max retry count
func (qs *QueueStore) CountFinishedChildren(parentID string, maxRetries int) int {
	query := `
		SELECT COUNT(*) 
		FROM queue_items 
		WHERE parent_id = ? 
		AND (status = 'completed' OR (status = 'failed' AND retry_count >= ?))
	`
	
	var count int
	err := qs.db.QueryRow(query, parentID, maxRetries).Scan(&count)
	if err != nil {
		return 0
	}
	
	return count
}
