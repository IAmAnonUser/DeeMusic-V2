package api

import (
	"context"
	"testing"
	"time"
)

func TestNewDeezerClient(t *testing.T) {
	client := NewDeezerClient(30 * time.Second)
	
	if client == nil {
		t.Fatal("NewDeezerClient returned nil")
	}
	
	if client.httpClient == nil {
		t.Error("HTTP client not initialized")
	}
	
	if client.rateLimiter == nil {
		t.Error("Rate limiter not initialized")
	}
	
	if client.IsAuthenticated() {
		t.Error("Client should not be authenticated initially")
	}
}

func TestAuthenticateEmptyARL(t *testing.T) {
	client := NewDeezerClient(30 * time.Second)
	ctx := context.Background()
	
	err := client.Authenticate(ctx, "")
	if err == nil {
		t.Error("Expected error for empty ARL token")
	}
}

func TestGetFormatCode(t *testing.T) {
	tests := []struct {
		quality  string
		expected string
	}{
		{QualityMP3128, "MP3_128"},
		{QualityMP3320, "MP3_320"},
		{QualityFLAC, "FLAC"},
		{"invalid", "MP3_320"}, // default
	}
	
	for _, tt := range tests {
		result := getFormatCode(tt.quality)
		if result != tt.expected {
			t.Errorf("getFormatCode(%s) = %s, want %s", tt.quality, result, tt.expected)
		}
	}
}

func TestGetFormatFromQuality(t *testing.T) {
	tests := []struct {
		quality  string
		expected string
	}{
		{QualityMP3128, "mp3"},
		{QualityMP3320, "mp3"},
		{QualityFLAC, "flac"},
	}
	
	for _, tt := range tests {
		result := getFormatFromQuality(tt.quality)
		if result != tt.expected {
			t.Errorf("getFormatFromQuality(%s) = %s, want %s", tt.quality, result, tt.expected)
		}
	}
}

func TestMillisecondsToLRCTimestamp(t *testing.T) {
	tests := []struct {
		ms       int
		expected string
	}{
		{0, "00:00.00"},
		{1000, "00:01.00"},
		{60000, "01:00.00"},
		{61500, "01:01.50"},
		{125750, "02:05.75"},
		{-100, "00:00.00"}, // negative should be treated as 0
	}
	
	for _, tt := range tests {
		result := millisecondsToLRCTimestamp(tt.ms)
		if result != tt.expected {
			t.Errorf("millisecondsToLRCTimestamp(%d) = %s, want %s", tt.ms, result, tt.expected)
		}
	}
}

func TestLRCTimestampToMilliseconds(t *testing.T) {
	tests := []struct {
		timestamp string
		expected  int
	}{
		{"00:00.00", 0},
		{"00:01.00", 1000},
		{"01:00.00", 60000},
		{"01:01.50", 61500},
		{"02:05.75", 125750},
		{"invalid", -1},
		{"00:00", -1}, // missing centiseconds
	}
	
	for _, tt := range tests {
		result := lrcTimestampToMilliseconds(tt.timestamp)
		if result != tt.expected {
			t.Errorf("lrcTimestampToMilliseconds(%s) = %d, want %d", tt.timestamp, result, tt.expected)
		}
	}
}

func TestParseLRCLyrics(t *testing.T) {
	lrcContent := `[00:00.00]First line
[00:05.50]Second line
[00:10.75]Third line`
	
	lines := ParseLRCLyrics(lrcContent)
	
	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines, got %d", len(lines))
	}
	
	if lines[0].Line != "First line" {
		t.Errorf("First line text = %s, want 'First line'", lines[0].Line)
	}
	
	if lines[0].Milliseconds != 0 {
		t.Errorf("First line ms = %d, want 0", lines[0].Milliseconds)
	}
	
	if lines[1].Milliseconds != 5500 {
		t.Errorf("Second line ms = %d, want 5500", lines[1].Milliseconds)
	}
	
	if lines[2].Milliseconds != 10750 {
		t.Errorf("Third line ms = %d, want 10750", lines[2].Milliseconds)
	}
}

func TestLyricsHasLyrics(t *testing.T) {
	tests := []struct {
		name     string
		lyrics   *Lyrics
		expected bool
	}{
		{
			name:     "No lyrics",
			lyrics:   &Lyrics{},
			expected: false,
		},
		{
			name: "Has synced lyrics",
			lyrics: &Lyrics{
				SyncedLyrics: "[00:00.00]Test",
			},
			expected: true,
		},
		{
			name: "Has unsynced lyrics",
			lyrics: &Lyrics{
				UnsyncedLyrics: "Test lyrics",
			},
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.lyrics.HasLyrics()
			if result != tt.expected {
				t.Errorf("HasLyrics() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLyricsHasSynchronizedLyrics(t *testing.T) {
	tests := []struct {
		name     string
		lyrics   *Lyrics
		expected bool
	}{
		{
			name:     "No lyrics",
			lyrics:   &Lyrics{},
			expected: false,
		},
		{
			name: "Has synced lyrics string",
			lyrics: &Lyrics{
				SyncedLyrics: "[00:00.00]Test",
			},
			expected: true,
		},
		{
			name: "Has synchronized array",
			lyrics: &Lyrics{
				Synchronized: []*LyricLine{
					{Line: "Test", Milliseconds: 0},
				},
			},
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.lyrics.HasSynchronizedLyrics()
			if result != tt.expected {
				t.Errorf("HasSynchronizedLyrics() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCache(t *testing.T) {
	c := newCache(1 * time.Second)
	
	// Test set and get
	c.set("key1", "value1")
	
	val, ok := c.get("key1")
	if !ok {
		t.Error("Expected to find key1 in cache")
	}
	
	if val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}
	
	// Test non-existent key
	_, ok = c.get("nonexistent")
	if ok {
		t.Error("Expected not to find nonexistent key")
	}
	
	// Test expiration
	time.Sleep(1100 * time.Millisecond)
	_, ok = c.get("key1")
	if ok {
		t.Error("Expected key1 to be expired")
	}
}
