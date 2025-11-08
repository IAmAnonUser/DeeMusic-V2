# DeeMusic Desktop Models

This directory contains the C# data models for the DeeMusic Desktop application. These models represent the data structures used throughout the application and are designed to match the JSON responses from the Go backend DLL.

## Model Files

### Music Content Models

- **Track.cs** - Represents a Deezer track with metadata (title, artist, album, duration, etc.)
- **Album.cs** - Represents a Deezer album with tracks and metadata
- **Artist.cs** - Represents a Deezer artist with profile information
- **Playlist.cs** - Represents a Deezer playlist with tracks and creator information
- **User.cs** - Represents a Deezer user (included in Playlist.cs)
- **Genre.cs** - Represents a music genre (included in Album.cs)

### Queue Models

- **QueueItem.cs** - Represents a download queue item with progress tracking
  - Implements `INotifyPropertyChanged` for real-time UI updates
  - Includes helper properties for UI binding (IsPending, IsDownloading, etc.)
  - Provides formatted display strings (DisplayName, StatusText, etc.)

- **QueueStats.cs** - Represents queue statistics (total, pending, downloading, completed, failed)
  - Implements `INotifyPropertyChanged` for real-time UI updates
  - Provides summary properties for display

### Settings Models

- **Settings.cs** - Main application configuration container
  - **DeezerSettings** - Deezer API credentials (ARL token)
  - **DownloadSettings** - Download preferences (quality, output directory, concurrent downloads, etc.)
  - **SpotifySettings** - Spotify API credentials for playlist conversion
  - **LyricsSettings** - Lyrics download and embedding preferences
  - **NetworkSettings** - Network configuration (timeout, retries, proxy, bandwidth limit)
  - **SystemSettings** - System integration settings (startup, tray, theme)
  - **LoggingSettings** - Logging configuration

### Utility Models

- **SearchResult.cs** - Generic search result wrapper with pagination support
- **ErrorResponse.cs** - Error response from backend API calls

## JSON Serialization

All models use `System.Text.Json` attributes for serialization:
- `[JsonPropertyName("property_name")]` - Maps C# properties to JSON field names
- Snake_case JSON fields are mapped to PascalCase C# properties

## Validation

Models include validation attributes where appropriate:
- `[Required]` - Marks required fields
- `[Range(min, max)]` - Validates numeric ranges
- `[RegularExpression(pattern)]` - Validates string patterns

## Property Change Notification

Models that need real-time UI updates implement `INotifyPropertyChanged`:
- **QueueItem** - For download progress updates
- **QueueStats** - For queue statistics updates

## Helper Properties

Many models include computed properties for UI display:
- `DisplayName` - Formatted display name (e.g., "Artist - Title")
- `FormattedDuration` - Human-readable duration (e.g., "3:45")
- `FormattedBytes` - Human-readable file sizes (e.g., "3.5 MB")
- `StatusText` - User-friendly status descriptions

## Usage Example

```csharp
// Deserialize a track from JSON
var track = JsonSerializer.Deserialize<Track>(jsonString);

// Access properties
Console.WriteLine(track.DisplayName); // "Artist - Title"
Console.WriteLine(track.FormattedDuration); // "3:45"

// Create a queue item
var queueItem = new QueueItem
{
    Id = track.Id,
    Type = "track",
    Title = track.Title,
    Artist = track.Artist?.Name ?? "",
    Status = "pending"
};

// Bind to UI (WPF)
queueItem.PropertyChanged += (s, e) =>
{
    if (e.PropertyName == nameof(QueueItem.Progress))
    {
        // Update progress bar
    }
};

// Deserialize settings
var settings = JsonSerializer.Deserialize<Settings>(settingsJson);
Console.WriteLine(settings.Download.Quality); // "MP3_320"
```

## Backend Compatibility

These models are designed to match the JSON structures returned by the Go backend DLL:
- Track, Album, Artist, Playlist match `internal/api/models.go`
- QueueItem, QueueStats match `internal/store/queue.go`
- Settings matches `internal/config/config.go`

Any changes to the Go backend data structures should be reflected in these C# models to maintain compatibility.
