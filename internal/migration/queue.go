package migration

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/deemusic/deemusic-go/internal/store"
	_ "github.com/mattn/go-sqlite3"
)

// PythonQueueItem represents a queue item from Python version
type PythonQueueItem struct {
	ID           string
	Type         string
	Title        string
	Artist       string
	Album        string
	Status       string
	Progress     int
	DownloadURL  string
	OutputPath   string
	ErrorMessage string
	RetryCount   int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CompletedAt  *time.Time
}

// PythonHistoryItem represents a download history item from Python version
type PythonHistoryItem struct {
	TrackID      string
	Title        string
	Artist       string
	Album        string
	FilePath     string
	FileSize     int64
	Quality      string
	DownloadedAt time.Time
}

// QueueMigrator handles migration of queue data from Python to Go
type QueueMigrator struct {
	pythonDBPath string
	goDBPath     string
	pythonDB     *sql.DB
	goDB         *sql.DB
	queueStore   *store.QueueStore
}

// NewQueueMigrator creates a new QueueMigrator
func NewQueueMigrator(pythonDBPath, goDBPath string) *QueueMigrator {
	return &QueueMigrator{
		pythonDBPath: pythonDBPath,
		goDBPath:     goDBPath,
	}
}

// Open opens both Python and Go databases
func (qm *QueueMigrator) Open() error {
	// Open Python database
	pythonDB, err := sql.Open("sqlite3", qm.pythonDBPath)
	if err != nil {
		return fmt.Errorf("failed to open Python database: %w", err)
	}
	qm.pythonDB = pythonDB

	// Open Go database
	goDB, err := sql.Open("sqlite3", qm.goDBPath)
	if err != nil {
		pythonDB.Close()
		return fmt.Errorf("failed to open Go database: %w", err)
	}
	qm.goDB = goDB

	// Initialize Go database with migrations
	if err := store.RunMigrations(goDB); err != nil {
		pythonDB.Close()
		goDB.Close()
		return fmt.Errorf("failed to run Go database migrations: %w", err)
	}

	qm.queueStore = store.NewQueueStore(goDB)

	return nil
}

// Close closes both databases
func (qm *QueueMigrator) Close() error {
	var errs []error

	if qm.pythonDB != nil {
		if err := qm.pythonDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Python database: %w", err))
		}
	}

	if qm.goDB != nil {
		if err := qm.goDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Go database: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing databases: %v", errs)
	}

	return nil
}

// ReadPythonQueue reads queue items from Python database
func (qm *QueueMigrator) ReadPythonQueue() ([]*PythonQueueItem, error) {
	// Try different possible table names and schemas
	queries := []string{
		`SELECT id, type, title, artist, album, status, progress, download_url, 
		        output_path, error_message, retry_count, created_at, updated_at, completed_at
		 FROM queue_items ORDER BY created_at ASC`,
		
		`SELECT id, type, title, artist, album, status, progress, url, 
		        path, error, retries, created, updated, completed
		 FROM downloads ORDER BY created ASC`,
		
		`SELECT id, item_type, title, artist, album, status, progress, url, 
		        output_path, error_msg, retry_count, created_at, updated_at, completed_at
		 FROM queue ORDER BY created_at ASC`,
	}

	var items []*PythonQueueItem
	var lastErr error

	for _, query := range queries {
		rows, err := qm.pythonDB.Query(query)
		if err != nil {
			lastErr = err
			continue
		}
		defer rows.Close()

		items, err = qm.scanPythonQueueItems(rows)
		if err != nil {
			lastErr = err
			continue
		}

		return items, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to read Python queue (tried multiple schemas): %w", lastErr)
	}

	return items, nil
}

// scanPythonQueueItems scans queue items from rows
func (qm *QueueMigrator) scanPythonQueueItems(rows *sql.Rows) ([]*PythonQueueItem, error) {
	items := []*PythonQueueItem{}

	for rows.Next() {
		item := &PythonQueueItem{}
		var completedAt sql.NullTime
		var errorMessage sql.NullString

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
			&errorMessage,
			&item.RetryCount,
			&item.CreatedAt,
			&item.UpdatedAt,
			&completedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan queue item: %w", err)
		}

		if errorMessage.Valid {
			item.ErrorMessage = errorMessage.String
		}
		if completedAt.Valid {
			item.CompletedAt = &completedAt.Time
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating queue rows: %w", err)
	}

	return items, nil
}

// ConvertToGoQueueItem converts Python queue item to Go format
func (qm *QueueMigrator) ConvertToGoQueueItem(pythonItem *PythonQueueItem) *store.QueueItem {
	goItem := &store.QueueItem{
		ID:           pythonItem.ID,
		Type:         qm.mapItemType(pythonItem.Type),
		Title:        pythonItem.Title,
		Artist:       pythonItem.Artist,
		Album:        pythonItem.Album,
		Status:       qm.mapStatus(pythonItem.Status),
		Progress:     pythonItem.Progress,
		DownloadURL:  pythonItem.DownloadURL,
		OutputPath:   pythonItem.OutputPath,
		ErrorMessage: pythonItem.ErrorMessage,
		RetryCount:   pythonItem.RetryCount,
		CreatedAt:    pythonItem.CreatedAt,
		UpdatedAt:    pythonItem.UpdatedAt,
		CompletedAt:  pythonItem.CompletedAt,
	}

	return goItem
}

// mapItemType maps Python item types to Go item types
func (qm *QueueMigrator) mapItemType(pythonType string) string {
	typeMap := map[string]string{
		"track":    "track",
		"album":    "album",
		"playlist": "playlist",
		"song":     "track",
	}

	if mapped, ok := typeMap[pythonType]; ok {
		return mapped
	}

	return "track" // Default
}

// mapStatus maps Python status values to Go status values
func (qm *QueueMigrator) mapStatus(pythonStatus string) string {
	statusMap := map[string]string{
		"pending":     "pending",
		"downloading": "downloading",
		"completed":   "completed",
		"failed":      "failed",
		"queued":      "pending",
		"in_progress": "downloading",
		"done":        "completed",
		"error":       "failed",
	}

	if mapped, ok := statusMap[pythonStatus]; ok {
		return mapped
	}

	return "pending" // Default
}

// ImportQueueItems imports queue items into Go database
func (qm *QueueMigrator) ImportQueueItems(items []*PythonQueueItem) error {
	for _, pythonItem := range items {
		goItem := qm.ConvertToGoQueueItem(pythonItem)
		
		if err := qm.queueStore.Add(goItem); err != nil {
			// If item already exists, try updating instead
			if err := qm.queueStore.Update(goItem); err != nil {
				return fmt.Errorf("failed to import queue item %s: %w", goItem.ID, err)
			}
		}
	}

	return nil
}

// ReadPythonHistory reads download history from Python database
func (qm *QueueMigrator) ReadPythonHistory() ([]*PythonHistoryItem, error) {
	queries := []string{
		`SELECT track_id, title, artist, album, file_path, file_size, quality, downloaded_at
		 FROM download_history ORDER BY downloaded_at DESC`,
		
		`SELECT id, title, artist, album, path, size, quality, timestamp
		 FROM history ORDER BY timestamp DESC`,
	}

	var items []*PythonHistoryItem
	var lastErr error

	for _, query := range queries {
		rows, err := qm.pythonDB.Query(query)
		if err != nil {
			lastErr = err
			continue
		}
		defer rows.Close()

		items, err = qm.scanPythonHistoryItems(rows)
		if err != nil {
			lastErr = err
			continue
		}

		return items, nil
	}

	// History is optional, so if table doesn't exist, return empty list
	if lastErr != nil {
		return []*PythonHistoryItem{}, nil
	}

	return items, nil
}

// scanPythonHistoryItems scans history items from rows
func (qm *QueueMigrator) scanPythonHistoryItems(rows *sql.Rows) ([]*PythonHistoryItem, error) {
	items := []*PythonHistoryItem{}

	for rows.Next() {
		item := &PythonHistoryItem{}

		err := rows.Scan(
			&item.TrackID,
			&item.Title,
			&item.Artist,
			&item.Album,
			&item.FilePath,
			&item.FileSize,
			&item.Quality,
			&item.DownloadedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan history item: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating history rows: %w", err)
	}

	return items, nil
}

// ImportHistory imports download history into Go database
func (qm *QueueMigrator) ImportHistory(items []*PythonHistoryItem) error {
	for _, item := range items {
		err := qm.queueStore.AddToHistory(
			item.TrackID,
			item.Title,
			item.Artist,
			item.Album,
			item.FilePath,
			item.Quality,
			item.FileSize,
		)
		
		if err != nil {
			// Log but don't fail on history import errors
			fmt.Printf("Warning: failed to import history item %s: %v\n", item.TrackID, err)
		}
	}

	return nil
}

// Migrate performs the complete queue migration
func (qm *QueueMigrator) Migrate() error {
	// Open databases
	if err := qm.Open(); err != nil {
		return fmt.Errorf("failed to open databases: %w", err)
	}
	defer qm.Close()

	// Read Python queue
	queueItems, err := qm.ReadPythonQueue()
	if err != nil {
		return fmt.Errorf("failed to read Python queue: %w", err)
	}

	// Import queue items
	if err := qm.ImportQueueItems(queueItems); err != nil {
		return fmt.Errorf("failed to import queue items: %w", err)
	}

	// Read Python history
	historyItems, err := qm.ReadPythonHistory()
	if err != nil {
		return fmt.Errorf("failed to read Python history: %w", err)
	}

	// Import history
	if err := qm.ImportHistory(historyItems); err != nil {
		return fmt.Errorf("failed to import history: %w", err)
	}

	return nil
}

// GetMigrationStats returns statistics about what will be migrated
func (qm *QueueMigrator) GetMigrationStats() (map[string]int, error) {
	stats := make(map[string]int)

	// Open Python database temporarily
	pythonDB, err := sql.Open("sqlite3", qm.pythonDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Python database: %w", err)
	}
	defer pythonDB.Close()

	// Try to count queue items
	var queueCount int
	queries := []string{
		"SELECT COUNT(*) FROM queue_items",
		"SELECT COUNT(*) FROM downloads",
		"SELECT COUNT(*) FROM queue",
	}

	for _, query := range queries {
		err = pythonDB.QueryRow(query).Scan(&queueCount)
		if err == nil {
			break
		}
	}
	if err != nil && err != sql.ErrNoRows {
		queueCount = 0 // If table doesn't exist, assume 0
	}
	stats["queue_items"] = queueCount

	// Try to count history items
	var historyCount int
	historyQueries := []string{
		"SELECT COUNT(*) FROM download_history",
		"SELECT COUNT(*) FROM history",
	}

	for _, query := range historyQueries {
		err = pythonDB.QueryRow(query).Scan(&historyCount)
		if err == nil {
			break
		}
	}
	if err != nil && err != sql.ErrNoRows {
		historyCount = 0 // If table doesn't exist, assume 0
	}
	stats["history_items"] = historyCount

	return stats, nil
}
