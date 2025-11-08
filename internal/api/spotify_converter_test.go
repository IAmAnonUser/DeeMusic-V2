package api

import (
	"testing"
)

func TestSpotifyConverter_normalizeString(t *testing.T) {
	sc := &SpotifyConverter{}
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic normalization",
			input:    "Hello World",
			expected: "hello world",
		},
		{
			name:     "Remove feat",
			input:    "Song feat. Artist",
			expected: "song artist",
		},
		{
			name:     "Remove ft",
			input:    "Song ft. Artist",
			expected: "song artist",
		},
		{
			name:     "Remove parentheses",
			input:    "Song (Remix)",
			expected: "song remix",
		},
		{
			name:     "Remove brackets",
			input:    "Song [Official Video]",
			expected: "song official video",
		},
		{
			name:     "Multiple spaces",
			input:    "Song   With    Spaces",
			expected: "song with spaces",
		},
		{
			name:     "Complex case",
			input:    "Song feat. Artist (Remix) [2023]",
			expected: "song artist remix 2023",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sc.normalizeString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestSpotifyConverter_stringSimilarity(t *testing.T) {
	sc := &SpotifyConverter{}
	
	tests := []struct {
		name     string
		s1       string
		s2       string
		minScore float64
	}{
		{
			name:     "Identical strings",
			s1:       "hello",
			s2:       "hello",
			minScore: 1.0,
		},
		{
			name:     "Similar strings",
			s1:       "hello world",
			s2:       "hello world!",
			minScore: 0.8,
		},
		{
			name:     "Substring match",
			s1:       "hello",
			s2:       "hello world",
			minScore: 0.4,
		},
		{
			name:     "Different strings",
			s1:       "abc",
			s2:       "xyz",
			minScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := sc.stringSimilarity(tt.s1, tt.s2)
			if score < tt.minScore {
				t.Errorf("Expected score >= %.2f, got %.2f", tt.minScore, score)
			}
		})
	}
}

func TestSpotifyConverter_levenshteinDistance(t *testing.T) {
	sc := &SpotifyConverter{}
	
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected int
	}{
		{
			name:     "Identical strings",
			s1:       "hello",
			s2:       "hello",
			expected: 0,
		},
		{
			name:     "One character difference",
			s1:       "hello",
			s2:       "hallo",
			expected: 1,
		},
		{
			name:     "Empty string",
			s1:       "",
			s2:       "hello",
			expected: 5,
		},
		{
			name:     "Completely different",
			s1:       "abc",
			s2:       "xyz",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := sc.levenshteinDistance(tt.s1, tt.s2)
			if distance != tt.expected {
				t.Errorf("Expected distance %d, got %d", tt.expected, distance)
			}
		})
	}
}

func TestSpotifyConverter_calculateMatchScore(t *testing.T) {
	sc := &SpotifyConverter{}
	
	tests := []struct {
		name             string
		spotifyTitle     string
		spotifyArtist    string
		spotifyAlbum     string
		spotifyDuration  int
		deezerTitle      string
		deezerArtist     string
		deezerAlbum      string
		deezerDuration   int
		minScore         float64
	}{
		{
			name:            "Perfect match",
			spotifyTitle:    "Bohemian Rhapsody",
			spotifyArtist:   "Queen",
			spotifyAlbum:    "A Night at the Opera",
			spotifyDuration: 354000, // milliseconds
			deezerTitle:     "Bohemian Rhapsody",
			deezerArtist:    "Queen",
			deezerAlbum:     "A Night at the Opera",
			deezerDuration:  354, // seconds
			minScore:        0.95,
		},
		{
			name:            "Good match with slight differences",
			spotifyTitle:    "Bohemian Rhapsody",
			spotifyArtist:   "Queen",
			spotifyAlbum:    "A Night at the Opera",
			spotifyDuration: 354000,
			deezerTitle:     "Bohemian Rhapsody (Remastered)",
			deezerArtist:    "Queen",
			deezerAlbum:     "A Night at the Opera (Deluxe)",
			deezerDuration:  355,
			minScore:        0.7,
		},
		{
			name:            "Poor match",
			spotifyTitle:    "Song A",
			spotifyArtist:   "Artist A",
			spotifyAlbum:    "Album A",
			spotifyDuration: 180000,
			deezerTitle:     "Song B",
			deezerArtist:    "Artist B",
			deezerAlbum:     "Album B",
			deezerDuration:  240,
			minScore:        0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := sc.calculateMatchScore(
				tt.spotifyTitle, tt.spotifyArtist, tt.spotifyAlbum, tt.spotifyDuration,
				tt.deezerTitle, tt.deezerArtist, tt.deezerAlbum, tt.deezerDuration,
			)
			
			if score < tt.minScore {
				t.Errorf("Expected score >= %.2f, got %.2f", tt.minScore, score)
			}
		})
	}
}

func TestSpotifyConverter_buildSearchQuery(t *testing.T) {
	sc := &SpotifyConverter{}
	
	tests := []struct {
		name     string
		track    *SpotifyTrack
		expected string
	}{
		{
			name: "Basic track",
			track: &SpotifyTrack{
				Name: "Bohemian Rhapsody",
				Artists: []SpotifyArtist{
					{Name: "Queen"},
				},
			},
			expected: "Queen Bohemian Rhapsody",
		},
		{
			name: "Track with multiple artists",
			track: &SpotifyTrack{
				Name: "Song Title",
				Artists: []SpotifyArtist{
					{Name: "Artist 1"},
					{Name: "Artist 2"},
				},
			},
			expected: "Artist 1 Song Title",
		},
		{
			name: "Track with no artists",
			track: &SpotifyTrack{
				Name:    "Song Title",
				Artists: []SpotifyArtist{},
			},
			expected: "Song Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := sc.buildSearchQuery(tt.track)
			if query != tt.expected {
				t.Errorf("Expected query '%s', got '%s'", tt.expected, query)
			}
		})
	}
}

func TestNewSpotifyConverter(t *testing.T) {
	spotifyClient := NewSpotifyClient("id", "secret", 30)
	deezerClient := NewDeezerClient(30)
	
	converter := NewSpotifyConverter(spotifyClient, deezerClient)
	
	if converter == nil {
		t.Fatal("Expected converter to be created")
	}
	
	if converter.spotifyClient != spotifyClient {
		t.Error("Expected spotifyClient to be set")
	}
	
	if converter.deezerClient != deezerClient {
		t.Error("Expected deezerClient to be set")
	}
}
