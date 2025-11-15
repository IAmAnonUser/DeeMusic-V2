package api

import (
	"context"
	"testing"
	"time"
)

func TestParsePlaylistURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedID  string
		expectError bool
	}{
		{
			name:        "Standard HTTPS URL",
			url:         "https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M",
			expectedID:  "37i9dQZF1DXcBWIGoYBM5M",
			expectError: false,
		},
		{
			name:        "HTTPS URL with query params",
			url:         "https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M?si=abc123",
			expectedID:  "37i9dQZF1DXcBWIGoYBM5M",
			expectError: false,
		},
		{
			name:        "Spotify URI format",
			url:         "spotify:playlist:37i9dQZF1DXcBWIGoYBM5M",
			expectedID:  "37i9dQZF1DXcBWIGoYBM5M",
			expectError: false,
		},
		{
			name:        "Invalid URL format",
			url:         "https://example.com/playlist/123",
			expectedID:  "",
			expectError: true,
		},
		{
			name:        "Invalid Spotify URL",
			url:         "https://open.spotify.com/track/123",
			expectedID:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := ParsePlaylistURL(tt.url)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if id != tt.expectedID {
					t.Errorf("Expected ID %s, got %s", tt.expectedID, id)
				}
			}
		})
	}
}

func TestNewSpotifyClient(t *testing.T) {
	client := NewSpotifyClient("test_id", "test_secret", 30*time.Second)
	
	if client == nil {
		t.Fatal("Expected client to be created")
	}
	
	if client.clientID != "test_id" {
		t.Errorf("Expected clientID to be 'test_id', got '%s'", client.clientID)
	}
	
	if client.clientSecret != "test_secret" {
		t.Errorf("Expected clientSecret to be 'test_secret', got '%s'", client.clientSecret)
	}
	
	if client.httpClient == nil {
		t.Error("Expected httpClient to be initialized")
	}
	
	if client.rateLimiter == nil {
		t.Error("Expected rateLimiter to be initialized")
	}
}

func TestSpotifyClient_AuthenticateValidation(t *testing.T) {
	tests := []struct {
		name         string
		clientID     string
		clientSecret string
		expectError  bool
	}{
		{
			name:         "Empty credentials",
			clientID:     "",
			clientSecret: "",
			expectError:  true,
		},
		{
			name:         "Empty client ID",
			clientID:     "",
			clientSecret: "secret",
			expectError:  true,
		},
		{
			name:         "Empty client secret",
			clientID:     "id",
			clientSecret: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewSpotifyClient(tt.clientID, tt.clientSecret, 30*time.Second)
			ctx := context.Background()
			
			err := client.Authenticate(ctx)
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
		})
	}
}
