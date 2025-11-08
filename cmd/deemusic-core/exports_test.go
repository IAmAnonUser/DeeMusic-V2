package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/deemusic/deemusic-go/internal/config"
)

// TestCheckInitialized tests the initialization check function
func TestCheckInitialized(t *testing.T) {
	// Reset state
	mu.Lock()
	initialized = false
	mu.Unlock()
	
	if checkInitialized() {
		t.Error("Should not be initialized initially")
	}
	
	// Set initialized
	mu.Lock()
	initialized = true
	mu.Unlock()
	
	if !checkInitialized() {
		t.Error("Should be initialized after setting")
	}
	
	// Reset for other tests
	mu.Lock()
	initialized = false
	mu.Unlock()
}

// TestCallbackNotifier tests the callback notifier
func TestCallbackNotifier(t *testing.T) {
	notifier := &CallbackNotifier{}
	
	// Test with nil callbacks (should not crash)
	callbackMu.Lock()
	progressCb = nil
	statusCb = nil
	queueUpdateCb = nil
	callbackMu.Unlock()
	
	// These should not crash
	notifier.NotifyProgress("test", 50, 100, 200)
	notifier.NotifyStarted("test")
	notifier.NotifyCompleted("test")
	notifier.NotifyFailed("test", os.ErrNotExist)
	notifier.notifyQueueUpdate()
}

// TestGlobalState tests global state management
func TestGlobalState(t *testing.T) {
	// Test initial state
	mu.RLock()
	if initialized {
		t.Error("Should not be initialized at start")
	}
	mu.RUnlock()
	
	// Test callback mutex
	callbackMu.Lock()
	callbackMu.Unlock()
	
	// Verify all global variables exist
	if ctx != nil {
		t.Log("Context exists")
	}
	if cancel != nil {
		t.Log("Cancel function exists")
	}
}

// TestConfigManagement tests configuration handling
func TestConfigManagement(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := setupTestConfig(t, tmpDir)
	
	// Load config
	loadedCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// Verify config values
	if loadedCfg.Download.Quality != "MP3_320" {
		t.Errorf("Expected quality MP3_320, got %s", loadedCfg.Download.Quality)
	}
	
	if loadedCfg.System.Theme != "dark" {
		t.Errorf("Expected theme dark, got %s", loadedCfg.System.Theme)
	}
}

// TestThreadSafety tests concurrent access to global state
func TestThreadSafety(t *testing.T) {
	var wg sync.WaitGroup
	
	// Test concurrent reads of initialized state
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = checkInitialized()
		}()
	}
	
	wg.Wait()
	
	// Test concurrent callback access
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			callbackMu.RLock()
			_ = progressCb
			_ = statusCb
			_ = queueUpdateCb
			callbackMu.RUnlock()
		}()
	}
	
	wg.Wait()
}

// Helper function to setup test config
func setupTestConfig(t *testing.T, tmpDir string) string {
	configPath := filepath.Join(tmpDir, "settings.json")
	
	cfg := &config.Config{
		Deezer: config.DeezerConfig{
			ARL: "",
		},
		Download: config.DownloadConfig{
			OutputDir:           filepath.Join(tmpDir, "downloads"),
			Quality:             "MP3_320",
			ConcurrentDownloads: 3,
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
	
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	
	return configPath
}
