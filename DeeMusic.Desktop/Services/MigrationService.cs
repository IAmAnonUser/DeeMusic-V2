using System;
using System.Runtime.InteropServices;
using System.Text.Json;
using System.Threading.Tasks;

namespace DeeMusic.Desktop.Services
{
    /// <summary>
    /// Service for migrating data from Python DeeMusic to the new standalone version
    /// </summary>
    public class MigrationService
    {
        // P/Invoke declarations for migration functions
        [DllImport("deemusic-core.dll", CallingConvention = CallingConvention.Cdecl)]
        private static extern int CheckMigrationNeeded();

        [DllImport("deemusic-core.dll", CallingConvention = CallingConvention.Cdecl)]
        private static extern IntPtr DetectPythonInstallation();

        [DllImport("deemusic-core.dll", CallingConvention = CallingConvention.Cdecl)]
        private static extern IntPtr GetMigrationStats();

        [DllImport("deemusic-core.dll", CallingConvention = CallingConvention.Cdecl)]
        private static extern IntPtr PerformMigration(ProgressCallback progressCallback);

        // Delegate for progress callback
        [UnmanagedFunctionPointer(CallingConvention.Cdecl)]
        private delegate void ProgressCallback(IntPtr message, int progress, long bytesProcessed, long totalBytes);

        /// <summary>
        /// Event raised when migration progress is updated
        /// </summary>
        public event EventHandler<MigrationProgressEventArgs>? ProgressUpdated;

        /// <summary>
        /// Checks if migration from Python version is needed
        /// </summary>
        public async Task<bool> IsMigrationNeededAsync()
        {
            return await Task.Run(() =>
            {
                try
                {
                    int result = CheckMigrationNeeded();
                    return result == 1;
                }
                catch (Exception ex)
                {
                    System.Diagnostics.Debug.WriteLine($"Error checking migration: {ex.Message}");
                    return false;
                }
            });
        }

        /// <summary>
        /// Detects Python DeeMusic installation
        /// </summary>
        public async Task<PythonInstallationInfo?> DetectPythonInstallationAsync()
        {
            return await Task.Run(() =>
            {
                try
                {
                    IntPtr resultPtr = DetectPythonInstallation();
                    if (resultPtr == IntPtr.Zero)
                        return null;

                    string json = Marshal.PtrToStringUTF8(resultPtr) ?? "{}";
                    GoBackend.FreeString(resultPtr);

                    var result = JsonSerializer.Deserialize<PythonInstallationInfo>(json);
                    return result;
                }
                catch (Exception ex)
                {
                    System.Diagnostics.Debug.WriteLine($"Error detecting Python installation: {ex.Message}");
                    return null;
                }
            });
        }

        /// <summary>
        /// Gets statistics about what will be migrated
        /// </summary>
        public async Task<MigrationStats?> GetMigrationStatsAsync()
        {
            return await Task.Run(() =>
            {
                try
                {
                    IntPtr resultPtr = GetMigrationStats();
                    if (resultPtr == IntPtr.Zero)
                        return null;

                    string json = Marshal.PtrToStringUTF8(resultPtr) ?? "{}";
                    GoBackend.FreeString(resultPtr);

                    var stats = JsonSerializer.Deserialize<MigrationStats>(json);
                    return stats;
                }
                catch (Exception ex)
                {
                    System.Diagnostics.Debug.WriteLine($"Error getting migration stats: {ex.Message}");
                    return null;
                }
            });
        }

        /// <summary>
        /// Performs the migration from Python to standalone version
        /// </summary>
        public async Task<MigrationResult> PerformMigrationAsync()
        {
            return await Task.Run(() =>
            {
                try
                {
                    // Create progress callback
                    ProgressCallback callback = (messagePtr, progress, bytesProcessed, totalBytes) =>
                    {
                        try
                        {
                            string message = Marshal.PtrToStringUTF8(messagePtr) ?? "";
                            
                            // Raise progress event on UI thread
                            System.Windows.Application.Current?.Dispatcher.Invoke(() =>
                            {
                                ProgressUpdated?.Invoke(this, new MigrationProgressEventArgs
                                {
                                    Message = message,
                                    Progress = progress
                                });
                            });
                        }
                        catch (Exception ex)
                        {
                            System.Diagnostics.Debug.WriteLine($"Error in progress callback: {ex.Message}");
                        }
                    };

                    // Perform migration
                    IntPtr resultPtr = PerformMigration(callback);
                    if (resultPtr == IntPtr.Zero)
                    {
                        return new MigrationResult
                        {
                            Success = false,
                            Error = "Migration failed: No result returned"
                        };
                    }

                    string json = Marshal.PtrToStringUTF8(resultPtr) ?? "{}";
                    GoBackend.FreeString(resultPtr);

                    var result = JsonSerializer.Deserialize<MigrationResult>(json);
                    return result ?? new MigrationResult
                    {
                        Success = false,
                        Error = "Failed to parse migration result"
                    };
                }
                catch (Exception ex)
                {
                    System.Diagnostics.Debug.WriteLine($"Error performing migration: {ex.Message}");
                    return new MigrationResult
                    {
                        Success = false,
                        Error = $"Migration error: {ex.Message}"
                    };
                }
            });
        }
    }

    /// <summary>
    /// Information about detected Python installation
    /// </summary>
    public class PythonInstallationInfo
    {
        public string? data_dir { get; set; }
        public bool has_settings { get; set; }
        public bool has_queue { get; set; }
        public string? settings_path { get; set; }
        public string? queue_path { get; set; }
        public string? error { get; set; }

        public bool HasError => !string.IsNullOrEmpty(error);
    }

    /// <summary>
    /// Statistics about migration data
    /// </summary>
    public class MigrationStats
    {
        public int queue_items { get; set; }
        public int history_items { get; set; }
        public string? error { get; set; }

        public bool HasError => !string.IsNullOrEmpty(error);
        public int TotalItems => queue_items + history_items;
    }

    /// <summary>
    /// Result of migration operation
    /// </summary>
    public class MigrationResult
    {
        public bool Success { get; set; }
        public bool settings_migrated { get; set; }
        public bool queue_migrated { get; set; }
        public string? backup_path { get; set; }
        public string? Error { get; set; }

        public bool HasError => !string.IsNullOrEmpty(Error);
    }

    /// <summary>
    /// Event args for migration progress updates
    /// </summary>
    public class MigrationProgressEventArgs : EventArgs
    {
        public string Message { get; set; } = "";
        public int Progress { get; set; }
    }
}
