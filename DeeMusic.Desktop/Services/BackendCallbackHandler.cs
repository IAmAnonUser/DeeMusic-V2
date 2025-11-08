using System;
using System.Text.Json;
using System.Windows;
using System.Windows.Threading;
using Microsoft.Extensions.Logging;
using DeeMusic.Desktop.Models;

namespace DeeMusic.Desktop.Services
{
    /// <summary>
    /// Handles callbacks from the Go backend and marshals them to the UI thread
    /// </summary>
    public class BackendCallbackHandler : IDisposable
    {
        private readonly Dispatcher _dispatcher;
        private readonly ILogger<BackendCallbackHandler>? _logger;
        
        // Keep references to delegates to prevent garbage collection
        private readonly ProgressCallback _progressCallback;
        private readonly StatusCallback _statusCallback;
        private readonly QueueUpdateCallback _queueUpdateCallback;

        #region Events

        /// <summary>
        /// Raised when download progress is updated
        /// </summary>
        public event EventHandler<ProgressUpdateEventArgs>? ProgressUpdated;

        /// <summary>
        /// Raised when download status changes
        /// </summary>
        public event EventHandler<StatusUpdateEventArgs>? StatusChanged;

        /// <summary>
        /// Raised when queue statistics are updated
        /// </summary>
        public event EventHandler<QueueStatsEventArgs>? QueueStatsUpdated;

        #endregion

        public BackendCallbackHandler(ILogger<BackendCallbackHandler>? logger = null)
        {
            _dispatcher = Application.Current?.Dispatcher ?? Dispatcher.CurrentDispatcher;
            _logger = logger;

            // Create delegate instances and keep references
            _progressCallback = OnProgressCallback;
            _statusCallback = OnStatusCallback;
            _queueUpdateCallback = OnQueueUpdateCallback;

            // Register callbacks with Go backend
            RegisterCallbacks();
            
            // Log initialization
            LoggingService.Instance.LogInfo("Backend callback handler initialized");
        }

        /// <summary>
        /// Register all callbacks with the Go backend
        /// </summary>
        private void RegisterCallbacks()
        {
            try
            {
                GoBackend.SetProgressCallback(_progressCallback);
                GoBackend.SetStatusCallback(_statusCallback);
                GoBackend.SetQueueUpdateCallback(_queueUpdateCallback);
                
                _logger?.LogInformation("Backend callbacks registered successfully");
                LoggingService.Instance.LogInfo("Backend callbacks registered successfully");
            }
            catch (Exception ex)
            {
                _logger?.LogError(ex, "Failed to register backend callbacks");
                LoggingService.Instance.LogError("Failed to register backend callbacks", ex);
                throw;
            }
        }

        #region Callback Handlers

        /// <summary>
        /// Called by Go backend when download progress is updated
        /// </summary>
        private void OnProgressCallback(string itemID, int progress, long bytesProcessed, long totalBytes)
        {
            try
            {
                // Marshal to UI thread
                _dispatcher.BeginInvoke(() =>
                {
                    try
                    {
                        var args = new ProgressUpdateEventArgs
                        {
                            ItemID = itemID,
                            Progress = progress,
                            BytesProcessed = bytesProcessed,
                            TotalBytes = totalBytes,
                            Speed = CalculateSpeed(bytesProcessed, totalBytes)
                        };

                        ProgressUpdated?.Invoke(this, args);
                    }
                    catch (Exception ex)
                    {
                        _logger?.LogError(ex, "Error handling progress update for item {ItemID}", itemID);
                        LoggingService.Instance.LogError($"Error handling progress update for item {itemID}", ex);
                    }
                });
            }
            catch (Exception ex)
            {
                _logger?.LogError(ex, "Error marshaling progress callback to UI thread");
                LoggingService.Instance.LogError("Error marshaling progress callback to UI thread", ex);
            }
        }

        /// <summary>
        /// Called by Go backend when download status changes
        /// </summary>
        private void OnStatusCallback(string itemID, string status, string? errorMsg)
        {
            try
            {
                // Log status changes
                if (!string.IsNullOrEmpty(errorMsg))
                {
                    LoggingService.Instance.LogWarning($"Download status changed for {itemID}: {status} - {errorMsg}");
                }
                
                // Marshal to UI thread
                _dispatcher.BeginInvoke(() =>
                {
                    try
                    {
                        var args = new StatusUpdateEventArgs
                        {
                            ItemID = itemID,
                            Status = status,
                            ErrorMessage = errorMsg
                        };

                        StatusChanged?.Invoke(this, args);
                    }
                    catch (Exception ex)
                    {
                        _logger?.LogError(ex, "Error handling status update for item {ItemID}", itemID);
                        LoggingService.Instance.LogError($"Error handling status update for item {itemID}", ex);
                    }
                });
            }
            catch (Exception ex)
            {
                _logger?.LogError(ex, "Error marshaling status callback to UI thread");
                LoggingService.Instance.LogError("Error marshaling status callback to UI thread", ex);
            }
        }

        /// <summary>
        /// Called by Go backend when queue statistics are updated
        /// </summary>
        private void OnQueueUpdateCallback(string statsJson)
        {
            try
            {
                // Marshal to UI thread
                _dispatcher.BeginInvoke(() =>
                {
                    try
                    {
                        // Deserialize queue stats
                        var stats = JsonSerializer.Deserialize<QueueStats>(statsJson, new JsonSerializerOptions
                        {
                            PropertyNameCaseInsensitive = true
                        });

                        if (stats != null)
                        {
                            var args = new QueueStatsEventArgs
                            {
                                Stats = stats
                            };

                            QueueStatsUpdated?.Invoke(this, args);
                        }
                        else
                        {
                            LoggingService.Instance.LogWarning("Queue stats deserialization returned null");
                        }
                    }
                    catch (Exception ex)
                    {
                        _logger?.LogError(ex, "Error handling queue stats update");
                        LoggingService.Instance.LogError("Error handling queue stats update", ex);
                    }
                });
            }
            catch (Exception ex)
            {
                _logger?.LogError(ex, "Error marshaling queue update callback to UI thread");
                LoggingService.Instance.LogError("Error marshaling queue update callback to UI thread", ex);
            }
        }

        #endregion

        #region Helper Methods

        /// <summary>
        /// Calculate download speed in a human-readable format
        /// </summary>
        private string CalculateSpeed(long bytesProcessed, long totalBytes)
        {
            // This is a simplified calculation
            // In a real implementation, you'd track time and calculate actual speed
            if (totalBytes == 0)
                return "0 B/s";

            // For now, return empty string - actual speed calculation would require timing
            return string.Empty;
        }

        #endregion

        #region IDisposable

        private bool _disposed;

        public void Dispose()
        {
            if (_disposed)
                return;

            try
            {
                // Unregister callbacks by setting them to null
                GoBackend.SetProgressCallback(null!);
                GoBackend.SetStatusCallback(null!);
                GoBackend.SetQueueUpdateCallback(null!);
                
                _logger?.LogInformation("Backend callbacks unregistered");
                LoggingService.Instance.LogInfo("Backend callbacks unregistered");
            }
            catch (Exception ex)
            {
                _logger?.LogError(ex, "Error unregistering callbacks");
                LoggingService.Instance.LogError("Error unregistering callbacks", ex);
            }

            _disposed = true;
        }

        #endregion
    }

    #region Event Args Classes

    /// <summary>
    /// Event arguments for progress updates
    /// </summary>
    public class ProgressUpdateEventArgs : EventArgs
    {
        public required string ItemID { get; init; }
        public int Progress { get; init; }
        public long BytesProcessed { get; init; }
        public long TotalBytes { get; init; }
        public string Speed { get; init; } = string.Empty;
    }

    /// <summary>
    /// Event arguments for status updates
    /// </summary>
    public class StatusUpdateEventArgs : EventArgs
    {
        public required string ItemID { get; init; }
        public required string Status { get; init; }
        public string? ErrorMessage { get; init; }
    }

    /// <summary>
    /// Event arguments for queue statistics updates
    /// </summary>
    public class QueueStatsEventArgs : EventArgs
    {
        public required QueueStats Stats { get; init; }
    }

    #endregion
}
