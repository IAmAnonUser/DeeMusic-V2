package api

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// GetLyrics retrieves lyrics for a track (both synchronized and plain text)
func (c *DeezerClient) GetLyrics(ctx context.Context, trackID string) (*Lyrics, error) {
	if trackID == "" {
		return nil, fmt.Errorf("track ID cannot be empty")
	}

	// Check cache
	cacheKey := fmt.Sprintf("lyrics_%s", trackID)
	if cached, ok := responseCache.get(cacheKey); ok {
		return cached.(*Lyrics), nil
	}

	// Try to get lyrics from private API
	params := map[string]interface{}{
		"sng_id": trackID,
	}

	result, err := c.doPrivateAPIRequest(ctx, "song.getLyrics", params)
	if err != nil {
		// Lyrics might not be available, return empty lyrics instead of error
		return &Lyrics{
			ID:      trackID,
			TrackID: trackID,
		}, nil
	}

	// Extract lyrics from results
	results, ok := result["results"].(map[string]interface{})
	if !ok {
		return &Lyrics{
			ID:      trackID,
			TrackID: trackID,
		}, nil
	}

	lyrics := &Lyrics{
		ID:      trackID,
		TrackID: trackID,
	}

	// Extract synchronized lyrics
	if syncLyrics, ok := results["LYRICS_SYNC_JSON"].([]interface{}); ok && len(syncLyrics) > 0 {
		lyrics.Synchronized = parseSynchronizedLyrics(syncLyrics)
		lyrics.SyncedLyrics = formatLyricsAsLRC(lyrics.Synchronized)
	}

	// Extract plain text lyrics
	if textLyrics, ok := results["LYRICS_TEXT"].(string); ok {
		lyrics.UnsyncedLyrics = textLyrics
	}

	// Extract metadata
	if writers, ok := results["LYRICS_WRITERS"].(string); ok {
		lyrics.Writers = writers
	}

	if copyright, ok := results["LYRICS_COPYRIGHTS"].(string); ok {
		lyrics.Copyright = copyright
	}

	// If no lyrics found at all, try alternative method
	if lyrics.SyncedLyrics == "" && lyrics.UnsyncedLyrics == "" {
		// Try getting from track data
		trackLyrics, err := c.getLyricsFromTrackData(ctx, trackID)
		if err == nil && trackLyrics != nil {
			lyrics = trackLyrics
		}
	}

	// Cache result (even if empty, to avoid repeated failed requests)
	responseCache.set(cacheKey, lyrics)

	return lyrics, nil
}

// getLyricsFromTrackData attempts to get lyrics from track data
func (c *DeezerClient) getLyricsFromTrackData(ctx context.Context, trackID string) (*Lyrics, error) {
	params := map[string]interface{}{
		"sng_id": trackID,
	}

	result, err := c.doPrivateAPIRequest(ctx, "song.getData", params)
	if err != nil {
		return nil, err
	}

	results, ok := result["results"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	lyrics := &Lyrics{
		ID:      trackID,
		TrackID: trackID,
	}

	// Check if lyrics are available
	if lyricsID, ok := results["LYRICS_ID"].(float64); ok && lyricsID > 0 {
		// Lyrics exist, try to fetch them
		lyricsIDStr := strconv.FormatFloat(lyricsID, 'f', 0, 64)
		return c.getLyricsByID(ctx, lyricsIDStr)
	}

	return lyrics, nil
}

// getLyricsByID retrieves lyrics by lyrics ID
func (c *DeezerClient) getLyricsByID(ctx context.Context, lyricsID string) (*Lyrics, error) {
	params := map[string]interface{}{
		"lyrics_id": lyricsID,
	}

	result, err := c.doPrivateAPIRequest(ctx, "lyrics.getData", params)
	if err != nil {
		return nil, err
	}

	results, ok := result["results"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	lyrics := &Lyrics{
		ID: lyricsID,
	}

	// Extract synchronized lyrics
	if syncLyrics, ok := results["LYRICS_SYNC_JSON"].([]interface{}); ok && len(syncLyrics) > 0 {
		lyrics.Synchronized = parseSynchronizedLyrics(syncLyrics)
		lyrics.SyncedLyrics = formatLyricsAsLRC(lyrics.Synchronized)
	}

	// Extract plain text lyrics
	if textLyrics, ok := results["LYRICS_TEXT"].(string); ok {
		lyrics.UnsyncedLyrics = textLyrics
	}

	return lyrics, nil
}

// parseSynchronizedLyrics parses synchronized lyrics from API response
func parseSynchronizedLyrics(data []interface{}) []*LyricLine {
	var lines []*LyricLine

	for _, item := range data {
		lineData, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		line := &LyricLine{}

		// Extract line text
		if text, ok := lineData["line"].(string); ok {
			line.Line = text
		}

		// Extract milliseconds
		if ms, ok := lineData["milliseconds"].(string); ok {
			if msInt, err := strconv.Atoi(ms); err == nil {
				line.Milliseconds = msInt
			}
		} else if ms, ok := lineData["milliseconds"].(float64); ok {
			line.Milliseconds = int(ms)
		}

		// Extract duration
		if duration, ok := lineData["duration"].(string); ok {
			if durInt, err := strconv.Atoi(duration); err == nil {
				line.Duration = durInt
			}
		} else if duration, ok := lineData["duration"].(float64); ok {
			line.Duration = int(duration)
		}

		// Generate LRC timestamp
		line.LrcTimestamp = millisecondsToLRCTimestamp(line.Milliseconds)

		lines = append(lines, line)
	}

	return lines
}

// formatLyricsAsLRC formats synchronized lyrics as LRC format
func formatLyricsAsLRC(lines []*LyricLine) string {
	if len(lines) == 0 {
		return ""
	}

	var lrcBuilder strings.Builder

	for _, line := range lines {
		if line.LrcTimestamp != "" {
			lrcBuilder.WriteString(fmt.Sprintf("[%s]%s\n", line.LrcTimestamp, line.Line))
		}
	}

	return lrcBuilder.String()
}

// millisecondsToLRCTimestamp converts milliseconds to LRC timestamp format [mm:ss.xx]
func millisecondsToLRCTimestamp(ms int) string {
	if ms < 0 {
		ms = 0
	}

	totalSeconds := ms / 1000
	milliseconds := (ms % 1000) / 10

	minutes := totalSeconds / 60
	seconds := totalSeconds % 60

	return fmt.Sprintf("%02d:%02d.%02d", minutes, seconds, milliseconds)
}

// ParseLRCLyrics parses LRC format lyrics into structured format
func ParseLRCLyrics(lrcContent string) []*LyricLine {
	var lines []*LyricLine

	for _, line := range strings.Split(lrcContent, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse LRC line format: [mm:ss.xx]text
		if !strings.HasPrefix(line, "[") {
			continue
		}

		closeBracket := strings.Index(line, "]")
		if closeBracket == -1 {
			continue
		}

		timestamp := line[1:closeBracket]
		text := strings.TrimSpace(line[closeBracket+1:])

		// Parse timestamp
		ms := lrcTimestampToMilliseconds(timestamp)
		if ms < 0 {
			continue
		}

		lines = append(lines, &LyricLine{
			Line:         text,
			Milliseconds: ms,
			LrcTimestamp: timestamp,
		})
	}

	return lines
}

// lrcTimestampToMilliseconds converts LRC timestamp to milliseconds
func lrcTimestampToMilliseconds(timestamp string) int {
	// Format: mm:ss.xx or mm:ss.xxx
	parts := strings.Split(timestamp, ":")
	if len(parts) != 2 {
		return -1
	}

	minutes, err := strconv.Atoi(parts[0])
	if err != nil {
		return -1
	}

	secondsParts := strings.Split(parts[1], ".")
	if len(secondsParts) != 2 {
		return -1
	}

	seconds, err := strconv.Atoi(secondsParts[0])
	if err != nil {
		return -1
	}

	// Handle both .xx and .xxx formats
	centiseconds := secondsParts[1]
	if len(centiseconds) == 2 {
		centiseconds += "0"
	}
	ms, err := strconv.Atoi(centiseconds)
	if err != nil {
		return -1
	}

	totalMs := (minutes*60+seconds)*1000 + ms
	return totalMs
}

// HasLyrics checks if lyrics are available for a track
func (l *Lyrics) HasLyrics() bool {
	return l.SyncedLyrics != "" || l.UnsyncedLyrics != ""
}

// HasSynchronizedLyrics checks if synchronized lyrics are available
func (l *Lyrics) HasSynchronizedLyrics() bool {
	return len(l.Synchronized) > 0 || l.SyncedLyrics != ""
}

// GetPlainTextLyrics returns plain text lyrics (prefers unsynced, falls back to synced)
func (l *Lyrics) GetPlainTextLyrics() string {
	if l.UnsyncedLyrics != "" {
		return l.UnsyncedLyrics
	}

	// Convert synchronized lyrics to plain text
	if len(l.Synchronized) > 0 {
		var builder strings.Builder
		for _, line := range l.Synchronized {
			builder.WriteString(line.Line)
			builder.WriteString("\n")
		}
		return builder.String()
	}

	return ""
}

// SaveAsLRC saves lyrics in LRC format
func (l *Lyrics) SaveAsLRC() string {
	if l.SyncedLyrics != "" {
		return l.SyncedLyrics
	}

	if len(l.Synchronized) > 0 {
		return formatLyricsAsLRC(l.Synchronized)
	}

	return ""
}

// SaveAsText saves lyrics as plain text
func (l *Lyrics) SaveAsText() string {
	return l.GetPlainTextLyrics()
}
