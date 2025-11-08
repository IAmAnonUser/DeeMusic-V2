# DeeMusic Core DLL

This directory contains the Go backend compiled as a C-shared library (DLL) for use with the C# WPF frontend.

## Building

To build the DLL:

```bash
go build -buildmode=c-shared -o deemusic-core.dll cmd/deemusic-core/main.go
```

This will generate two files:
- `deemusic-core.dll` - The shared library
- `deemusic-core.h` - C header file with function declarations

## Exported Functions

### Initialization

- `int InitializeApp(char* configPath)` - Initialize the backend with config file path
- `void ShutdownApp()` - Shutdown and cleanup resources

### Callbacks

- `void SetProgressCallback(ProgressCallback callback)` - Set progress update callback
- `void SetStatusCallback(StatusCallback callback)` - Set status change callback
- `void SetQueueUpdateCallback(QueueUpdateCallback callback)` - Set queue stats callback

### Search & Browse

- `char* Search(char* query, char* searchType, int limit)` - Search for tracks/albums/artists/playlists
- `char* GetAlbum(char* albumID)` - Get album details
- `char* GetArtist(char* artistID)` - Get artist details
- `char* GetPlaylist(char* playlistID)` - Get playlist details
- `char* GetCharts(int limit)` - Get Deezer charts

### Downloads

- `int DownloadTrack(char* trackID, char* quality)` - Download a track
- `int DownloadAlbum(char* albumID, char* quality)` - Download an album
- `int DownloadPlaylist(char* playlistID, char* quality)` - Download a playlist
- `char* ConvertSpotifyURL(char* url)` - Convert Spotify URL (not yet implemented)

### Queue Management

- `char* GetQueue(int offset, int limit, char* filter)` - Get queue items with pagination
- `char* GetQueueStats()` - Get queue statistics
- `int PauseDownload(char* itemID)` - Pause a download
- `int ResumeDownload(char* itemID)` - Resume a download
- `int CancelDownload(char* itemID)` - Cancel a download
- `int RetryDownload(char* itemID)` - Retry a failed download
- `int ClearCompleted()` - Clear completed downloads

### Settings

- `char* GetSettings()` - Get current settings as JSON
- `int UpdateSettings(char* settingsJSON)` - Update settings from JSON
- `char* GetDownloadPath()` - Get download directory path
- `int SetDownloadPath(char* path)` - Set download directory path

### Utility

- `char* GetVersion()` - Get version string
- `void FreeString(char* str)` - Free a string allocated by Go

## Return Values

### Integer Returns
- `0` - Success
- `-1` - Not initialized
- `-2` - Operation failed
- `-3` - Validation error
- `-4` - Save error

### String Returns
All string-returning functions return JSON-encoded data or error objects.
Strings must be freed using `FreeString()` after use.

## Callback Types

```c
typedef void (*ProgressCallback)(char* itemID, int progress, long long bytesProcessed, long long totalBytes);
typedef void (*StatusCallback)(char* itemID, char* status, char* errorMsg);
typedef void (*QueueUpdateCallback)(char* statsJson);
```

## Memory Management

- All strings returned by Go functions must be freed using `FreeString()`
- Callback parameters are managed by Go and should not be freed by C#
- The DLL manages its own internal memory

## Thread Safety

- All exported functions are thread-safe
- Callbacks may be invoked from Go goroutines
- C# must marshal callbacks to the UI thread if needed

## Example Usage (C#)

```csharp
// Initialize
int result = InitializeApp("C:\\path\\to\\settings.json");

// Set callbacks
SetProgressCallback((itemID, progress, bytes, total) => {
    Console.WriteLine($"Progress: {progress}%");
});

// Search
IntPtr resultPtr = Search("Daft Punk", "artist", 10);
string json = Marshal.PtrToStringAnsi(resultPtr);
FreeString(resultPtr);

// Download
DownloadTrack("123456", "MP3_320");

// Cleanup
ShutdownApp();
```
