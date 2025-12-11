using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Linq;
using System.Runtime.CompilerServices;
using System.Threading.Tasks;
using System.Windows.Input;
using CommunityToolkit.Mvvm.Input;
using DeeMusic.Desktop.Models;
using DeeMusic.Desktop.Services;

namespace DeeMusic.Desktop.ViewModels
{
    /// <summary>
    /// ViewModel for the download queue
    /// Manages queue items, progress updates, and queue operations
    /// Optimized for large queues (10,000+ items)
    /// </summary>
    public class QueueViewModel : INotifyPropertyChanged, IDisposable
    {
        private readonly DeeMusicService _service;
        private string _statusFilter = "all";
        private bool _isLoading;
        private int _currentOffset;
        private const int PageSize = 100; // Load 100 items per page
        private const int MaxInMemoryItems = 1000; // Maximum items to keep in memory
        private bool _hasMoreItems = true;
        private TrayService? _trayService;
        private int _totalItems;
        private int _currentPage = 1;
        
        // Periodic refresh timer to keep UI in sync with backend
        private System.Windows.Threading.DispatcherTimer? _refreshTimer;
        private const int RefreshIntervalSeconds = 5; // Refresh every 5 seconds when downloads are active (reduced frequency for better performance)
        private bool _hasActiveDownloads;
        
        // Progress update throttling to prevent UI flooding
        private readonly Dictionary<string, DateTime> _lastProgressUpdate = new();

        public event PropertyChangedEventHandler? PropertyChanged;

        #region Properties

        /// <summary>
        /// Gets the collection of queue items
        /// </summary>
        public ObservableCollection<QueueItem> QueueItems { get; } = new();

        /// <summary>
        /// Gets the queue statistics
        /// </summary>
        public QueueStats QueueStats { get; private set; } = new();

        /// <summary>
        /// Gets or sets the status filter
        /// </summary>
        public string StatusFilter
        {
            get => _statusFilter;
            set
            {
                if (_statusFilter != value)
                {
                    _statusFilter = value;
                    OnPropertyChanged();
                    
                    // Reload queue with new filter
                    _ = LoadQueueAsync();
                }
            }
        }

        /// <summary>
        /// Gets whether the queue is loading
        /// </summary>
        public bool IsLoading
        {
            get => _isLoading;
            private set
            {
                if (_isLoading != value)
                {
                    _isLoading = value;
                    OnPropertyChanged();
                }
            }
        }

        /// <summary>
        /// Gets whether there are more items to load
        /// </summary>
        public bool HasMoreItems
        {
            get => _hasMoreItems;
            private set
            {
                if (_hasMoreItems != value)
                {
                    _hasMoreItems = value;
                    OnPropertyChanged();
                }
            }
        }

        /// <summary>
        /// Gets the total number of items in the queue
        /// </summary>
        public int TotalItems
        {
            get => _totalItems;
            private set
            {
                if (_totalItems != value)
                {
                    _totalItems = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(TotalPages));
                    OnPropertyChanged(nameof(PageInfo));
                    OnPropertyChanged(nameof(ShowPagination));
                }
            }
        }

        /// <summary>
        /// Gets the current page number
        /// </summary>
        public int CurrentPage
        {
            get => _currentPage;
            private set
            {
                if (_currentPage != value)
                {
                    _currentPage = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(PageInfo));
                    OnPropertyChanged(nameof(CanLoadPrevious));
                    OnPropertyChanged(nameof(CanLoadNext));
                }
            }
        }

        /// <summary>
        /// Gets the total number of pages
        /// </summary>
        public int TotalPages => TotalItems > 0 ? (int)Math.Ceiling((double)TotalItems / PageSize) : 0;

        /// <summary>
        /// Gets whether pagination should be shown
        /// </summary>
        public bool ShowPagination => TotalItems > PageSize;

        /// <summary>
        /// Gets whether the previous page can be loaded
        /// </summary>
        public bool CanLoadPrevious => CurrentPage > 1;

        /// <summary>
        /// Gets whether the next page can be loaded
        /// </summary>
        public bool CanLoadNext => CurrentPage < TotalPages;

        /// <summary>
        /// Gets the page information string
        /// </summary>
        public string PageInfo => $"Page {CurrentPage} of {TotalPages} ({TotalItems} items)";

        /// <summary>
        /// Gets whether the queue is empty
        /// </summary>
        public bool IsQueueEmpty => QueueItems.Count == 0 && !IsLoading;

        /// <summary>
        /// Gets the queue statistics with proper property names
        /// </summary>
        public QueueStats Stats => QueueStats;
        
        /// <summary>
        /// Gets the total number of albums/playlists in the queue
        /// </summary>
        public int TotalTracksInQueue
        {
            get
            {
                return QueueItems
                    .Count(i => i.Type == "album" || i.Type == "playlist");
            }
        }
        
        /// <summary>
        /// Gets the number of completed albums/playlists in the queue
        /// </summary>
        public int CompletedTracksInQueue
        {
            get
            {
                return QueueItems
                    .Count(i => (i.Type == "album" || i.Type == "playlist") && i.Status == "completed");
            }
        }

        /// <summary>
        /// Gets the available status filters
        /// </summary>
        public List<string> StatusFilters { get; } = new()
        {
            "all",
            "pending",
            "downloading",
            "completed",
            "failed"
        };

        #endregion

        #region Commands

        /// <summary>
        /// Command to pause a download
        /// </summary>
        public ICommand PauseCommand { get; }

        /// <summary>
        /// Command to resume a download
        /// </summary>
        public ICommand ResumeCommand { get; }

        /// <summary>
        /// Command to cancel a download
        /// </summary>
        public ICommand CancelCommand { get; }

        /// <summary>
        /// Command to retry a failed download
        /// </summary>
        public ICommand RetryCommand { get; }

        /// <summary>
        /// Command to clear completed downloads
        /// </summary>
        public ICommand ClearCompletedCommand { get; }

        /// <summary>
        /// Command to load more queue items (pagination)
        /// </summary>
        public ICommand LoadMoreCommand { get; }

        /// <summary>
        /// Command to load the next page
        /// </summary>
        public ICommand LoadNextPageCommand { get; }

        /// <summary>
        /// Command to load the previous page
        /// </summary>
        public ICommand LoadPreviousPageCommand { get; }

        /// <summary>
        /// Command to refresh the queue
        /// </summary>
        public ICommand RefreshCommand { get; }

        /// <summary>
        /// Command to pause all downloads
        /// </summary>
        public ICommand PauseAllCommand { get; }

        /// <summary>
        /// Command to resume all downloads
        /// </summary>
        public ICommand ResumeAllCommand { get; }
        
        /// <summary>
        /// Command to clear all items from queue
        /// </summary>
        public ICommand ClearAllCommand { get; }
        
        /// <summary>
        /// Command to retry all failed downloads
        /// </summary>
        public ICommand RetryFailedCommand { get; }

        #endregion

        public QueueViewModel(DeeMusicService service)
        {
            _service = service ?? throw new ArgumentNullException(nameof(service));

            // Initialize commands
            PauseCommand = new AsyncRelayCommand<QueueItem>(PauseDownloadAsync);
            ResumeCommand = new AsyncRelayCommand<QueueItem>(ResumeDownloadAsync);
            CancelCommand = new AsyncRelayCommand<QueueItem>(CancelDownloadAsync);
            RetryCommand = new AsyncRelayCommand<QueueItem>(RetryDownloadAsync);
            ClearCompletedCommand = new AsyncRelayCommand(ClearCompletedAsync);
            LoadMoreCommand = new AsyncRelayCommand(LoadMoreItemsAsync);
            LoadNextPageCommand = new AsyncRelayCommand(LoadNextPageAsync);
            LoadPreviousPageCommand = new AsyncRelayCommand(LoadPreviousPageAsync);
            RefreshCommand = new AsyncRelayCommand(LoadQueueAsync);
            PauseAllCommand = new AsyncRelayCommand(PauseAllDownloadsAsync);
            ResumeAllCommand = new AsyncRelayCommand(ResumeAllDownloadsAsync);
            ClearAllCommand = new AsyncRelayCommand(ClearAllAsync);
            RetryFailedCommand = new AsyncRelayCommand(RetryAllFailedAsync);

            // Subscribe to backend events
            _service.ProgressUpdated += OnProgressUpdated;
            _service.StatusChanged += OnStatusChanged;
            _service.QueueStatsUpdated += OnQueueStatsUpdated;
            
            // Initialize periodic refresh timer
            InitializeRefreshTimer();
        }
        
        /// <summary>
        /// Initialize the periodic refresh timer
        /// </summary>
        private void InitializeRefreshTimer()
        {
            _refreshTimer = new System.Windows.Threading.DispatcherTimer
            {
                Interval = TimeSpan.FromSeconds(RefreshIntervalSeconds)
            };
            _refreshTimer.Tick += OnRefreshTimerTick;
            // Timer will be started when we detect active downloads
        }
        
        /// <summary>
        /// Handle refresh timer tick - poll backend for updates
        /// </summary>
        private async void OnRefreshTimerTick(object? sender, EventArgs e)
        {
            // Skip if already loading or if UI is busy
            if (IsLoading)
                return;
            
            // Temporarily stop timer during refresh to prevent overlapping refreshes
            if (_refreshTimer != null)
            {
                _refreshTimer.Stop();
            }
                
            try
            {
                // Get fresh queue stats
                var stats = await _service.GetQueueStatsAsync();
                if (stats != null)
                {
                    // Update individual properties instead of replacing the entire object
                    QueueStats.Total = stats.Total;
                    QueueStats.Pending = stats.Pending;
                    QueueStats.Downloading = stats.Downloading;
                    QueueStats.Completed = stats.Completed;
                    QueueStats.Failed = stats.Failed;
                    
                    // Check if there are active downloads
                    _hasActiveDownloads = stats.Downloading > 0 || stats.Pending > 0;
                    
                    // Stop timer if no active downloads
                    if (!_hasActiveDownloads && _refreshTimer != null)
                    {
                        _refreshTimer.Stop();
                    }
                }
                
                // Refresh the current page to get updated progress
                if (_hasActiveDownloads)
                {
                    await RefreshCurrentPageAsync();
                }
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Refresh timer tick failed: {ex.Message}");
            }
            finally
            {
                // Restart timer if there are still active downloads
                if (_hasActiveDownloads && _refreshTimer != null && !_refreshTimer.IsEnabled)
                {
                    _refreshTimer.Start();
                }
            }
        }
        
        /// <summary>
        /// Refresh the current page without full reload
        /// </summary>
        private async Task RefreshCurrentPageAsync()
        {
            try
            {
                var filter = StatusFilter == "all" ? null : StatusFilter;
                // Only fetch current page for better performance (was PageSize * 10)
                var response = await _service.GetQueueAsync<QueueResponse>(
                    _currentOffset, PageSize, filter);

                if (response?.Items != null)
                {
                    // Do filtering and processing on background thread
                    var albumItems = response.Items
                        .Where(i => i.Type == "album" || i.Type == "playlist")
                        .OrderBy(i => i.CreatedAt)
                        .Take(PageSize)
                        .ToList();
                    
                    // Pre-compute new IDs set on background thread
                    var newIds = new HashSet<string>(albumItems.Select(a => a.Id));
                    
                    // Build lookup dictionary on background thread for O(1) access
                    var newItemsDict = albumItems.ToDictionary(a => a.Id);
                    
                    // Minimize UI thread work - use low priority to avoid blocking user input
                    await System.Windows.Application.Current.Dispatcher.InvokeAsync(() =>
                    {
                        // Build existing items lookup for O(1) access
                        var existingDict = new Dictionary<string, int>();
                        for (int i = 0; i < QueueItems.Count; i++)
                        {
                            existingDict[QueueItems[i].Id] = i;
                        }
                        
                        // Track items to remove (collect first, remove later to avoid index shifting)
                        var indicesToRemove = new List<int>();
                        
                        // Update existing items
                        foreach (var kvp in existingDict)
                        {
                            if (newItemsDict.TryGetValue(kvp.Key, out var newItem))
                            {
                                var existing = QueueItems[kvp.Value];
                                
                                // Only update if values actually changed (reduces property change notifications)
                                bool changed = false;
                                if (existing.TotalTracks != newItem.TotalTracks)
                                {
                                    existing.TotalTracks = newItem.TotalTracks;
                                    changed = true;
                                }
                                if (existing.CompletedTracks != newItem.CompletedTracks)
                                {
                                    existing.CompletedTracks = newItem.CompletedTracks;
                                    changed = true;
                                }
                                if (existing.Progress != newItem.Progress)
                                {
                                    existing.Progress = newItem.Progress;
                                    changed = true;
                                }
                                if (existing.ErrorMessage != newItem.ErrorMessage)
                                {
                                    existing.ErrorMessage = newItem.ErrorMessage;
                                    changed = true;
                                }
                                if (existing.Status != newItem.Status)
                                {
                                    existing.Status = newItem.Status;
                                    changed = true;
                                }
                                
                                // Only update background color if something changed
                                if (changed)
                                {
                                    existing.UpdateComputedBackgroundColor();
                                }
                                
                                // Replace completed items to force WPF refresh (only if status just changed to completed)
                                if (changed && existing.Status == "completed")
                                {
                                    newItem.UpdateComputedBackgroundColor();
                                    QueueItems[kvp.Value] = newItem;
                                }
                            }
                            else
                            {
                                // Item no longer exists
                                indicesToRemove.Add(kvp.Value);
                            }
                        }
                        
                        // Remove items in reverse order to maintain correct indices
                        indicesToRemove.Sort();
                        for (int i = indicesToRemove.Count - 1; i >= 0; i--)
                        {
                            QueueItems.RemoveAt(indicesToRemove[i]);
                        }
                        
                        // Add new items that don't exist yet
                        foreach (var newItem in albumItems)
                        {
                            if (!existingDict.ContainsKey(newItem.Id))
                            {
                                newItem.UpdateComputedBackgroundColor();
                                QueueItems.Add(newItem);
                            }
                        }
                    }, System.Windows.Threading.DispatcherPriority.Background);
                    
                    // Update stats
                    TotalItems = response.Total;
                    OnPropertyChanged(nameof(IsQueueEmpty));
                    OnPropertyChanged(nameof(TotalTracksInQueue));
                    OnPropertyChanged(nameof(CompletedTracksInQueue));
                }
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogWarning($"RefreshCurrentPageAsync failed: {ex.Message}");
            }
        }
        
        /// <summary>
        /// Start the refresh timer if not already running
        /// </summary>
        private void StartRefreshTimerIfNeeded()
        {
            if (_refreshTimer != null && !_refreshTimer.IsEnabled)
            {
                _hasActiveDownloads = true;
                _refreshTimer.Start();
            }
        }

        /// <summary>
        /// Set the tray service for notifications
        /// </summary>
        public void SetTrayService(TrayService trayService)
        {
            _trayService = trayService;
        }

        #region Queue Loading

        /// <summary>
        /// Load the queue from the backend
        /// </summary>
        public async Task LoadQueueAsync()
        {
            // Prevent concurrent loads
            if (IsLoading)
            {
                LoggingService.Instance.LogInfo("LoadQueueAsync skipped - already loading");
                return;
            }
            
            // Log full stack trace to see what's calling this
            var stackTrace = new System.Diagnostics.StackTrace(true);
            var frames = new System.Text.StringBuilder();
            for (int i = 1; i < Math.Min(10, stackTrace.FrameCount); i++)
            {
                var frame = stackTrace.GetFrame(i);
                var method = frame?.GetMethod();
                if (method != null)
                {
                    frames.AppendLine($"  [{i}] {method.DeclaringType?.Name}.{method.Name}");
                }
            }
            LoggingService.Instance.LogInfo($"LoadQueueAsync called from:\n{frames}");
            
            IsLoading = true;
            _currentOffset = 0;
            CurrentPage = 1;

            try
            {
                LoggingService.Instance.LogInfo("Loading queue stats...");
                // Load queue stats
                var stats = await _service.GetQueueStatsAsync();
                if (stats != null)
                {
                    // Update individual properties instead of replacing the entire object
                    // This ensures WPF bindings detect the changes
                    QueueStats.Total = stats.Total;
                    QueueStats.Pending = stats.Pending;
                    QueueStats.Downloading = stats.Downloading;
                    QueueStats.Completed = stats.Completed;
                    QueueStats.Failed = stats.Failed;
                    
                    LoggingService.Instance.LogInfo($"Queue stats loaded: Total={stats.Total}, Pending={stats.Pending}, Downloading={stats.Downloading}, Completed={stats.Completed}, Failed={stats.Failed}");
                    LoggingService.Instance.LogInfo($"[DEBUG] After setting QueueStats: Stats.Total={Stats.Total}, QueueStats.Total={QueueStats.Total}");
                }
                else
                {
                    LoggingService.Instance.LogWarning("Queue stats returned null");
                }

                // Load first page of queue items
                LoggingService.Instance.LogInfo("Loading first page of queue items...");
                await LoadPageAsync(0);
                LoggingService.Instance.LogInfo($"Queue loaded successfully: {QueueItems.Count} items");
                
                // Start refresh timer if there are any items in the queue
                _hasActiveDownloads = (QueueStats?.Downloading ?? 0) > 0 || (QueueStats?.Pending ?? 0) > 0;
                if (_hasActiveDownloads || QueueItems.Count > 0)
                {
                    StartRefreshTimerIfNeeded();
                }
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Failed to load queue: {ex.Message}");
                LoggingService.Instance.LogError("Failed to load queue", ex);
            }
            finally
            {
                IsLoading = false;
            }
        }

        /// <summary>
        /// Load a specific page of queue items
        /// </summary>
        private async Task LoadPageAsync(int offset)
        {
            // Don't check IsLoading here - it's already set by the caller
            // if (IsLoading)
            //     return;

            try
            {
                var filter = StatusFilter == "all" ? null : StatusFilter;
                LoggingService.Instance.LogInfo($"Loading queue page: offset={offset}, pageSize={PageSize}, filter={filter}");
                
                // Get all items (we'll filter client-side to only show albums)
                var response = await _service.GetQueueAsync<QueueResponse>(
                    offset, PageSize * 10, filter); // Get more items since we're filtering

                if (response != null)
                {
                    LoggingService.Instance.LogInfo($"Queue response received: Total={response.Total}, Items={response.Items?.Count ?? 0}");
                    
                    // Add new items - albums and playlists (not individual tracks)
                    if (response.Items != null)
                    {
                        // Do filtering on background thread
                        var albumItems = response.Items
                            .Where(i => i.Type == "album" || i.Type == "playlist")
                            .OrderBy(i => i.CreatedAt)
                            .Take(PageSize)
                            .ToList();
                        
                        // Pre-compute sets on background thread
                        var newIds = new HashSet<string>(albumItems.Select(a => a.Id));
                        var newItemsDict = albumItems.ToDictionary(a => a.Id);
                        
                        // Minimize UI thread work with background priority
                        await System.Windows.Application.Current.Dispatcher.InvokeAsync(() =>
                        {
                            // Build lookup dictionary for fast O(1) access
                            var existingDict = new Dictionary<string, int>();
                            for (int i = 0; i < QueueItems.Count; i++)
                            {
                                existingDict[QueueItems[i].Id] = i;
                            }
                            
                            // Collect indices to remove
                            var indicesToRemove = new List<int>();
                            
                            // Update existing items and mark removals
                            foreach (var kvp in existingDict)
                            {
                                if (newItemsDict.TryGetValue(kvp.Key, out var newItem))
                                {
                                    var existing = QueueItems[kvp.Value];
                                    // Update only if changed
                                    bool changed = false;
                                    if (existing.TotalTracks != newItem.TotalTracks) { existing.TotalTracks = newItem.TotalTracks; changed = true; }
                                    if (existing.CompletedTracks != newItem.CompletedTracks) { existing.CompletedTracks = newItem.CompletedTracks; changed = true; }
                                    if (existing.Progress != newItem.Progress) { existing.Progress = newItem.Progress; changed = true; }
                                    if (existing.ErrorMessage != newItem.ErrorMessage) { existing.ErrorMessage = newItem.ErrorMessage; changed = true; }
                                    if (existing.Status != newItem.Status) { existing.Status = newItem.Status; changed = true; }
                                    if (changed) existing.UpdateComputedBackgroundColor();
                                }
                                else
                                {
                                    indicesToRemove.Add(kvp.Value);
                                }
                            }
                            
                            // Remove in reverse order
                            indicesToRemove.Sort();
                            for (int i = indicesToRemove.Count - 1; i >= 0; i--)
                            {
                                QueueItems.RemoveAt(indicesToRemove[i]);
                            }
                            
                            // Add new items
                            foreach (var newItem in albumItems)
                            {
                                if (!existingDict.ContainsKey(newItem.Id))
                                {
                                    newItem.UpdateComputedBackgroundColor();
                                    QueueItems.Add(newItem);
                                }
                            }
                        }, System.Windows.Threading.DispatcherPriority.Background);
                    }

                    // Update pagination info
                    TotalItems = response.Total;
                    _currentOffset = offset;
                    HasMoreItems = offset + response.Items?.Count < response.Total;
                    
                    OnPropertyChanged(nameof(IsQueueEmpty));
                    OnPropertyChanged(nameof(TotalTracksInQueue));
                    OnPropertyChanged(nameof(CompletedTracksInQueue));
                }
                else
                {
                    LoggingService.Instance.LogWarning("Queue response is null");
                }
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Failed to load page: {ex.Message}");
                LoggingService.Instance.LogError($"Failed to load queue page at offset {offset}", ex);
            }
            // Don't set IsLoading = false here - it's managed by LoadQueueAsync
        }

        /// <summary>
        /// Load more queue items (infinite scroll - deprecated in favor of pagination)
        /// </summary>
        private async Task LoadMoreItemsAsync()
        {
            if (IsLoading || !HasMoreItems)
                return;

            IsLoading = true;

            try
            {
                // Check if we're approaching the memory limit
                if (QueueItems.Count >= MaxInMemoryItems)
                {
                    // Don't load more items, use pagination instead
                    System.Diagnostics.Debug.WriteLine($"Memory limit reached ({MaxInMemoryItems} items). Use pagination to view more items.");
                    HasMoreItems = false;
                    return;
                }

                var filter = StatusFilter == "all" ? null : StatusFilter;
                var response = await _service.GetQueueAsync<QueueResponse>(
                    _currentOffset, PageSize, filter);

                if (response?.Items != null && response.Items.Count > 0)
                {
                    foreach (var item in response.Items)
                    {
                        QueueItems.Add(item);
                    }

                    _currentOffset += response.Items.Count;
                    TotalItems = response.Total;
                    HasMoreItems = _currentOffset < response.Total && QueueItems.Count < MaxInMemoryItems;
                }
                else
                {
                    HasMoreItems = false;
                }
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Failed to load more items: {ex.Message}");
                HasMoreItems = false;
            }
            finally
            {
                IsLoading = false;
            }
        }

        /// <summary>
        /// Load the next page of queue items
        /// </summary>
        private async Task LoadNextPageAsync()
        {
            if (!CanLoadNext || IsLoading)
                return;

            CurrentPage++;
            var offset = (CurrentPage - 1) * PageSize;
            await LoadPageAsync(offset);
        }

        /// <summary>
        /// Load the previous page of queue items
        /// </summary>
        private async Task LoadPreviousPageAsync()
        {
            if (!CanLoadPrevious || IsLoading)
                return;

            CurrentPage--;
            var offset = (CurrentPage - 1) * PageSize;
            await LoadPageAsync(offset);
        }

        #endregion

        #region Queue Operations

        /// <summary>
        /// Pause a download
        /// </summary>
        private async Task PauseDownloadAsync(QueueItem? item)
        {
            if (item == null || !item.CanPause)
                return;

            try
            {
                await _service.PauseDownloadAsync(item.Id);
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Failed to pause download: {ex.Message}");
            }
        }

        /// <summary>
        /// Resume a download
        /// </summary>
        private async Task ResumeDownloadAsync(QueueItem? item)
        {
            if (item == null || !item.CanResume)
                return;

            try
            {
                await _service.ResumeDownloadAsync(item.Id);
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Failed to resume download: {ex.Message}");
            }
        }

        /// <summary>
        /// Cancel a download
        /// </summary>
        private async Task CancelDownloadAsync(QueueItem? item)
        {
            if (item == null)
                return;

            try
            {
                await _service.CancelDownloadAsync(item.Id);
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Failed to cancel download: {ex.Message}");
            }
        }

        /// <summary>
        /// Retry a failed download
        /// </summary>
        private async Task RetryDownloadAsync(QueueItem? item)
        {
            if (item == null || !item.CanRetry)
                return;

            try
            {
                await _service.RetryDownloadAsync(item.Id);
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Failed to retry download: {ex.Message}");
            }
        }

        /// <summary>
        /// Clear all completed downloads
        /// </summary>
        private async Task ClearCompletedAsync()
        {
            try
            {
                // Remove completed items from UI immediately to prevent visual glitches
                await System.Windows.Application.Current.Dispatcher.InvokeAsync(() =>
                {
                    var completedItems = QueueItems.Where(i => i.Status == "completed").ToList();
                    foreach (var item in completedItems)
                    {
                        QueueItems.Remove(item);
                    }
                });
                
                // Then clear from backend
                await _service.ClearCompletedAsync();
                
                // Reload the queue to ensure consistency
                await LoadQueueAsync();
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Failed to clear completed: {ex.Message}");
            }
        }

        /// <summary>
        /// Pause all active downloads
        /// </summary>
        private async Task PauseAllDownloadsAsync()
        {
            try
            {
                var activeItems = QueueItems.Where(i => i.CanPause).ToList();
                foreach (var item in activeItems)
                {
                    await _service.PauseDownloadAsync(item.Id);
                }
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Failed to pause all downloads: {ex.Message}");
            }
        }

        /// <summary>
        /// Resume all paused downloads
        /// </summary>
        private async Task ResumeAllDownloadsAsync()
        {
            try
            {
                var pausedItems = QueueItems.Where(i => i.CanResume).ToList();
                foreach (var item in pausedItems)
                {
                    await _service.ResumeDownloadAsync(item.Id);
                }
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Failed to resume all downloads: {ex.Message}");
            }
        }
        
        /// <summary>
        /// Clear all items from the queue
        /// </summary>
        private async Task ClearAllAsync()
        {
            try
            {
                LoggingService.Instance.LogInfo("Stopping all downloads and clearing queue");
                
                // Stop all active downloads and clear the entire queue
                int result = Services.GoBackend.StopAllDownloads();
                if (result != 0)
                {
                    LoggingService.Instance.LogError($"Failed to stop all downloads: error code {result}");
                }
                
                // Reload queue to reflect changes
                await LoadQueueAsync();
                
                LoggingService.Instance.LogInfo("All downloads stopped and queue cleared");
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to clear all queue items", ex);
                System.Diagnostics.Debug.WriteLine($"Failed to clear all: {ex.Message}");
            }
        }
        
        /// <summary>
        /// Retry all failed downloads
        /// </summary>
        private async Task RetryAllFailedAsync()
        {
            try
            {
                LoggingService.Instance.LogInfo("Retrying all failed downloads");
                
                var failedItems = QueueItems.Where(i => i.CanRetry).ToList();
                
                foreach (var item in failedItems)
                {
                    await _service.RetryDownloadAsync(item.Id);
                }
                
                LoggingService.Instance.LogInfo($"Retried {failedItems.Count} failed downloads");
                
                // Reload queue to show updated statuses
                await Task.Delay(500);
                await LoadQueueAsync();
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to retry failed downloads", ex);
                System.Diagnostics.Debug.WriteLine($"Failed to retry all failed: {ex.Message}");
            }
        }

        /// <summary>
        /// Get failed tracks for an album/playlist
        /// </summary>
        public async Task<List<FailedTrack>?> GetFailedTracksAsync(string parentId)
        {
            try
            {
                return await _service.GetFailedTracksAsync(parentId);
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Failed to get failed tracks for {parentId}", ex);
                return null;
            }
        }

        #endregion

        #region Event Handlers

        /// <summary>
        /// Handle progress updates from the backend
        /// </summary>
        private void OnProgressUpdated(object? sender, ProgressUpdateEventArgs e)
        {
            // Throttle progress updates to prevent UI flooding
            // Only process if it's been at least 200ms since last update for this item
            var now = DateTime.Now;
            if (_lastProgressUpdate.TryGetValue(e.ItemID, out var lastUpdate))
            {
                if ((now - lastUpdate).TotalMilliseconds < 200)
                {
                    return; // Skip this update, too soon
                }
            }
            _lastProgressUpdate[e.ItemID] = now;
            
            // Ensure UI updates happen on the UI thread
            System.Windows.Application.Current?.Dispatcher.InvokeAsync(() =>
            {
                // Mark that we have active downloads and start timer
                _hasActiveDownloads = true;
                StartRefreshTimerIfNeeded();
                
                // Find the queue item and update its progress
                var item = QueueItems.FirstOrDefault(i => i.Id == e.ItemID);
                if (item == null)
                {
                    // Item not found - this can happen if the queue was reloaded
                    // The refresh timer will pick up the update
                    return;
                }

                // Only update progress if it increased - backend sometimes sends stale/incorrect values
                if (e.Progress > item.Progress)
                {
                    item.Progress = e.Progress;
                }
                
                // For albums/playlists, BytesProcessed and TotalBytes represent track counts
                if (item.Type == "album" || item.Type == "playlist")
                {
                    var oldCompleted = item.CompletedTracks;
                    var oldTotal = item.TotalTracks;
                    
                    item.CompletedTracks = (int)e.BytesProcessed;
                    item.TotalTracks = (int)e.TotalBytes;
                    
                    // Only notify if counts actually changed
                    if (oldCompleted != item.CompletedTracks || oldTotal != item.TotalTracks)
                    {
                        // Notify total track counts changed
                        OnPropertyChanged(nameof(TotalTracksInQueue));
                        OnPropertyChanged(nameof(CompletedTracksInQueue));
                    }
                }
                else
                {
                    // For individual tracks, these are actual bytes
                    item.BytesDownloaded = e.BytesProcessed;
                    item.TotalBytes = e.TotalBytes;
                }
                
                // Calculate download speed
                if (e.TotalBytes > 0 && e.BytesProcessed > 0)
                {
                    // Speed calculation would be done by the backend
                    // For now, just update the display
                    item.DownloadSpeed = FormatSpeed(e.BytesProcessed, e.TotalBytes);
                }
            });
        }

        /// <summary>
        /// Handle status changes from the backend
        /// </summary>
        private void OnStatusChanged(object? sender, StatusUpdateEventArgs e)
        {
            // Ensure UI updates happen on the UI thread with lower priority to not block user interactions
            System.Windows.Application.Current?.Dispatcher.InvokeAsync(async () =>
            {
                // If status is "started" or "downloading", ensure timer is running
                if (e.Status == "started" || e.Status == "downloading")
                {
                    _hasActiveDownloads = true;
                    StartRefreshTimerIfNeeded();
                }
                
                // Find the queue item and update its status
                var item = QueueItems.FirstOrDefault(i => i.Id == e.ItemID);
                if (item == null)
                {
                    // Item not found - the refresh timer will pick up the update
                    return;
                }

                var previousStatus = item.Status;
                
                // Skip if status hasn't actually changed
                if (previousStatus == e.Status)
                {
                    return;
                }
                
                // For album completion, fetch fresh data and replace item
                if (e.Status == "completed" && previousStatus != "completed" && item.IsAlbumOrPlaylist)
                {
                    try
                    {
                        var response = await _service.GetQueueAsync<QueueResponse>(0, 1000, null);
                        var freshItem = response?.Items?.FirstOrDefault(i => i.Id == e.ItemID);
                        
                        if (freshItem != null)
                        {
                            freshItem.Status = "completed";
                            freshItem.CompletedAt = DateTime.Now;
                            freshItem.UpdateComputedBackgroundColor();
                            
                            var index = QueueItems.IndexOf(item);
                            if (index >= 0)
                            {
                                QueueItems.RemoveAt(index);
                                QueueItems.Insert(index, freshItem);
                                
                                // Fire PropertyChanged for DataTrigger properties
                                freshItem.OnPropertyChanged(nameof(freshItem.IsPartialSuccess));
                                freshItem.OnPropertyChanged(nameof(freshItem.IsCompleted));
                                freshItem.OnPropertyChanged(nameof(freshItem.IsFailed));
                            }
                            
                            // Show notification
                            bool isPartial = freshItem.IsPartialSuccess;
                            var notificationTitle = isPartial 
                                ? $"{freshItem.Title} (Partial - {freshItem.CompletedTracks}/{freshItem.TotalTracks} tracks)"
                                : freshItem.Title ?? "Track";
                            _trayService?.ShowDownloadCompleted(notificationTitle);
                            
                            OnPropertyChanged(nameof(TotalTracksInQueue));
                            OnPropertyChanged(nameof(CompletedTracksInQueue));
                            return;
                        }
                    }
                    catch (Exception ex)
                    {
                        LoggingService.Instance.LogWarning($"Failed to fetch fresh data for completed album: {ex.Message}");
                    }
                }
                
                // Fallback: update status on existing item
                item.Status = e.Status;
                
                if (!string.IsNullOrEmpty(e.ErrorMessage))
                {
                    item.ErrorMessage = e.ErrorMessage;
                }

                // If status changed to completed, update completion time and show notification
                if (e.Status == "completed" && previousStatus != "completed")
                {
                    item.CompletedAt = DateTime.Now;
                    
                    // Check IsPartialSuccess AFTER track counts and status are both set
                    bool isPartial = item.IsAlbumOrPlaylist && item.CompletedTracks < item.TotalTracks;
                    
                    // Only set progress to 100% for single tracks or fully completed albums
                    if (!item.IsAlbumOrPlaylist || !isPartial)
                    {
                        item.Progress = 100;
                    }
                    
                    LoggingService.Instance.LogInfo($"Item completed: {item.Title}, CompletedTracks={item.CompletedTracks}/{item.TotalTracks}, IsPartialSuccess={isPartial}, Progress={item.Progress}%");
                    
                    // Force background color update
                    item.UpdateComputedBackgroundColor();
                    
                    // Notify stats changed
                    OnPropertyChanged(nameof(TotalTracksInQueue));
                    OnPropertyChanged(nameof(CompletedTracksInQueue));
                    
                    // Show tray notification for download completion
                    var notificationTitle = isPartial 
                        ? $"{item.Title} (Partial - {item.CompletedTracks}/{item.TotalTracks} tracks)"
                        : item.Title ?? "Track";
                    _trayService?.ShowDownloadCompleted(notificationTitle);
                }
                
                // If status changed to failed, show error notification
                if (e.Status == "failed" && previousStatus != "failed")
                {
                    _trayService?.ShowDownloadError(item.Title ?? "Track", e.ErrorMessage ?? "Unknown error");
                }
            });
        }

        /// <summary>
        /// Handle queue statistics updates from the backend
        /// </summary>
        private void OnQueueStatsUpdated(object? sender, QueueStatsEventArgs e)
        {
            if (e.Stats != null)
            {
                // Update individual properties instead of replacing the entire object
                QueueStats.Total = e.Stats.Total;
                QueueStats.Pending = e.Stats.Pending;
                QueueStats.Downloading = e.Stats.Downloading;
                QueueStats.Completed = e.Stats.Completed;
                QueueStats.Failed = e.Stats.Failed;
            }
        }

        /// <summary>
        /// Format download speed for display
        /// </summary>
        private string FormatSpeed(long bytesProcessed, long totalBytes)
        {
            // Simple placeholder - actual speed calculation would be done by backend
            if (totalBytes > 0)
            {
                var percentage = (double)bytesProcessed / totalBytes * 100;
                return $"{percentage:F1}%";
            }
            return "0%";
        }

        #endregion

        #region IDisposable

        private bool _disposed;

        public void Dispose()
        {
            if (_disposed)
                return;

            // Stop and dispose refresh timer
            if (_refreshTimer != null)
            {
                _refreshTimer.Stop();
                _refreshTimer.Tick -= OnRefreshTimerTick;
                _refreshTimer = null;
            }

            // Unsubscribe from events
            _service.ProgressUpdated -= OnProgressUpdated;
            _service.StatusChanged -= OnStatusChanged;
            _service.QueueStatsUpdated -= OnQueueStatsUpdated;

            _disposed = true;
        }

        #endregion

        protected virtual void OnPropertyChanged([CallerMemberName] string? propertyName = null)
        {
            PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
        }
    }

    /// <summary>
    /// Response wrapper for queue items with pagination metadata
    /// </summary>
    public class QueueResponse
    {
        public List<QueueItem> Items { get; set; } = new();
        public int Total { get; set; }
        public int Offset { get; set; }
        public int Limit { get; set; }
    }
}
