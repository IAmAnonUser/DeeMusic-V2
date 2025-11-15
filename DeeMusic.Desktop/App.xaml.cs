using System;
using System.IO;
using System.Text.Json;
using System.Threading.Tasks;
using System.Windows;
using System.Windows.Threading;
using DeeMusic.Desktop.Services;
using DeeMusic.Desktop.Models;
using DeeMusic.Desktop.ViewModels;

namespace DeeMusic.Desktop
{
    /// <summary>
    /// Interaction logic for App.xaml
    /// </summary>
    public partial class App : Application
    {
        private TrayService? _trayService;
        private bool _startMinimized;

        protected override void OnStartup(StartupEventArgs e)
        {
            base.OnStartup(e);
            
            // Enable hardware acceleration for better performance
            System.Windows.Media.RenderOptions.ProcessRenderMode = System.Windows.Interop.RenderMode.Default;
            
            // Set up global exception handlers
            SetupExceptionHandlers();
            
            // Check for command line arguments
            CheckCommandLineArguments(e.Args);
            
            // Check for migration before showing main window
            CheckAndPerformMigration();
            
            // Initialize theme from settings
            InitializeTheme();
        }

        protected override void OnActivated(EventArgs e)
        {
            base.OnActivated(e);
            
            // Tray service disabled for debugging - app will close immediately
            // Mark startup as complete for performance monitoring
            PerformanceMonitor.Instance.MarkStartupComplete();
            PerformanceMonitor.Instance.LogCurrentMetrics("Startup Complete");
        }

        /// <summary>
        /// Checks command line arguments for startup options
        /// </summary>
        private void CheckCommandLineArguments(string[] args)
        {
            foreach (var arg in args)
            {
                if (arg.Equals("--minimized", StringComparison.OrdinalIgnoreCase) ||
                    arg.Equals("-minimized", StringComparison.OrdinalIgnoreCase) ||
                    arg.Equals("/minimized", StringComparison.OrdinalIgnoreCase))
                {
                    _startMinimized = true;
                    break;
                }
            }
        }

        protected override void OnExit(ExitEventArgs e)
        {
            try
            {
                LoggingService.Instance.LogInfo("Application exiting - cleaning up resources");
                
                // Note: Go backend shutdown is handled by DeeMusicService.Dispose()
                // which is called from MainWindow.OnClosing -> MainViewModel.OnWindowClosing
                
                // Cleanup tray service
                _trayService?.Dispose();
                
                // Give services time to cleanup
                System.Threading.Thread.Sleep(100);
                
                // Dispose logging service
                LoggingService.Instance.Dispose();
                
                // Force kill any remaining DeeMusic processes
                KillRemainingProcesses();
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Error during exit: {ex.Message}");
            }
            finally
            {
                base.OnExit(e);
            }
            
            // Force terminate the current process immediately
            // This ensures no background threads keep the process alive
            var currentProcess = System.Diagnostics.Process.GetCurrentProcess();
            try
            {
                currentProcess.Kill();
            }
            catch
            {
                // If Kill fails, use Environment.Exit as fallback
                System.Environment.Exit(0);
            }
        }
        
        /// <summary>
        /// Kill any remaining DeeMusic processes to ensure clean shutdown
        /// </summary>
        private void KillRemainingProcesses()
        {
            try
            {
                var currentProcess = System.Diagnostics.Process.GetCurrentProcess();
                var processes = System.Diagnostics.Process.GetProcessesByName("DeeMusic.Desktop");
                
                foreach (var process in processes)
                {
                    try
                    {
                        // Don't kill ourselves
                        if (process.Id != currentProcess.Id)
                        {
                            process.Kill();
                            process.WaitForExit(1000);
                        }
                    }
                    catch
                    {
                        // Ignore errors killing individual processes
                    }
                }
            }
            catch
            {
                // Ignore errors in cleanup
            }
        }
        
        /// <summary>
        /// Set up global exception handlers
        /// </summary>
        private void SetupExceptionHandlers()
        {
            // Handle unhandled exceptions in UI thread
            DispatcherUnhandledException += (sender, e) =>
            {
                LoggingService.Instance.LogCritical("Unhandled UI exception", e.Exception);
                ErrorHandler.HandleException(e.Exception, "UI Thread");
                e.Handled = true; // Prevent application crash
            };
            
            // Handle unhandled exceptions in background threads
            AppDomain.CurrentDomain.UnhandledException += (sender, e) =>
            {
                if (e.ExceptionObject is Exception ex)
                {
                    LoggingService.Instance.LogCritical("Unhandled domain exception", ex);
                    ErrorHandler.HandleException(ex, "Background Thread");
                }
            };
            
            // Handle task exceptions
            TaskScheduler.UnobservedTaskException += (sender, e) =>
            {
                LoggingService.Instance.LogCritical("Unobserved task exception", e.Exception);
                ErrorHandler.HandleException(e.Exception, "Task");
                e.SetObserved(); // Prevent application crash
            };
            
            LoggingService.Instance.LogInfo("Global exception handlers configured");
        }

        /// <summary>
        /// Initializes the application theme from settings
        /// </summary>
        private void InitializeTheme()
        {
            try
            {
                // Try to load theme from settings
                var settingsPath = GetSettingsPath();
                if (File.Exists(settingsPath))
                {
                    var json = File.ReadAllText(settingsPath);
                    var settings = JsonSerializer.Deserialize<Settings>(json);
                    
                    if (settings?.System?.Theme != null)
                    {
                        ThemeManager.Instance.Initialize(settings.System.Theme);
                        LoggingService.Instance.LogInfo($"Theme initialized: {settings.System.Theme}");
                        return;
                    }
                }
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Error loading theme from settings: {ex.Message}");
                LoggingService.Instance.LogWarning("Error loading theme from settings", ex);
            }

            // Default to dark theme if settings not found or error occurred
            ThemeManager.Instance.Initialize("dark");
            LoggingService.Instance.LogInfo("Theme initialized: dark (default)");
        }

        /// <summary>
        /// Initializes the system tray icon
        /// </summary>
        private void InitializeTray()
        {
            try
            {
                // Wait for MainWindow to be created
                if (MainWindow is MainWindow mainWindow && mainWindow.DataContext is MainViewModel mainViewModel)
                {
                    _trayService = new TrayService(mainViewModel);
                    _trayService.Initialize();
                    
                    // Store reference in MainWindow for access
                    mainWindow.Tag = _trayService;
                    
                    // Pass tray service to MainViewModel for notifications
                    mainViewModel.SetTrayService(_trayService);
                    
                    LoggingService.Instance.LogInfo("System tray initialized");
                }
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Error initializing tray: {ex.Message}");
                LoggingService.Instance.LogError("Error initializing tray", ex);
            }
        }

        /// <summary>
        /// Checks if migration is needed and shows migration window
        /// </summary>
        private async void CheckAndPerformMigration()
        {
            try
            {
                LoggingService.Instance.LogInfo("Checking for migration");
                var migrationService = new MigrationService();
                bool migrationNeeded = await migrationService.IsMigrationNeededAsync();

                if (migrationNeeded)
                {
                    LoggingService.Instance.LogInfo("Migration needed, showing migration window");
                    // Show migration window before main window
                    var migrationWindow = new Views.MigrationWindow();
                    migrationWindow.ShowDialog();
                }
                else
                {
                    LoggingService.Instance.LogInfo("No migration needed");
                }
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Error checking migration: {ex.Message}");
                LoggingService.Instance.LogError("Error checking migration", ex);
                // Continue with normal startup even if migration check fails
            }
        }

        /// <summary>
        /// Gets the path to the settings file
        /// </summary>
        private string GetSettingsPath()
        {
            var appDataPath = Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData);
            var deeMusicPath = Path.Combine(appDataPath, "DeeMusicV2");
            return Path.Combine(deeMusicPath, "settings.json");
        }
    }
}
