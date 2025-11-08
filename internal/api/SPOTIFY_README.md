# Spotify Integration

This package provides Spotify API integration for DeeMusic, enabling conversion of Spotify playlists to Deezer tracks.

## Features

- **Spotify API Client**: OAuth authentication and API access
- **Playlist Fetching**: Retrieve complete Spotify playlists with all tracks
- **Track Search**: Search for tracks on Spotify
- **Playlist Conversion**: Convert Spotify playlists to Deezer tracks with fuzzy matching
- **Match Confidence**: Calculate match confidence scores for track conversions

## Components

### SpotifyClient

Handles authentication and API requests to Spotify.

**Key Methods:**
- `Authenticate(ctx)` - Authenticate using Client Credentials flow
- `GetPlaylist(ctx, playlistID)` - Fetch a complete playlist
- `SearchTrack(ctx, query, limit)` - Search for tracks

**Authentication:**
Uses Spotify's Client Credentials OAuth flow. Requires:
- Client ID
- Client Secret

Tokens are automatically refreshed when expired.

### SpotifyConverter

Converts Spotify playlists to Deezer tracks using intelligent matching.

**Key Methods:**
- `ConvertPlaylist(ctx, playlistURL)` - Convert entire playlist
- `ConvertTrack(ctx, spotifyTrack)` - Convert single track

**Matching Algorithm:**
The converter uses a weighted scoring system:
- **Title similarity**: 40% weight
- **Artist similarity**: 35% weight
- **Album similarity**: 15% weight
- **Duration similarity**: 10% weight

Tracks are matched if confidence score is ≥ 0.5 (50%).

### URL Parsing

Supports multiple Spotify URL formats:
- `https://open.spotify.com/playlist/ID`
- `https://open.spotify.com/playlist/ID?si=...`
- `spotify:playlist:ID`

## Usage

### Basic Playlist Conversion

```go
import (
    "context"
    "time"
    "github.com/yourusername/deemusic/internal/api"
)

// Create clients
spotifyClient := api.NewSpotifyClient("client_id", "client_secret", 30*time.Second)
deezerClient := api.NewDeezerClient(30 * time.Second)

ctx := context.Background()

// Authenticate
spotifyClient.Authenticate(ctx)
deezerClient.Authenticate(ctx, "arl_token")

// Create converter
converter := api.NewSpotifyConverter(spotifyClient, deezerClient)

// Convert playlist
result, err := converter.ConvertPlaylist(ctx, "https://open.spotify.com/playlist/...")
if err != nil {
    // Handle error
}

// Access results
fmt.Printf("Matched %d/%d tracks (%.1f%%)\n",
    result.MatchedTracks,
    result.TotalTracks,
    result.SuccessRate*100,
)

for _, trackResult := range result.Results {
    if trackResult.Matched {
        fmt.Printf("✓ %s - Confidence: %.0f%%\n",
            trackResult.DeezerTrack.Title,
            trackResult.Confidence*100,
        )
    }
}
```

### Get Spotify Playlist

```go
// Parse URL
playlistID, err := api.ParsePlaylistURL(playlistURL)

// Fetch playlist
playlist, err := spotifyClient.GetPlaylist(ctx, playlistID)

fmt.Printf("Playlist: %s\n", playlist.Name)
fmt.Printf("Tracks: %d\n", playlist.Tracks.Total)

for _, item := range playlist.Tracks.Items {
    fmt.Printf("- %s by %s\n",
        item.Track.Name,
        item.Track.Artists[0].Name,
    )
}
```

### Search Tracks

```go
tracks, err := spotifyClient.SearchTrack(ctx, "Bohemian Rhapsody", 10)

for _, track := range tracks {
    fmt.Printf("%s - %s\n",
        track.Artists[0].Name,
        track.Name,
    )
}
```

### Convert Single Track

```go
spotifyTrack := &api.SpotifyTrack{
    Name: "Song Title",
    Artists: []api.SpotifyArtist{
        {Name: "Artist Name"},
    },
    Album: api.SpotifyAlbum{
        Name: "Album Name",
    },
    Duration: 180000, // milliseconds
}

result, err := converter.ConvertTrack(ctx, spotifyTrack)

if result.Matched {
    fmt.Printf("Found on Deezer: %s (ID: %s)\n",
        result.DeezerTrack.Title,
        result.DeezerTrack.ID,
    )
}
```

## Configuration

### Spotify Credentials

To use the Spotify API, you need to:

1. Create a Spotify Developer account at https://developer.spotify.com
2. Create a new application
3. Get your Client ID and Client Secret
4. Add them to your configuration

### Rate Limiting

Both clients implement rate limiting:
- **Spotify**: 10 requests per second
- **Deezer**: 10 requests per second

Rate limits are automatically enforced using token bucket algorithm.

## Data Models

### SpotifyTrack
```go
type SpotifyTrack struct {
    ID       string
    Name     string
    Artists  []SpotifyArtist
    Album    SpotifyAlbum
    Duration int // milliseconds
    ISRC     string
    URI      string
}
```

### SpotifyPlaylist
```go
type SpotifyPlaylist struct {
    ID          string
    Name        string
    Description string
    Owner       SpotifyUser
    Tracks      SpotifyPlaylistTracks
    Images      []SpotifyImage
    URI         string
}
```

### ConversionResult
```go
type ConversionResult struct {
    SpotifyTrack  *SpotifyTrack
    DeezerTrack   *Track
    Matched       bool
    Confidence    float64 // 0.0 to 1.0
    ErrorMessage  string
}
```

### PlaylistConversionResult
```go
type PlaylistConversionResult struct {
    SpotifyPlaylist *SpotifyPlaylist
    Results         []*ConversionResult
    TotalTracks     int
    MatchedTracks   int
    SuccessRate     float64
}
```

## Error Handling

The integration handles various error scenarios:

- **Authentication failures**: Invalid credentials
- **Rate limiting**: Automatic retry with backoff
- **Token expiration**: Automatic token refresh
- **Network errors**: Proper error propagation
- **Invalid URLs**: URL parsing validation
- **No matches found**: Graceful handling with error messages

## Testing

Run tests with:
```bash
go test ./internal/api -v -run TestSpotify
```

Tests cover:
- URL parsing
- Client initialization
- String normalization
- Similarity calculations
- Match scoring
- Levenshtein distance

## Performance Considerations

- **Pagination**: Large playlists are fetched in pages automatically
- **Connection pooling**: HTTP connections are reused
- **Rate limiting**: Prevents API throttling
- **Concurrent requests**: Safe for concurrent use with mutex protection
- **Memory efficiency**: Streaming approach for large playlists

## Matching Algorithm Details

### String Normalization
- Convert to lowercase
- Remove featuring/feat/ft variations
- Remove parentheses and brackets
- Remove extra whitespace
- Normalize hyphens

### Similarity Calculation
Uses Levenshtein distance algorithm to calculate edit distance between strings, then converts to similarity score (0.0 to 1.0).

### Match Scoring
Weighted combination of:
1. **Title match** (40%): How similar are the track titles?
2. **Artist match** (35%): How similar are the artist names?
3. **Album match** (15%): How similar are the album names?
4. **Duration match** (10%): How close are the durations?

A track is considered matched if the total score is ≥ 0.5.

## Future Enhancements

Potential improvements:
- ISRC-based matching for exact matches
- User authentication flow for private playlists
- Batch conversion optimization
- Caching of conversion results
- Alternative matching strategies
- Machine learning-based matching

## Requirements

This implementation satisfies requirement 7.2:
> THE Go Application SHALL support Spotify playlist conversion with automatic track matching

## See Also

- [Deezer API Documentation](./README.md)
- [Example Usage](./spotify_example_usage.go)
- [API Models](./models.go)
