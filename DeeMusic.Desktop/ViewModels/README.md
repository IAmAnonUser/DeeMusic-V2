# ViewModels

This directory contains the MVVM ViewModels for the DeeMusic Desktop application. ViewModels act as the bridge between the UI (Views) and the business logic (Services/Models).

## Overview

All ViewModels follow the MVVM pattern and implement `INotifyPropertyChanged` to support data binding. They use the CommunityToolkit.Mvvm library for command implementations.

## ViewModels

### MainViewModel

**Purpose**: Main application ViewModel that manages navigation, application state, and window lifecycle.

**Key Features**:
- Navigation between different views (Search, Queue, Settings)
- Application initialization and shutdown
- Window lifecycle management (loaded, closing)
- Hosts child ViewModels

**Properties**:
- `CurrentView`: The currently displayed view
- `CurrentPage`: Name of the current page
- `IsInitialized`: Whether the backend is initialized
- `SearchViewModel`: Instance of SearchViewModel
- `QueueViewModel`: Instance of QueueViewModel
- `SettingsViewModel`: Instance of SettingsViewModel

**Commands**:
- `NavigateCommand`: Navigate to a specific page
- `WindowLoadedCommand`: Handle window loaded event
- `WindowClosingCommand`: Handle window closing event

**Usage**:
```csharp
var service = new DeeMusicService();
var mainViewModel = new MainViewModel(service);

// Navigate to queue
mainViewModel.NavigateCommand.Execute("Queue");
```

### SearchViewModel

**Purpose**: Manages search functionality, results, and download operations from search results.

**Key Features**:
- Search for tracks, albums, artists, and playlists
- Filter search results by type
- Download items directly from search results
- View details of search results

**Properties**:
- `SearchQuery`: Current search query
- `SelectedSearchType`: Selected search type (track, album, artist, playlist)
- `IsSearching`: Whether a search is in progress
- `TrackResults`: Collection of track search results
- `AlbumResults`: Collection of album search results
- `ArtistResults`: Collection of artist search results
- `PlaylistResults`: Collection of playlist search results

**Commands**:
- `SearchCommand`: Execute a search
- `DownloadTrackCommand`: Download a track
- `DownloadAlbumCommand`: Download an album
- `DownloadPlaylistCommand`: Download a playlist
- `ViewDetailsCommand`: View details of a result

**Usage**:
```csharp
var searchViewModel = new SearchViewModel(service);
searchViewModel.SearchQuery = "Daft Punk";
searchViewModel.SelectedSearchType = "track";
await searchViewModel.SearchCommand.ExecuteAsync(null);
```

### QueueViewModel

**Purpose**: Manages the download queue, progress updates, and queue operations.

**Key Features**:
- Display queue items with real-time progress updates
- Pagination for large queues (loads 100 items at a time)
- Filter queue by status (all, pending, downloading, completed, failed)
- Queue operations (pause, resume, cancel, retry)
- Real-time updates via backend callbacks
- Queue statistics display

**Properties**:
- `QueueItems`: Observable collection of queue items
- `QueueStats`: Queue statistics (total, pending, downloading, completed, failed)
- `StatusFilter`: Current status filter
- `IsLoading`: Whether queue is loading
- `HasMoreItems`: Whether there are more items to load

**Commands**:
- `PauseCommand`: Pause a download
- `ResumeCommand`: Resume a paused download
- `CancelCommand`: Cancel a download
- `RetryCommand`: Retry a failed download
- `ClearCompletedCommand`: Clear all completed downloads
- `LoadMoreCommand`: Load more queue items (pagination)
- `RefreshCommand`: Refresh the queue

**Event Handling**:
- Subscribes to `ProgressUpdated` events from backend
- Subscribes to `StatusChanged` events from backend
- Subscribes to `QueueStatsUpdated` events from backend

**Usage**:
```csharp
var queueViewModel = new QueueViewModel(service);
await queueViewModel.LoadQueueAsync();

// Pause a download
await queueViewModel.PauseCommand.ExecuteAsync(queueItem);

// Load more items
await queueViewModel.LoadMoreCommand.ExecuteAsync(null);
```

**Important**: QueueViewModel implements `IDisposable` and must be disposed to unsubscribe from backend events.

### SettingsViewModel

**Purpose**: Manages application settings, validation, and persistence.

**Key Features**:
- Load and save settings
- Real-time validation
- Track unsaved changes
- Browse for download folder
- Reset to default settings

**Properties**:
- `Settings`: Complete settings object
- `IsSaving`: Whether settings are being saved
- `HasUnsavedChanges`: Whether there are unsaved changes
- `ValidationError`: Current validation error message

**Convenience Properties** (for easier binding):
- `DeezerARL`: Deezer ARL token
- `DownloadPath`: Download directory path
- `Quality`: Download quality (MP3_320, FLAC)
- `ConcurrentDownloads`: Number of concurrent downloads
- `EmbedArtwork`: Whether to embed artwork
- `ArtworkSize`: Artwork size in pixels
- `LyricsEnabled`: Whether lyrics are enabled
- `EmbedLyrics`: Whether to embed lyrics in files
- `SaveLyricsFile`: Whether to save lyrics as separate files
- `Theme`: UI theme (dark, light)
- `MinimizeToTray`: Whether to minimize to system tray
- `StartWithWindows`: Whether to start with Windows

**Commands**:
- `SaveCommand`: Save settings (only enabled when there are valid unsaved changes)
- `ResetCommand`: Reset settings to defaults
- `BrowseFolderCommand`: Browse for download folder
- `OpenDownloadFolderCommand`: Open download folder in Explorer

**Validation**:
- Uses DataAnnotations for validation
- Validates settings before saving
- Displays validation errors to user

**Usage**:
```csharp
var settingsViewModel = new SettingsViewModel(service);
await settingsViewModel.LoadSettingsAsync();

// Change a setting
settingsViewModel.Quality = "FLAC";

// Save changes
if (settingsViewModel.HasUnsavedChanges)
{
    await settingsViewModel.SaveCommand.ExecuteAsync(null);
}
```

## Common Patterns

### Property Change Notification

All ViewModels implement `INotifyPropertyChanged`:

```csharp
private string _myProperty;
public string MyProperty
{
    get => _myProperty;
    set
    {
        if (_myProperty != value)
        {
            _myProperty = value;
            OnPropertyChanged();
        }
    }
}
```

### Commands

Commands use CommunityToolkit.Mvvm:

```csharp
// Synchronous command
public ICommand MyCommand { get; }
MyCommand = new RelayCommand(ExecuteMyCommand);

// Async command
public ICommand MyAsyncCommand { get; }
MyAsyncCommand = new AsyncRelayCommand(ExecuteMyAsyncCommand);

// Command with parameter
public ICommand MyParameterCommand { get; }
MyParameterCommand = new RelayCommand<MyType>(ExecuteMyParameterCommand);
```

### Error Handling

All ViewModels use try-catch blocks and log errors:

```csharp
try
{
    await _service.SomeOperationAsync();
}
catch (Exception ex)
{
    System.Diagnostics.Debug.WriteLine($"Operation failed: {ex.Message}");
    // TODO: Show error to user via dialog or notification
}
```

## Dependencies

- **CommunityToolkit.Mvvm**: For RelayCommand and AsyncRelayCommand
- **DeeMusicService**: High-level service wrapper for backend operations
- **Models**: Data models (Track, Album, QueueItem, Settings, etc.)

## Data Binding

ViewModels are designed to be bound to XAML views:

```xaml
<Window DataContext="{Binding MainViewModel}">
    <TextBox Text="{Binding SearchViewModel.SearchQuery, UpdateSourceTrigger=PropertyChanged}" />
    <Button Command="{Binding SearchViewModel.SearchCommand}" />
    <ListBox ItemsSource="{Binding QueueViewModel.QueueItems}" />
</Window>
```

## Thread Safety

- All property changes are made on the UI thread
- Backend callbacks are marshaled to the UI thread by the service layer
- Async operations use `await` to avoid blocking the UI

## Testing

ViewModels can be unit tested by:
1. Mocking the `DeeMusicService`
2. Testing property change notifications
3. Testing command execution
4. Testing validation logic

## Next Steps

After implementing ViewModels, the next tasks are:

1. **Task 8**: Design and implement XAML views
2. **Task 9**: Implement custom WPF controls
3. **Task 10**: Implement theme system

## Status

✅ **Task 7 Complete**: All MVVM ViewModels have been implemented:
- MainViewModel (navigation and lifecycle)
- SearchViewModel (search and results)
- QueueViewModel (queue management and progress)
- SettingsViewModel (settings management)

All ViewModels are ready for UI binding and have no compilation errors.
