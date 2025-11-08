# Metadata Package Implementation

## Overview

The metadata package has been successfully implemented with comprehensive support for audio file metadata, artwork, and lyrics management for both MP3 and FLAC formats.

## Completed Features

### 1. Metadata Tagging (Task 6.1)

**Files Created:**
- `metadata.go` - Core metadata management

**Functionality:**
- ✅ MetadataManager for ID3 tag operations
- ✅ ApplyMetadata for MP3 (ID3v2.4) and FLAC (Vorbis comments)
- ✅ Support for all metadata fields:
  - Title, Artist, Album, Album Artist
  - Track Number, Disc Number (multi-disc support)
  - Year, Genre
  - ISRC, Label, Copyright
- ✅ GetMetadata to read existing metadata
- ✅ RemoveMetadata to strip all tags

**Key Implementation Details:**
- Uses `github.com/bogem/id3v2/v2` for MP3 ID3v2.4 tags
- Uses `github.com/go-flac/flacvorbis` for FLAC Vorbis comments
- Proper handling of multi-disc albums with disc numbers
- Preserves existing metadata when updating specific fields

### 2. Artwork Download and Embedding (Task 6.2)

**Files Created:**
- `artwork.go` - Artwork management and caching

**Functionality:**
- ✅ DownloadAndEmbedArtwork with configurable sizes
- ✅ Artwork caching to avoid re-downloading
- ✅ Image resizing to configured dimensions using Lanczos3 algorithm
- ✅ Artwork embedding in both MP3 (APIC frames) and FLAC (PICTURE blocks)
- ✅ Cache management:
  - GetCacheSize - Get total cache size
  - ClearCache - Clear all cached artwork
  - CleanOldCache - Remove old cache entries

**Key Implementation Details:**
- Uses `github.com/nfnt/resize` for high-quality image resizing
- MD5-based cache keys for efficient lookup
- Supports JPEG and PNG formats
- Automatic MIME type detection
- Thread-safe caching operations
- Preserves existing metadata when adding artwork

### 3. Lyrics Embedding (Task 6.3)

**Files Created:**
- `lyrics.go` - Lyrics management and embedding

**Functionality:**
- ✅ Embed synchronized lyrics in USLT/SYLT ID3 frames (MP3)
- ✅ Embed lyrics in Vorbis comments (FLAC)
- ✅ Save lyrics as separate .lrc and .txt files when configured
- ✅ Support configurable lyrics language preference
- ✅ LRC format parsing and generation
- ✅ GetLyrics to read embedded lyrics
- ✅ RemoveLyrics to strip lyrics from files
- ✅ LoadLyricsFromFiles to load from separate files

**Key Implementation Details:**
- Full LRC format support with timestamp conversion
- SYLT frame creation for synchronized lyrics in MP3
- Custom SYNCEDLYRICS field for FLAC synchronized lyrics
- Bidirectional conversion between milliseconds and LRC timestamps
- Support for both synchronized and unsynchronized lyrics

## File Structure

```
internal/metadata/
├── metadata.go          # Core metadata tagging
├── artwork.go           # Artwork download and caching
├── lyrics.go            # Lyrics embedding and management
├── metadata_test.go     # Unit tests
├── example_usage.go     # Usage examples
├── README.md            # Package documentation
└── IMPLEMENTATION.md    # This file
```

## Dependencies Added

```
github.com/bogem/id3v2/v2 v2.1.4
github.com/go-flac/flacvorbis v0.2.0
github.com/go-flac/go-flac v1.0.0
github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646
```

## Testing

All unit tests pass successfully:
- ✅ TestNewManager
- ✅ TestTrackMetadata
- ✅ TestFileExists
- ✅ TestLyricsConfig
- ✅ TestLyrics
- ✅ TestMillisecondsToLRCTimestamp
- ✅ TestLRCTimestampToMilliseconds
- ✅ TestWriteUint32BE
- ✅ TestArtworkCache
- ✅ TestNewArtworkCacheErrors

## Usage Examples

### Basic Metadata Tagging

```go
manager := metadata.NewManager(&metadata.Config{
    EmbedArtwork: true,
    ArtworkSize:  1200,
})

metadata := &metadata.TrackMetadata{
    Title:       "Song Title",
    Artist:      "Artist Name",
    Album:       "Album Name",
    TrackNumber: 1,
    DiscNumber:  1,
    Year:        2024,
}

err := manager.ApplyMetadata("song.mp3", metadata)
```

### Artwork Embedding

```go
err := manager.DownloadAndEmbedArtwork(
    "song.mp3",
    "https://example.com/artwork.jpg",
    1200,
)
```

### Lyrics Embedding

```go
lyrics := &metadata.Lyrics{
    SyncedLyrics:   "[00:12.00]First line\n[00:15.50]Second line",
    UnsyncedLyrics: "First line\nSecond line",
}

err := manager.EmbedLyrics("song.mp3", lyrics, &metadata.LyricsConfig{
    EmbedInFile:      true,
    SaveSeparateFile: true,
    Language:         "eng",
})
```

## Format Support

### MP3 (ID3v2.4)
- ✅ All standard text frames (TIT2, TPE1, TALB, etc.)
- ✅ APIC frames for artwork
- ✅ USLT frames for unsynchronized lyrics
- ✅ SYLT frames for synchronized lyrics
- ✅ Multi-disc support (TPOS frame)

### FLAC (Vorbis Comments)
- ✅ All standard Vorbis comment fields
- ✅ PICTURE metadata blocks for artwork
- ✅ LYRICS field for unsynchronized lyrics
- ✅ Custom SYNCEDLYRICS field for synchronized lyrics
- ✅ Multi-disc support (DISCNUMBER field)

## Performance Considerations

- Streaming I/O for large files
- Efficient caching to minimize network requests
- High-quality Lanczos3 resizing algorithm
- Minimal memory footprint
- Thread-safe operations on different files

## Integration Points

The metadata package integrates with:
- **Download Manager**: Apply metadata after download completion
- **Deezer API**: Use track/album data for metadata
- **Config Manager**: Use artwork size and lyrics settings
- **Queue Store**: Track metadata application status

## Next Steps

This package is ready for integration with the download manager (Task 5) to automatically apply metadata, artwork, and lyrics to downloaded tracks.

## Requirements Satisfied

- ✅ Requirement 7.4: Embed metadata (ID3 tags) and high-resolution artwork
- ✅ Requirement 13.2: Embed synchronized lyrics in audio file metadata
- ✅ Requirement 13.3: Save lyrics as separate .lrc and .txt files
- ✅ Requirement 13.4: Support configurable lyrics language preference
