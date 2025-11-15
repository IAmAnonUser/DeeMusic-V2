# Deezer API Client

This package provides a complete Go client for interacting with the Deezer API, including both public and private endpoints.

## Features

- **Authentication**: ARL token-based authentication with automatic token refresh
- **Rate Limiting**: Built-in rate limiting using token bucket algorithm (10 requests/second)
- **Caching**: Response caching for frequently accessed data (10-minute TTL)
- **Search**: Search for tracks, albums, artists, and playlists
- **Metadata**: Retrieve detailed information about tracks, albums, artists, and playlists
- **Download URLs**: Generate download URLs with quality selection (MP3_128, MP3_320, FLAC)
- **Lyrics**: Fetch synchronized and plain text lyrics with LRC format support

## Usage

### Creating a Client

```go
import (
    "context"
    "time"
    "internal/api"
)

// Create a new client with 30-second timeout
client := api.NewDeezerClient(30 * time.Second)

// Authenticate with ARL token
ctx := context.Background()
err := client.Authenticate(ctx, "your-arl-token-here")
if err != nil {
    log.Fatal(err)
}
```

### Searching

```go
// Search for tracks
tracks, err := client.SearchTracks(ctx, "Daft Punk", 25)
if err != nil {
    log.Fatal(err)
}

// Search for albums
albums, err := client.SearchAlbums(ctx, "Random Access Memories", 10)

// Search for artists
artists, err := client.SearchArtists(ctx, "Daft Punk", 10)

// Search for playlists
playlists, err := client.SearchPlaylists(ctx, "Electronic", 10)
```

### Getting Metadata

```go
// Get album details with tracks
album, err := client.GetAlbum(ctx, "302127")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Album: %s by %s\n", album.Title, album.Artist.Name)
fmt.Printf("Tracks: %d\n", len(album.Tracks.Data))

// Get artist details
artist, err := client.GetArtist(ctx, "27")

// Get playlist details with tracks
playlist, err := client.GetPlaylist(ctx, "1234567890")

// Get track details
track, err := client.GetTrack(ctx, "3135556")
```

### Getting Download URLs

```go
// Get download URL for a track
downloadURL, err := client.GetTrackDownloadURL(ctx, "3135556", api.QualityMP3320)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Download URL: %s\n", downloadURL.URL)
fmt.Printf("Format: %s\n", downloadURL.Format)
fmt.Printf("Quality: %s\n", downloadURL.Quality)

// Available qualities:
// - api.QualityMP3128  (128 kbps MP3)
// - api.QualityMP3320  (320 kbps MP3)
// - api.QualityFLAC    (Lossless FLAC)
```

### Getting Lyrics

```go
// Get lyrics for a track
lyrics, err := client.GetLyrics(ctx, "3135556")
if err != nil {
    log.Fatal(err)
}

// Check if lyrics are available
if lyrics.HasLyrics() {
    // Get synchronized lyrics in LRC format
    if lyrics.HasSynchronizedLyrics() {
        lrcContent := lyrics.SaveAsLRC()
        fmt.Println("LRC Lyrics:")
        fmt.Println(lrcContent)
    }
    
    // Get plain text lyrics
    plainText := lyrics.GetPlainTextLyrics()
    fmt.Println("Plain Text Lyrics:")
    fmt.Println(plainText)
}
```

### Token Refresh

```go
// Check if authenticated
if !client.IsAuthenticated() {
    // Refresh token
    err := client.RefreshToken(ctx)
    if err != nil {
        log.Fatal(err)
    }
}
```

## Data Models

### Track
- ID, Title, Artist, Album
- Duration, TrackNumber, DiscNumber
- ISRC, PreviewURL, CoverURL
- Availability status

### Album
- ID, Title, Artist
- TrackCount, Duration, ReleaseDate
- Cover images (small, medium, big, XL)
- Track list

### Artist
- ID, Name
- Pictures (small, medium, big, XL)
- Link to artist page

### Playlist
- ID, Title, Description
- TrackCount, Duration
- Creator information
- Track list

### Lyrics
- Synchronized lyrics (LRC format)
- Plain text lyrics
- Writers, Copyright information
- Helper methods for format conversion

## Caching

The client automatically caches responses for:
- Search results (10 minutes)
- Track metadata (10 minutes)
- Album metadata (10 minutes)
- Artist metadata (10 minutes)
- Playlist metadata (10 minutes)
- Lyrics (10 minutes)

Cache is automatically cleaned up every 5 minutes.

## Rate Limiting

The client implements rate limiting using a token bucket algorithm:
- 10 requests per second
- Burst capacity: 10 requests
- Automatic waiting when limit is reached

## Error Handling

The client returns descriptive errors for:
- Authentication failures
- Network errors
- API errors
- Invalid parameters
- Rate limiting
- Unavailable content

Always check for errors and handle them appropriately:

```go
downloadURL, err := client.GetTrackDownloadURL(ctx, trackID, quality)
if err != nil {
    if strings.Contains(err.Error(), "not available") {
        // Track is not available for download
    } else if strings.Contains(err.Error(), "authentication") {
        // Token expired, refresh needed
        client.RefreshToken(ctx)
    } else {
        // Other error
        log.Printf("Error: %v", err)
    }
}
```

## Thread Safety

The DeezerClient is thread-safe and can be used concurrently from multiple goroutines. Internal state is protected with read-write mutexes.

## Requirements

- Go 1.21+
- golang.org/x/time/rate (for rate limiting)

## Notes

- ARL tokens are 192-character hexadecimal strings obtained from Deezer cookies
- Download URLs are temporary and should be used immediately
- Some content may not be available in all regions
- Respect Deezer's Terms of Service when using this client
