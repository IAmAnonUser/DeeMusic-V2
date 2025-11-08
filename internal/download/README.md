# Download Package

The download package provides a robust, concurrent download management system for DeeMusic. It handles downloading and decrypting audio files from Deezer with progress tracking, retry logic, and WebSocket notifications.

## Components

### WorkerPool
Manages a pool of worker goroutines for concurrent downloads with configurable limits.

**Features:**
- Configurable number of concurrent workers
- Job queuing with buffered channels
- Graceful shutdown with context cancellation
- Individual job cancellation
- Active job tracking

### Manager
Orchestrates all download operations, coordinating the worker pool, queue store, Deezer API, and decryption processor.

**Features:**
- Track, album, and playlist downloads
- Pause, resume, and cancel operations
- Automatic retry with exponential backoff
- Queue persistence via SQLite
- Progress tracking and notifications
- Download statistics

### ProgressNotifier
Handles real-time progress tracking and WebSocket broadcasting to connected clients.

**Features:**
- Real-time progress updates with speed and ETA calculation
- Status notifications (started, completed, failed)
- WebSocket client management
- Download statistics (success rate, active downloads)
- Thread-safe operations

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      Manager                             │
│  ┌────────────────────────────────────────────────┐    │
│  │  DownloadTrack / DownloadAlbum / DownloadPlaylist │  │
│  │  PauseDownload / ResumeDownload / CancelDownload  │  │
│  └────────────────┬───────────────────────────────┘    │
│                   │                                      │
│  ┌────────────────▼───────────────┐                     │
│  │        WorkerPool               │                     │
│  │  ┌──────┐ ┌──────┐ ┌──────┐   │                     │
│  │  │Worker│ │Worker│ │Worker│   │                     │
│  │  └──┬───┘ └──┬───┘ └──┬───┘   │                     │
│  └─────┼────────┼────────┼────────┘                     │
│        │        │        │                              │
│  ┌─────▼────────▼────────▼────────┐                     │
│  │    Job Processing               │                     │
│  │  - Download & Decrypt           │                     │
│  │  - Progress Tracking            │                     │
│  │  - Error Handling               │                     │
│  └─────┬───────────────────────────┘                     │
│        │                                                 │
│  ┌─────▼───────────────────────────┐                     │
│  │    ProgressNotifier             │                     │
│  │  - WebSocket Broadcasting       │                     │
│  │  - Statistics Tracking          │                     │
│  └─────────────────────────────────┘                     │
└─────────────────────────────────────────────────────────┘
```

## Usage

### Basic Setup

```go
import (
    "context"
    "github.com/yourusername/deemusic/internal/api"
    "github.com/yourusername/deemusic/internal/config"
    "github.com/yourusername/deemusic/internal/download"
    "github.com/yourusername/deemusic/internal/store"
)

// Load configuration
cfg, err := config.Load("")
if err != nil {
    log.Fatal(err)
}

// Initialize database
db, err := store.InitDB(store.GetDBPath())
if err != nil {
    log.Fatal(err)
}

// Create queue store
queueStore := store.NewQueueStore(db)

// Create Deezer API client
deezerAPI := api.NewDeezerClient(30 * time.Second)
err = deezerAPI.Authenticate(context.Background(), cfg.Deezer.ARL)
if err != nil {
    log.Fatal(err)
}

// Create progress notifier
notifier := download.NewProgressNotifier()
notifier.Start()

// Create download manager
manager := download.NewManager(cfg, queueStore, deezerAPI, notifier)

// Start the manager
ctx := context.Background()
err = manager.Start(ctx)
if err != nil {
    log.Fatal(err)
}
defer manager.Stop()
```

### Download Operations

```go
// Download a single track
err := manager.DownloadTrack(ctx, "123456789")
if err != nil {
    log.Printf("Failed to queue track: %v", err)
}

// Download an album
err = manager.DownloadAlbum(ctx, "987654321")
if err != nil {
    log.Printf("Failed to queue album: %v", err)
}

// Download a playlist
err = manager.DownloadPlaylist(ctx, "555555555")
if err != nil {
    log.Printf("Failed to queue playlist: %v", err)
}
```

### Queue Management

```go
// Pause a download
err := manager.PauseDownload("track_123456789")
if err != nil {
    log.Printf("Failed to pause: %v", err)
}

// Resume a download
err = manager.ResumeDownload("track_123456789")
if err != nil {
    log.Printf("Failed to resume: %v", err)
}

// Cancel a download
err = manager.CancelDownload("track_123456789")
if err != nil {
    log.Printf("Failed to cancel: %v", err)
}
```

### Statistics

```go
// Get download statistics
stats, err := manager.GetStats()
if err != nil {
    log.Printf("Failed to get stats: %v", err)
} else {
    fmt.Printf("Active downloads: %d\n", stats["active_downloads"])
    fmt.Printf("Queue total: %d\n", stats["queue_total"])
    fmt.Printf("Completed: %d\n", stats["queue_completed"])
}

// Get notifier statistics
notifierStats := notifier.GetStats()
fmt.Printf("Success rate: %.1f%%\n", notifierStats["success_rate"])
```

### WebSocket Integration

```go
// Register a WebSocket client
client := download.NewClient("client-123")
notifier.Register(client)

// Listen for messages
go func() {
    for data := range client.SendChan {
        // Send data to WebSocket connection
        ws.WriteMessage(websocket.TextMessage, data)
    }
}()

// Unregister when done
defer notifier.Unregister(client)
```

## Job Types

- **JobTypeTrack**: Downloads a single track
- **JobTypeAlbum**: Downloads all tracks in an album
- **JobTypePlaylist**: Downloads all tracks in a playlist

## Status Values

Queue items can have the following statuses:
- `pending`: Waiting to be processed
- `downloading`: Currently being downloaded
- `completed`: Successfully downloaded
- `failed`: Download failed (will retry if under limit)

## Error Handling

The download manager implements automatic retry logic with exponential backoff:
- Network errors: Retry with increasing delays
- Authentication errors: Attempt token refresh
- Rate limiting: Wait and retry
- Decryption errors: Mark as failed (no retry)

## Configuration

Key configuration options:
- `Download.ConcurrentDownloads`: Number of concurrent workers (default: 8)
- `Download.OutputDir`: Output directory for downloads
- `Download.Quality`: Audio quality (MP3_320 or FLAC)
- `Network.Timeout`: HTTP request timeout in seconds
- `Network.MaxRetries`: Maximum retry attempts for failed downloads

## Thread Safety

All components are thread-safe and can be safely accessed from multiple goroutines:
- WorkerPool uses sync.Map for active jobs
- Manager uses sync.RWMutex for paused jobs
- ProgressNotifier uses sync.RWMutex for clients and stats

## Performance Considerations

- Worker pool size should match available CPU cores and network bandwidth
- Job and result channels are buffered for smoother operation
- Progress updates are throttled to avoid overwhelming WebSocket clients
- Statistics are calculated incrementally to minimize overhead

## Testing

See `example_usage.go` for complete working examples.
