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
                    QueueStats = stats;
                    OnPropertyChanged(nameof(QueueStats));
                    OnPropertyChanged(nameof(Stats));
                    LoggingService.Instance.LogInfo($"Queue stats loaded: Total={stats.Total}, Pending={stats.Pending}, Downloading={stats.Downloading}, Completed={stats.Completed}, Failed={stats.Failed}");
                }
                else
                {
                    LoggingService.Instance.LogWarning("Queue stats returned null");
                }

                // Load first page of queue items
                LoggingService.Instance.LogInfo("Loading first page of queue items...");
                await LoadPageAsync(0);
                LoggingService.Instance.LogInfo($"Queue loaded successfully: {QueueItems.Count} items");
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
                        var albumItems = response.Items.Where(i => i.Type == "album" || i.Type == "playlist").OrderBy(i => i.CreatedAt).Take(PageSize).ToList();
                        
                        // Ensure collection modifications happen on UI thread
                        await System.Windows.Application.Current.Dispatcher.InvokeAsync(() =>
                        {
                            // Build lookup dictionary for fast access
                            var existingDict = QueueItems.ToDictionary(q => q.Id);
                            var newIds = new HashSet<string>(albumItems.Select(a => a.Id));
                            
                            // Remove items that no longer exist
                            for (int i = QueueItems.Count - 1; i >= 0; i--)
                            {
                                if (!newIds.Contains(QueueItems[i].Id))
                                {
                                    QueueItems.RemoveAt(i);
                                }
                            }
                            
                            // Update existing items and add new ones
                            foreach (var newItem in albumItems)
                            {
                                if (existingDict.TryGetValue(newItem.Id, out var existing))
                                {
                                    // Update existing item properties
                                    existing.Status = newItem.Status;
                                    
                                    // Only update progress if it increased (never decrease)
                                    if (newItem.Progress > existing.Progress)
                                    {
                                        LoggingService.Instance.LogInfo($"Progress UPDATE: {newItem.Title} {existing.Progress}% -> {newItem.Progress}%");
                                        existing.Progress = newItem.Progress;
                                    }
                                    else if (newItem.Progress < existing.Progress)
                                    {
                                        LoggingService.Instance.LogWarning($"Progress BLOCKED: {newItem.Title} would decrease from {existing.Progress}% to {newItem.Progress}%");
                                    }
                                    
                                    existing.TotalTracks = newItem.TotalTracks;
                                    existing.CompletedTracks = newItem.CompletedTracks;
                                    existing.ErrorMessage = newItem.ErrorMessage;
                                }
                                else
                                {
                                    // Add new item at the end (oldest first)
                                    QueueItems.Add(newItem);
                                    LoggingService.Instance.LogInfo($"Added queue item: ID={newItem.Id}, Title={newItem.Title}, TotalTracks={newItem.TotalTracks}");
                                }
                            }
                        });
                        
                        LoggingService.Instance.LogInfo($"Filtered to {albumItems.Count} albums from {response.Items.Count} total items");
                    }

                    // Update pagination info
                    TotalItems = response.Total;
                    _currentOffset = offset;
                    HasMoreItems = offset + response.Items?.Count < response.Total;
                    
                    OnPropertyChanged(nameof(IsQueueEmpty));
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
                await _service.ClearCompletedAsync();
                
                // Remove completed items from the collection
                var completedItems = QueueItems.Where(i => i.IsCompleted).ToList();
                foreach (var item in completedItems)
                {
                    QueueItems.Remove(item);
                }
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

        #endregion

        #region Event Handlers

        /// <summary>
        /// Handle progress updates from the backend
        /// </summary>
        private void OnProgressUpdated(object? sender, ProgressUpdateEventArgs e)
        {
            // Find the queue item and update its progress
            var item = QueueItems.FirstOrDefault(i => i.Id == e.ItemID);
            if (item != null)
            {
                // Only update progress if it increased - backend sometimes sends stale/incorrect values
                if (e.Progress > item.Progress)
                {
                    item.Progress = e.Progress;
                }
                
                item.BytesDownloaded = e.BytesProcessed;
                item.TotalBytes = e.TotalBytes;
                
                // Calculate download speed
                if (e.TotalBytes > 0 && e.BytesProcessed > 0)
                {
                    // Speed calculation would be done by the backend
                    // For now, just update the display
                    item.DownloadSpeed = FormatSpeed(e.BytesProcessed, e.TotalBytes);
                }
            }
        }

        /// <summary>
        /// Handle status changes from the backend
        /// </summary>
        private void OnStatusChanged(object? sender, StatusUpdateEventArgs e)
        {
            // Find the queue item and update its status
            var item = QueueItems.FirstOrDefault(i => i.Id == e.ItemID);
            if (item != null)
            {
                var previousStatus = item.Status;
                item.Status = e.Status;
                
                if (!string.IsNullOrEmpty(e.ErrorMessage))
                {
                    item.ErrorMessage = e.ErrorMessage;
                }

                // If status changed to completed, update completion time and show notification
                if (e.Status == "completed" && previousStatus != "completed")
                {
                    item.CompletedAt = DateTime.Now;
                    item.Progress = 100; // Ensure progress is 100%
                    
                    LoggingService.Instance.LogInfo($"Item completed: {item.Title}, Status={item.Status}, IsCompleted={item.IsCompleted}, Progress={item.Progress}%");
                    
                    // Show tray notification for download completion
                    _trayService?.ShowDownloadCompleted(item.Title ?? "Track");
                }
                
                // If status changed to failed, show error notification
                if (e.Status == "failed" && previousStatus != "failed")
                {
                    _trayService?.ShowDownloadError(item.Title ?? "Track", e.ErrorMessage ?? "Unknown error");
                }
            }
        }

        /// <summary>
        /// Handle queue statistics updates from the backend
        /// </summary>
        private void OnQueueStatsUpdated(object? sender, QueueStatsEventArgs e)
        {
            if (e.Stats != null)
            {
                QueueStats = e.Stats;
                OnPropertyChanged(nameof(QueueStats));
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
