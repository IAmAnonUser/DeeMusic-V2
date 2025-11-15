package api

import (
	"context"
	"fmt"
	"strings"
)

// ConversionResult represents the result of converting a Spotify track to Deezer
type ConversionResult struct {
	SpotifyTrack  *SpotifyTrack `json:"spotify_track"`
	DeezerTrack   *Track        `json:"deezer_track,omitempty"`
	Matched       bool          `json:"matched"`
	Confidence    float64       `json:"confidence"` // 0.0 to 1.0
	ErrorMessage  string        `json:"error_message,omitempty"`
}

// PlaylistConversionResult represents the result of converting an entire playlist
type PlaylistConversionResult struct {
	SpotifyPlaylist *SpotifyPlaylist    `json:"spotify_playlist"`
	Results         []*ConversionResult `json:"results"`
	TotalTracks     int                 `json:"total_tracks"`
	MatchedTracks   int                 `json:"matched_tracks"`
	SuccessRate     float64             `json:"success_rate"`
}

// SpotifyConverter handles conversion of Spotify playlists to Deezer
type SpotifyConverter struct {
	spotifyClient *SpotifyClient
	deezerClient  *DeezerClient
}

// NewSpotifyConverter creates a new Spotify to Deezer converter
func NewSpotifyConverter(spotifyClient *SpotifyClient, deezerClient *DeezerClient) *SpotifyConverter {
	return &SpotifyConverter{
		spotifyClient: spotifyClient,
		deezerClient:  deezerClient,
	}
}

// ConvertPlaylist converts a Spotify playlist to Deezer tracks
func (sc *SpotifyConverter) ConvertPlaylist(ctx context.Context, playlistURL string) (*PlaylistConversionResult, error) {
	// Parse playlist URL
	playlistID, err := ParsePlaylistURL(playlistURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse playlist URL: %w", err)
	}

	// Fetch Spotify playlist
	playlist, err := sc.spotifyClient.GetPlaylist(ctx, playlistID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Spotify playlist: %w", err)
	}

	// Convert each track
	results := make([]*ConversionResult, 0, len(playlist.Tracks.Items))
	matchedCount := 0

	for _, item := range playlist.Tracks.Items {
		if item.Track.ID == "" {
			// Skip unavailable tracks
			results = append(results, &ConversionResult{
				SpotifyTrack: &item.Track,
				Matched:      false,
				Confidence:   0.0,
				ErrorMessage: "Track unavailable on Spotify",
			})
			continue
		}

		result, err := sc.ConvertTrack(ctx, &item.Track)
		if err != nil {
			result = &ConversionResult{
				SpotifyTrack: &item.Track,
				Matched:      false,
				Confidence:   0.0,
				ErrorMessage: err.Error(),
			}
		}

		if result.Matched {
			matchedCount++
		}

		results = append(results, result)
	}

	successRate := 0.0
	if len(results) > 0 {
		successRate = float64(matchedCount) / float64(len(results))
	}

	return &PlaylistConversionResult{
		SpotifyPlaylist: playlist,
		Results:         results,
		TotalTracks:     len(results),
		MatchedTracks:   matchedCount,
		SuccessRate:     successRate,
	}, nil
}

// ConvertTrack converts a single Spotify track to a Deezer track
func (sc *SpotifyConverter) ConvertTrack(ctx context.Context, spotifyTrack *SpotifyTrack) (*ConversionResult, error) {
	// Build search query
	query := sc.buildSearchQuery(spotifyTrack)

	// Search on Deezer
	searchResults, err := sc.deezerClient.SearchTracks(ctx, query, 10)
	if err != nil {
		return nil, fmt.Errorf("Deezer search failed: %w", err)
	}

	if len(searchResults) == 0 {
		return &ConversionResult{
			SpotifyTrack: spotifyTrack,
			Matched:      false,
			Confidence:   0.0,
			ErrorMessage: "No matches found on Deezer",
		}, nil
	}

	// Find best match using fuzzy matching
	bestMatch, confidence := sc.findBestMatch(spotifyTrack, searchResults)

	if bestMatch == nil || confidence < 0.5 {
		return &ConversionResult{
			SpotifyTrack: spotifyTrack,
			Matched:      false,
			Confidence:   confidence,
			ErrorMessage: "No confident match found",
		}, nil
	}

	return &ConversionResult{
		SpotifyTrack: spotifyTrack,
		DeezerTrack:  bestMatch,
		Matched:      true,
		Confidence:   confidence,
	}, nil
}

// buildSearchQuery builds a search query from Spotify track info
func (sc *SpotifyConverter) buildSearchQuery(track *SpotifyTrack) string {
	// Primary artist
	artist := ""
	if len(track.Artists) > 0 {
		artist = track.Artists[0].Name
	}

	// Build query: "artist track"
	query := fmt.Sprintf("%s %s", artist, track.Name)
	
	// Clean up query
	query = strings.TrimSpace(query)
	
	return query
}

// findBestMatch finds the best matching Deezer track using fuzzy matching
func (sc *SpotifyConverter) findBestMatch(spotifyTrack *SpotifyTrack, deezerTracks []*Track) (*Track, float64) {
	var bestMatch *Track
	var bestScore float64 = 0.0

	spotifyArtist := ""
	if len(spotifyTrack.Artists) > 0 {
		spotifyArtist = spotifyTrack.Artists[0].Name
	}

	for _, deezerTrack := range deezerTracks {
		score := sc.calculateMatchScore(
			spotifyTrack.Name,
			spotifyArtist,
			spotifyTrack.Album.Name,
			spotifyTrack.Duration,
			deezerTrack.Title,
			deezerTrack.Artist.Name,
			deezerTrack.Album.Title,
			deezerTrack.Duration,
		)

		if score > bestScore {
			bestScore = score
			bestMatch = deezerTrack
		}
	}

	return bestMatch, bestScore
}

// calculateMatchScore calculates a match score between Spotify and Deezer tracks
func (sc *SpotifyConverter) calculateMatchScore(
	spotifyTitle, spotifyArtist, spotifyAlbum string, spotifyDuration int,
	deezerTitle, deezerArtist, deezerAlbum string, deezerDuration int,
) float64 {
	var score float64 = 0.0

	// Title similarity (40% weight)
	titleScore := sc.stringSimilarity(
		sc.normalizeString(spotifyTitle),
		sc.normalizeString(deezerTitle),
	)
	score += titleScore * 0.4

	// Artist similarity (35% weight)
	artistScore := sc.stringSimilarity(
		sc.normalizeString(spotifyArtist),
		sc.normalizeString(deezerArtist),
	)
	score += artistScore * 0.35

	// Album similarity (15% weight)
	albumScore := sc.stringSimilarity(
		sc.normalizeString(spotifyAlbum),
		sc.normalizeString(deezerAlbum),
	)
	score += albumScore * 0.15

	// Duration similarity (10% weight)
	// Spotify duration is in milliseconds, Deezer in seconds
	spotifyDurationSec := spotifyDuration / 1000
	durationDiff := abs(spotifyDurationSec - deezerDuration)
	durationScore := 1.0
	if durationDiff > 5 {
		durationScore = 1.0 - (float64(durationDiff) / float64(spotifyDurationSec))
		if durationScore < 0 {
			durationScore = 0
		}
	}
	score += durationScore * 0.1

	return score
}

// stringSimilarity calculates similarity between two strings using Levenshtein-like approach
func (sc *SpotifyConverter) stringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// Simple similarity: check for substring match
	if strings.Contains(s1, s2) || strings.Contains(s2, s1) {
		shorter := len(s1)
		if len(s2) < shorter {
			shorter = len(s2)
		}
		longer := len(s1)
		if len(s2) > longer {
			longer = len(s2)
		}
		return float64(shorter) / float64(longer)
	}

	// Calculate Levenshtein distance
	distance := sc.levenshteinDistance(s1, s2)
	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}

	similarity := 1.0 - (float64(distance) / float64(maxLen))
	if similarity < 0 {
		similarity = 0
	}

	return similarity
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func (sc *SpotifyConverter) levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// normalizeString normalizes a string for comparison
func (sc *SpotifyConverter) normalizeString(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	
	// Remove common variations
	replacements := map[string]string{
		" feat. ":  " ",
		" ft. ":    " ",
		" featuring ": " ",
		"(":        "",
		")":        "",
		"[":        "",
		"]":        "",
		" - ":      " ",
	}
	
	for old, new := range replacements {
		s = strings.ReplaceAll(s, old, new)
	}
	
	// Remove extra spaces
	s = strings.Join(strings.Fields(s), " ")
	
	return s
}

// Helper functions
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
