package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/deemusic/deemusic-go/internal/config"
)

// TestDatabasePersistence verifies that queue data persists across database connections
func TestDatabasePersistence(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "persistence_test.db")

	// First connection - add data
	db1, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	store1 := NewQueueStore(db1)

	// Add test items
	items := []*QueueItem{
		{
			ID:              "track-1",
			Type:            "track",
			Title:           "Test Track 1",
			Artist:          "Test Artist",
			Album:           "Test Album",
			Status:          "pending",
			Progress:        0,
			BytesDownloaded: 0,
			TotalBytes:      1024000,
		},
		{
			ID:              "track-2",
			Type:            "track",
			Title:           "Test Track 2",
			Artist:          "Test Artist 2",
			Album:           "Test Album 2",
			Status:          "downloading",
			Progress:        45,
			BytesDownloaded: 460800,
			TotalBytes:      1024000,
			PartialFilePath: "/tmp/partial.mp3",
		},
		{
			ID:       "track-3",
			Type:     "track",
			Title:    "Test Track 3",
			Artist:   "Test Artist 3",
			Album:    "Test Album 3",
			Status:   "completed",
			Progress: 100,
		},
	}

	for _, item := range items {
		if err := store1.Add(item); err != nil {
			t.Fatalf("Failed to add item: %v", err)
		}
	}

	// Close first connection
	db1.Close()

	// Second connection - verify data persists
	db2, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	store2 := NewQueueStore(db2)

	// Verify all items are still there
	for _, originalItem := range items {
		retrieved, err := store2.GetByID(originalItem.ID)
		if err != nil {
			t.Errorf("Failed to retrieve item %s after restart: %v", originalItem.ID, err)
			continue
		}

		if retrieved.Title != originalItem.Title {
			t.Errorf("Item %s: expected title %s, got %s", originalItem.ID, originalItem.Title, retrieved.Title)
		}
		if retrieved.Status != originalItem.Status {
			t.Errorf("Item %s: expected status %s, got %s", originalItem.ID, originalItem.Status, retrieved.Status)
		}
		if retrieved.Progress != originalItem.Progress {
			t.Errorf("Item %s: expected progress %d, got %d", originalItem.ID, originalItem.Progress, retrieved.Progress)
		}
		if retrieved.BytesDownloaded != originalItem.BytesDownloaded {
			t.Errorf("Item %s: expected bytes downloaded %d, got %d", originalItem.ID, originalItem.BytesDownloaded, retrieved.BytesDownloaded)
		}
	}

	// Verify stats
	stats, err := store2.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Total != 3 {
		t.Errorf("Expected total 3, got %d", stats.Total)
	}
	if stats.Pending != 1 {
		t.Errorf("Expected pending 1, got %d", stats.Pending)
	}
	if stats.Downloading != 1 {
		t.Errorf("Expected downloading 1, got %d", stats.Downloading)
	}
	if stats.Completed != 1 {
		t.Errorf("Expected completed 1, got %d", stats.Completed)
	}
}

// TestResumableDownloadsPersistence verifies that resumable download info persists
func TestResumableDownloadsPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "resumable_test.db")

	// First connection - add resumable download
	db1, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	store1 := NewQueueStore(db1)

	item := &QueueItem{
		ID:              "resumable-1",
		Type:            "track",
		Title:           "Resumable Track",
		Artist:          "Test Artist",
		Status:          "failed",
		Progress:        60,
		PartialFilePath: "/tmp/partial_download.mp3",
		BytesDownloaded: 6144000,
		TotalBytes:      10240000,
	}

	if err := store1.Add(item); err != nil {
		t.Fatalf("Failed to add item: %v", err)
	}

	db1.Close()

	// Second connection - verify resumable info persists
	db2, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	store2 := NewQueueStore(db2)

	// Get resumable downloads
	resumable, err := store2.GetResumableDownloads(10)
	if err != nil {
		t.Fatalf("Failed to get resumable downloads: %v", err)
	}

	if len(resumable) != 1 {
		t.Fatalf("Expected 1 resumable download, got %d", len(resumable))
	}

	retrieved := resumable[0]
	if retrieved.ID != item.ID {
		t.Errorf("Expected ID %s, got %s", item.ID, retrieved.ID)
	}
	if retrieved.PartialFilePath != item.PartialFilePath {
		t.Errorf("Expected partial path %s, got %s", item.PartialFilePath, retrieved.PartialFilePath)
	}
	if retrieved.BytesDownloaded != item.BytesDownloaded {
		t.Errorf("Expected bytes downloaded %d, got %d", item.BytesDownloaded, retrieved.BytesDownloaded)
	}
	if !retrieved.IsResumable() {
		t.Error("Item should be resumable")
	}
}

// TestDownloadHistoryPersistence verifies that download history persists
func TestDownloadHistoryPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "history_test.db")

	// First connection - add history
	db1, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	store1 := NewQueueStore(db1)

	// Add history entries
	entries := []struct {
		trackID  string
		title    string
		artist   string
		album    string
		filePath string
		quality  string
		fileSize int64
	}{
		{"123", "Track 1", "Artist 1", "Album 1", "/music/track1.mp3", "MP3_320", 8192000},
		{"456", "Track 2", "Artist 2", "Album 2", "/music/track2.flac", "FLAC", 32768000},
	}

	for _, entry := range entries {
		err := store1.AddToHistory(
			entry.trackID,
			entry.title,
			entry.artist,
			entry.album,
			entry.filePath,
			entry.quality,
			entry.fileSize,
		)
		if err != nil {
			t.Fatalf("Failed to add history entry: %v", err)
		}
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	db1.Close()

	// Second connection - verify history persists
	db2, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	store2 := NewQueueStore(db2)

	history, err := store2.GetHistory(0, 10)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	if len(history) != 2 {
		t.Fatalf("Expected 2 history entries, got %d", len(history))
	}

	// Verify entries (should be in reverse chronological order)
	if history[0]["title"] != "Track 2" {
		t.Errorf("Expected first entry to be Track 2, got %s", history[0]["title"])
	}
	if history[1]["title"] != "Track 1" {
		t.Errorf("Expected second entry to be Track 1, got %s", history[1]["title"])
	}
}

// TestConfigCachePersistence verifies that config cache persists
func TestConfigCachePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "config_cache_test.db")

	// First connection - set cache values
	db1, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	store1 := NewQueueStore(db1)

	cacheEntries := map[string]string{
		"last_arl_check":    "2024-01-15T10:30:00Z",
		"last_update_check": "2024-01-15T09:00:00Z",
		"user_preferences":  `{"theme":"dark","language":"en"}`,
	}

	for key, value := range cacheEntries {
		if err := store1.SetConfigCache(key, value); err != nil {
			t.Fatalf("Failed to set config cache: %v", err)
		}
	}

	db1.Close()

	// Second connection - verify cache persists
	db2, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	store2 := NewQueueStore(db2)

	for key, expectedValue := range cacheEntries {
		value, err := store2.GetConfigCache(key)
		if err != nil {
			t.Errorf("Failed to get config cache for key %s: %v", key, err)
			continue
		}
		if value != expectedValue {
			t.Errorf("Key %s: expected value %s, got %s", key, expectedValue, value)
		}
	}
}

// TestMigrationPersistence verifies that migrations are tracked correctly
func TestMigrationPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "migration_test.db")

	// First connection - run migrations
	db1, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	if err := RunMigrations(db1); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Check migration version
	version1, err := getCurrentVersion(db1)
	if err != nil {
		t.Fatalf("Failed to get version: %v", err)
	}

	db1.Close()

	// Second connection - verify migrations don't re-run
	db2, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	if err := RunMigrations(db2); err != nil {
		t.Fatalf("Failed to run migrations on second connection: %v", err)
	}

	version2, err := getCurrentVersion(db2)
	if err != nil {
		t.Fatalf("Failed to get version on second connection: %v", err)
	}

	if version1 != version2 {
		t.Errorf("Migration version changed: %d -> %d", version1, version2)
	}

	// Verify all expected tables exist
	tables := []string{"queue_items", "download_history", "config_cache", "schema_migrations"}
	for _, table := range tables {
		var count int
		err := db2.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Errorf("Failed to check table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("Table %s does not exist", table)
		}
	}
}

// TestSettingsPersistence verifies that settings persist across saves and loads
func TestSettingsPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "settings.json")

	// Create test config
	cfg := &config.Config{
		Deezer: config.DeezerConfig{
			ARL: "test_arl_token",
		},
		Download: config.DownloadConfig{
			OutputDir:           "/test/downloads",
			Quality:             "FLAC",
			ConcurrentDownloads: 12,
			EmbedArtwork:        true,
			ArtworkSize:         1200,
			FilenameTemplate:    "{artist} - {title}",
			FolderStructure: map[string]string{
				"track":    "{artist}/{album}",
				"album":    "{artist}/{album}",
				"playlist": "Playlists/{playlist}",
			},
		},
		Lyrics: config.LyricsConfig{
			Enabled:          true,
			EmbedInFile:      true,
			SaveSeparateFile: false,
			Language:         "en",
		},
		Network: config.NetworkConfig{
			Timeout:          30,
			MaxRetries:       3,
			BandwidthLimit:   0,
			ConnectionsPerDL: 1,
		},
		System: config.SystemConfig{
			RunOnStartup:   true,
			MinimizeToTray: true,
			StartMinimized: false,
			Theme:          "dark",
			Language:       "en",
		},
		Logging: config.LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "file",
			FilePath:   "/test/logs/app.log",
			MaxSizeMB:  100,
			MaxBackups: 3,
			MaxAgeDays: 30,
			Compress:   true,
		},
	}

	// Save config
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load config
	loadedCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify all settings persisted correctly
	if loadedCfg.Download.Quality != cfg.Download.Quality {
		t.Errorf("Quality: expected %s, got %s", cfg.Download.Quality, loadedCfg.Download.Quality)
	}
	if loadedCfg.Download.ConcurrentDownloads != cfg.Download.ConcurrentDownloads {
		t.Errorf("ConcurrentDownloads: expected %d, got %d", cfg.Download.ConcurrentDownloads, loadedCfg.Download.ConcurrentDownloads)
	}
	if loadedCfg.Download.OutputDir != cfg.Download.OutputDir {
		t.Errorf("OutputDir: expected %s, got %s", cfg.Download.OutputDir, loadedCfg.Download.OutputDir)
	}
	if loadedCfg.System.Theme != cfg.System.Theme {
		t.Errorf("Theme: expected %s, got %s", cfg.System.Theme, loadedCfg.System.Theme)
	}
	if loadedCfg.System.RunOnStartup != cfg.System.RunOnStartup {
		t.Errorf("RunOnStartup: expected %v, got %v", cfg.System.RunOnStartup, loadedCfg.System.RunOnStartup)
	}
	if loadedCfg.Logging.Level != cfg.Logging.Level {
		t.Errorf("LogLevel: expected %s, got %s", cfg.Logging.Level, loadedCfg.Logging.Level)
	}
}

// TestLargeQueuePersistence verifies that large queues persist correctly
func TestLargeQueuePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "large_queue_test.db")

	// First connection - add many items
	db1, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	store1 := NewQueueStore(db1)

	// Add 1000 items
	itemCount := 1000
	for i := 0; i < itemCount; i++ {
		item := &QueueItem{
			ID:       fmt.Sprintf("track-%d", i),
			Type:     "track",
			Title:    fmt.Sprintf("Track %d", i),
			Artist:   fmt.Sprintf("Artist %d", i%100),
			Album:    fmt.Sprintf("Album %d", i%50),
			Status:   "pending",
			Progress: 0,
		}
		if err := store1.Add(item); err != nil {
			t.Fatalf("Failed to add item %d: %v", i, err)
		}
	}

	db1.Close()

	// Second connection - verify all items persist
	db2, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()

	store2 := NewQueueStore(db2)

	stats, err := store2.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Total != itemCount {
		t.Errorf("Expected total %d, got %d", itemCount, stats.Total)
	}

	// Verify pagination works
	page1, err := store2.GetAll(0, 100)
	if err != nil {
		t.Fatalf("Failed to get page 1: %v", err)
	}
	if len(page1) != 100 {
		t.Errorf("Expected 100 items in page 1, got %d", len(page1))
	}

	page2, err := store2.GetAll(100, 100)
	if err != nil {
		t.Fatalf("Failed to get page 2: %v", err)
	}
	if len(page2) != 100 {
		t.Errorf("Expected 100 items in page 2, got %d", len(page2))
	}
}
