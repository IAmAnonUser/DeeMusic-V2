using System;
using System.Runtime.InteropServices;
using System.Text;

namespace DeeMusic.Desktop.Services
{
    /// <summary>
    /// P/Invoke wrapper for the Go backend DLL (deemusic-core.dll)
    /// Provides direct access to all exported Go functions
    /// </summary>
    public static class GoBackend
    {
        private const string DllName = "deemusic-core.dll";

        #region Initialization and Lifecycle

        /// <summary>
        /// Initialize the Go backend with configuration file path
        /// </summary>
        /// <param name="configPath">Path to the configuration JSON file</param>
        /// <returns>0 on success, negative error code on failure</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern int InitializeApp(string configPath);

        /// <summary>
        /// Shutdown the Go backend and cleanup resources
        /// </summary>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern void ShutdownApp();

        #endregion

        #region Callback Registration

        /// <summary>
        /// Set the progress callback for download progress updates
        /// </summary>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern void SetProgressCallback(ProgressCallback callback);

        /// <summary>
        /// Set the status callback for download status changes
        /// </summary>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern void SetStatusCallback(StatusCallback callback);

        /// <summary>
        /// Set the queue update callback for queue statistics changes
        /// </summary>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern void SetQueueUpdateCallback(QueueUpdateCallback callback);

        #endregion

        #region Search and Browse

        /// <summary>
        /// Search for tracks, albums, artists, or playlists
        /// </summary>
        /// <param name="query">Search query string</param>
        /// <param name="searchType">Type: "track", "album", "artist", or "playlist"</param>
        /// <param name="limit">Maximum number of results</param>
        /// <returns>Pointer to JSON string (must be freed with FreeString)</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern IntPtr Search(string query, string searchType, int limit);

        /// <summary>
        /// Get album details by ID
        /// </summary>
        /// <param name="albumID">Deezer album ID</param>
        /// <returns>Pointer to JSON string (must be freed with FreeString)</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern IntPtr GetAlbum(string albumID);

        /// <summary>
        /// Get artist details by ID
        /// </summary>
        /// <param name="artistID">Deezer artist ID</param>
        /// <returns>Pointer to JSON string (must be freed with FreeString)</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern IntPtr GetArtist(string artistID);

        /// <summary>
        /// Get artist albums by ID
        /// </summary>
        /// <param name="artistID">Deezer artist ID</param>
        /// <param name="limit">Maximum number of albums to return</param>
        /// <returns>Pointer to JSON string (must be freed with FreeString)</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern IntPtr GetArtistAlbums(string artistID, int limit);

        /// <summary>
        /// Get playlist details by ID
        /// </summary>
        /// <param name="playlistID">Deezer playlist ID</param>
        /// <returns>Pointer to JSON string (must be freed with FreeString)</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern IntPtr GetPlaylist(string playlistID);

        /// <summary>
        /// Get Deezer charts
        /// </summary>
        /// <param name="limit">Maximum number of results</param>
        /// <returns>Pointer to JSON string (must be freed with FreeString)</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern IntPtr GetCharts(int limit);

        /// <summary>
        /// Get editorial releases (new releases)
        /// </summary>
        /// <param name="limit">Maximum number of results</param>
        /// <returns>Pointer to JSON string (must be freed with FreeString)</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern IntPtr GetEditorialReleases(int limit);

        #endregion

        #region Download Operations

        /// <summary>
        /// Download a single track
        /// </summary>
        /// <param name="trackID">Deezer track ID</param>
        /// <param name="quality">Quality setting (e.g., "MP3_320", "FLAC")</param>
        /// <returns>0 on success, negative error code on failure</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern int DownloadTrack(string trackID, string? quality);

        /// <summary>
        /// Download an entire album
        /// </summary>
        /// <param name="albumID">Deezer album ID</param>
        /// <param name="quality">Quality setting (e.g., "MP3_320", "FLAC")</param>
        /// <returns>0 on success, negative error code on failure</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern int DownloadAlbum(string albumID, string? quality);

        /// <summary>
        /// Download an entire playlist
        /// </summary>
        /// <param name="playlistID">Deezer playlist ID</param>
        /// <param name="quality">Quality setting (e.g., "MP3_320", "FLAC")</param>
        /// <returns>0 on success, negative error code on failure</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern int DownloadPlaylist(string playlistID, string? quality);
        
        /// <summary>
        /// Download a custom playlist (e.g., from Spotify import)
        /// </summary>
        /// <param name="playlistJSON">JSON containing playlist metadata and track IDs</param>
        /// <param name="quality">Quality setting (e.g., "MP3_320", "FLAC")</param>
        /// <returns>0 on success, negative error code on failure</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern int DownloadCustomPlaylist(string playlistJSON, string? quality);

        /// <summary>
        /// Convert Spotify URL to Deezer tracks
        /// </summary>
        /// <param name="url">Spotify playlist or track URL</param>
        /// <returns>Pointer to JSON string (must be freed with FreeString)</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern IntPtr ConvertSpotifyURL(string url);

        #endregion

        #region Queue Management

        /// <summary>
        /// Get queue items with pagination
        /// </summary>
        /// <param name="offset">Starting offset</param>
        /// <param name="limit">Maximum number of items</param>
        /// <param name="filter">Filter by status (e.g., "pending", null for all)</param>
        /// <returns>Pointer to JSON string (must be freed with FreeString)</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern IntPtr GetQueue(int offset, int limit, string? filter);

        /// <summary>
        /// Get queue statistics
        /// </summary>
        /// <returns>Pointer to JSON string (must be freed with FreeString)</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern IntPtr GetQueueStats();

        /// <summary>
        /// Pause a download
        /// </summary>
        /// <param name="itemID">Queue item ID</param>
        /// <returns>0 on success, negative error code on failure</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern int PauseDownload(string itemID);

        /// <summary>
        /// Resume a paused download
        /// </summary>
        /// <param name="itemID">Queue item ID</param>
        /// <returns>0 on success, negative error code on failure</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern int ResumeDownload(string itemID);

        /// <summary>
        /// Cancel a download
        /// </summary>
        /// <param name="itemID">Queue item ID</param>
        /// <returns>0 on success, negative error code on failure</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern int CancelDownload(string itemID);

        /// <summary>
        /// Retry a failed download
        /// </summary>
        /// <param name="itemID">Queue item ID</param>
        /// <returns>0 on success, negative error code on failure</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern int RetryDownload(string itemID);

        /// <summary>
        /// Clear all completed downloads from queue
        /// </summary>
        /// <returns>0 on success, negative error code on failure</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern int ClearCompleted();

        /// <summary>
        /// Stop all active downloads and clear the entire queue
        /// </summary>
        /// <returns>0 on success, negative error code on failure</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern int StopAllDownloads();

        #endregion

        #region Settings Management

        /// <summary>
        /// Get current settings as JSON
        /// </summary>
        /// <returns>Pointer to JSON string (must be freed with FreeString)</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern IntPtr GetSettings();

        /// <summary>
        /// Update settings from JSON
        /// </summary>
        /// <param name="settingsJSON">Settings as JSON string</param>
        /// <returns>0 on success, negative error code on failure</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern int UpdateSettings(string settingsJSON);

        /// <summary>
        /// Get current download path
        /// </summary>
        /// <returns>Pointer to string (must be freed with FreeString)</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern IntPtr GetDownloadPath();

        /// <summary>
        /// Set download path
        /// </summary>
        /// <param name="path">New download directory path</param>
        /// <returns>0 on success, negative error code on failure</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl, CharSet = CharSet.Ansi)]
        public static extern int SetDownloadPath(string path);

        #endregion

        #region System

        /// <summary>
        /// Get backend version
        /// </summary>
        /// <returns>Pointer to version string (must be freed with FreeString)</returns>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern IntPtr GetVersion();

        /// <summary>
        /// Free a string allocated by Go
        /// </summary>
        /// <param name="str">Pointer to string to free</param>
        [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
        public static extern void FreeString(IntPtr str);

        #endregion

        #region Helper Methods

        /// <summary>
        /// Convert IntPtr to managed string and free the Go-allocated memory
        /// </summary>
        /// <param name="ptr">Pointer to Go-allocated string</param>
        /// <returns>Managed string or null if pointer is invalid</returns>
        public static string? PtrToStringAndFree(IntPtr ptr)
        {
            if (ptr == IntPtr.Zero)
                return null;

            try
            {
                // Marshal the string from unmanaged memory
                string? result = Marshal.PtrToStringAnsi(ptr);
                return result;
            }
            finally
            {
                // Always free the Go-allocated memory
                FreeString(ptr);
            }
        }

        /// <summary>
        /// Get error message for error codes
        /// </summary>
        /// <param name="errorCode">Error code from Go function</param>
        /// <returns>Human-readable error message</returns>
        public static string GetErrorMessage(int errorCode)
        {
            return errorCode switch
            {
                0 => "Success",
                -1 => "Backend not initialized or invalid state",
                -2 => "Operation failed",
                -3 => "Invalid configuration",
                -4 => "Database error",
                -5 => "Migration failed",
                -6 => "Failed to start download manager",
                -7 => "Authentication failed",
                -8 => "Network error",
                -9 => "File system error",
                -10 => "Invalid parameter",
                -11 => "Resource not found",
                -12 => "Permission denied",
                -13 => "Timeout",
                -14 => "Rate limit exceeded",
                -15 => "Item already in queue",
                _ => $"Unknown error code: {errorCode}"
            };
        }
        
        /// <summary>
        /// Get detailed error message with context
        /// </summary>
        /// <param name="errorCode">Error code from Go function</param>
        /// <param name="operation">Operation that failed</param>
        /// <returns>Detailed error message</returns>
        public static string GetDetailedErrorMessage(int errorCode, string operation)
        {
            var baseMessage = GetErrorMessage(errorCode);
            return $"{operation} failed: {baseMessage} (Error code: {errorCode})";
        }
        
        /// <summary>
        /// Check if error code represents a transient error that can be retried
        /// </summary>
        /// <param name="errorCode">Error code from Go function</param>
        /// <returns>True if error is transient and can be retried</returns>
        public static bool IsTransientError(int errorCode)
        {
            return errorCode switch
            {
                -8 => true,  // Network error
                -13 => true, // Timeout
                -14 => true, // Rate limit
                _ => false
            };
        }

        #endregion
    }

    #region Callback Delegates

    /// <summary>
    /// Callback delegate for download progress updates
    /// </summary>
    /// <param name="itemID">Queue item ID</param>
    /// <param name="progress">Progress percentage (0-100)</param>
    /// <param name="bytesProcessed">Bytes downloaded so far</param>
    /// <param name="totalBytes">Total bytes to download</param>
    [UnmanagedFunctionPointer(CallingConvention.Cdecl)]
    public delegate void ProgressCallback(
        [MarshalAs(UnmanagedType.LPStr)] string itemID,
        int progress,
        long bytesProcessed,
        long totalBytes);

    /// <summary>
    /// Callback delegate for download status changes
    /// </summary>
    /// <param name="itemID">Queue item ID</param>
    /// <param name="status">New status (e.g., "started", "completed", "failed")</param>
    /// <param name="errorMsg">Error message if status is "failed", null otherwise</param>
    [UnmanagedFunctionPointer(CallingConvention.Cdecl)]
    public delegate void StatusCallback(
        [MarshalAs(UnmanagedType.LPStr)] string itemID,
        [MarshalAs(UnmanagedType.LPStr)] string status,
        [MarshalAs(UnmanagedType.LPStr)] string? errorMsg);

    /// <summary>
    /// Callback delegate for queue statistics updates
    /// </summary>
    /// <param name="statsJson">Queue statistics as JSON string</param>
    [UnmanagedFunctionPointer(CallingConvention.Cdecl)]
    public delegate void QueueUpdateCallback(
        [MarshalAs(UnmanagedType.LPStr)] string statsJson);

    #endregion
}
