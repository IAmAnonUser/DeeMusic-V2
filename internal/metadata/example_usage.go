package metadata

import (
	"fmt"
	"log"
	"time"
)

// ExampleBasicMetadata demonstrates basic metadata tagging
func ExampleBasicMetadata() {
	// Create metadata manager
	manager := NewManager(&Config{
		EmbedArtwork: true,
		ArtworkSize:  1200,
	})

	// Prepare metadata
	metadata := &TrackMetadata{
		Title:       "Bohemian Rhapsody",
		Artist:      "Queen",
		Album:       "A Night at the Opera",
		AlbumArtist: "Queen",
		TrackNumber: 11,
		DiscNumber:  1,
		Year:        1975,
		Genre:       "Rock",
		ISRC:        "GBUM71029604",
		Label:       "EMI",
		Copyright:   "© 1975 Queen Productions Ltd.",
	}

	// Apply metadata to MP3 file
	err := manager.ApplyMetadata("song.mp3", metadata)
	if err != nil {
		log.Printf("Failed to apply metadata: %v", err)
		return
	}

	fmt.Println("Metadata applied successfully")
}

// ExampleArtworkEmbedding demonstrates artwork download and embedding
func ExampleArtworkEmbedding() {
	manager := NewManager(&Config{
		EmbedArtwork: true,
		ArtworkSize:  1200,
	})

	// Download and embed artwork
	artworkURL := "https://e-cdns-images.dzcdn.net/images/cover/1234567890/1200x1200.jpg"
	err := manager.DownloadAndEmbedArtwork("song.mp3", artworkURL, 1200)
	if err != nil {
		log.Printf("Failed to embed artwork: %v", err)
		return
	}

	fmt.Println("Artwork embedded successfully")
}

// ExampleArtworkCache demonstrates artwork caching
func ExampleArtworkCache() {
	// Create artwork cache
	cache, err := NewArtworkCache("/path/to/cache")
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}

	// Download artwork (will be cached)
	artworkURL := "https://e-cdns-images.dzcdn.net/images/cover/1234567890/1200x1200.jpg"
	artworkData, mimeType, err := cache.DownloadArtwork(artworkURL, 1200)
	if err != nil {
		log.Printf("Failed to download artwork: %v", err)
		return
	}

	fmt.Printf("Downloaded %d bytes of %s\n", len(artworkData), mimeType)

	// Get cache statistics
	size, err := cache.GetCacheSize()
	if err != nil {
		log.Printf("Failed to get cache size: %v", err)
		return
	}
	fmt.Printf("Cache size: %d bytes\n", size)

	// Clean old cache entries (older than 30 days)
	err = cache.CleanOldCache(30 * 24 * time.Hour)
	if err != nil {
		log.Printf("Failed to clean cache: %v", err)
		return
	}

	fmt.Println("Cache cleaned successfully")
}

// ExampleLyricsEmbedding demonstrates lyrics embedding
func ExampleLyricsEmbedding() {
	manager := NewManager(nil)

	// Prepare lyrics
	lyrics := &Lyrics{
		SyncedLyrics: `[00:12.00]Is this the real life?
[00:16.00]Is this just fantasy?
[00:19.50]Caught in a landslide
[00:22.00]No escape from reality`,
		UnsyncedLyrics: `Is this the real life?
Is this just fantasy?
Caught in a landslide
No escape from reality`,
	}

	// Embed lyrics in file and save separate files
	err := manager.EmbedLyrics("song.mp3", lyrics, &LyricsConfig{
		EmbedInFile:      true,
		SaveSeparateFile: true,
		Language:         "eng",
	})
	if err != nil {
		log.Printf("Failed to embed lyrics: %v", err)
		return
	}

	fmt.Println("Lyrics embedded successfully")
}

// ExampleReadMetadata demonstrates reading metadata from a file
func ExampleReadMetadata() {
	manager := NewManager(nil)

	// Read metadata from file
	metadata, err := manager.GetMetadata("song.mp3")
	if err != nil {
		log.Printf("Failed to read metadata: %v", err)
		return
	}

	// Display metadata
	fmt.Printf("Title: %s\n", metadata.Title)
	fmt.Printf("Artist: %s\n", metadata.Artist)
	fmt.Printf("Album: %s\n", metadata.Album)
	fmt.Printf("Track: %d\n", metadata.TrackNumber)
	fmt.Printf("Year: %d\n", metadata.Year)
}

// ExampleReadLyrics demonstrates reading lyrics from a file
func ExampleReadLyrics() {
	manager := NewManager(nil)

	// Read lyrics from file
	lyrics, err := manager.GetLyrics("song.mp3")
	if err != nil {
		log.Printf("Failed to read lyrics: %v", err)
		return
	}

	// Display lyrics
	if lyrics.SyncedLyrics != "" {
		fmt.Println("Synchronized lyrics:")
		fmt.Println(lyrics.SyncedLyrics)
	}

	if lyrics.UnsyncedLyrics != "" {
		fmt.Println("Unsynchronized lyrics:")
		fmt.Println(lyrics.UnsyncedLyrics)
	}
}

// ExampleMultiDiscAlbum demonstrates handling multi-disc albums
func ExampleMultiDiscAlbum() {
	manager := NewManager(&Config{
		EmbedArtwork: true,
		ArtworkSize:  1200,
	})

	// Disc 1, Track 1
	metadata1 := &TrackMetadata{
		Title:       "Track 1",
		Artist:      "Artist Name",
		Album:       "Greatest Hits",
		AlbumArtist: "Artist Name",
		TrackNumber: 1,
		DiscNumber:  1,
		Year:        2024,
	}

	err := manager.ApplyMetadata("disc1_track1.mp3", metadata1)
	if err != nil {
		log.Printf("Failed to apply metadata: %v", err)
		return
	}

	// Disc 2, Track 1
	metadata2 := &TrackMetadata{
		Title:       "Track 1",
		Artist:      "Artist Name",
		Album:       "Greatest Hits",
		AlbumArtist: "Artist Name",
		TrackNumber: 1,
		DiscNumber:  2,
		Year:        2024,
	}

	err = manager.ApplyMetadata("disc2_track1.mp3", metadata2)
	if err != nil {
		log.Printf("Failed to apply metadata: %v", err)
		return
	}

	fmt.Println("Multi-disc album metadata applied successfully")
}

// ExampleCompleteWorkflow demonstrates a complete metadata workflow
func ExampleCompleteWorkflow() {
	// Initialize manager
	manager := NewManager(&Config{
		EmbedArtwork: true,
		ArtworkSize:  1200,
	})

	// Initialize artwork cache
	cache, err := NewArtworkCache("/path/to/cache")
	if err != nil {
		log.Printf("Failed to create cache: %v", err)
		return
	}

	// Download artwork
	artworkURL := "https://e-cdns-images.dzcdn.net/images/cover/1234567890/1200x1200.jpg"
	artworkData, mimeType, err := cache.DownloadArtwork(artworkURL, 1200)
	if err != nil {
		log.Printf("Failed to download artwork: %v", err)
		return
	}

	// Prepare complete metadata
	metadata := &TrackMetadata{
		Title:       "Bohemian Rhapsody",
		Artist:      "Queen",
		Album:       "A Night at the Opera",
		AlbumArtist: "Queen",
		TrackNumber: 11,
		DiscNumber:  1,
		Year:        1975,
		Genre:       "Rock",
		ISRC:        "GBUM71029604",
		Label:       "EMI",
		Copyright:   "© 1975 Queen Productions Ltd.",
		ArtworkData: artworkData,
		ArtworkMIME: mimeType,
	}

	// Apply metadata
	err = manager.ApplyMetadata("song.mp3", metadata)
	if err != nil {
		log.Printf("Failed to apply metadata: %v", err)
		return
	}

	// Prepare and embed lyrics
	lyrics := &Lyrics{
		SyncedLyrics: `[00:12.00]Is this the real life?
[00:16.00]Is this just fantasy?`,
		UnsyncedLyrics: `Is this the real life?
Is this just fantasy?`,
	}

	err = manager.EmbedLyrics("song.mp3", lyrics, &LyricsConfig{
		EmbedInFile:      true,
		SaveSeparateFile: true,
		Language:         "eng",
	})
	if err != nil {
		log.Printf("Failed to embed lyrics: %v", err)
		return
	}

	fmt.Println("Complete workflow finished successfully")
}

// ExampleFLACMetadata demonstrates FLAC metadata handling
func ExampleFLACMetadata() {
	manager := NewManager(&Config{
		EmbedArtwork: true,
		ArtworkSize:  1200,
	})

	// Prepare metadata for FLAC
	metadata := &TrackMetadata{
		Title:       "High Quality Track",
		Artist:      "Artist Name",
		Album:       "Lossless Album",
		AlbumArtist: "Artist Name",
		TrackNumber: 1,
		Year:        2024,
		Genre:       "Classical",
	}

	// Apply metadata to FLAC file
	err := manager.ApplyMetadata("song.flac", metadata)
	if err != nil {
		log.Printf("Failed to apply FLAC metadata: %v", err)
		return
	}

	// Embed lyrics in FLAC
	lyrics := &Lyrics{
		UnsyncedLyrics: "Lyrics text here",
	}

	err = manager.EmbedLyrics("song.flac", lyrics, &LyricsConfig{
		EmbedInFile: true,
		Language:    "eng",
	})
	if err != nil {
		log.Printf("Failed to embed FLAC lyrics: %v", err)
		return
	}

	fmt.Println("FLAC metadata applied successfully")
}

// ExampleRemoveMetadata demonstrates removing metadata
func ExampleRemoveMetadata() {
	manager := NewManager(nil)

	// Remove all metadata from file
	err := manager.RemoveMetadata("song.mp3")
	if err != nil {
		log.Printf("Failed to remove metadata: %v", err)
		return
	}

	// Remove only lyrics
	err = manager.RemoveLyrics("song.mp3")
	if err != nil {
		log.Printf("Failed to remove lyrics: %v", err)
		return
	}

	fmt.Println("Metadata removed successfully")
}
