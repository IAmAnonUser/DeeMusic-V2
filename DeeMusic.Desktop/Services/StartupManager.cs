using System;
using System.Diagnostics;
using System.IO;
using System.Reflection;
using Microsoft.Win32;

namespace DeeMusic.Desktop.Services
{
    /// <summary>
    /// Manages Windows startup integration for the application
    /// </summary>
    public class StartupManager
    {
        private const string AppName = "DeeMusic";
        private const string RegistryKeyPath = @"SOFTWARE\Microsoft\Windows\CurrentVersion\Run";
        
        private static StartupManager? _instance;
        private static readonly object _lock = new();

        /// <summary>
        /// Gets the singleton instance of the StartupManager
        /// </summary>
        public static StartupManager Instance
        {
            get
            {
                if (_instance == null)
                {
                    lock (_lock)
                    {
                        _instance ??= new StartupManager();
                    }
                }
                return _instance;
            }
        }

        private StartupManager()
        {
        }

        /// <summary>
        /// Enables the application to start with Windows
        /// </summary>
        /// <param name="startMinimized">Whether to start the application minimized</param>
        /// <returns>True if successful, false otherwise</returns>
        public bool EnableStartup(bool startMinimized = false)
        {
            try
            {
                string executablePath = GetExecutablePath();
                if (string.IsNullOrEmpty(executablePath))
                {
                    Debug.WriteLine("Failed to get executable path");
                    return false;
                }

                // Build command line arguments
                string commandLine = $"\"{executablePath}\"";
                if (startMinimized)
                {
                    commandLine += " --minimized";
                }

                using var key = Registry.CurrentUser.OpenSubKey(RegistryKeyPath, writable: true);
                if (key == null)
                {
                    Debug.WriteLine("Failed to open registry key");
                    return false;
                }

                key.SetValue(AppName, commandLine, RegistryValueKind.String);
                Debug.WriteLine($"Startup enabled: {commandLine}");
                return true;
            }
            catch (Exception ex)
            {
                Debug.WriteLine($"Failed to enable startup: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Disables the application from starting with Windows
        /// </summary>
        /// <returns>True if successful, false otherwise</returns>
        public bool DisableStartup()
        {
            try
            {
                using var key = Registry.CurrentUser.OpenSubKey(RegistryKeyPath, writable: true);
                if (key == null)
                {
                    Debug.WriteLine("Failed to open registry key");
                    return false;
                }

                // Check if the value exists before trying to delete it
                if (key.GetValue(AppName) != null)
                {
                    key.DeleteValue(AppName, throwOnMissingValue: false);
                    Debug.WriteLine("Startup disabled");
                }
                
                return true;
            }
            catch (Exception ex)
            {
                Debug.WriteLine($"Failed to disable startup: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Checks if the application is set to start with Windows
        /// </summary>
        /// <returns>True if startup is enabled, false otherwise</returns>
        public bool IsStartupEnabled()
        {
            try
            {
                using var key = Registry.CurrentUser.OpenSubKey(RegistryKeyPath, writable: false);
                if (key == null)
                {
                    return false;
                }

                var value = key.GetValue(AppName) as string;
                return !string.IsNullOrEmpty(value);
            }
            catch (Exception ex)
            {
                Debug.WriteLine($"Failed to check startup status: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Updates the startup configuration
        /// </summary>
        /// <param name="enabled">Whether startup should be enabled</param>
        /// <param name="startMinimized">Whether to start minimized</param>
        /// <returns>True if successful, false otherwise</returns>
        public bool UpdateStartup(bool enabled, bool startMinimized = false)
        {
            if (enabled)
            {
                return EnableStartup(startMinimized);
            }
            else
            {
                return DisableStartup();
            }
        }

        /// <summary>
        /// Gets the full path to the application executable
        /// </summary>
        private string GetExecutablePath()
        {
            try
            {
                // Get the location of the executing assembly
                var assembly = Assembly.GetExecutingAssembly();
                var location = assembly.Location;

                // If running as a single-file app, use Process.GetCurrentProcess().MainModule.FileName
                if (string.IsNullOrEmpty(location) || location.EndsWith(".dll", StringComparison.OrdinalIgnoreCase))
                {
                    var process = Process.GetCurrentProcess();
                    location = process.MainModule?.FileName ?? string.Empty;
                }

                // Ensure we have an .exe path
                if (!string.IsNullOrEmpty(location) && location.EndsWith(".dll", StringComparison.OrdinalIgnoreCase))
                {
                    location = Path.ChangeExtension(location, ".exe");
                }

                return location;
            }
            catch (Exception ex)
            {
                Debug.WriteLine($"Failed to get executable path: {ex.Message}");
                return string.Empty;
            }
        }

        /// <summary>
        /// Gets the current startup command line from the registry
        /// </summary>
        /// <returns>The command line string, or null if not set</returns>
        public string? GetStartupCommandLine()
        {
            try
            {
                using var key = Registry.CurrentUser.OpenSubKey(RegistryKeyPath, writable: false);
                return key?.GetValue(AppName) as string;
            }
            catch (Exception ex)
            {
                Debug.WriteLine($"Failed to get startup command line: {ex.Message}");
                return null;
            }
        }
    }
}
