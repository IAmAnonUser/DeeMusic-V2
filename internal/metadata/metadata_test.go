package metadata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	// Test with nil config
	manager := NewManager(nil)
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}
	if manager.config == nil {
		t.Fatal("Manager config is nil")
	}
	if !manager.config.EmbedArtwork {
		t.Error("Default EmbedArtwork should be true")
	}
	if manager.config.ArtworkSize != 1200 {
		t.Errorf("Default ArtworkSize should be 1200, got %d", manager.config.ArtworkSize)
	}

	// Test with custom config
	customConfig := &Config{
		EmbedArtwork: false,
		ArtworkSize:  800,
	}
	manager = NewManager(customConfig)
	if manager.config.EmbedArtwork {
		t.Error("Custom EmbedArtwork should be false")
	}
	if manager.config.ArtworkSize != 800 {
		t.Errorf("Custom ArtworkSize should be 800, got %d", manager.config.ArtworkSize)
	}
}

func TestTrackMetadata(t *testing.T) {
	metadata := &TrackMetadata{
		Title:       "Test Song",
		Artist:      "Test Artist",
		Album:       "Test Album",
		AlbumArtist: "Test Album Artist",
		TrackNumber: 1,
		DiscNumber:  1,
		Year:        2024,
		Genre:       "Test Genre",
		ISRC:        "TEST12345678",
		Label:       "Test Label",
		Copyright:   "© 2024 Test",
	}

	if metadata.Title != "Test Song" {
		t.Errorf("Expected Title 'Test Song', got '%s'", metadata.Title)
	}
	if metadata.TrackNumber != 1 {
		t.Errorf("Expected TrackNumber 1, got %d", metadata.TrackNumber)
	}
	if metadata.Year != 2024 {
		t.Errorf("Expected Year 2024, got %d", metadata.Year)
	}
}

func TestFileExists(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	
	// File doesn't exist yet
	if FileExists(tmpFile) {
		t.Error("FileExists should return false for non-existent file")
	}

	// Create the file
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// File should exist now
	if !FileExists(tmpFile) {
		t.Error("FileExists should return true for existing file")
	}
}

func TestLyricsConfig(t *testing.T) {
	config := &LyricsConfig{
		EmbedInFile:      true,
		SaveSeparateFile: false,
		Language:         "eng",
	}

	if !config.EmbedInFile {
		t.Error("EmbedInFile should be true")
	}
	if config.SaveSeparateFile {
		t.Error("SaveSeparateFile should be false")
	}
	if config.Language != "eng" {
		t.Errorf("Expected Language 'eng', got '%s'", config.Language)
	}
}

func TestLyrics(t *testing.T) {
	lyrics := &Lyrics{
		SyncedLyrics:   "[00:12.00]First line\n[00:15.50]Second line",
		UnsyncedLyrics: "First line\nSecond line",
		Language:       "eng",
	}

	if lyrics.SyncedLyrics == "" {
		t.Error("SyncedLyrics should not be empty")
	}
	if lyrics.UnsyncedLyrics == "" {
		t.Error("UnsyncedLyrics should not be empty")
	}
	if lyrics.Language != "eng" {
		t.Errorf("Expected Language 'eng', got '%s'", lyrics.Language)
	}
}

func TestMillisecondsToLRCTimestamp(t *testing.T) {
	manager := NewManager(nil)

	tests := []struct {
		ms       int
		expected string
	}{
		{0, "00:00.00"},
		{1000, "00:01.00"},
		{12000, "00:12.00"},
		{15500, "00:15.50"},
		{60000, "01:00.00"},
		{125500, "02:05.50"},
	}

	for _, test := range tests {
		result := manager.millisecondsToLRCTimestamp(test.ms)
		if result != test.expected {
			t.Errorf("millisecondsToLRCTimestamp(%d) = %s, expected %s", test.ms, result, test.expected)
		}
	}
}

func TestLRCTimestampToMilliseconds(t *testing.T) {
	manager := NewManager(nil)

	tests := []struct {
		timestamp string
		expected  int
	}{
		{"00:00.00", 0},
		{"00:01.00", 1000},
		{"00:12.00", 12000},
		{"00:15.50", 15500},
		{"01:00.00", 60000},
		{"02:05.50", 125500},
	}

	for _, test := range tests {
		result := manager.lrcTimestampToMilliseconds(test.timestamp)
		if result != test.expected {
			t.Errorf("lrcTimestampToMilliseconds(%s) = %d, expected %d", test.timestamp, result, test.expected)
		}
	}

	// Test invalid formats
	invalid := []string{
		"invalid",
		"00",
		"00:00",
		"00:00:00",
		"aa:bb.cc",
	}

	for _, ts := range invalid {
		result := manager.lrcTimestampToMilliseconds(ts)
		if result != -1 {
			t.Errorf("lrcTimestampToMilliseconds(%s) should return -1 for invalid format, got %d", ts, result)
		}
	}
}

func TestWriteUint32BE(t *testing.T) {
	tests := []struct {
		value    uint32
		expected []byte
	}{
		{0, []byte{0, 0, 0, 0}},
		{1, []byte{0, 0, 0, 1}},
		{256, []byte{0, 0, 1, 0}},
		{65536, []byte{0, 1, 0, 0}},
		{16777216, []byte{1, 0, 0, 0}},
		{0x12345678, []byte{0x12, 0x34, 0x56, 0x78}},
	}

	for _, test := range tests {
		result := make([]byte, 4)
		writeUint32BE(result, test.value)
		for i := 0; i < 4; i++ {
			if result[i] != test.expected[i] {
				t.Errorf("writeUint32BE(%d) = %v, expected %v", test.value, result, test.expected)
				break
			}
		}
	}
}

func TestArtworkCache(t *testing.T) {
	tmpDir := t.TempDir()
	
	cache, err := NewArtworkCache(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create artwork cache: %v", err)
	}

	if cache.cacheDir != tmpDir {
		t.Errorf("Expected cache dir %s, got %s", tmpDir, cache.cacheDir)
	}

	// Test cache key generation
	key1 := cache.generateCacheKey("http://example.com/image.jpg", 1200)
	key2 := cache.generateCacheKey("http://example.com/image.jpg", 1200)
	key3 := cache.generateCacheKey("http://example.com/other.jpg", 1200)

	if key1 != key2 {
		t.Error("Same URL and size should generate same cache key")
	}
	if key1 == key3 {
		t.Error("Different URLs should generate different cache keys")
	}
}

func TestNewArtworkCacheErrors(t *testing.T) {
	// Test with empty cache dir
	_, err := NewArtworkCache("")
	if err == nil {
		t.Error("NewArtworkCache should return error for empty cache dir")
	}
}
