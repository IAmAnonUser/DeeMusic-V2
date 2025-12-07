package api

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// GetTrackDownloadURL retrieves the download URL for a track with specified quality
// Automatically falls back to lower quality if requested quality is not available
func (c *DeezerClient) GetTrackDownloadURL(ctx context.Context, trackID string, quality string) (*DownloadURL, error) {
	if trackID == "" {
		return nil, fmt.Errorf("track ID cannot be empty")
	}

	// Validate quality
	if quality != QualityMP3128 && quality != QualityMP3320 && quality != QualityFLAC {
		return nil, fmt.Errorf("invalid quality: %s (must be MP3_128, MP3_320, or FLAC)", quality)
	}

	// Get track info first to get MD5 origin
	track, err := c.GetTrack(ctx, trackID)
	if err != nil {
		return nil, fmt.Errorf("failed to get track info: %w", err)
	}

	if !track.Available {
		return nil, fmt.Errorf("track is not available for download")
	}

	// Get track token from private API
	trackToken, err := c.getTrackToken(ctx, trackID)
	if err != nil {
		return nil, fmt.Errorf("failed to get track token: %w", err)
	}

	// Define quality fallback order based on requested quality
	var qualityFallback []string
	switch quality {
	case QualityFLAC:
		qualityFallback = []string{QualityFLAC, QualityMP3320, QualityMP3128}
	case QualityMP3320:
		qualityFallback = []string{QualityMP3320, QualityMP3128}
	case QualityMP3128:
		qualityFallback = []string{QualityMP3128}
	}

	// Try each quality in fallback order
	var lastErr error
	for _, tryQuality := range qualityFallback {
		mediaURL, err := c.getMediaURL(ctx, trackID, trackToken, tryQuality)
		if err == nil {
			// Success! Log if we used fallback quality
			if tryQuality != quality {
				if logFile, logErr := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); logErr == nil {
					fmt.Fprintf(logFile, "[%s] Quality fallback: requested %s, using %s for track %s\n", 
						time.Now().Format("2006-01-02 15:04:05"), quality, tryQuality, trackID)
					logFile.Close()
				}
			}
			
			return &DownloadURL{
				TrackID: trackID,
				Quality: tryQuality, // Return actual quality used
				URL:     mediaURL,
				Format:  getFormatFromQuality(tryQuality),
			}, nil
		}
		lastErr = err
		
		// Log the attempt
		if logFile, logErr := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); logErr == nil {
			fmt.Fprintf(logFile, "[%s] Quality %s not available for track %s, trying next quality...\n", 
				time.Now().Format("2006-01-02 15:04:05"), tryQuality, trackID)
			logFile.Close()
		}
	}

	// All qualities failed
	return nil, fmt.Errorf("failed to get media URL (tried all qualities): %w", lastErr)
}

// getTrackToken retrieves the track token needed for download URL generation
func (c *DeezerClient) getTrackToken(ctx context.Context, trackID string) (string, error) {
	// Use doPrivateAPIRequest which handles authentication properly
	result, err := c.doPrivateAPIRequest(ctx, "deezer.pageTrack", map[string]interface{}{
		"sng_id": trackID,
	})
	
	if err != nil {
		return "", fmt.Errorf("pageTrack request failed: %w", err)
	}

	// Log the response to debug file
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-api-response.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		responseJSON, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintf(logFile, "[%s] deezer.pageTrack response for track %s:\n%s\n\n", time.Now().Format("2006-01-02 15:04:05"), trackID, string(responseJSON))
		logFile.Close()
	}

	// Extract track token from results
	results, ok := result["results"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format: missing results")
	}

	// Get DATA object which contains track information
	data, ok := results["DATA"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format: missing DATA in results")
	}

	trackToken, ok := data["TRACK_TOKEN"].(string)
	if !ok || trackToken == "" {
		// Log what we actually got
		if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-api-response.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			dataJSON, _ := json.MarshalIndent(data, "", "  ")
			fmt.Fprintf(logFile, "[%s] DATA object (no TRACK_TOKEN found):\n%s\n\n", time.Now().Format("2006-01-02 15:04:05"), string(dataJSON))
			logFile.Close()
		}
		return "", fmt.Errorf("track token not found in response")
	}

	return trackToken, nil
}

// getMediaURL retrieves the actual media URL for downloading
func (c *DeezerClient) getMediaURL(ctx context.Context, trackID, trackToken, quality string) (string, error) {
	c.mu.RLock()
	licenseToken := c.licenseToken
	arl := c.arl
	c.mu.RUnlock()

	if licenseToken == "" {
		return "", fmt.Errorf("license token not available")
	}

	// Map quality to format code
	formatCode := getFormatCode(quality)

	// Build payload for media API
	payload := map[string]interface{}{
		"license_token": licenseToken,
		"media": []map[string]interface{}{
			{
				"type":   "FULL",
				"formats": []map[string]interface{}{
					{
						"cipher": "BF_CBC_STRIPE",
						"format": formatCode,
					},
				},
			},
		},
		"track_tokens": []string{trackToken},
	}

	// Use the media.deezer.com endpoint directly (like Python V1)
	mediaURL := "https://media.deezer.com/v1/get_url"
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", mediaURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Cookie", "arl="+arl)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return "", fmt.Errorf("media API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("media API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode media response: %w", err)
	}

	// Check for errors in response
	if errData, ok := result["error"].([]interface{}); ok && len(errData) > 0 {
		return "", fmt.Errorf("media API error: %v", errData)
	}

	// Extract URL from response - new structure
	data, ok := result["data"].([]interface{})
	if !ok || len(data) == 0 {
		return "", fmt.Errorf("no data in media response")
	}

	trackData, ok := data[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid track data format")
	}

	// Check for errors in track data
	if errors, ok := trackData["errors"].([]interface{}); ok && len(errors) > 0 {
		errorInfo := errors[0].(map[string]interface{})
		errorCode := errorInfo["code"]
		errorMsg := errorInfo["message"]
		return "", fmt.Errorf("track error %v: %v", errorCode, errorMsg)
	}

	media, ok := trackData["media"].([]interface{})
	if !ok || len(media) == 0 {
		return "", fmt.Errorf("no media information available")
	}

	mediaInfo, ok := media[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid media info format")
	}

	sources, ok := mediaInfo["sources"].([]interface{})
	if !ok || len(sources) == 0 {
		return "", fmt.Errorf("no media sources available")
	}

	source, ok := sources[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid source format")
	}

	downloadURL, ok := source["url"].(string)
	if !ok || downloadURL == "" {
		return "", fmt.Errorf("media URL not found")
	}

	return downloadURL, nil
}

// getLegacyDownloadURL generates download URL using legacy method (fallback)
func (c *DeezerClient) getLegacyDownloadURL(trackID, md5Origin string, quality string) (string, error) {
	// This is a fallback method using the legacy URL generation
	// Format: https://e-cdns-proxy-{server}.dzcdn.net/mobile/1/{hash}
	
	formatCode := getFormatCode(quality)
	
	// Generate hash
	hash := generateURLHash(trackID, md5Origin, formatCode)
	
	// Select server (simple round-robin based on track ID)
	trackNum, _ := strconv.Atoi(trackID)
	server := trackNum % 3
	
	url := fmt.Sprintf("https://e-cdns-proxy-%d.dzcdn.net/mobile/1/%s", server, hash)
	
	return url, nil
}

// generateURLHash generates the hash for legacy download URL
func generateURLHash(trackID, md5Origin, formatCode string) string {
	// Hash format: MD5(md5Origin + "¤" + formatCode + "¤" + trackID + "¤" + md5Origin)
	data := fmt.Sprintf("%s¤%s¤%s¤%s", md5Origin, formatCode, trackID, md5Origin)
	
	hash := md5.Sum([]byte(data))
	hashStr := hex.EncodeToString(hash[:])
	
	// Build final hash string
	parts := []string{
		hashStr,
		"¤",
		data,
		"¤",
	}
	
	finalData := strings.Join(parts, "")
	finalHash := md5.Sum([]byte(finalData))
	
	return hex.EncodeToString(finalHash[:])
}

// getFormatCode returns the format code for a given quality
func getFormatCode(quality string) string {
	switch quality {
	case QualityMP3128:
		return "MP3_128"
	case QualityMP3320:
		return "MP3_320"
	case QualityFLAC:
		return "FLAC"
	default:
		return "MP3_320"
	}
}

// getFormatFromQuality returns the file format for a given quality
func getFormatFromQuality(quality string) string {
	switch quality {
	case QualityFLAC:
		return "flac"
	default:
		return "mp3"
	}
}

// GetArtistTopTracks retrieves an artist's top tracks
func (c *DeezerClient) GetArtistTopTracks(ctx context.Context, artistID string, limit int) ([]*Track, error) {
	if artistID == "" {
		return nil, fmt.Errorf("artist ID cannot be empty")
	}

	if limit <= 0 {
		limit = 25
	}

	// Check cache
	cacheKey := fmt.Sprintf("artist_top_%s_%d", artistID, limit)
	if cached, ok := responseCache.get(cacheKey); ok {
		return cached.([]*Track), nil
	}

	result, err := c.doPublicAPIRequest(ctx, fmt.Sprintf("/artist/%s/top", artistID), nil)
	if err != nil {
		return nil, fmt.Errorf("get artist top tracks failed: %w", err)
	}

	// Parse tracks
	dataBytes, err := json.Marshal(result["data"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal track data: %w", err)
	}

	var tracks []*Track
	if err := json.Unmarshal(dataBytes, &tracks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tracks: %w", err)
	}

	// Limit results
	if len(tracks) > limit {
		tracks = tracks[:limit]
	}

	// Cache result
	responseCache.set(cacheKey, tracks)

	return tracks, nil
}

// GetArtistAlbums retrieves ALL of an artist's releases (albums, singles, EPs)
// Handles pagination to fetch all results
func (c *DeezerClient) GetArtistAlbums(ctx context.Context, artistID string, limit int) ([]*Album, error) {
	if artistID == "" {
		return nil, fmt.Errorf("artist ID cannot be empty")
	}

	if limit <= 0 {
		limit = 500 // Default to high limit to get all releases
	}

	// Check cache
	cacheKey := fmt.Sprintf("artist_albums_%s_%d", artistID, limit)
	if cached, ok := responseCache.get(cacheKey); ok {
		return cached.([]*Album), nil
	}

	var allAlbums []*Album
	index := 0
	batchSize := 100 // Deezer API max per request
	
	// Fetch albums in batches until we have enough or no more results
	for len(allAlbums) < limit {
		params := url.Values{}
		params.Set("limit", strconv.Itoa(batchSize))
		params.Set("index", strconv.Itoa(index))
		
		result, err := c.doPublicAPIRequest(ctx, fmt.Sprintf("/artist/%s/albums", artistID), params)
		if err != nil {
			return nil, fmt.Errorf("get artist albums failed: %w", err)
		}

		// Parse albums from this batch
		dataBytes, err := json.Marshal(result["data"])
		if err != nil {
			return nil, fmt.Errorf("failed to marshal album data: %w", err)
		}

		var batchAlbums []*Album
		if err := json.Unmarshal(dataBytes, &batchAlbums); err != nil {
			return nil, fmt.Errorf("failed to unmarshal albums: %w", err)
		}

		// If no albums returned, we've reached the end
		if len(batchAlbums) == 0 {
			break
		}

		allAlbums = append(allAlbums, batchAlbums...)
		
		// Check if there are more results
		if result["next"] == nil || result["next"] == "" {
			break
		}
		
		// Move to next batch
		index += batchSize
	}

	// Limit to requested amount
	if len(allAlbums) > limit {
		allAlbums = allAlbums[:limit]
	}

	// Cache result
	responseCache.set(cacheKey, allAlbums)

	return allAlbums, nil
}
