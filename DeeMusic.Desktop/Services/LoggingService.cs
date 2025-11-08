using System;
using System.IO;
using System.Text;
using System.Threading;

namespace DeeMusic.Desktop.Services
{
    /// <summary>
    /// File-based logging service for debugging and error tracking
    /// </summary>
    public class LoggingService : IDisposable
    {
        private static LoggingService? _instance;
        private static readonly object _lock = new object();
        
        private readonly string _logFilePath;
        private readonly SemaphoreSlim _writeSemaphore = new SemaphoreSlim(1, 1);
        private readonly int _maxLogSizeBytes = 10 * 1024 * 1024; // 10 MB
        private readonly int _maxLogFiles = 5;
        
        public enum LogLevel
        {
            Debug,
            Info,
            Warning,
            Error,
            Critical
        }
        
        public static LoggingService Instance
        {
            get
            {
                if (_instance == null)
                {
                    lock (_lock)
                    {
                        _instance ??= new LoggingService();
                    }
                }
                return _instance;
            }
        }
        
        private LoggingService()
        {
            var appDataPath = Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData);
            var logsPath = Path.Combine(appDataPath, "DeeMusicV2", "logs");
            
            // Ensure logs directory exists
            Directory.CreateDirectory(logsPath);
            
            // Create log file with timestamp
            var timestamp = DateTime.Now.ToString("yyyyMMdd");
            _logFilePath = Path.Combine(logsPath, $"deemusic_{timestamp}.log");
            
            // Rotate old logs if needed
            RotateLogsIfNeeded(logsPath);
            
            // Write startup message
            LogInfo("=== DeeMusic Desktop Started ===");
        }
        
        /// <summary>
        /// Log a debug message
        /// </summary>
        public void LogDebug(string message, Exception? exception = null)
        {
            Log(LogLevel.Debug, message, exception);
        }
        
        /// <summary>
        /// Log an informational message
        /// </summary>
        public void LogInfo(string message, Exception? exception = null)
        {
            Log(LogLevel.Info, message, exception);
        }
        
        /// <summary>
        /// Log a warning message
        /// </summary>
        public void LogWarning(string message, Exception? exception = null)
        {
            Log(LogLevel.Warning, message, exception);
        }
        
        /// <summary>
        /// Log an error message
        /// </summary>
        public void LogError(string message, Exception? exception = null)
        {
            Log(LogLevel.Error, message, exception);
        }
        
        /// <summary>
        /// Log a critical error message
        /// </summary>
        public void LogCritical(string message, Exception? exception = null)
        {
            Log(LogLevel.Critical, message, exception);
        }
        
        /// <summary>
        /// Log a message with specified level
        /// </summary>
        private async void Log(LogLevel level, string message, Exception? exception = null)
        {
            try
            {
                await _writeSemaphore.WaitAsync();
                
                var sb = new StringBuilder();
                sb.Append($"[{DateTime.Now:yyyy-MM-dd HH:mm:ss.fff}] ");
                sb.Append($"[{level}] ");
                sb.AppendLine(message);
                
                if (exception != null)
                {
                    sb.AppendLine($"Exception: {exception.GetType().Name}");
                    sb.AppendLine($"Message: {exception.Message}");
                    sb.AppendLine($"StackTrace: {exception.StackTrace}");
                    
                    if (exception.InnerException != null)
                    {
                        sb.AppendLine($"Inner Exception: {exception.InnerException.GetType().Name}");
                        sb.AppendLine($"Inner Message: {exception.InnerException.Message}");
                    }
                }
                
                // Check file size and rotate if needed
                CheckAndRotateLog();
                
                // Write to file
                await File.AppendAllTextAsync(_logFilePath, sb.ToString());
            }
            catch
            {
                // Silently fail - don't throw exceptions from logging
            }
            finally
            {
                _writeSemaphore.Release();
            }
        }
        
        /// <summary>
        /// Check log file size and rotate if needed
        /// </summary>
        private void CheckAndRotateLog()
        {
            try
            {
                if (File.Exists(_logFilePath))
                {
                    var fileInfo = new FileInfo(_logFilePath);
                    if (fileInfo.Length > _maxLogSizeBytes)
                    {
                        // Rotate the log file
                        var timestamp = DateTime.Now.ToString("yyyyMMdd_HHmmss");
                        var rotatedPath = _logFilePath.Replace(".log", $"_{timestamp}.log");
                        File.Move(_logFilePath, rotatedPath);
                    }
                }
            }
            catch
            {
                // Silently fail
            }
        }
        
        /// <summary>
        /// Rotate old log files to keep only the most recent ones
        /// </summary>
        private void RotateLogsIfNeeded(string logsPath)
        {
            try
            {
                var logFiles = Directory.GetFiles(logsPath, "deemusic_*.log");
                
                if (logFiles.Length > _maxLogFiles)
                {
                    // Sort by creation time (oldest first)
                    Array.Sort(logFiles, (a, b) => 
                        File.GetCreationTime(a).CompareTo(File.GetCreationTime(b)));
                    
                    // Delete oldest files
                    for (int i = 0; i < logFiles.Length - _maxLogFiles; i++)
                    {
                        try
                        {
                            File.Delete(logFiles[i]);
                        }
                        catch
                        {
                            // Continue if we can't delete a file
                        }
                    }
                }
            }
            catch
            {
                // Silently fail
            }
        }
        
        /// <summary>
        /// Get the current log file path
        /// </summary>
        public string GetLogFilePath() => _logFilePath;
        
        /// <summary>
        /// Open the logs folder in Windows Explorer
        /// </summary>
        public void OpenLogsFolder()
        {
            try
            {
                var logsPath = Path.GetDirectoryName(_logFilePath);
                if (logsPath != null && Directory.Exists(logsPath))
                {
                    System.Diagnostics.Process.Start("explorer.exe", logsPath);
                }
            }
            catch (Exception ex)
            {
                LogError("Failed to open logs folder", ex);
            }
        }
        
        public void Dispose()
        {
            LogInfo("=== DeeMusic Desktop Shutdown ===");
            _writeSemaphore?.Dispose();
        }
    }
}
