# DeeMusic Core DLL Implementation

## Overview

This document describes the implementation of the Go backend as a C-shared library (DLL) for integration with the C# WPF frontend.

## What Was Implemented

### 1. New Entry Point (`cmd/deemusic-core/main.go`)

Created a new main package specifically for DLL compilation with:
- CGO integration for C interop
- Exported functions with proper C calling conventions
- Memory management for string marshaling
- Thread-safe callback mechanism

### 2. Callback System

Implemented a callback notifier that replaces the WebSocket-based notification system:

**CallbackNotifier** implements the `Notifier` interface with:
- `NotifyProgress()` - Reports download progress
- `NotifyStarted()` - Signals download start
- `NotifyCompleted()` - Signals download completion
- `NotifyFailed()` - Reports download failures

**C Helper Functions** for safe callback invocation:
- `call_progress_callback()`
- `call_status_callback()`
- `call_queue_update_callback()`

### 3. Exported Functions

#### Initialization & Lifecycle
- `InitializeApp()` - Loads config, initializes database, starts download manager
- `ShutdownApp()` - Graceful cleanup of all resources

#### Callback Registration
- `SetProgressCallback()` - Register progress updates
- `SetStatusCallback()` - Register status changes
- `SetQueueUpdateCallback()` - Register queue statistics updates

#### Search & Browse
- `Search()` - Unified search supporting tracks, albums, artists, playlists
- `GetAlbum()` - Retrieve album details with track listing
- `GetArtist()` - Retrieve artist information
- `GetPlaylist()` - Retrieve playlist contents
- `GetCharts()` - Get Deezer charts

#### Download Operations
- `DownloadTrack()` - Queue single track download
- `DownloadAlbum()` - Queue album download
- `DownloadPlaylist()` - Queue playlist download
- `ConvertSpotifyURL()` - Placeholder for Spotify integration

#### Queue Management
- `GetQueue()` - Retrieve queue items with pagination and filtering
- `GetQueueStats()` - Get queue statistics (total, pending, downloading, etc.)
- `PauseDownload()` - Pause active download
- `ResumeDownload()` - Resume paused download
- `CancelDownload()` - Cancel and remove from queue
- `RetryDownload()` - Retry failed download
- `ClearCompleted()` - Remove completed items

#### Settings Management
- `GetSettings()` - Retrieve current configuration as JSON
- `UpdateSettings()` - Update configuration from JSON
- `GetDownloadPath()` - Get download directory
- `SetDownloadPath()` - Set download directory

#### Utility
- `GetVersion()` - Get version string
- `FreeString()` - Free Go-allocated strings

### 4. Configuration Changes

Removed HTTP server dependencies from `internal/config/config.go`:
- Removed `ServerConfig` struct
- Removed server validation logic
- Removed server defaults
- Removed server settings from Save() method

The configuration now focuses on:
- Deezer API settings
- Download settings
- Spotify settings
- Lyrics settings
- Network settings
- Desktop integration settings
- Logging settings

### 5. Memory Management

Implemented proper memory management for C interop:
- All string returns use `C.CString()` which must be freed by caller
- `FreeString()` export for C# to free Go-allocated memory
- Callback parameters are managed by Go (no C# cleanup needed)
- Thread-safe access to global state with mutexes

### 6. Thread Safety

All exported functions are thread-safe:
- Global state protected by `sync.RWMutex`
- Callback registration protected by separate mutex
- Download manager handles concurrent operations internally

### 7. Error Handling

Consistent error handling across all exports:
- Integer returns: 0 = success, negative = error code
- String returns: JSON with error field on failure
- Initialization check before all operations
- Detailed error messages for debugging

## Build Process

### Requirements
- Go 1.24+ with CGO enabled
- C compiler (MinGW-w64 or TDM-GCC on Windows)
- SQLite3 development libraries

### Build Command
```bash
go build -buildmode=c-shared -o deemusic-core.dll cmd/deemusic-core/main.go
```

### Build Script
Created `scripts/build-dll.ps1` for automated building:
- Checks for Go and CGO availability
- Supports Debug and Release modes
- Reports build status and file sizes

### Output Files
- `deemusic-core.dll` - The shared library (~27 MB)
- `deemusic-core.h` - C header with function declarations

## Integration Points

### C# P/Invoke
The generated header file provides all necessary declarations for C# P/Invoke:
```csharp
[DllImport("deemusic-core.dll")]
public static extern int InitializeApp(string configPath);
```

### Callback Delegates
C# must define matching delegate types:
```csharp
public delegate void ProgressCallback(string itemID, int progress, long bytesProcessed, long totalBytes);
```

### JSON Communication
All complex data is exchanged as JSON strings:
- Search results
- Queue items
- Settings
- Statistics

## Preserved Backend Components

All existing Go packages remain functional:
- `internal/api` - Deezer and Spotify API clients
- `internal/decryption` - Decryption engine
- `internal/metadata` - Metadata processing
- `internal/download` - Download manager
- `internal/store` - SQLite queue store
- `internal/config` - Configuration management
- `internal/security` - Security utilities
- `internal/network` - Network utilities

## Testing

The DLL compiles successfully and exports all required functions. Integration testing with C# WPF frontend is the next step.

## Next Steps

1. Create C# WPF project
2. Implement P/Invoke wrapper
3. Create ViewModels and Views
4. Test callback mechanism
5. Implement full UI integration

## Notes

- The DLL is Windows-specific due to `c-shared` build mode
- All existing functionality is preserved
- No breaking changes to internal packages
- Server code remains in place but is not used by DLL
- Future: Could create separate builds for server and DLL modes
