using System;
using System.ComponentModel;
using System.IO;
using System.Runtime.InteropServices;
using System.Text.Json;
using System.Threading.Tasks;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Interop;
using System.Windows.Media;
using DeeMusic.Desktop.Models;
using DeeMusic.Desktop.Services;
using DeeMusic.Desktop.ViewModels;

namespace DeeMusic.Desktop
{
    /// <summary>
    /// Interaction logic for MainWindow.xaml
    /// </summary>
    public partial class MainWindow : Window
    {
        private bool _minimizeToTray = true;
        private bool _isClosing = false;
        
        // CRITICAL: Static field to prevent GC from collecting the service
        // The service MUST live for the entire application lifetime
        private static DeeMusicService? _staticService;
        private readonly DeeMusicService _service;

        // P/Invoke for proper maximize behavior
        [DllImport("user32.dll")]
        private static extern bool GetMonitorInfo(IntPtr hMonitor, ref MONITORINFO lpmi);

        [DllImport("user32.dll")]
        private static extern IntPtr MonitorFromWindow(IntPtr hwnd, uint dwFlags);

        private const uint MONITOR_DEFAULTTONEAREST = 2;

        [StructLayout(LayoutKind.Sequential)]
        public struct RECT
        {
            public int Left;
            public int Top;
            public int Right;
            public int Bottom;
        }

        [StructLayout(LayoutKind.Sequential)]
        public struct MONITORINFO
        {
            public uint Size;
            public RECT Monitor;
            public RECT WorkArea;
            public uint Flags;
        }

        public MainWindow()
        {
            InitializeComponent();
            
            try
            {
                // Initialize notification service
                NotificationService.Instance.Initialize(RootGrid);
                
                // Initialize DataContext with DeeMusicService
                // IMPORTANT: Use static field to absolutely prevent garbage collection
                if (_staticService == null)
                {
                    _staticService = new DeeMusicService();
                }
                _service = _staticService;
                
                var mainViewModel = new MainViewModel(_service);
                DataContext = mainViewModel;
                
                // Subscribe to settings requested event
                mainViewModel.ShowSettingsRequested += (s, e) => SettingsButton_Click(this, new RoutedEventArgs());
                
                // Subscribe to search box events to enable search from any page
                SearchBox.GotFocus += SearchBox_GotFocus;
                SearchBox.PreviewKeyDown += SearchBox_PreviewKeyDown;
                SearchBox.TextChanged += SearchBox_TextChanged;
                
                SearchBox.Focus();
                
                // Load minimize to tray setting
                LoadMinimizeToTraySetting();
                
                // Check if initial setup is needed
                Loaded += MainWindow_Loaded;
                
                // Maximize window on startup (respecting taskbar)
                Loaded += (s, e) =>
                {
                    MaximizeToWorkArea();
                };
                
                // Prevent actual WindowState.Maximized from covering taskbar
                StateChanged += MainWindow_StateChanged;
            }
            catch (Exception ex)
            {
                MessageBox.Show(
                    $"Failed to initialize application:\n\n{ex.Message}\n\nInner Exception: {ex.InnerException?.Message}\n\nStack Trace:\n{ex.StackTrace}",
                    "Initialization Error",
                    MessageBoxButton.OK,
                    MessageBoxImage.Error);
                throw;
            }
        }

        private async void MainWindow_Loaded(object sender, RoutedEventArgs e)
        {
            var mainViewModel = DataContext as MainViewModel;
            if (mainViewModel == null)
            {
                LoggingService.Instance.LogError("MainViewModel is null in MainWindow_Loaded");
                return;
            }
            
            LoggingService.Instance.LogInfo("MainWindow_Loaded - checking settings before initialization");
            
            // Check if required settings are configured BEFORE initializing backend
            // This prevents the backend from loading a stale config
            bool settingsConfigured = await AreRequiredSettingsConfigured();
            LoggingService.Instance.LogInfo($"Settings configured: {settingsConfigured}");
            
            if (!settingsConfigured)
            {
                LoggingService.Instance.LogInfo("Settings not configured, showing welcome dialog");
                ShowWelcomeSettingsDialog();
                
                // After welcome dialog, check again
                settingsConfigured = await AreRequiredSettingsConfigured();
            }
            
            // Now initialize the backend with the correct settings
            if (settingsConfigured)
            {
                LoggingService.Instance.LogInfo("Initializing backend with configured settings");
                await mainViewModel.InitializeAsync();
                
                // Check for updates if enabled
                CheckForUpdatesOnStartup(mainViewModel);
            }
            else
            {
                LoggingService.Instance.LogWarning("Settings still not configured after welcome dialog");
            }
        }

        /// <summary>
        /// Check for updates on startup if enabled
        /// </summary>
        private async void CheckForUpdatesOnStartup(MainViewModel mainViewModel)
        {
            try
            {
                if (mainViewModel.SettingsViewModel.CheckForUpdates)
                {
                    // Check for updates every time the app opens
                    LoggingService.Instance.LogInfo("Checking for updates on startup...");
                    
                    var updateInfo = await UpdateService.Instance.CheckForUpdatesAsync();
                    
                    // Update last check time
                    mainViewModel.SettingsViewModel.Settings.System.LastUpdateCheck = DateTime.Now;
                    await mainViewModel.SettingsViewModel.ForceSaveAsync();
                    
                    if (updateInfo != null)
                    {
                        LoggingService.Instance.LogInfo($"Update available: v{updateInfo.Version}");
                        
                        // Show update notification
                        NotificationService.Instance.ShowInfo($"Update available: v{updateInfo.Version}");
                        
                        // Show update dialog on UI thread
                        await Dispatcher.InvokeAsync(async () =>
                        {
                            var result = MessageBox.Show(
                                this,
                                $"🎉 A new version of DeeMusic is available!\n\n" +
                                $"Current Version: v{mainViewModel.SettingsViewModel.CurrentVersion}\n" +
                                $"New Version: v{updateInfo.Version}\n\n" +
                                $"Would you like to update now?\n\n" +
                                $"The update will download in the background and restart the app when ready.",
                                "Update Available",
                                MessageBoxButton.YesNo,
                                MessageBoxImage.Information);

                            if (result == MessageBoxResult.Yes)
                            {
                                try
                                {
                                    // Show downloading notification
                                    NotificationService.Instance.ShowInfo("Downloading update in background...");
                                    LoggingService.Instance.LogInfo("User accepted update, starting download...");
                                    
                                    var progress = new Progress<int>(p =>
                                    {
                                        if (p % 20 == 0 || p == 100) // Update every 20%
                                        {
                                            NotificationService.Instance.ShowInfo($"Downloading update: {p}%");
                                            LoggingService.Instance.LogInfo($"Update download progress: {p}%");
                                        }
                                    });
                                    
                                    var downloadPath = await UpdateService.Instance.DownloadUpdateAsync(updateInfo, progress);
                                    
                                    if (downloadPath != null)
                                    {
                                        LoggingService.Instance.LogInfo($"Update downloaded successfully to: {downloadPath}");
                                        NotificationService.Instance.ShowSuccess("Update downloaded! Restarting...");
                                        
                                        // Give user a moment to see the success message
                                        await Task.Delay(1000);
                                        
                                        // Apply update (this will restart the app)
                                        LoggingService.Instance.LogInfo("Applying update and restarting application...");
                                        UpdateService.Instance.ApplyUpdate(downloadPath);
                                    }
                                    else
                                    {
                                        LoggingService.Instance.LogError("Update download returned null path");
                                        NotificationService.Instance.ShowError("Failed to download update");
                                        MessageBox.Show(
                                            this,
                                            "Failed to download the update.\n\nPlease try again later or download manually from GitHub.",
                                            "Update Error",
                                            MessageBoxButton.OK,
                                            MessageBoxImage.Error);
                                    }
                                }
                                catch (Exception ex)
                                {
                                    LoggingService.Instance.LogError("Failed to download/install update", ex);
                                    NotificationService.Instance.ShowError("Update failed");
                                    MessageBox.Show(
                                        this,
                                        $"Failed to update: {ex.Message}\n\nPlease try again later or download manually from GitHub.",
                                        "Update Error",
                                        MessageBoxButton.OK,
                                        MessageBoxImage.Error);
                                }
                            }
                            else
                            {
                                LoggingService.Instance.LogInfo("User declined update");
                                NotificationService.Instance.ShowInfo("Update postponed. Check Settings to update later.");
                            }
                        });
                    }
                    else
                    {
                        LoggingService.Instance.LogInfo("No updates available");
                    }
                }
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to check for updates on startup", ex);
            }
        }

        /// <summary>
        /// Check if required settings (ARL and download path) are configured
        /// </summary>
        private async System.Threading.Tasks.Task<bool> AreRequiredSettingsConfigured()
        {
            try
            {
                var mainViewModel = DataContext as MainViewModel;
                if (mainViewModel?.SettingsViewModel == null)
                    return false;

                // Load settings
                await mainViewModel.SettingsViewModel.LoadSettingsAsync();

                var settings = mainViewModel.SettingsViewModel.Settings;
                
                var arl = settings?.Deezer?.ARL;
                
                // Check if ARL is set and not a placeholder
                bool hasArl = !string.IsNullOrWhiteSpace(arl) && 
                             arl != "CREDENTIAL_MANAGER" &&
                             arl != "your_arl_token_here" &&
                             arl.Length >= 100; // Real ARL tokens are typically 192 characters
                
                // Check if download path is set
                bool hasDownloadPath = !string.IsNullOrWhiteSpace(settings?.Download?.OutputDir);
                
                LoggingService.Instance.LogInfo($"ARL check: hasArl={hasArl}, arl='{arl?.Substring(0, Math.Min(20, arl?.Length ?? 0))}...', length={arl?.Length ?? 0}");
                LoggingService.Instance.LogInfo($"Download path check: hasDownloadPath={hasDownloadPath}, path='{settings?.Download?.OutputDir}'");
                
                return hasArl && hasDownloadPath;
            }
            catch
            {
                return false;
            }
        }

        /// <summary>
        /// Show welcome dialog with settings
        /// </summary>
        private async void ShowWelcomeSettingsDialog()
        {
            var mainViewModel = DataContext as MainViewModel;
            if (mainViewModel?.SettingsViewModel == null)
                return;

            // Create welcome setup dialog
            var inputDialog = new Window
            {
                Title = "Welcome to DeeMusic - Initial Setup",
                Width = 650,
                Height = 550,
                WindowStartupLocation = WindowStartupLocation.CenterOwner,
                Owner = this,
                Background = Background,
                Foreground = Foreground,
                ResizeMode = ResizeMode.NoResize
            };

            var grid = new Grid { Margin = new Thickness(20) };
            grid.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto }); // Welcome text
            grid.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto }); // ARL label
            grid.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto }); // ARL textbox
            grid.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto }); // Download path label
            grid.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto }); // Download path + browse
            grid.RowDefinitions.Add(new RowDefinition { Height = new GridLength(1, GridUnitType.Star) }); // Instructions
            grid.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto }); // Buttons

            var welcomeText = new TextBlock
            {
                Text = "Welcome to DeeMusic!\n\nPlease configure the required settings to get started:",
                TextWrapping = TextWrapping.Wrap,
                FontSize = 14,
                Margin = new Thickness(0, 0, 0, 20)
            };
            Grid.SetRow(welcomeText, 0);

            var arlLabel = new TextBlock
            {
                Text = "Deezer ARL Token (Required):",
                FontWeight = FontWeights.SemiBold,
                Margin = new Thickness(0, 0, 0, 8)
            };
            Grid.SetRow(arlLabel, 1);

            var arlTextBox = new TextBox
            {
                Margin = new Thickness(0, 0, 0, 16),
                Padding = new Thickness(8),
                FontSize = 13
            };
            Grid.SetRow(arlTextBox, 2);

            var downloadPathLabel = new TextBlock
            {
                Text = "Download Directory:",
                FontWeight = FontWeights.SemiBold,
                Margin = new Thickness(0, 0, 0, 8)
            };
            Grid.SetRow(downloadPathLabel, 3);

            var downloadPathGrid = new Grid { Margin = new Thickness(0, 0, 0, 16) };
            downloadPathGrid.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
            downloadPathGrid.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
            
            var defaultDownloadPath = System.IO.Path.Combine(
                Environment.GetFolderPath(Environment.SpecialFolder.MyMusic),
                "DeeMusic");
            
            var downloadPathTextBox = new TextBox
            {
                Text = defaultDownloadPath,
                Padding = new Thickness(8),
                FontSize = 13,
                IsReadOnly = true
            };
            Grid.SetColumn(downloadPathTextBox, 0);

            var browseButton = new Button
            {
                Content = "Browse",
                Padding = new Thickness(15, 8, 15, 8),
                Margin = new Thickness(8, 0, 0, 0),
                FontSize = 13
            };
            Grid.SetColumn(browseButton, 1);
            
            browseButton.Click += (s, e) =>
            {
                var dialog = new System.Windows.Forms.FolderBrowserDialog
                {
                    Description = "Select download directory",
                    SelectedPath = downloadPathTextBox.Text,
                    ShowNewFolderButton = true
                };
                
                if (dialog.ShowDialog() == System.Windows.Forms.DialogResult.OK)
                {
                    downloadPathTextBox.Text = dialog.SelectedPath;
                }
            };

            downloadPathGrid.Children.Add(downloadPathTextBox);
            downloadPathGrid.Children.Add(browseButton);
            Grid.SetRow(downloadPathGrid, 4);

            var instructionsText = new TextBlock
            {
                Text = "How to find your ARL token:\n" +
                       "1. Log into deezer.com in your browser\n" +
                       "2. Open Developer Tools (F12)\n" +
                       "3. Go to Application → Cookies\n" +
                       "4. Copy the 'arl' cookie value\n\n" +
                       "You can change other settings later from the Settings menu.",
                TextWrapping = TextWrapping.Wrap,
                FontSize = 12,
                Foreground = Brushes.Gray,
                Margin = new Thickness(0, 0, 0, 20)
            };
            Grid.SetRow(instructionsText, 5);

            var buttonPanel = new StackPanel 
            { 
                Orientation = Orientation.Horizontal,
                HorizontalAlignment = HorizontalAlignment.Right
            };
            
            var moreSettingsButton = new Button
            {
                Content = "More Settings",
                Padding = new Thickness(20, 10, 20, 10),
                FontSize = 14,
                Margin = new Thickness(0, 0, 10, 0)
            };
            
            var skipButton = new Button
            {
                Content = "Skip",
                Padding = new Thickness(20, 10, 20, 10),
                FontSize = 14,
                Margin = new Thickness(0, 0, 10, 0)
            };

            var saveButton = new Button
            {
                Content = "Save and Continue",
                Padding = new Thickness(20, 10, 20, 10),
                FontSize = 14
            };
            
            buttonPanel.Children.Add(moreSettingsButton);
            buttonPanel.Children.Add(skipButton);
            buttonPanel.Children.Add(saveButton);
            Grid.SetRow(buttonPanel, 6);
            
            LoggingService.Instance.LogInfo("Welcome dialog buttons created");

            bool saved = false;
            
            // More Settings button - opens full settings dialog
            moreSettingsButton.Click += (s, e) =>
            {
                inputDialog.Close();
                SettingsButton_Click(this, new RoutedEventArgs());
            };
            
            // Skip button - closes dialog without saving
            skipButton.Click += (s, e) =>
            {
                var result = MessageBox.Show(
                    "Are you sure you want to skip setup? You'll need to configure settings later to use DeeMusic.",
                    "Skip Setup",
                    MessageBoxButton.YesNo,
                    MessageBoxImage.Question);
                    
                if (result == MessageBoxResult.Yes)
                {
                    inputDialog.Close();
                }
            };
            
            // Save button - saves ARL and download path
            saveButton.Click += async (s, e) =>
            {
                try
                {
                    LoggingService.Instance.LogInfo("=== Save button clicked ===");
                    
                    var arl = arlTextBox.Text.Trim();
                    var downloadPath = downloadPathTextBox.Text.Trim();
                    
                    LoggingService.Instance.LogInfo($"ARL entered, length: {arl.Length}");
                    LoggingService.Instance.LogInfo($"Download path: {downloadPath}");
                    
                    if (string.IsNullOrWhiteSpace(arl))
                    {
                        MessageBox.Show("Please enter your ARL token.", "Required", MessageBoxButton.OK, MessageBoxImage.Warning);
                        return;
                    }

                    if (arl.Length < 100)
                    {
                        var result = MessageBox.Show("The ARL token seems too short. Are you sure this is correct?", "Confirm", MessageBoxButton.YesNo, MessageBoxImage.Question);
                        if (result == MessageBoxResult.No)
                            return;
                    }
                    
                    if (string.IsNullOrWhiteSpace(downloadPath))
                    {
                        MessageBox.Show("Please select a download directory.", "Required", MessageBoxButton.OK, MessageBoxImage.Warning);
                        return;
                    }

                    // Save the ARL and download path
                    LoggingService.Instance.LogInfo($"Setting ARL property: {arl.Substring(0, Math.Min(20, arl.Length))}...");
                    mainViewModel.SettingsViewModel.DeezerARL = arl;
                    mainViewModel.SettingsViewModel.DownloadPath = downloadPath;
                    
                    LoggingService.Instance.LogInfo($"ARL property set, current value: {mainViewModel.SettingsViewModel.DeezerARL.Substring(0, Math.Min(20, mainViewModel.SettingsViewModel.DeezerARL.Length))}...");
                    LoggingService.Instance.LogInfo($"Download path set: {mainViewModel.SettingsViewModel.DownloadPath}");
                    LoggingService.Instance.LogInfo($"HasUnsavedChanges: {mainViewModel.SettingsViewModel.HasUnsavedChanges}");

                    // Call the save method directly
                    LoggingService.Instance.LogInfo("Calling ForceSaveAsync...");
                    await mainViewModel.SettingsViewModel.ForceSaveAsync();
                    LoggingService.Instance.LogInfo("ForceSaveAsync completed");
                    
                    // Immediately reload to ensure we have the saved value
                    LoggingService.Instance.LogInfo("Reloading settings to verify save...");
                    await mainViewModel.SettingsViewModel.LoadSettingsAsync();
                    LoggingService.Instance.LogInfo($"Settings reloaded, ARL length: {mainViewModel.SettingsViewModel.DeezerARL.Length}");
                    
                    saved = true;
                    LoggingService.Instance.LogInfo("Save completed successfully");
                    inputDialog.Close();
                }
                catch (Exception ex)
                {
                    LoggingService.Instance.LogError("Error in save button click", ex);
                    MessageBox.Show($"Error saving settings: {ex.Message}", "Error", MessageBoxButton.OK, MessageBoxImage.Error);
                }
            };

            grid.Children.Add(welcomeText);
            grid.Children.Add(arlLabel);
            grid.Children.Add(arlTextBox);
            grid.Children.Add(downloadPathLabel);
            grid.Children.Add(downloadPathGrid);
            grid.Children.Add(instructionsText);
            grid.Children.Add(buttonPanel);

            inputDialog.Content = grid;
            
            LoggingService.Instance.LogInfo("Showing welcome input dialog");
            System.Diagnostics.Debug.WriteLine("Showing welcome input dialog");
            inputDialog.ShowDialog();
            LoggingService.Instance.LogInfo($"Welcome input dialog closed, saved: {saved}");
            System.Diagnostics.Debug.WriteLine($"Welcome input dialog closed, saved: {saved}");
            
            if (saved)
            {
                // After dialog closes, re-check configuration and refresh UI
                await mainViewModel.SearchViewModel.InitializeAsync();
            }
        }

        /// <summary>
        /// Maximize window to work area (respecting taskbar)
        /// </summary>
        private void MaximizeToWorkArea()
        {
            var hwnd = new WindowInteropHelper(this).Handle;
            var monitor = MonitorFromWindow(hwnd, MONITOR_DEFAULTTONEAREST);
            
            if (monitor != IntPtr.Zero)
            {
                var monitorInfo = new MONITORINFO { Size = (uint)Marshal.SizeOf(typeof(MONITORINFO)) };
                if (GetMonitorInfo(monitor, ref monitorInfo))
                {
                    var workArea = monitorInfo.WorkArea;
                    
                    // Set window position and size to work area (excludes taskbar)
                    Left = workArea.Left;
                    Top = workArea.Top;
                    Width = workArea.Right - workArea.Left;
                    Height = workArea.Bottom - workArea.Top;
                    
                    WindowState = WindowState.Normal; // Keep as Normal but sized to work area
                }
            }
        }

        /// <summary>
        /// Handle window state changes to prevent covering taskbar
        /// </summary>
        private void MainWindow_StateChanged(object? sender, EventArgs e)
        {
            // Allow minimization to work normally
            if (WindowState == WindowState.Minimized)
            {
                return;
            }
            
            // If someone tries to maximize the window, maximize to work area instead
            if (WindowState == WindowState.Maximized)
            {
                WindowState = WindowState.Normal;
                MaximizeToWorkArea();
            }
        }

        private void TopBar_MouseLeftButtonDown(object sender, MouseButtonEventArgs e)
        {
            if (e.ClickCount == 2)
            {
                if (WindowState == WindowState.Normal && 
                    Width == SystemParameters.WorkArea.Width && 
                    Height == SystemParameters.WorkArea.Height)
                {
                    // Currently "maximized" to work area, restore to default size
                    Width = 1280;
                    Height = 900;
                    WindowStartupLocation = WindowStartupLocation.CenterScreen;
                }
                else
                {
                    // Maximize to work area
                    MaximizeToWorkArea();
                }
            }
            else
            {
                DragMove();
            }
        }

        private void MinimizeButton_Click(object sender, RoutedEventArgs e)
        {
            WindowState = WindowState.Minimized;
        }

        private void MaximizeButton_Click(object sender, RoutedEventArgs e)
        {
            WindowState = WindowState == WindowState.Maximized ? WindowState.Normal : WindowState.Maximized;
        }

        private void CloseButton_Click(object sender, RoutedEventArgs e)
        {
            // Always close the application immediately (tray disabled for debugging)
            _isClosing = true;
            
            // Get current process ID before closing
            var currentPid = System.Diagnostics.Process.GetCurrentProcess().Id;
            
            try
            {
                Close();
            }
            catch
            {
                // If normal close fails, force terminate
                Application.Current.Shutdown();
            }
            
            // Nuclear option: Use taskkill to force terminate the process tree
            // This ensures no background threads or DLLs keep the process alive
            try
            {
                // Kill the main process
                var psi = new System.Diagnostics.ProcessStartInfo
                {
                    FileName = "taskkill",
                    Arguments = $"/F /PID {currentPid} /T",
                    CreateNoWindow = true,
                    UseShellExecute = false,
                    RedirectStandardOutput = true,
                    RedirectStandardError = true
                };
                System.Diagnostics.Process.Start(psi);
                
                // Also kill any lingering dotnet.exe and conhost.exe processes
                // These are spawned by the Go DLL and don't die with the main process
                System.Threading.Tasks.Task.Run(() =>
                {
                    System.Threading.Thread.Sleep(100); // Wait for main process to die
                    
                    try
                    {
                        var killDotnet = new System.Diagnostics.ProcessStartInfo
                        {
                            FileName = "taskkill",
                            Arguments = "/F /IM dotnet.exe",
                            CreateNoWindow = true,
                            UseShellExecute = false
                        };
                        System.Diagnostics.Process.Start(killDotnet);
                    }
                    catch { }
                    
                    try
                    {
                        var killConhost = new System.Diagnostics.ProcessStartInfo
                        {
                            FileName = "taskkill",
                            Arguments = "/F /IM conhost.exe",
                            CreateNoWindow = true,
                            UseShellExecute = false
                        };
                        System.Diagnostics.Process.Start(killConhost);
                    }
                    catch { }
                });
            }
            catch
            {
                // Ignore errors - the process might already be dead
            }
        }

        private async void SettingsButton_Click(object sender, RoutedEventArgs e)
        {
            var mainViewModel = DataContext as ViewModels.MainViewModel;
            if (mainViewModel?.SettingsViewModel == null)
                return;

            // Don't reload settings - use what's already in memory
            // Reloading would overwrite any unsaved changes and could load stale data from backend
            LoggingService.Instance.LogInfo("Opening settings dialog");

            // Calculate safe window size based on work area (screen minus taskbar)
            var workArea = SystemParameters.WorkArea;
            var maxWidth = workArea.Width * 0.9; // 90% of work area width
            var maxHeight = workArea.Height * 0.9; // 90% of work area height
            
            var settingsWidth = Math.Min(1000, maxWidth); // Increased from 900 to fit all tabs
            var settingsHeight = Math.Min(900, maxHeight);

            // Create and show settings dialog
            var settingsWindow = new Window
            {
                Title = "Settings",
                Width = settingsWidth,
                Height = settingsHeight,
                WindowStartupLocation = WindowStartupLocation.CenterScreen, // Use CenterScreen instead of CenterOwner
                Owner = this,
                Content = new Views.SettingsView { DataContext = mainViewModel.SettingsViewModel },
                Background = Background,
                Foreground = Foreground,
                ResizeMode = ResizeMode.CanResize,
                MinWidth = Math.Min(850, maxWidth), // Increased from 700 to ensure all tabs are visible
                MinHeight = Math.Min(600, maxHeight),
                MaxWidth = maxWidth, // Prevent window from exceeding work area
                MaxHeight = maxHeight
            };
            
            // Ensure window stays within work area bounds
            settingsWindow.Loaded += (s, args) =>
            {
                // Double-check position after window is loaded
                if (settingsWindow.Left < workArea.Left)
                    settingsWindow.Left = workArea.Left;
                if (settingsWindow.Top < workArea.Top)
                    settingsWindow.Top = workArea.Top;
                if (settingsWindow.Left + settingsWindow.ActualWidth > workArea.Right)
                    settingsWindow.Left = workArea.Right - settingsWindow.ActualWidth;
                if (settingsWindow.Top + settingsWindow.ActualHeight > workArea.Bottom)
                    settingsWindow.Top = workArea.Bottom - settingsWindow.ActualHeight;
            };
            
            settingsWindow.ShowDialog();
            
            // After dialog closes, re-check configuration in case settings were updated
            // Trigger a refresh of the SearchViewModel to update the UI
            if (mainViewModel.IsInitialized)
            {
                await mainViewModel.SearchViewModel.InitializeAsync();
            }
        }

        protected override void OnClosing(CancelEventArgs e)
        {
            // Always close immediately (tray disabled for debugging)
            _isClosing = true;
            
            try
            {
                LoggingService.Instance.LogInfo("MainWindow closing - cleaning up");
                
                // Dispose tray service if it exists
                if (Tag is TrayService trayService)
                {
                    trayService.Dispose();
                }
                
                // Dispose MainViewModel and its services
                if (DataContext is MainViewModel mainViewModel)
                {
                    // This will dispose QueueViewModel and DeeMusicService
                    mainViewModel.WindowClosingCommand?.Execute(null);
                }
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Error during window closing", ex);
            }
            
            base.OnClosing(e);
        }

        /// <summary>
        /// Load the minimize to tray setting from configuration
        /// </summary>
        private void LoadMinimizeToTraySetting()
        {
            try
            {
                var settingsPath = GetSettingsPath();
                if (File.Exists(settingsPath))
                {
                    var json = File.ReadAllText(settingsPath);
                    var settings = JsonSerializer.Deserialize<Settings>(json);
                    
                    if (settings?.System != null)
                    {
                        _minimizeToTray = settings.System.MinimizeToTray;
                    }
                }
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Error loading minimize to tray setting: {ex.Message}");
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

        /// <summary>
        /// Update the minimize to tray setting
        /// </summary>
        public void UpdateMinimizeToTray(bool minimizeToTray)
        {
            _minimizeToTray = minimizeToTray;
        }

        /// <summary>
        /// Handle search box getting focus - navigate to search view when clicked
        /// </summary>
        private void SearchBox_GotFocus(object sender, RoutedEventArgs e)
        {
            try
            {
                var mainViewModel = DataContext as MainViewModel;
                if (mainViewModel == null)
                {
                    LoggingService.Instance.LogWarning("SearchBox_GotFocus: MainViewModel is null");
                    return;
                }

                bool isOnSearchView = mainViewModel.CurrentView == mainViewModel.SearchViewModel;
                LoggingService.Instance.LogInfo($"SearchBox_GotFocus: CurrentPage={mainViewModel.CurrentPage}, IsOnSearchView={isOnSearchView}");

                // Navigate to search view when user clicks the search box
                // Use Dispatcher to delay navigation until after current input is processed
                if (!isOnSearchView)
                {
                    Dispatcher.BeginInvoke(new Action(() =>
                    {
                        mainViewModel.NavigateCommand?.Execute("Search");
                        // Restore focus to search box after navigation
                        SearchBox.Focus();
                    }), System.Windows.Threading.DispatcherPriority.Input);
                }
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("SearchBox_GotFocus error", ex);
            }
        }

        /// <summary>
        /// Handle text changes in search box - not used, navigation happens on focus
        /// </summary>
        private void SearchBox_TextChanged(object sender, TextChangedEventArgs e)
        {
            // Empty - navigation happens in GotFocus when user clicks the search box
            // Typing will work normally since we're already on the search view
        }

        /// <summary>
        /// Handle key presses in search box - execute search on Enter
        /// </summary>
        private void SearchBox_PreviewKeyDown(object sender, KeyEventArgs e)
        {
            try
            {
                var mainViewModel = DataContext as MainViewModel;
                if (mainViewModel == null)
                    return;

                // If user presses Enter, execute search regardless of current view
                if (e.Key == Key.Enter)
                {
                    bool isOnSearchView = mainViewModel.CurrentView == mainViewModel.SearchViewModel;
                    if (!isOnSearchView)
                    {
                        // Navigate to search view first
                        mainViewModel.NavigateCommand?.Execute("Search");
                    }
                    // Execute search command
                    mainViewModel.SearchViewModel?.SearchCommand?.Execute(null);
                    e.Handled = true;
                    return;
                }
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("SearchBox_PreviewKeyDown error", ex);
            }
        }
    }
}
