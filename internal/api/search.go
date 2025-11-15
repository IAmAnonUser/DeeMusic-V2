package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// Cache for frequently accessed data
type cache struct {
	data      map[string]cacheEntry
	mu        sync.RWMutex
	ttl       time.Duration
}

type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

func newCache(ttl time.Duration) *cache {
	c := &cache{
		data: make(map[string]cacheEntry),
		ttl:  ttl,
	}
	
	// Start cleanup goroutine
	go c.cleanup()
	
	return c
}

func (c *cache) get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, exists := c.data[key]
	if !exists {
		return nil, false
	}
	
	if time.Now().After(entry.expiration) {
		return nil, false
	}
	
	return entry.value, true
}

func (c *cache) set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.data[key] = cacheEntry{
		value:      value,
		expiration: time.Now().Add(c.ttl),
	}
}

func (c *cache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.data {
			if now.After(entry.expiration) {
				delete(c.data, key)
			}
		}
		c.mu.Unlock()
	}
}

// Initialize cache in DeezerClient
var responseCache = newCache(10 * time.Minute)

// SearchTracks searches for tracks on Deezer
func (c *DeezerClient) SearchTracks(ctx context.Context, query string, limit int) ([]*Track, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	
	if limit <= 0 {
		limit = 25
	}
	
	// Check cache
	cacheKey := fmt.Sprintf("search_tracks_%s_%d", query, limit)
	if cached, ok := responseCache.get(cacheKey); ok {
		return cached.([]*Track), nil
	}
	
	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", strconv.Itoa(limit))
	
	result, err := c.doPublicAPIRequest(ctx, "/search/track", params)
	if err != nil {
		return nil, fmt.Errorf("search tracks failed: %w", err)
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
	
	// Normalize track numbers for all tracks
	for _, track := range tracks {
		if track != nil {
			actualTrackNum := track.GetTrackNumber()
			if actualTrackNum > 0 {
				track.TrackNumber = actualTrackNum
			}
			track.TrackPosition = 0 // Clear to avoid confusion
		}
	}
	
	// Cache result
	responseCache.set(cacheKey, tracks)
	
	return tracks, nil
}

// SearchAlbums searches for albums on Deezer
func (c *DeezerClient) SearchAlbums(ctx context.Context, query string, limit int) ([]*Album, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	
	if limit <= 0 {
		limit = 25
	}
	
	// Check cache
	cacheKey := fmt.Sprintf("search_albums_%s_%d", query, limit)
	if cached, ok := responseCache.get(cacheKey); ok {
		return cached.([]*Album), nil
	}
	
	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", strconv.Itoa(limit))
	
	result, err := c.doPublicAPIRequest(ctx, "/search/album", params)
	if err != nil {
		return nil, fmt.Errorf("search albums failed: %w", err)
	}
	
	// Parse albums
	dataBytes, err := json.Marshal(result["data"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal album data: %w", err)
	}
	
	var albums []*Album
	if err := json.Unmarshal(dataBytes, &albums); err != nil {
		return nil, fmt.Errorf("failed to unmarshal albums: %w", err)
	}
	
	// Cache result
	responseCache.set(cacheKey, albums)
	
	return albums, nil
}

// SearchArtists searches for artists on Deezer
func (c *DeezerClient) SearchArtists(ctx context.Context, query string, limit int) ([]*Artist, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	
	if limit <= 0 {
		limit = 25
	}
	
	// Check cache
	cacheKey := fmt.Sprintf("search_artists_%s_%d", query, limit)
	if cached, ok := responseCache.get(cacheKey); ok {
		return cached.([]*Artist), nil
	}
	
	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", strconv.Itoa(limit))
	
	result, err := c.doPublicAPIRequest(ctx, "/search/artist", params)
	if err != nil {
		return nil, fmt.Errorf("search artists failed: %w", err)
	}
	
	// Parse artists
	dataBytes, err := json.Marshal(result["data"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal artist data: %w", err)
	}
	
	var artists []*Artist
	if err := json.Unmarshal(dataBytes, &artists); err != nil {
		return nil, fmt.Errorf("failed to unmarshal artists: %w", err)
	}
	
	// Cache result
	responseCache.set(cacheKey, artists)
	
	return artists, nil
}

// SearchPlaylists searches for playlists on Deezer
func (c *DeezerClient) SearchPlaylists(ctx context.Context, query string, limit int) ([]*Playlist, error) {
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	
	if limit <= 0 {
		limit = 25
	}
	
	// Check cache
	cacheKey := fmt.Sprintf("search_playlists_%s_%d", query, limit)
	if cached, ok := responseCache.get(cacheKey); ok {
		return cached.([]*Playlist), nil
	}
	
	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", strconv.Itoa(limit))
	
	result, err := c.doPublicAPIRequest(ctx, "/search/playlist", params)
	if err != nil {
		return nil, fmt.Errorf("search playlists failed: %w", err)
	}
	
	// Parse playlists
	dataBytes, err := json.Marshal(result["data"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal playlist data: %w", err)
	}
	
	var playlists []*Playlist
	if err := json.Unmarshal(dataBytes, &playlists); err != nil {
		return nil, fmt.Errorf("failed to unmarshal playlists: %w", err)
	}
	
	// Cache result
	responseCache.set(cacheKey, playlists)
	
	return playlists, nil
}

// GetAlbum retrieves full album details including tracks
func (c *DeezerClient) GetAlbum(ctx context.Context, albumID string) (*Album, error) {
	if albumID == "" {
		return nil, fmt.Errorf("album ID cannot be empty")
	}
	
	// Check cache
	cacheKey := fmt.Sprintf("album_%s", albumID)
	if cached, ok := responseCache.get(cacheKey); ok {
		return cached.(*Album), nil
	}
	
	result, err := c.doPublicAPIRequest(ctx, "/album/"+albumID, nil)
	if err != nil {
		return nil, fmt.Errorf("get album failed: %w", err)
	}
	
	// Parse album
	albumBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal album data: %w", err)
	}
	
	var album Album
	if err := json.Unmarshal(albumBytes, &album); err != nil {
		return nil, fmt.Errorf("failed to unmarshal album: %w", err)
	}
	
	// Fix track positions if they're 0 (use array index + 1)
	// Also normalize to use TrackNumber field for consistency
	if album.Tracks != nil && album.Tracks.Data != nil {
		for i, track := range album.Tracks.Data {
			actualTrackNum := track.GetTrackNumber()
			if actualTrackNum == 0 {
				track.TrackNumber = i + 1
			} else {
				track.TrackNumber = actualTrackNum
			}
			// Clear TrackPosition to avoid confusion
			track.TrackPosition = 0
		}
	}
	
	// Cache result
	responseCache.set(cacheKey, &album)
	
	return &album, nil
}

// GetArtist retrieves full artist details
func (c *DeezerClient) GetArtist(ctx context.Context, artistID string) (*Artist, error) {
	if artistID == "" {
		return nil, fmt.Errorf("artist ID cannot be empty")
	}
	
	// Check cache
	cacheKey := fmt.Sprintf("artist_%s", artistID)
	if cached, ok := responseCache.get(cacheKey); ok {
		return cached.(*Artist), nil
	}
	
	result, err := c.doPublicAPIRequest(ctx, "/artist/"+artistID, nil)
	if err != nil {
		return nil, fmt.Errorf("get artist failed: %w", err)
	}
	
	// Parse artist
	artistBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal artist data: %w", err)
	}
	
	var artist Artist
	if err := json.Unmarshal(artistBytes, &artist); err != nil {
		return nil, fmt.Errorf("failed to unmarshal artist: %w", err)
	}
	
	// Cache result
	responseCache.set(cacheKey, &artist)
	
	return &artist, nil
}

// GetPlaylist retrieves full playlist details including tracks
func (c *DeezerClient) GetPlaylist(ctx context.Context, playlistID string) (*Playlist, error) {
	if playlistID == "" {
		return nil, fmt.Errorf("playlist ID cannot be empty")
	}
	
	// Check cache
	cacheKey := fmt.Sprintf("playlist_%s", playlistID)
	if cached, ok := responseCache.get(cacheKey); ok {
		return cached.(*Playlist), nil
	}
	
	result, err := c.doPublicAPIRequest(ctx, "/playlist/"+playlistID, nil)
	if err != nil {
		return nil, fmt.Errorf("get playlist failed: %w", err)
	}
	
	// Parse playlist
	playlistBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal playlist data: %w", err)
	}
	
	var playlist Playlist
	if err := json.Unmarshal(playlistBytes, &playlist); err != nil {
		return nil, fmt.Errorf("failed to unmarshal playlist: %w", err)
	}
	
	// Cache result
	responseCache.set(cacheKey, &playlist)
	
	return &playlist, nil
}

// GetTrack retrieves full track details
func (c *DeezerClient) GetTrack(ctx context.Context, trackID string) (*Track, error) {
	if trackID == "" {
		return nil, fmt.Errorf("track ID cannot be empty")
	}
	
	// Check cache
	cacheKey := fmt.Sprintf("track_%s", trackID)
	if cached, ok := responseCache.get(cacheKey); ok {
		return cached.(*Track), nil
	}
	
	result, err := c.doPublicAPIRequest(ctx, "/track/"+trackID, nil)
	if err != nil {
		return nil, fmt.Errorf("get track failed: %w", err)
	}
	
	// Parse track
	trackBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal track data: %w", err)
	}
	
	var track Track
	if err := json.Unmarshal(trackBytes, &track); err != nil {
		return nil, fmt.Errorf("failed to unmarshal track: %w", err)
	}
	
	// Normalize track number (prefer track_number over track_position)
	actualTrackNum := track.GetTrackNumber()
	if actualTrackNum > 0 {
		track.TrackNumber = actualTrackNum
	}
	track.TrackPosition = 0 // Clear to avoid confusion
	
	// Cache result
	responseCache.set(cacheKey, &track)
	
	return &track, nil
}
