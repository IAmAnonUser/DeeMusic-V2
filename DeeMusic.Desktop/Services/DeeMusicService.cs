using System;
using System.Collections.Generic;
using System.Linq;
using System.Text.Json;
using System.Threading.Tasks;
using Microsoft.Extensions.Logging;
using DeeMusic.Desktop.Models;

namespace DeeMusic.Desktop.Services
{
    /// <summary>
    /// High-level service wrapper for the Go backend
    /// Provides async methods, JSON deserialization, error handling, and retry logic
    /// </summary>
    public class DeeMusicService : IDisposable
    {
        private readonly ILogger<DeeMusicService>? _logger;
        private readonly BackendCallbackHandler _callbackHandler;
        private bool _initialized;
        private readonly int _maxRetries = 3;
        private readonly int _retryDelayMs = 1000;

        #region Events (forwarded from callback handler)

        public event EventHandler<ProgressUpdateEventArgs>? ProgressUpdated
        {
            add => _callbackHandler.ProgressUpdated += value;
            remove => _callbackHandler.ProgressUpdated -= value;
        }

        public event EventHandler<StatusUpdateEventArgs>? StatusChanged
        {
            add => _callbackHandler.StatusChanged += value;
            remove => _callbackHandler.StatusChanged -= value;
        }

        public event EventHandler<QueueStatsEventArgs>? QueueStatsUpdated
        {
            add => _callbackHandler.QueueStatsUpdated += value;
            remove => _callbackHandler.QueueStatsUpdated -= value;
        }

        #endregion

        public DeeMusicService(ILogger<DeeMusicService>? logger = null)
        {
            _logger = logger;
            _callbackHandler = new BackendCallbackHandler(
                logger as ILogger<BackendCallbackHandler>);
            
            LoggingService.Instance.LogInfo("DeeMusicService created");
        }

        #region Initialization

        /// <summary>
        /// Initialize the backend with configuration file
        /// </summary>
        public async Task<bool> InitializeAsync(string configPath)
        {
            LoggingService.Instance.LogInfo($"Initializing backend with config: {configPath}");
            
            return await ExecuteWithRetryAsync(async () =>
            {
                await Task.Run(() =>
                {
                    var result = GoBackend.InitializeApp(configPath);
                    if (result != 0)
                    {
                        var errorMsg = GoBackend.GetDetailedErrorMessage(result, "Backend initialization");
                        _logger?.LogError("Failed to initialize backend: {Error}", errorMsg);
                        LoggingService.Instance.LogError(errorMsg);
                        throw new BackendException($"Initialization failed: {errorMsg}", result);
                    }

                    _initialized = true;
                    _logger?.LogInformation("Backend initialized successfully");
                    LoggingService.Instance.LogInfo("Backend initialized successfully");
                });

                return true;
            });
        }

        /// <summary>
        /// Shutdown the backend
        /// </summary>
        public void Shutdown()
        {
            if (!_initialized)
                return;

            try
            {
                LoggingService.Instance.LogInfo("Shutting down backend");
                GoBackend.ShutdownApp();
                _initialized = false;
                _logger?.LogInformation("Backend shutdown successfully");
                LoggingService.Instance.LogInfo("Backend shutdown successfully");
            }
            catch (Exception ex)
            {
                _logger?.LogError(ex, "Error during backend shutdown");
                LoggingService.Instance.LogError("Error during backend shutdown", ex);
            }
        }

        #endregion

        #region Search and Browse

        /// <summary>
        /// Search for tracks, albums, artists, or playlists
        /// </summary>
        public async Task<T?> SearchAsync<T>(string query, string searchType, int limit = 50)
        {
            LoggingService.Instance.LogInfo($"SearchAsync called: query='{query}', type='{searchType}', limit={limit}");
            EnsureInitialized();

            return await ExecuteWithRetryAsync(async () =>
            {
                return await Task.Run(() =>
                {
                    LoggingService.Instance.LogInfo("Calling GoBackend.Search...");
                    var ptr = GoBackend.Search(query, searchType, limit);
                    LoggingService.Instance.LogInfo($"GoBackend.Search returned pointer: {ptr}");
                    
                    var json = GoBackend.PtrToStringAndFree(ptr);
                    LoggingService.Instance.LogInfo($"Search JSON (length: {json?.Length ?? 0}): {json?.Substring(0, Math.Min(500, json?.Length ?? 0))}");

                    if (string.IsNullOrEmpty(json))
                    {
                        _logger?.LogWarning("Search returned empty result");
                        LoggingService.Instance.LogWarning("Search returned empty result");
                        return default;
                    }

                    var result = DeserializeJson<T>(json);
                    LoggingService.Instance.LogInfo($"Search deserialized successfully, type: {result?.GetType().Name}");
                    return result;
                });
            });
        }

        /// <summary>
        /// Get album details
        /// </summary>
        public async Task<T?> GetAlbumAsync<T>(string albumID)
        {
            EnsureInitialized();

            return await ExecuteWithRetryAsync(async () =>
            {
                return await Task.Run(() =>
                {
                    var ptr = GoBackend.GetAlbum(albumID);
                    var json = GoBackend.PtrToStringAndFree(ptr);

                    if (string.IsNullOrEmpty(json))
                        return default;

                    return DeserializeJson<T>(json);
                });
            });
        }

        /// <summary>
        /// Get artist details
        /// </summary>
        public async Task<T?> GetArtistAsync<T>(string artistID)
        {
            EnsureInitialized();

            return await ExecuteWithRetryAsync(async () =>
            {
                return await Task.Run(() =>
                {
                    var ptr = GoBackend.GetArtist(artistID);
                    var json = GoBackend.PtrToStringAndFree(ptr);

                    if (string.IsNullOrEmpty(json))
                        return default;

                    return DeserializeJson<T>(json);
                });
            });
        }

        /// <summary>
        /// Get playlist details
        /// </summary>
        public async Task<T?> GetPlaylistAsync<T>(string playlistID)
        {
            EnsureInitialized();

            return await ExecuteWithRetryAsync(async () =>
            {
                return await Task.Run(() =>
                {
                    var ptr = GoBackend.GetPlaylist(playlistID);
                    var json = GoBackend.PtrToStringAndFree(ptr);

                    if (string.IsNullOrEmpty(json))
                        return default;

                    return DeserializeJson<T>(json);
                });
            });
        }

        /// <summary>
        /// Get artist albums
        /// </summary>
        public async Task<T?> GetArtistAlbumsAsync<T>(string artistID, int limit = 100)
        {
            EnsureInitialized();

            return await ExecuteWithRetryAsync(async () =>
            {
                return await Task.Run(() =>
                {
                    var ptr = GoBackend.GetArtistAlbums(artistID, limit);
                    var json = GoBackend.PtrToStringAndFree(ptr);

                    if (string.IsNullOrEmpty(json))
                        return default;

                    return DeserializeJson<T>(json);
                });
            });
        }

        /// <summary>
        /// Get Deezer charts
        /// </summary>
        public async Task<T?> GetChartsAsync<T>(int limit = 25)
        {
            EnsureInitialized();

            return await ExecuteWithRetryAsync(async () =>
            {
                return await Task.Run(() =>
                {
                    var ptr = GoBackend.GetCharts(limit);
                    var json = GoBackend.PtrToStringAndFree(ptr);

                    if (string.IsNullOrEmpty(json))
                        return default;

                    return DeserializeJson<T>(json);
                });
            });
        }

        /// <summary>
        /// Get editorial releases (new releases)
        /// </summary>
        public async Task<T?> GetEditorialReleasesAsync<T>(int limit = 25)
        {
            EnsureInitialized();

            return await ExecuteWithRetryAsync(async () =>
            {
                return await Task.Run(() =>
                {
                    var ptr = GoBackend.GetEditorialReleases(limit);
                    var json = GoBackend.PtrToStringAndFree(ptr);

                    if (string.IsNullOrEmpty(json))
                        return default;

                    return DeserializeJson<T>(json);
                });
            });
        }

        #endregion

        #region Download Operations

        /// <summary>
        /// Download a track
        /// </summary>
        public async Task DownloadTrackAsync(string trackID, string? quality = null)
        {
            EnsureInitialized();

            await ExecuteWithRetryAsync(async () =>
            {
                await Task.Run(() =>
                {
                    var result = GoBackend.DownloadTrack(trackID, quality);
                    if (result != 0)
                    {
                        var errorMsg = GoBackend.GetErrorMessage(result);
                        throw new BackendException($"Failed to download track: {errorMsg}", result);
                    }
                });
            });
        }

        /// <summary>
        /// Download an album
        /// </summary>
        public async Task DownloadAlbumAsync(string albumID, string? quality = null)
        {
            EnsureInitialized();

            await ExecuteWithRetryAsync(async () =>
            {
                await Task.Run(() =>
                {
                    var result = GoBackend.DownloadAlbum(albumID, quality);
                    if (result != 0)
                    {
                        var errorMsg = GoBackend.GetErrorMessage(result);
                        throw new BackendException($"Failed to download album: {errorMsg}", result);
                    }
                });
            });
        }

        /// <summary>
        /// Download a playlist
        /// </summary>
        public async Task DownloadPlaylistAsync(string playlistID, string? quality = null)
        {
            EnsureInitialized();

            await ExecuteWithRetryAsync(async () =>
            {
                await Task.Run(() =>
                {
                    var result = GoBackend.DownloadPlaylist(playlistID, quality);
                    if (result != 0)
                    {
                        var errorMsg = GoBackend.GetErrorMessage(result);
                        throw new BackendException($"Failed to download playlist: {errorMsg}", result);
                    }
                });
            });
        }
        
        /// <summary>
        /// Download a custom playlist (e.g., from Spotify import)
        /// </summary>
        public async Task DownloadCustomPlaylistAsync(string playlistID, string title, string creator, List<string> trackIDs, string pictureUrl = "", string? quality = null)
        {
            EnsureInitialized();

            await ExecuteWithRetryAsync(async () =>
            {
                await Task.Run(() =>
                {
                    var playlistData = new
                    {
                        id = playlistID,
                        title = title,
                        description = "",
                        creator = creator,
                        track_ids = trackIDs,
                        picture_url = pictureUrl
                    };
                    
                    var json = System.Text.Json.JsonSerializer.Serialize(playlistData);
                    var result = GoBackend.DownloadCustomPlaylist(json, quality);
                    if (result != 0)
                    {
                        var errorMsg = GoBackend.GetErrorMessage(result);
                        throw new BackendException($"Failed to download custom playlist: {errorMsg}", result);
                    }
                });
            });
        }

        /// <summary>
        /// Convert Spotify URL to Deezer tracks
        /// </summary>
        public async Task<T?> ConvertSpotifyURLAsync<T>(string url)
        {
            EnsureInitialized();

            return await ExecuteWithRetryAsync(async () =>
            {
                return await Task.Run(() =>
                {
                    var ptr = GoBackend.ConvertSpotifyURL(url);
                    var json = GoBackend.PtrToStringAndFree(ptr);

                    if (string.IsNullOrEmpty(json))
                        return default;

                    return DeserializeJson<T>(json);
                });
            });
        }

        #endregion

        #region Queue Management

        /// <summary>
        /// Get queue items with pagination
        /// </summary>
        public async Task<T?> GetQueueAsync<T>(int offset = 0, int limit = 100, string? filter = null)
        {
            LoggingService.Instance.LogInfo($"GetQueueAsync called: offset={offset}, limit={limit}, filter={filter}");
            EnsureInitialized();

            return await ExecuteWithRetryAsync(async () =>
            {
                return await Task.Run(() =>
                {
                    LoggingService.Instance.LogInfo("Calling GoBackend.GetQueue...");
                    var ptr = GoBackend.GetQueue(offset, limit, filter);
                    var json = GoBackend.PtrToStringAndFree(ptr);

                    LoggingService.Instance.LogInfo($"GetQueue returned JSON (length: {json?.Length ?? 0})");

                    if (string.IsNullOrEmpty(json))
                    {
                        LoggingService.Instance.LogWarning("GetQueue returned null or empty JSON");
                        return default;
                    }

                    // Log completed items with their track counts for debugging
                    if (json.Contains("\"status\":\"completed\""))
                    {
                        // Find and log completed album data
                        try
                        {
                            var tempDoc = System.Text.Json.JsonDocument.Parse(json);
                            if (tempDoc.RootElement.TryGetProperty("items", out var items))
                            {
                                foreach (var item in items.EnumerateArray())
                                {
                                    if (item.TryGetProperty("status", out var status) && status.GetString() == "completed")
                                    {
                                        var id = item.TryGetProperty("id", out var idProp) ? idProp.GetString() : "?";
                                        var title = item.TryGetProperty("title", out var titleProp) ? titleProp.GetString() : "?";
                                        var type = item.TryGetProperty("type", out var typeProp) ? typeProp.GetString() : "?";
                                        var completed = item.TryGetProperty("completed_tracks", out var compProp) ? compProp.GetInt32() : -1;
                                        var total = item.TryGetProperty("total_tracks", out var totalProp) ? totalProp.GetInt32() : -1;
                                        
                                        LoggingService.Instance.LogInfo($"[RAW JSON] Completed item: ID={id}, Title={title}, Type={type}, CompletedTracks={completed}, TotalTracks={total}");
                                    }
                                }
                            }
                        }
                        catch (Exception ex)
                        {
                            LoggingService.Instance.LogWarning($"Failed to parse JSON for logging: {ex.Message}");
                        }
                    }

                    var result = DeserializeJson<T>(json);
                    LoggingService.Instance.LogInfo($"Deserialized result type: {result?.GetType().Name}");
                    return result;
                });
            });
        }

        /// <summary>
        /// Get queue statistics
        /// </summary>
        public async Task<QueueStats?> GetQueueStatsAsync()
        {
            EnsureInitialized();

            return await ExecuteWithRetryAsync(async () =>
            {
                return await Task.Run(() =>
                {
                    var ptr = GoBackend.GetQueueStats();
                    var json = GoBackend.PtrToStringAndFree(ptr);

                    if (string.IsNullOrEmpty(json))
                        return null;

                    return DeserializeJson<QueueStats>(json);
                });
            });
        }

        /// <summary>
        /// Get failed tracks for an album/playlist
        /// </summary>
        public async Task<List<FailedTrack>?> GetFailedTracksAsync(string parentId)
        {
            EnsureInitialized();

            return await ExecuteWithRetryAsync(async () =>
            {
                return await Task.Run(() =>
                {
                    var ptr = GoBackend.GetFailedTracks(parentId);
                    var json = GoBackend.PtrToStringAndFree(ptr);

                    if (string.IsNullOrEmpty(json))
                        return new List<FailedTrack>();

                    return DeserializeJson<List<FailedTrack>>(json) ?? new List<FailedTrack>();
                });
            });
        }

        /// <summary>
        /// Pause a download
        /// </summary>
        public async Task PauseDownloadAsync(string itemID)
        {
            EnsureInitialized();

            await ExecuteWithRetryAsync(async () =>
            {
                await Task.Run(() =>
                {
                    var result = GoBackend.PauseDownload(itemID);
                    if (result != 0)
                    {
                        var errorMsg = GoBackend.GetErrorMessage(result);
                        throw new BackendException($"Failed to pause download: {errorMsg}", result);
                    }
                });
            });
        }

        /// <summary>
        /// Resume a download
        /// </summary>
        public async Task ResumeDownloadAsync(string itemID)
        {
            EnsureInitialized();

            await ExecuteWithRetryAsync(async () =>
            {
                await Task.Run(() =>
                {
                    var result = GoBackend.ResumeDownload(itemID);
                    if (result != 0)
                    {
                        var errorMsg = GoBackend.GetErrorMessage(result);
                        throw new BackendException($"Failed to resume download: {errorMsg}", result);
                    }
                });
            });
        }

        /// <summary>
        /// Cancel a download
        /// </summary>
        public async Task CancelDownloadAsync(string itemID)
        {
            EnsureInitialized();

            await ExecuteWithRetryAsync(async () =>
            {
                await Task.Run(() =>
                {
                    var result = GoBackend.CancelDownload(itemID);
                    if (result != 0)
                    {
                        var errorMsg = GoBackend.GetErrorMessage(result);
                        throw new BackendException($"Failed to cancel download: {errorMsg}", result);
                    }
                });
            });
        }

        /// <summary>
        /// Retry a failed download
        /// </summary>
        public async Task RetryDownloadAsync(string itemID)
        {
            EnsureInitialized();

            await ExecuteWithRetryAsync(async () =>
            {
                await Task.Run(() =>
                {
                    var result = GoBackend.RetryDownload(itemID);
                    if (result != 0)
                    {
                        var errorMsg = GoBackend.GetErrorMessage(result);
                        throw new BackendException($"Failed to retry download: {errorMsg}", result);
                    }
                });
            });
        }

        /// <summary>
        /// Clear completed downloads
        /// </summary>
        public async Task ClearCompletedAsync()
        {
            EnsureInitialized();

            await ExecuteWithRetryAsync(async () =>
            {
                await Task.Run(() =>
                {
                    var result = GoBackend.ClearCompleted();
                    if (result != 0)
                    {
                        var errorMsg = GoBackend.GetErrorMessage(result);
                        throw new BackendException($"Failed to clear completed: {errorMsg}", result);
                    }
                });
            });
        }

        #endregion

        #region Settings Management

        /// <summary>
        /// Get current settings
        /// </summary>
        public async Task<T?> GetSettingsAsync<T>()
        {
            EnsureInitialized();

            return await ExecuteWithRetryAsync(async () =>
            {
                return await Task.Run(() =>
                {
                    var ptr = GoBackend.GetSettings();
                    var json = GoBackend.PtrToStringAndFree(ptr);

                    if (string.IsNullOrEmpty(json))
                        return default;

                    return DeserializeJson<T>(json);
                });
            });
        }

        /// <summary>
        /// Update settings
        /// </summary>
        public async Task UpdateSettingsAsync<T>(T settings)
        {
            EnsureInitialized();

            await ExecuteWithRetryAsync(async () =>
            {
                await Task.Run(() =>
                {
                    var json = JsonSerializer.Serialize(settings, new JsonSerializerOptions
                    {
                        PropertyNamingPolicy = JsonNamingPolicy.CamelCase,
                        WriteIndented = true
                    });

                    var result = GoBackend.UpdateSettings(json);
                    if (result != 0)
                    {
                        var errorMsg = GoBackend.GetErrorMessage(result);
                        throw new BackendException($"Failed to update settings: {errorMsg}", result);
                    }
                });
            });
        }

        /// <summary>
        /// Get download path
        /// </summary>
        public async Task<string?> GetDownloadPathAsync()
        {
            EnsureInitialized();

            return await ExecuteWithRetryAsync(async () =>
            {
                return await Task.Run(() =>
                {
                    var ptr = GoBackend.GetDownloadPath();
                    return GoBackend.PtrToStringAndFree(ptr);
                });
            });
        }

        /// <summary>
        /// Set download path
        /// </summary>
        public async Task SetDownloadPathAsync(string path)
        {
            EnsureInitialized();

            await ExecuteWithRetryAsync(async () =>
            {
                await Task.Run(() =>
                {
                    var result = GoBackend.SetDownloadPath(path);
                    if (result != 0)
                    {
                        var errorMsg = GoBackend.GetErrorMessage(result);
                        throw new BackendException($"Failed to set download path: {errorMsg}", result);
                    }
                });
            });
        }

        #endregion

        #region System

        /// <summary>
        /// Get backend version
        /// </summary>
        public async Task<string?> GetVersionAsync()
        {
            return await Task.Run(() =>
            {
                var ptr = GoBackend.GetVersion();
                return GoBackend.PtrToStringAndFree(ptr);
            });
        }

        #endregion

        #region Helper Methods

        private void EnsureInitialized()
        {
            if (!_initialized)
            {
                throw new InvalidOperationException("Backend is not initialized. Call InitializeAsync first.");
            }
        }

        private T? DeserializeJson<T>(string json)
        {
            try
            {
                // Check for error response
                if (json.Contains("\"error\""))
                {
                    var errorResponse = JsonSerializer.Deserialize<ErrorResponse>(json, new JsonSerializerOptions
                    {
                        PropertyNameCaseInsensitive = true
                    });

                    if (errorResponse?.Error != null)
                    {
                        _logger?.LogError("Backend returned error: {Error}", errorResponse.Error);
                        LoggingService.Instance.LogError($"Backend returned error: {errorResponse.Error}");
                        throw new BackendException(errorResponse.Error);
                    }
                }

                return JsonSerializer.Deserialize<T>(json, new JsonSerializerOptions
                {
                    PropertyNameCaseInsensitive = true
                });
            }
            catch (JsonException ex)
            {
                _logger?.LogError(ex, "Failed to deserialize JSON: {Json}", json);
                
                // Log the first 500 characters of JSON for debugging
                var jsonPreview = json.Length > 500 ? json.Substring(0, 500) + "..." : json;
                LoggingService.Instance.LogError($"Failed to deserialize JSON (length: {json.Length}). Preview: {jsonPreview}", ex);
                LoggingService.Instance.LogError($"Deserialization error: {ex.Message}");
                LoggingService.Instance.LogError($"Target type: {typeof(T).FullName}");
                
                throw new BackendException("Failed to deserialize response", ex);
            }
        }

        private async Task<T> ExecuteWithRetryAsync<T>(Func<Task<T>> operation)
        {
            Exception? lastException = null;

            for (int i = 0; i < _maxRetries; i++)
            {
                try
                {
                    return await operation();
                }
                catch (BackendException backendEx)
                {
                    // Check if it's a transient error that can be retried
                    if (backendEx.ErrorCode.HasValue && GoBackend.IsTransientError(backendEx.ErrorCode.Value))
                    {
                        lastException = backendEx;
                        _logger?.LogWarning(backendEx, "Transient error, attempt {Attempt} of {MaxRetries}", i + 1, _maxRetries);
                        LoggingService.Instance.LogWarning($"Transient error (attempt {i + 1}/{_maxRetries}): {backendEx.Message}");
                        
                        if (i < _maxRetries - 1)
                        {
                            var delay = _retryDelayMs * (i + 1); // Exponential backoff
                            await Task.Delay(delay);
                            continue;
                        }
                    }
                    
                    // Non-transient backend exceptions should not be retried
                    LoggingService.Instance.LogError($"Non-transient backend error: {backendEx.Message}", backendEx);
                    throw;
                }
                catch (Exception ex)
                {
                    lastException = ex;
                    _logger?.LogWarning(ex, "Operation failed, attempt {Attempt} of {MaxRetries}", i + 1, _maxRetries);
                    LoggingService.Instance.LogWarning($"Operation failed (attempt {i + 1}/{_maxRetries}): {ex.Message}");

                    if (i < _maxRetries - 1)
                    {
                        await Task.Delay(_retryDelayMs * (i + 1)); // Exponential backoff
                    }
                }
            }

            _logger?.LogError(lastException, "Operation failed after {MaxRetries} attempts", _maxRetries);
            LoggingService.Instance.LogError($"Operation failed after {_maxRetries} attempts", lastException);
            throw new BackendException("Operation failed after multiple retries", lastException!);
        }

        private async Task ExecuteWithRetryAsync(Func<Task> operation)
        {
            await ExecuteWithRetryAsync(async () =>
            {
                await operation();
                return true;
            });
        }

        #endregion

        #region IDisposable

        private bool _disposed;

        public void Dispose()
        {
            if (_disposed)
                return;

            Shutdown();
            _callbackHandler?.Dispose();

            _disposed = true;
        }

        #endregion
    }

    #region Exception Classes

    /// <summary>
    /// Exception thrown by backend operations
    /// </summary>
    public class BackendException : Exception
    {
        public int? ErrorCode { get; }

        public BackendException(string message) : base(message)
        {
        }

        public BackendException(string message, int errorCode) : base(message)
        {
            ErrorCode = errorCode;
        }

        public BackendException(string message, Exception innerException) : base(message, innerException)
        {
        }
    }

    /// <summary>
    /// Error response from backend
    /// </summary>
    internal class ErrorResponse
    {
        public string? Error { get; set; }
    }

    #endregion
}
