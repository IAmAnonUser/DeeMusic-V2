package api

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExampleUsage demonstrates how to use the Deezer API client
func ExampleUsage() {
	// Create a new client with 30-second timeout
	client := NewDeezerClient(30 * time.Second)

	// Context for all operations
	ctx := context.Background()

	// Authenticate with ARL token
	// Note: Replace with actual ARL token from Deezer cookies
	arl := "your-192-character-arl-token-here"
	if err := client.Authenticate(ctx, arl); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	fmt.Println("✓ Authenticated successfully")

	// Example 1: Search for tracks
	fmt.Println("\n--- Searching for tracks ---")
	tracks, err := client.SearchTracks(ctx, "Daft Punk Get Lucky", 5)
	if err != nil {
		log.Printf("Search failed: %v", err)
	} else {
		for i, track := range tracks {
			fmt.Printf("%d. %s - %s (%s)\n", i+1, track.Artist.Name, track.Title, track.Album.Title)
		}
	}

	// Example 2: Get album details
	fmt.Println("\n--- Getting album details ---")
	album, err := client.GetAlbum(ctx, "302127") // Random Access Memories
	if err != nil {
		log.Printf("Get album failed: %v", err)
	} else {
		fmt.Printf("Album: %s by %s\n", album.Title, album.Artist.Name)
		fmt.Printf("Tracks: %d\n", album.TrackCount)
		fmt.Printf("Release Date: %s\n", album.ReleaseDate)
		if album.Tracks != nil && len(album.Tracks.Data) > 0 {
			fmt.Println("\nFirst 3 tracks:")
			for i, track := range album.Tracks.Data {
				if i >= 3 {
					break
				}
				fmt.Printf("  %d. %s (Track %d)\n", i+1, track.Title, track.TrackNumber)
			}
		}
	}

	// Example 3: Get artist details
	fmt.Println("\n--- Getting artist details ---")
	artist, err := client.GetArtist(ctx, "27") // Daft Punk
	if err != nil {
		log.Printf("Get artist failed: %v", err)
	} else {
		fmt.Printf("Artist: %s\n", artist.Name)
		fmt.Printf("Link: %s\n", artist.Link)
	}

	// Example 4: Get download URL
	fmt.Println("\n--- Getting download URL ---")
	trackID := "3135556" // Get Lucky
	downloadURL, err := client.GetTrackDownloadURL(ctx, trackID, QualityMP3320)
	if err != nil {
		log.Printf("Get download URL failed: %v", err)
	} else {
		fmt.Printf("Track ID: %s\n", downloadURL.TrackID)
		fmt.Printf("Quality: %s\n", downloadURL.Quality)
		fmt.Printf("Format: %s\n", downloadURL.Format)
		fmt.Printf("URL: %s...\n", downloadURL.URL[:50]) // Show first 50 chars
	}

	// Example 5: Get lyrics
	fmt.Println("\n--- Getting lyrics ---")
	lyrics, err := client.GetLyrics(ctx, trackID)
	if err != nil {
		log.Printf("Get lyrics failed: %v", err)
	} else {
		if lyrics.HasLyrics() {
			fmt.Println("✓ Lyrics found")
			if lyrics.HasSynchronizedLyrics() {
				fmt.Println("✓ Synchronized lyrics available")
				_ = lyrics.SaveAsLRC() // LRC content available
				lines := len(lyrics.Synchronized)
				fmt.Printf("  Lines: %d\n", lines)
				if lines > 0 {
					fmt.Printf("  First line: [%s] %s\n",
						lyrics.Synchronized[0].LrcTimestamp,
						lyrics.Synchronized[0].Line)
				}
			}
			if lyrics.UnsyncedLyrics != "" {
				fmt.Println("✓ Plain text lyrics available")
				plainText := lyrics.GetPlainTextLyrics()
				if len(plainText) > 100 {
					fmt.Printf("  Preview: %s...\n", plainText[:100])
				} else {
					fmt.Printf("  Preview: %s\n", plainText)
				}
			}
		} else {
			fmt.Println("✗ No lyrics available for this track")
		}
	}

	// Example 6: Search albums
	fmt.Println("\n--- Searching for albums ---")
	albums, err := client.SearchAlbums(ctx, "Random Access Memories", 3)
	if err != nil {
		log.Printf("Search albums failed: %v", err)
	} else {
		for i, album := range albums {
			fmt.Printf("%d. %s - %s (%s)\n", i+1, album.Artist.Name, album.Title, album.ReleaseDate)
		}
	}

	// Example 7: Get playlist
	fmt.Println("\n--- Getting playlist details ---")
	// Note: Use a valid playlist ID
	playlist, err := client.GetPlaylist(ctx, "1362516245")
	if err != nil {
		log.Printf("Get playlist failed: %v", err)
	} else {
		fmt.Printf("Playlist: %s\n", playlist.Title)
		fmt.Printf("Description: %s\n", playlist.Description)
		fmt.Printf("Tracks: %d\n", playlist.TrackCount)
		fmt.Printf("Creator: %s\n", playlist.Creator.Name)
	}

	// Example 8: Check authentication status
	fmt.Println("\n--- Authentication status ---")
	if client.IsAuthenticated() {
		fmt.Println("✓ Client is authenticated")
	} else {
		fmt.Println("✗ Client is not authenticated")
	}

	// Example 9: Token refresh (if needed)
	fmt.Println("\n--- Token refresh ---")
	if err := client.RefreshToken(ctx); err != nil {
		log.Printf("Token refresh failed: %v", err)
	} else {
		fmt.Println("✓ Token refreshed successfully")
	}

	fmt.Println("\n--- Example completed ---")
}

// ExampleSearchAndDownload demonstrates a complete search and download workflow
func ExampleSearchAndDownload() {
	client := NewDeezerClient(30 * time.Second)
	ctx := context.Background()

	// Authenticate
	arl := "your-arl-token"
	if err := client.Authenticate(ctx, arl); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	// Search for a track
	query := "Daft Punk Get Lucky"
	tracks, err := client.SearchTracks(ctx, query, 1)
	if err != nil || len(tracks) == 0 {
		log.Fatalf("Search failed: %v", err)
	}

	track := tracks[0]
	fmt.Printf("Found: %s - %s\n", track.Artist.Name, track.Title)

	// Get download URL
	downloadInfo, err := client.GetTrackDownloadURL(ctx, track.ID.String(), QualityMP3320)
	if err != nil {
		log.Fatalf("Failed to get download URL: %v", err)
	}

	fmt.Printf("Download URL obtained: %s\n", downloadInfo.Format)

	// Get lyrics
	lyrics, err := client.GetLyrics(ctx, track.ID.String())
	if err != nil {
		log.Printf("Failed to get lyrics: %v", err)
	} else if lyrics.HasLyrics() {
		fmt.Println("Lyrics available")
	}

	// At this point, you would:
	// 1. Download the file from downloadURL.URL
	// 2. Decrypt it using the decryption package
	// 3. Apply metadata and artwork
	// 4. Embed lyrics if available
}

// ExampleAlbumDownload demonstrates downloading an entire album
func ExampleAlbumDownload() {
	client := NewDeezerClient(30 * time.Second)
	ctx := context.Background()

	// Authenticate
	arl := "your-arl-token"
	if err := client.Authenticate(ctx, arl); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	// Get album details
	albumID := "302127" // Random Access Memories
	album, err := client.GetAlbum(ctx, albumID)
	if err != nil {
		log.Fatalf("Failed to get album: %v", err)
	}

	fmt.Printf("Album: %s by %s\n", album.Title, album.Artist.Name)
	fmt.Printf("Tracks: %d\n", len(album.Tracks.Data))

	// Process each track
	for i, track := range album.Tracks.Data {
		fmt.Printf("\nTrack %d/%d: %s\n", i+1, len(album.Tracks.Data), track.Title)

		// Get download URL
		_, err := client.GetTrackDownloadURL(ctx, track.ID.String(), QualityMP3320)
		if err != nil {
			log.Printf("  Failed to get download URL: %v", err)
			continue
		}

		fmt.Printf("  ✓ Download URL obtained\n")

		// Get lyrics
		lyrics, err := client.GetLyrics(ctx, track.ID.String())
		if err == nil && lyrics.HasLyrics() {
			fmt.Printf("  ✓ Lyrics available\n")
		}

		// Here you would download and process the track
		// For this example, we just print the status
	}
}
