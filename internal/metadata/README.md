# Metadata Package

The metadata package provides comprehensive functionality for managing audio file metadata, artwork, and lyrics for both MP3 and FLAC formats.

## Features

- **Metadata Tagging**: Apply ID3v2 tags (MP3) and Vorbis comments (FLAC)
- **Artwork Management**: Download, resize, cache, and embed album artwork
- **Lyrics Support**: Embed synchronized (LRC) and unsynchronized lyrics
- **Multi-disc Support**: Handle disc numbers for multi-disc albums
- **Format Support**: MP3 (ID3v2.4) and FLAC (Vorbis comments)

## Components

### MetadataManager

The main component for applying metadata to audio files.

```go
manager := metadata.NewManager(&metadata.Config{
    EmbedArtwork: true,
    ArtworkSize:  1200,
})

// Apply metadata
err := manager.ApplyMetadata(filePath, &metadata.TrackMetadata{
    Title:       "Song Title",
    Artist:      "Artist Name",
    Album:       "Album Name",
    AlbumArtist: "Album Artist",
    TrackNumber: 1,
    DiscNumber:  1,
    Year:        2024,
    Genre:       "Pop",
    ISRC:        "USRC12345678",
    ArtworkData: imageBytes,
    ArtworkMIME: "image/jpeg",
})
```

### Artwork Management

Download and embed artwork with automatic caching and resizing.

```go
// Download and embed artwork
err := manager.DownloadAndEmbedArtwork(
    filePath,
    "https://example.com/artwork.jpg",
    1200, // target size in pixels
)

// Use artwork cache
cache, _ := metadata.NewArtworkCache("/path/to/cache")
artworkData, mimeType, err := cache.DownloadArtwork(url, 1200)

// Clear old cache
cache.CleanOldCache(30 * 24 * time.Hour) // 30 days
```

### Lyrics Embedding

Embed synchronized and unsynchronized lyrics in audio files.

```go
// Embed lyrics
err := manager.EmbedLyrics(filePath, &metadata.Lyrics{
    SyncedLyrics:   "[00:12.00]First line\n[00:15.50]Second line",
    UnsyncedLyrics: "First line\nSecond line",
}, &metadata.LyricsConfig{
    EmbedInFile:      true,
    SaveSeparateFile: true,
    Language:         "eng",
})

// Read lyrics from file
lyrics, err := manager.GetLyrics(filePath)
```

## Supported Metadata Fields

### Basic Fields
- Title
- Artist
- Album
- Album Artist
- Genre
- Year

### Track Information
- Track Number
- Disc Number (for multi-disc albums)
- Duration
- ISRC (International Standard Recording Code)

### Additional Fields
- Label/Publisher
- Copyright
- Artwork (embedded images)
- Lyrics (synchronized and unsynchronized)

## Format-Specific Details

### MP3 (ID3v2.4)
- Uses ID3v2.4 tags
- Artwork stored in APIC frames
- Synchronized lyrics in SYLT frames
- Unsynchronized lyrics in USLT frames

### FLAC (Vorbis Comments)
- Uses Vorbis comment metadata blocks
- Artwork stored in PICTURE metadata blocks
- Lyrics stored in LYRICS field
- Synchronized lyrics in custom SYNCEDLYRICS field

## Artwork Caching

The artwork cache prevents redundant downloads:

- Cache key based on URL and size
- Automatic resizing to target dimensions
- Configurable cache cleanup
- Thread-safe operations

```go
cache, _ := metadata.NewArtworkCache("/cache/dir")

// Get cache statistics
size, _ := cache.GetCacheSize()
fmt.Printf("Cache size: %d bytes\n", size)

// Clear entire cache
cache.ClearCache()
```

## Lyrics Format Support

### LRC Format (Synchronized)
```
[00:12.00]First line of lyrics
[00:15.50]Second line of lyrics
[00:20.00]Third line of lyrics
```

### Plain Text (Unsynchronized)
```
First line of lyrics
Second line of lyrics
Third line of lyrics
```

## Error Handling

All functions return descriptive errors:

```go
err := manager.ApplyMetadata(filePath, metadata)
if err != nil {
    log.Printf("Failed to apply metadata: %v", err)
}
```

## Best Practices

1. **Always validate file paths** before applying metadata
2. **Use artwork caching** to avoid redundant downloads
3. **Resize artwork** to reasonable dimensions (1200x1200 recommended)
4. **Preserve existing metadata** when only updating specific fields
5. **Handle multi-disc albums** by setting disc numbers
6. **Save separate lyrics files** for better compatibility

## Thread Safety

The metadata manager is safe for concurrent use when operating on different files. For the same file, serialize operations to avoid conflicts.

## Performance Considerations

- Artwork resizing uses Lanczos3 algorithm for quality
- Streaming I/O for large files
- Efficient caching to minimize network requests
- Minimal memory footprint

## Dependencies

- `github.com/bogem/id3v2/v2` - ID3v2 tag support
- `github.com/go-flac/flacvorbis` - FLAC Vorbis comments
- `github.com/go-flac/go-flac` - FLAC file parsing
- `github.com/nfnt/resize` - Image resizing
