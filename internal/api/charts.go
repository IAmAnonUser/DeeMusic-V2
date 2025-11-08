package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// GetChart retrieves chart data from Deezer
func (c *DeezerClient) GetChart(ctx context.Context, limit int) (*ChartData, error) {
	if limit < 1 || limit > 100 {
		limit = 25
	}

	// Check cache
	cacheKey := fmt.Sprintf("chart_%d", limit)
	if cached, ok := responseCache.get(cacheKey); ok {
		return cached.(*ChartData), nil
	}

	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", limit))

	result, err := c.doPublicAPIRequest(ctx, "/chart", params)
	if err != nil {
		return nil, fmt.Errorf("get chart failed: %w", err)
	}

	// Marshal and unmarshal to convert map to struct
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chart: %w", err)
	}

	var chart ChartData
	if err := json.Unmarshal(data, &chart); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chart: %w", err)
	}

	// Cache the result
	responseCache.set(cacheKey, &chart)

	return &chart, nil
}

// GetEditorialReleases retrieves editorial releases (new releases)
func (c *DeezerClient) GetEditorialReleases(ctx context.Context, limit int) ([]*Album, error) {
	if limit < 1 || limit > 100 {
		limit = 25
	}

	// Check cache
	cacheKey := fmt.Sprintf("editorial_releases_%d", limit)
	if cached, ok := responseCache.get(cacheKey); ok {
		return cached.([]*Album), nil
	}

	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", limit))

	// Use editorial/0/releases endpoint (0 is for all genres)
	result, err := c.doPublicAPIRequest(ctx, "/editorial/0/releases", params)
	if err != nil {
		return nil, fmt.Errorf("get editorial releases failed: %w", err)
	}

	// Marshal and unmarshal to convert map to struct
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal releases: %w", err)
	}

	var response struct {
		Data []*Album `json:"data"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal releases: %w", err)
	}

	// Cache the result
	responseCache.set(cacheKey, response.Data)

	return response.Data, nil
}

// GetEditorialCharts retrieves editorial charts (popular playlists)
func (c *DeezerClient) GetEditorialCharts(ctx context.Context, limit int) ([]*Playlist, error) {
	if limit < 1 || limit > 100 {
		limit = 25
	}

	// Check cache
	cacheKey := fmt.Sprintf("editorial_charts_%d", limit)
	if cached, ok := responseCache.get(cacheKey); ok {
		return cached.([]*Playlist), nil
	}

	params := url.Values{}
	params.Set("limit", fmt.Sprintf("%d", limit))

	// Use editorial/0/charts endpoint
	result, err := c.doPublicAPIRequest(ctx, "/editorial/0/charts", params)
	if err != nil {
		return nil, fmt.Errorf("get editorial charts failed: %w", err)
	}

	// Marshal and unmarshal to convert map to struct
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal charts: %w", err)
	}

	var response struct {
		Data []*Playlist `json:"data"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal charts: %w", err)
	}

	// Cache the result
	responseCache.set(cacheKey, response.Data)

	return response.Data, nil
}

// ChartData represents chart data from Deezer
type ChartData struct {
	Tracks    *TrackList    `json:"tracks"`
	Albums    *AlbumList    `json:"albums"`
	Artists   *ArtistList   `json:"artists"`
	Playlists *PlaylistList `json:"playlists"`
}

// TrackList represents a list of tracks
type TrackList struct {
	Data []*Track `json:"data"`
}

// AlbumList represents a list of albums
type AlbumList struct {
	Data []*Album `json:"data"`
}

// ArtistList represents a list of artists
type ArtistList struct {
	Data []*Artist `json:"data"`
}

// PlaylistList represents a list of playlists
type PlaylistList struct {
	Data []*Playlist `json:"data"`
}
