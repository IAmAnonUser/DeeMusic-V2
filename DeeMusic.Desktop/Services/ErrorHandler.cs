using System;
using System.IO;
using System.Net.Http;
using System.Windows;

namespace DeeMusic.Desktop.Services
{
    /// <summary>
    /// Centralized error handling service with user-friendly error messages
    /// </summary>
    public static class ErrorHandler
    {
        /// <summary>
        /// Handle an exception and show appropriate error dialog
        /// </summary>
        public static void HandleException(Exception ex, string context = "")
        {
            var logger = LoggingService.Instance;
            
            // Log the error
            logger.LogError($"Error in {context}: {ex.Message}", ex);
            
            // Determine error type and show appropriate message
            var (title, message, isCritical) = GetErrorMessage(ex, context);
            
            // Show error dialog
            ShowErrorDialog(title, message, isCritical);
        }
        
        /// <summary>
        /// Handle a backend exception with specific error code
        /// </summary>
        public static void HandleBackendException(BackendException ex, string context = "")
        {
            var logger = LoggingService.Instance;
            
            // Log the error with error code
            logger.LogError($"Backend error in {context} (Code: {ex.ErrorCode}): {ex.Message}", ex);
            
            // Get user-friendly message
            var (title, message, isCritical) = GetBackendErrorMessage(ex, context);
            
            // Show error dialog
            ShowErrorDialog(title, message, isCritical);
        }
        
        /// <summary>
        /// Show a user-friendly error dialog
        /// </summary>
        public static void ShowErrorDialog(string title, string message, bool isCritical = false)
        {
            try
            {
                Application.Current?.Dispatcher.Invoke(() =>
                {
                    var icon = isCritical ? MessageBoxImage.Error : MessageBoxImage.Warning;
                    MessageBox.Show(message, title, MessageBoxButton.OK, icon);
                });
            }
            catch
            {
                // If we can't show the dialog, at least log it
                LoggingService.Instance.LogError($"Failed to show error dialog: {title} - {message}");
            }
        }
        
        /// <summary>
        /// Get user-friendly error message based on exception type
        /// </summary>
        private static (string title, string message, bool isCritical) GetErrorMessage(Exception ex, string context)
        {
            return ex switch
            {
                BackendException backendEx => GetBackendErrorMessage(backendEx, context),
                
                InvalidOperationException => (
                    "Operation Error",
                    $"The operation could not be completed: {ex.Message}\n\nPlease try again.",
                    false
                ),
                
                UnauthorizedAccessException => (
                    "Access Denied",
                    "DeeMusic doesn't have permission to access the required files or folders.\n\n" +
                    "Please check your folder permissions and try again.",
                    false
                ),
                
                IOException => (
                    "File Error",
                    $"A file operation failed: {ex.Message}\n\n" +
                    "Please check that you have enough disk space and the files are not in use.",
                    false
                ),
                
                HttpRequestException => (
                    "Network Error",
                    "Unable to connect to the server. Please check your internet connection and try again.",
                    false
                ),
                
                TimeoutException => (
                    "Timeout Error",
                    "The operation took too long to complete. Please check your internet connection and try again.",
                    false
                ),
                
                _ => (
                    "Unexpected Error",
                    $"An unexpected error occurred: {ex.Message}\n\n" +
                    "Please check the log files for more details.",
                    true
                )
            };
        }
        
        /// <summary>
        /// Get user-friendly error message for backend exceptions
        /// </summary>
        private static (string title, string message, bool isCritical) GetBackendErrorMessage(
            BackendException ex, string context)
        {
            // Check for specific error patterns in the message
            var errorMsg = ex.Message.ToLower();
            
            if (errorMsg.Contains("not initialized"))
            {
                return (
                    "Initialization Error",
                    "The application backend is not properly initialized.\n\n" +
                    "Please restart the application. If the problem persists, try reinstalling.",
                    true
                );
            }
            
            if (errorMsg.Contains("authentication") || errorMsg.Contains("arl"))
            {
                return (
                    "Authentication Error",
                    "Failed to authenticate with Deezer. Your ARL token may be invalid or expired.\n\n" +
                    "Please update your ARL token in Settings.",
                    false
                );
            }
            
            if (errorMsg.Contains("database"))
            {
                return (
                    "Database Error",
                    "A database error occurred. Your queue data may be corrupted.\n\n" +
                    "Please check the log files for details. You may need to reset the queue.",
                    true
                );
            }
            
            if (errorMsg.Contains("migration"))
            {
                return (
                    "Migration Error",
                    "Failed to migrate data from the previous version.\n\n" +
                    "Your old data is still safe. Please check the log files for details.",
                    false
                );
            }
            
            if (errorMsg.Contains("download") && errorMsg.Contains("failed"))
            {
                return (
                    "Download Error",
                    $"Failed to start download: {ex.Message}\n\n" +
                    "The track may not be available or your ARL token may be invalid.",
                    false
                );
            }
            
            if (errorMsg.Contains("not found") || errorMsg.Contains("404"))
            {
                return (
                    "Not Found",
                    "The requested content was not found on Deezer.\n\n" +
                    "It may have been removed or is not available in your region.",
                    false
                );
            }
            
            if (errorMsg.Contains("rate limit") || errorMsg.Contains("too many requests"))
            {
                return (
                    "Rate Limit",
                    "Too many requests to Deezer. Please wait a moment and try again.",
                    false
                );
            }
            
            // Check error code if available
            if (ex.ErrorCode.HasValue)
            {
                return ex.ErrorCode.Value switch
                {
                    -1 => ("Initialization Error", "Backend not initialized or invalid state.", true),
                    -2 => ("Operation Failed", $"The operation failed: {ex.Message}", false),
                    -3 => ("Configuration Error", "Invalid configuration. Please check your settings.", false),
                    -4 => ("Database Error", "Database operation failed. Your data may be corrupted.", true),
                    -5 => ("Migration Error", "Failed to migrate data from previous version.", false),
                    -6 => ("Download Manager Error", "Failed to start download manager.", true),
                    _ => ("Backend Error", $"Backend error (code {ex.ErrorCode}): {ex.Message}", false)
                };
            }
            
            // Default backend error message
            return (
                "Backend Error",
                $"A backend error occurred: {ex.Message}\n\n" +
                "Please try again. If the problem persists, check the log files.",
                false
            );
        }
        
        /// <summary>
        /// Show a confirmation dialog for critical operations
        /// </summary>
        public static bool ShowConfirmation(string title, string message)
        {
            try
            {
                var result = MessageBoxResult.No;
                Application.Current?.Dispatcher.Invoke(() =>
                {
                    result = MessageBox.Show(message, title, MessageBoxButton.YesNo, MessageBoxImage.Question);
                });
                return result == MessageBoxResult.Yes;
            }
            catch
            {
                return false;
            }
        }
        
        /// <summary>
        /// Show an information dialog
        /// </summary>
        public static void ShowInfo(string title, string message)
        {
            try
            {
                Application.Current?.Dispatcher.Invoke(() =>
                {
                    MessageBox.Show(message, title, MessageBoxButton.OK, MessageBoxImage.Information);
                });
            }
            catch
            {
                // Silently fail
            }
        }
    }
}
