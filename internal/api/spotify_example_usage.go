package api

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExampleSpotifyIntegration demonstrates how to use the Spotify integration
func ExampleSpotifyIntegration() {
	// Create clients
	spotifyClient := NewSpotifyClient("your_client_id", "your_client_secret", 30*time.Second)
	deezerClient := NewDeezerClient(30 * time.Second)
	
	ctx := context.Background()
	
	// Authenticate with Spotify
	if err := spotifyClient.Authenticate(ctx); err != nil {
		log.Fatalf("Failed to authenticate with Spotify: %v", err)
	}
	fmt.Println("✓ Authenticated with Spotify")
	
	// Authenticate with Deezer
	if err := deezerClient.Authenticate(ctx, "your_arl_token"); err != nil {
		log.Fatalf("Failed to authenticate with Deezer: %v", err)
	}
	fmt.Println("✓ Authenticated with Deezer")
	
	// Create converter
	converter := NewSpotifyConverter(spotifyClient, deezerClient)
	
	// Convert a Spotify playlist
	playlistURL := "https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M"
	
	fmt.Printf("\nConverting playlist: %s\n", playlistURL)
	result, err := converter.ConvertPlaylist(ctx, playlistURL)
	if err != nil {
		log.Fatalf("Failed to convert playlist: %v", err)
	}
	
	// Display results
	fmt.Printf("\n=== Conversion Results ===\n")
	fmt.Printf("Playlist: %s\n", result.SpotifyPlaylist.Name)
	fmt.Printf("Total tracks: %d\n", result.TotalTracks)
	fmt.Printf("Matched tracks: %d\n", result.MatchedTracks)
	fmt.Printf("Success rate: %.1f%%\n\n", result.SuccessRate*100)
	
	// Display individual track results
	fmt.Println("Track Matches:")
	for i, trackResult := range result.Results {
		spotifyTrack := trackResult.SpotifyTrack
		artistName := ""
		if len(spotifyTrack.Artists) > 0 {
			artistName = spotifyTrack.Artists[0].Name
		}
		
		if trackResult.Matched {
			fmt.Printf("%d. ✓ %s - %s (Confidence: %.0f%%)\n",
				i+1,
				artistName,
				spotifyTrack.Name,
				trackResult.Confidence*100,
			)
			fmt.Printf("   → Deezer: %s - %s\n",
				trackResult.DeezerTrack.Artist.Name,
				trackResult.DeezerTrack.Title,
			)
		} else {
			fmt.Printf("%d. ✗ %s - %s\n",
				i+1,
				artistName,
				spotifyTrack.Name,
			)
			if trackResult.ErrorMessage != "" {
				fmt.Printf("   Error: %s\n", trackResult.ErrorMessage)
			}
		}
	}
}

// ExampleGetSpotifyPlaylist demonstrates how to fetch a Spotify playlist
func ExampleGetSpotifyPlaylist() {
	client := NewSpotifyClient("your_client_id", "your_client_secret", 30*time.Second)
	ctx := context.Background()
	
	// Authenticate
	if err := client.Authenticate(ctx); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}
	
	// Parse playlist URL
	playlistURL := "https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M"
	playlistID, err := ParsePlaylistURL(playlistURL)
	if err != nil {
		log.Fatalf("Failed to parse URL: %v", err)
	}
	
	// Get playlist
	playlist, err := client.GetPlaylist(ctx, playlistID)
	if err != nil {
		log.Fatalf("Failed to get playlist: %v", err)
	}
	
	// Display playlist info
	fmt.Printf("Playlist: %s\n", playlist.Name)
	fmt.Printf("Owner: %s\n", playlist.Owner.DisplayName)
	fmt.Printf("Total tracks: %d\n", playlist.Tracks.Total)
	fmt.Println("\nTracks:")
	
	for i, item := range playlist.Tracks.Items {
		track := item.Track
		artistName := ""
		if len(track.Artists) > 0 {
			artistName = track.Artists[0].Name
		}
		fmt.Printf("%d. %s - %s (%d:%02d)\n",
			i+1,
			artistName,
			track.Name,
			track.Duration/60000,
			(track.Duration/1000)%60,
		)
	}
}

// ExampleSearchSpotifyTrack demonstrates how to search for tracks on Spotify
func ExampleSearchSpotifyTrack() {
	client := NewSpotifyClient("your_client_id", "your_client_secret", 30*time.Second)
	ctx := context.Background()
	
	// Authenticate
	if err := client.Authenticate(ctx); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}
	
	// Search for tracks
	query := "Bohemian Rhapsody Queen"
	tracks, err := client.SearchTrack(ctx, query, 5)
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}
	
	fmt.Printf("Search results for: %s\n\n", query)
	for i, track := range tracks {
		artistName := ""
		if len(track.Artists) > 0 {
			artistName = track.Artists[0].Name
		}
		fmt.Printf("%d. %s - %s\n", i+1, artistName, track.Name)
		fmt.Printf("   Album: %s\n", track.Album.Name)
		fmt.Printf("   Duration: %d:%02d\n\n",
			track.Duration/60000,
			(track.Duration/1000)%60,
		)
	}
}

// ExampleConvertSingleTrack demonstrates how to convert a single Spotify track
func ExampleConvertSingleTrack() {
	spotifyClient := NewSpotifyClient("your_client_id", "your_client_secret", 30*time.Second)
	deezerClient := NewDeezerClient(30 * time.Second)
	ctx := context.Background()
	
	// Authenticate both clients
	if err := spotifyClient.Authenticate(ctx); err != nil {
		log.Fatalf("Spotify auth failed: %v", err)
	}
	if err := deezerClient.Authenticate(ctx, "your_arl_token"); err != nil {
		log.Fatalf("Deezer auth failed: %v", err)
	}
	
	// Create converter
	converter := NewSpotifyConverter(spotifyClient, deezerClient)
	
	// Create a sample Spotify track
	spotifyTrack := &SpotifyTrack{
		Name: "Bohemian Rhapsody",
		Artists: []SpotifyArtist{
			{Name: "Queen"},
		},
		Album: SpotifyAlbum{
			Name: "A Night at the Opera",
		},
		Duration: 354000, // milliseconds
	}
	
	// Convert track
	result, err := converter.ConvertTrack(ctx, spotifyTrack)
	if err != nil {
		log.Fatalf("Conversion failed: %v", err)
	}
	
	// Display result
	if result.Matched {
		fmt.Printf("✓ Match found (Confidence: %.0f%%)\n", result.Confidence*100)
		fmt.Printf("Spotify: %s - %s\n",
			spotifyTrack.Artists[0].Name,
			spotifyTrack.Name,
		)
		fmt.Printf("Deezer:  %s - %s\n",
			result.DeezerTrack.Artist.Name,
			result.DeezerTrack.Title,
		)
		fmt.Printf("Deezer Track ID: %s\n", result.DeezerTrack.ID)
	} else {
		fmt.Printf("✗ No match found\n")
		if result.ErrorMessage != "" {
			fmt.Printf("Error: %s\n", result.ErrorMessage)
		}
	}
}
