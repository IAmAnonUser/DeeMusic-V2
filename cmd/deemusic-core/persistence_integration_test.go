package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/deemusic/deemusic-go/internal/config"
	"github.com/deemusic/deemusic-go/internal/store"
)

// TestAppRestartPersistence simulates a full app restart and verifies data persists
func TestAppRestartPersistence(t *testing.T) {
	// Create temporary directory for test data
	tmpDir := t.TempDir()
	
	// Set up paths
	configPath := filepath.Join(tmpDir, "settings.json")
	dbPath := filepath.Join(tmpDir, "data", "queue.db")
	
	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}
	
	// Create initial configuration
	initialConfig := &config.Config{
		Deezer: config.DeezerConfig{
			ARL: "test_arl_token_12345",
		},
		Download: config.DownloadConfig{
			OutputDir:           filepath.Join(tmpDir, "downloads"),
			Quality:             "MP3_320",
			ConcurrentDownloads: 8,
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
			RunOnStartup:   false,
			MinimizeToTray: true,
			StartMinimized: false,
			Theme:          "dark",
			Language:       "en",
		},
		Logging: config.LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "file",
			FilePath:   filepath.Join(tmpDir, "logs", "app.log"),
			MaxSizeMB:  100,
			MaxBackups: 3,
			MaxAgeDays: 30,
			Compress:   true,
		},
	}
	
	// Save initial config
	if err := initialConfig.Save(configPath); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}
	
	// === FIRST APP SESSION ===
	t.Log("Starting first app session...")
	
	// Initialize database
	db1, err := store.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	
	queueStore1 := store.NewQueueStore(db1)
	
	// Add some queue items
	testItems := []*store.QueueItem{
		{
			ID:              "track-001",
			Type:            "track",
			Title:           "Bohemian Rhapsody",
			Artist:          "Queen",
			Album:           "A Night at the Opera",
			Status:          "pending",
			Progress:        0,
			BytesDownloaded: 0,
			TotalBytes:      8192000,
		},
		{
			ID:              "track-002",
			Type:            "track",
			Title:           "Stairway to Heaven",
			Artist:          "Led Zeppelin",
			Album:           "Led Zeppelin IV",
			Status:          "failed",
			Progress:        35,
			BytesDownloaded: 2867200,
			TotalBytes:      8192000,
			PartialFilePath: filepath.Join(tmpDir, "partial", "track-002.mp3.part"),
		},
		{
			ID:       "track-003",
			Type:     "track",
			Title:    "Hotel California",
			Artist:   "Eagles",
			Album:    "Hotel California",
			Status:   "completed",
			Progress: 100,
		},
	}
	
	for _, item := range testItems {
		if err := queueStore1.Add(item); err != nil {
			t.Fatalf("Failed to add queue item: %v", err)
		}
	}
	
	// Add to download history
	if err := queueStore1.AddToHistory(
		"track-003",
		"Hotel California",
		"Eagles",
		"Hotel California",
		filepath.Join(tmpDir, "downloads", "Eagles", "Hotel California", "Hotel California.mp3"),
		"MP3_320",
		8192000,
	); err != nil {
		t.Fatalf("Failed to add to history: %v", err)
	}
	
	// Set some config cache
	if err := queueStore1.SetConfigCache("last_check", time.Now().Format(time.RFC3339)); err != nil {
		t.Fatalf("Failed to set config cache: %v", err)
	}
	
	// Get stats before closing
	stats1, err := queueStore1.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	
	t.Logf("First session stats: Total=%d, Pending=%d, Downloading=%d, Completed=%d, Failed=%d",
		stats1.Total, stats1.Pending, stats1.Downloading, stats1.Completed, stats1.Failed)
	
	// Close database (simulating app shutdown)
	db1.Close()
	t.Log("First app session ended (database closed)")
	
	// === SIMULATE APP RESTART ===
	time.Sleep(100 * time.Millisecond)
	
	// === SECOND APP SESSION ===
	t.Log("Starting second app session (after restart)...")
	
	// Load configuration
	loadedConfig, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config after restart: %v", err)
	}
	
	// Verify config persisted
	if loadedConfig.Download.Quality != initialConfig.Download.Quality {
		t.Errorf("Config quality mismatch: expected %s, got %s",
			initialConfig.Download.Quality, loadedConfig.Download.Quality)
	}
	if loadedConfig.System.Theme != initialConfig.System.Theme {
		t.Errorf("Config theme mismatch: expected %s, got %s",
			initialConfig.System.Theme, loadedConfig.System.Theme)
	}
	
	// Reopen database
	db2, err := store.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()
	
	queueStore2 := store.NewQueueStore(db2)
	
	// Verify queue items persisted
	for _, originalItem := range testItems {
		retrieved, err := queueStore2.GetByID(originalItem.ID)
		if err != nil {
			t.Errorf("Failed to retrieve item %s after restart: %v", originalItem.ID, err)
			continue
		}
		
		if retrieved.Title != originalItem.Title {
			t.Errorf("Item %s title mismatch: expected %s, got %s",
				originalItem.ID, originalItem.Title, retrieved.Title)
		}
		if retrieved.Status != originalItem.Status {
			t.Errorf("Item %s status mismatch: expected %s, got %s",
				originalItem.ID, originalItem.Status, retrieved.Status)
		}
		if retrieved.Progress != originalItem.Progress {
			t.Errorf("Item %s progress mismatch: expected %d, got %d",
				originalItem.ID, originalItem.Progress, retrieved.Progress)
		}
		
		// Verify resumable download info persisted
		if originalItem.PartialFilePath != "" {
			if retrieved.PartialFilePath != originalItem.PartialFilePath {
				t.Errorf("Item %s partial path mismatch: expected %s, got %s",
					originalItem.ID, originalItem.PartialFilePath, retrieved.PartialFilePath)
			}
			if retrieved.BytesDownloaded != originalItem.BytesDownloaded {
				t.Errorf("Item %s bytes downloaded mismatch: expected %d, got %d",
					originalItem.ID, originalItem.BytesDownloaded, retrieved.BytesDownloaded)
			}
			if !retrieved.IsResumable() {
				t.Errorf("Item %s should be resumable after restart", originalItem.ID)
			}
		}
	}
	
	// Verify stats match
	stats2, err := queueStore2.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats after restart: %v", err)
	}
	
	t.Logf("Second session stats: Total=%d, Pending=%d, Downloading=%d, Completed=%d, Failed=%d",
		stats2.Total, stats2.Pending, stats2.Downloading, stats2.Completed, stats2.Failed)
	
	if stats2.Total != stats1.Total {
		t.Errorf("Total count mismatch: expected %d, got %d", stats1.Total, stats2.Total)
	}
	if stats2.Pending != stats1.Pending {
		t.Errorf("Pending count mismatch: expected %d, got %d", stats1.Pending, stats2.Pending)
	}
	if stats2.Downloading != stats1.Downloading {
		t.Errorf("Downloading count mismatch: expected %d, got %d", stats1.Downloading, stats2.Downloading)
	}
	if stats2.Completed != stats1.Completed {
		t.Errorf("Completed count mismatch: expected %d, got %d", stats1.Completed, stats2.Completed)
	}
	
	// Verify download history persisted
	history, err := queueStore2.GetHistory(0, 10)
	if err != nil {
		t.Fatalf("Failed to get history after restart: %v", err)
	}
	
	if len(history) != 1 {
		t.Errorf("Expected 1 history entry, got %d", len(history))
	} else {
		if history[0]["title"] != "Hotel California" {
			t.Errorf("History title mismatch: expected Hotel California, got %s", history[0]["title"])
		}
	}
	
	// Verify config cache persisted
	cachedValue, err := queueStore2.GetConfigCache("last_check")
	if err != nil {
		t.Errorf("Failed to get config cache after restart: %v", err)
	} else {
		t.Logf("Config cache persisted: last_check=%s", cachedValue)
	}
	
	// Verify resumable downloads can be retrieved
	resumable, err := queueStore2.GetResumableDownloads(10)
	if err != nil {
		t.Fatalf("Failed to get resumable downloads: %v", err)
	}
	
	if len(resumable) != 1 {
		t.Errorf("Expected 1 resumable download, got %d", len(resumable))
	} else {
		if resumable[0].ID != "track-002" {
			t.Errorf("Expected resumable download track-002, got %s", resumable[0].ID)
		}
	}
	
	t.Log("Second app session completed successfully")
	t.Log("✓ All data persisted correctly across app restart")
}

// TestConfigUpdatePersistence verifies that config updates persist
func TestConfigUpdatePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "settings.json")
	
	// Create initial config
	cfg := &config.Config{
		Download: config.DownloadConfig{
			OutputDir:           "/initial/path",
			Quality:             "MP3_320",
			ConcurrentDownloads: 8,
			EmbedArtwork:        true,
			ArtworkSize:         1200,
			FilenameTemplate:    "{artist} - {title}",
			FolderStructure: map[string]string{
				"track": "{artist}/{album}",
			},
		},
		Lyrics: config.LyricsConfig{
			Enabled:     true,
			EmbedInFile: true,
			Language:    "en",
		},
		Network: config.NetworkConfig{
			Timeout:          30,
			MaxRetries:       3,
			ConnectionsPerDL: 1,
		},
		System: config.SystemConfig{
			Theme:    "dark",
			Language: "en",
		},
		Logging: config.LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "file",
			MaxSizeMB:  100,
			MaxBackups: 3,
			MaxAgeDays: 30,
		},
	}
	
	// Save initial config
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}
	
	// Load and modify
	loadedCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// Update settings
	loadedCfg.Download.Quality = "FLAC"
	loadedCfg.Download.ConcurrentDownloads = 12
	loadedCfg.System.Theme = "light"
	
	// Save updated config
	if err := loadedCfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save updated config: %v", err)
	}
	
	// Load again and verify updates persisted
	finalCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}
	
	if finalCfg.Download.Quality != "FLAC" {
		t.Errorf("Quality update did not persist: expected FLAC, got %s", finalCfg.Download.Quality)
	}
	if finalCfg.Download.ConcurrentDownloads != 12 {
		t.Errorf("ConcurrentDownloads update did not persist: expected 12, got %d", finalCfg.Download.ConcurrentDownloads)
	}
	if finalCfg.System.Theme != "light" {
		t.Errorf("Theme update did not persist: expected light, got %s", finalCfg.System.Theme)
	}
}

// TestDatabaseMigrationStability verifies migrations work correctly
func TestDatabaseMigrationStability(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "migration_test.db")
	
	// First initialization
	db1, err := store.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	
	// Add some data
	queueStore := store.NewQueueStore(db1)
	item := &store.QueueItem{
		ID:     "test-migration",
		Type:   "track",
		Title:  "Test Track",
		Status: "pending",
	}
	
	if err := queueStore.Add(item); err != nil {
		t.Fatalf("Failed to add item: %v", err)
	}
	
	db1.Close()
	
	// Reopen database (migrations should not re-run)
	db2, err := store.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()
	
	// Verify data is still there
	queueStore2 := store.NewQueueStore(db2)
	retrieved, err := queueStore2.GetByID("test-migration")
	if err != nil {
		t.Fatalf("Failed to retrieve item after migration: %v", err)
	}
	
	if retrieved.Title != item.Title {
		t.Errorf("Data corrupted after migration: expected %s, got %s", item.Title, retrieved.Title)
	}
	
	// Verify all expected columns exist (including migration 2 columns)
	var count int
	err = db2.QueryRow("SELECT COUNT(*) FROM pragma_table_info('queue_items') WHERE name IN ('partial_file_path', 'bytes_downloaded', 'total_bytes')").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to check columns: %v", err)
	}
	
	if count != 3 {
		t.Errorf("Expected 3 resume-related columns, found %d", count)
	}
}
