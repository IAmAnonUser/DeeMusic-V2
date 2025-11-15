# DeeMusic Backend Services

This directory contains the C# P/Invoke wrapper and service layer for communicating with the Go backend DLL.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                    C# Application                       │
│                                                         │
│  ┌───────────────────────────────────────────────────┐ │
│  │         DeeMusicService (High-level API)          │ │
│  │  - Async methods                                  │ │
│  │  - JSON deserialization                           │ │
│  │  - Error handling & retry logic                   │ │
│  │  - Event forwarding                               │ │
│  └───────────────────────────────────────────────────┘ │
│                         ↕                               │
│  ┌───────────────────────────────────────────────────┐ │
│  │      BackendCallbackHandler (Event Bridge)        │ │
│  │  - Callback registration                          │ │
│  │  - UI thread marshaling                           │ │
│  │  - Event publishing                               │ │
│  └───────────────────────────────────────────────────┘ │
│                         ↕                               │
│  ┌───────────────────────────────────────────────────┐ │
│  │       GoBackend (P/Invoke Declarations)           │ │
│  │  - DLL imports                                    │ │
│  │  - String marshaling                              │ │
│  │  - Memory management                              │ │
│  └───────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
                         ↕
              ┌──────────────────────┐
              │  deemusic-core.dll   │
              │    (Go Backend)      │
              └──────────────────────┘
```

## Components

### 1. GoBackendService.cs

Low-level P/Invoke wrapper that provides direct access to all Go DLL exports.

**Key Features:**
- P/Invoke declarations for all exported Go functions
- String marshaling helpers (`PtrToStringAndFree`)
- Memory management utilities
- Error code translation
- Callback delegate definitions

**Usage:**
```csharp
// Direct P/Invoke call (not recommended for application code)
var ptr = GoBackend.Search("query", "track", 50);
var json = GoBackend.PtrToStringAndFree(ptr);
```

### 2. BackendCallbackHandler.cs

Manages callbacks from the Go backend and marshals them to the UI thread.

**Key Features:**
- Callback registration with Go backend
- Thread-safe callback handling
- UI thread marshaling using Dispatcher
- Event-based notification system
- Proper delegate lifetime management

**Events:**
- `ProgressUpdated` - Download progress updates
- `StatusChanged` - Download status changes (started, completed, failed)
- `QueueStatsUpdated` - Queue statistics updates

**Usage:**
```csharp
var handler = new BackendCallbackHandler();
handler.ProgressUpdated += (sender, args) =>
{
    Console.WriteLine($"Item {args.ItemID}: {args.Progress}%");
};
```

### 3. DeeMusicService.cs

High-level service wrapper that provides a clean, async API for the application.

**Key Features:**
- Async/await pattern for all operations
- Automatic JSON deserialization
- Error handling with custom exceptions
- Retry logic with exponential backoff
- Type-safe generic methods
- Event forwarding from callback handler

**Usage:**
```csharp
var service = new DeeMusicService();

// Initialize
await service.InitializeAsync("config.json");

// Search
var tracks = await service.SearchAsync<TrackList>("Daft Punk", "track");

// Download
await service.DownloadTrackAsync("123456");

// Listen to events
service.ProgressUpdated += (sender, args) =>
{
    UpdateProgressBar(args.Progress);
};

// Cleanup
service.Dispose();
```

## API Reference

### Initialization

```csharp
Task<bool> InitializeAsync(string configPath)
void Shutdown()
```

### Search & Browse

```csharp
Task<T?> SearchAsync<T>(string query, string searchType, int limit = 50)
Task<T?> GetAlbumAsync<T>(string albumID)
Task<T?> GetArtistAsync<T>(string artistID)
Task<T?> GetPlaylistAsync<T>(string playlistID)
Task<T?> GetChartsAsync<T>(int limit = 25)
```

### Download Operations

```csharp
Task DownloadTrackAsync(string trackID, string? quality = null)
Task DownloadAlbumAsync(string albumID, string? quality = null)
Task DownloadPlaylistAsync(string playlistID, string? quality = null)
Task<T?> ConvertSpotifyURLAsync<T>(string url)
```

### Queue Management

```csharp
Task<T?> GetQueueAsync<T>(int offset = 0, int limit = 100, string? filter = null)
Task<QueueStats?> GetQueueStatsAsync()
Task PauseDownloadAsync(string itemID)
Task ResumeDownloadAsync(string itemID)
Task CancelDownloadAsync(string itemID)
Task RetryDownloadAsync(string itemID)
Task ClearCompletedAsync()
```

### Settings Management

```csharp
Task<T?> GetSettingsAsync<T>()
Task UpdateSettingsAsync<T>(T settings)
Task<string?> GetDownloadPathAsync()
Task SetDownloadPathAsync(string path)
```

### System

```csharp
Task<string?> GetVersionAsync()
```

## Memory Management

The Go backend allocates strings using C.CString, which must be freed by the C# side.

**Important:**
- All functions returning `IntPtr` (string pointers) must be freed using `FreeString`
- The helper method `PtrToStringAndFree` handles this automatically
- Never call `FreeString` twice on the same pointer

**Example:**
```csharp
// Manual memory management (not recommended)
var ptr = GoBackend.Search("query", "track", 50);
var json = Marshal.PtrToStringAnsi(ptr);
GoBackend.FreeString(ptr);

// Automatic memory management (recommended)
var json = GoBackend.PtrToStringAndFree(ptr);
```

## Error Handling

### Error Codes

Go functions return integer error codes:
- `0` - Success
- `-1` - Not initialized or invalid state
- `-2` - Operation failed
- `-3` - Invalid configuration
- `-4` - Database error
- `-5` - Migration failed
- `-6` - Failed to start download manager

### Exceptions

`BackendException` is thrown for backend errors:
```csharp
try
{
    await service.DownloadTrackAsync("invalid-id");
}
catch (BackendException ex)
{
    Console.WriteLine($"Error: {ex.Message}");
    if (ex.ErrorCode.HasValue)
    {
        Console.WriteLine($"Code: {ex.ErrorCode}");
    }
}
```

## Thread Safety

- All P/Invoke calls are thread-safe (Go backend handles synchronization)
- Callbacks are automatically marshaled to the UI thread
- Events are raised on the UI thread
- Multiple concurrent operations are supported

## Best Practices

1. **Always initialize before use:**
   ```csharp
   await service.InitializeAsync(configPath);
   ```

2. **Use the high-level service:**
   ```csharp
   // Good
   var tracks = await service.SearchAsync<TrackList>("query", "track");
   
   // Avoid (unless you need low-level control)
   var ptr = GoBackend.Search("query", "track", 50);
   ```

3. **Dispose properly:**
   ```csharp
   using var service = new DeeMusicService();
   // ... use service
   // Automatically disposed
   ```

4. **Handle errors gracefully:**
   ```csharp
   try
   {
       await service.DownloadTrackAsync(trackID);
   }
   catch (BackendException ex)
   {
       ShowErrorDialog(ex.Message);
   }
   ```

5. **Subscribe to events for real-time updates:**
   ```csharp
   service.ProgressUpdated += OnProgressUpdated;
   service.StatusChanged += OnStatusChanged;
   ```

## Testing

To test the P/Invoke wrapper:

1. Ensure `deemusic-core.dll` is in the output directory
2. Create a test configuration file
3. Initialize and call methods:

```csharp
var service = new DeeMusicService();
await service.InitializeAsync("test-config.json");

var version = await service.GetVersionAsync();
Console.WriteLine($"Backend version: {version}");

service.Dispose();
```

## Requirements Satisfied

This implementation satisfies the following requirements from the spec:

- **Requirement 2.3**: P/Invoke declarations for all Go exports ✓
- **Requirement 4.2**: Direct function calls without HTTP overhead ✓
- **Requirement 4.3**: Direct callbacks for progress updates ✓
- **Requirement 4.4**: Real-time UI updates without polling ✓
- **Requirement 4.1**: Support for all existing features ✓

### 4. ThemeManager.cs

Manages application theme switching with smooth transitions.

**Key Features:**
- Singleton pattern for global access
- Dynamic theme switching between dark and light modes
- Smooth fade animations during theme transitions
- Material Design theme integration
- Theme persistence support

**Usage:**
```csharp
// Get instance
var themeManager = ThemeManager.Instance;

// Apply theme
themeManager.ApplyTheme("dark", animate: true);

// Toggle theme
var newTheme = themeManager.ToggleTheme(animate: true);

// Initialize from settings
themeManager.Initialize("dark");

// Get current theme
var currentTheme = themeManager.CurrentTheme;
```

**Theme Files:**
- `Resources/Styles/DarkTheme.xaml` - Dark theme color definitions
- `Resources/Styles/LightTheme.xaml` - Light theme color definitions

**Integration:**
- Automatically updates Material Design theme
- Applies smooth fade transitions (300ms)
- Thread-safe theme switching
- Integrates with Settings system

### 5. TrayService.cs

Manages system tray integration for the application.

**Key Features:**
- System tray icon with context menu
- Minimize to tray functionality
- Tray notifications for download completion
- Quick actions from tray menu

**See:** `TRAY_INTEGRATION.md` for detailed documentation

### 6. StartupManager.cs

Manages Windows startup integration using the registry.

**Key Features:**
- Singleton pattern for global access
- Enable/disable Windows startup
- Support for start minimized option
- Registry-based implementation (no admin rights required)
- Automatic synchronization with settings

**Usage:**
```csharp
// Get instance
var startupManager = StartupManager.Instance;

// Enable startup
startupManager.EnableStartup(startMinimized: true);

// Disable startup
startupManager.DisableStartup();

// Check if enabled
bool isEnabled = startupManager.IsStartupEnabled();

// Update startup configuration
startupManager.UpdateStartup(enabled: true, startMinimized: false);
```

**Registry Details:**
- **Path:** `HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\Run`
- **Value Name:** `DeeMusic`
- **Value Format:** `"C:\Path\To\DeeMusic.Desktop.exe" [--minimized]`

**Integration:**
- Automatically called when settings change
- Syncs with settings on application startup
- Handles command line arguments for minimized start
- Silent error handling (logs to Debug output)

**See:** `STARTUP_INTEGRATION.md` for detailed documentation

## Next Steps

After implementing this P/Invoke wrapper, the next tasks are:

1. **Task 6**: Create C# data models for Track, Album, Artist, etc.
2. **Task 7**: Implement MVVM ViewModels
3. **Task 8**: Design and implement XAML views

The service layer is now ready to be consumed by ViewModels and UI components.
