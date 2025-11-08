package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	spotifyAuthURL = "https://accounts.spotify.com/api/token"
	spotifyAPIURL  = "https://api.spotify.com/v1"
)

// SpotifyClient handles all Spotify API interactions
type SpotifyClient struct {
	httpClient   *http.Client
	clientID     string
	clientSecret string
	accessToken  string
	tokenExpiry  time.Time
	rateLimiter  *rate.Limiter
	mu           sync.RWMutex
}

// SpotifyTrack represents a Spotify track
type SpotifyTrack struct {
	ID       string              `json:"id"`
	Name     string              `json:"name"`
	Artists  []SpotifyArtist     `json:"artists"`
	Album    SpotifyAlbum        `json:"album"`
	Duration int                 `json:"duration_ms"`
	ISRC     string              `json:"external_ids.isrc"`
	URI      string              `json:"uri"`
}

// SpotifyArtist represents a Spotify artist
type SpotifyArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URI  string `json:"uri"`
}

// SpotifyAlbum represents a Spotify album
type SpotifyAlbum struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	ReleaseDate string   `json:"release_date"`
	Images      []SpotifyImage `json:"images"`
	URI         string   `json:"uri"`
}

// SpotifyImage represents an album/playlist image
type SpotifyImage struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

// SpotifyPlaylist represents a Spotify playlist
type SpotifyPlaylist struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Owner       SpotifyUser         `json:"owner"`
	Tracks      SpotifyPlaylistTracks `json:"tracks"`
	Images      []SpotifyImage      `json:"images"`
	URI         string              `json:"uri"`
}

// SpotifyPlaylistTracks represents playlist tracks container
type SpotifyPlaylistTracks struct {
	Total int                    `json:"total"`
	Items []SpotifyPlaylistItem  `json:"items"`
	Next  string                 `json:"next"`
}

// SpotifyPlaylistItem represents a track in a playlist
type SpotifyPlaylistItem struct {
	Track SpotifyTrack `json:"track"`
}

// SpotifyUser represents a Spotify user
type SpotifyUser struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// NewSpotifyClient creates a new Spotify API client
func NewSpotifyClient(clientID, clientSecret string, timeout time.Duration) *SpotifyClient {
	return &SpotifyClient{
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		clientID:     clientID,
		clientSecret: clientSecret,
		rateLimiter:  rate.NewLimiter(rate.Every(100*time.Millisecond), 10), // 10 requests per second
	}
}

// Authenticate authenticates with Spotify using Client Credentials flow
func (c *SpotifyClient) Authenticate(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.clientID == "" || c.clientSecret == "" {
		return fmt.Errorf("client ID and secret are required")
	}

	// Prepare request
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, "POST", spotifyAuthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}

	// Set authorization header
	auth := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	c.accessToken = result.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)

	return nil
}

// ensureAuthenticated checks if token is valid and refreshes if needed
func (c *SpotifyClient) ensureAuthenticated(ctx context.Context) error {
	c.mu.RLock()
	needsRefresh := time.Now().After(c.tokenExpiry.Add(-5 * time.Minute))
	c.mu.RUnlock()

	if needsRefresh {
		return c.Authenticate(ctx)
	}

	return nil
}

// doRequest performs an HTTP request with rate limiting and authentication
func (c *SpotifyClient) doRequest(ctx context.Context, method, endpoint string, params url.Values) (*http.Response, error) {
	// Ensure we're authenticated
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Build URL
	apiURL := spotifyAPIURL + endpoint
	if len(params) > 0 {
		apiURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, apiURL, nil)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	token := c.accessToken
	c.mu.RUnlock()

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Check for authentication errors
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		// Try to refresh token and retry once
		if err := c.Authenticate(ctx); err != nil {
			return nil, fmt.Errorf("token refresh failed: %w", err)
		}
		return c.doRequest(ctx, method, endpoint, params)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// GetPlaylist retrieves a Spotify playlist by ID
func (c *SpotifyClient) GetPlaylist(ctx context.Context, playlistID string) (*SpotifyPlaylist, error) {
	endpoint := fmt.Sprintf("/playlists/%s", playlistID)
	params := url.Values{}
	params.Set("fields", "id,name,description,owner(id,display_name),tracks(total,items(track(id,name,artists,album,duration_ms,uri)),next),images,uri")

	resp, err := c.doRequest(ctx, "GET", endpoint, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var playlist SpotifyPlaylist
	if err := json.NewDecoder(resp.Body).Decode(&playlist); err != nil {
		return nil, fmt.Errorf("failed to decode playlist: %w", err)
	}

	// Fetch all tracks if there are more pages
	for playlist.Tracks.Next != "" {
		moreTracks, err := c.getPlaylistTracksPage(ctx, playlist.Tracks.Next)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch additional tracks: %w", err)
		}
		playlist.Tracks.Items = append(playlist.Tracks.Items, moreTracks.Items...)
		playlist.Tracks.Next = moreTracks.Next
	}

	return &playlist, nil
}

// getPlaylistTracksPage fetches a page of playlist tracks from a URL
func (c *SpotifyClient) getPlaylistTracksPage(ctx context.Context, pageURL string) (*SpotifyPlaylistTracks, error) {
	// Ensure we're authenticated
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	token := c.accessToken
	c.mu.RUnlock()

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch tracks page with status %d: %s", resp.StatusCode, string(body))
	}

	var tracks SpotifyPlaylistTracks
	if err := json.NewDecoder(resp.Body).Decode(&tracks); err != nil {
		return nil, fmt.Errorf("failed to decode tracks page: %w", err)
	}

	return &tracks, nil
}

// SearchTrack searches for a track on Spotify
func (c *SpotifyClient) SearchTrack(ctx context.Context, query string, limit int) ([]*SpotifyTrack, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("type", "track")
	params.Set("limit", fmt.Sprintf("%d", limit))

	resp, err := c.doRequest(ctx, "GET", "/search", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Tracks struct {
			Items []*SpotifyTrack `json:"items"`
		} `json:"tracks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	return result.Tracks.Items, nil
}

// ParsePlaylistURL extracts the playlist ID from a Spotify URL
func ParsePlaylistURL(playlistURL string) (string, error) {
	// Support formats:
	// https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M
	// spotify:playlist:37i9dQZF1DXcBWIGoYBM5M
	
	if strings.HasPrefix(playlistURL, "spotify:playlist:") {
		return strings.TrimPrefix(playlistURL, "spotify:playlist:"), nil
	}

	if strings.Contains(playlistURL, "open.spotify.com/playlist/") {
		parts := strings.Split(playlistURL, "/playlist/")
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid Spotify playlist URL format")
		}
		// Remove query parameters if present
		playlistID := strings.Split(parts[1], "?")[0]
		return playlistID, nil
	}

	return "", fmt.Errorf("unsupported Spotify URL format")
}
