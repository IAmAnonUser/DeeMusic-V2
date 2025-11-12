using System;
using System.ComponentModel;
using System.IO;
using System.Text.Json;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
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
            }
            else
            {
                LoggingService.Instance.LogWarning("Settings still not configured after welcome dialog");
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

            // Create a simple input dialog for ARL
            var inputDialog = new Window
            {
                Title = "Welcome to DeeMusic - Initial Setup",
                Width = 600,
                Height = 400,
                WindowStartupLocation = WindowStartupLocation.CenterOwner,
                Owner = this,
                Background = Background,
                Foreground = Foreground,
                ResizeMode = ResizeMode.NoResize
            };

            var grid = new Grid { Margin = new Thickness(20) };
            grid.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
            grid.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
            grid.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
            grid.RowDefinitions.Add(new RowDefinition { Height = new GridLength(1, GridUnitType.Star) });
            grid.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });

            var welcomeText = new TextBlock
            {
                Text = "Welcome to DeeMusic!\n\nTo get started, please enter your Deezer ARL token:",
                TextWrapping = TextWrapping.Wrap,
                FontSize = 14,
                Margin = new Thickness(0, 0, 0, 20)
            };
            Grid.SetRow(welcomeText, 0);

            var arlLabel = new TextBlock
            {
                Text = "Deezer ARL Token:",
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

            var instructionsText = new TextBlock
            {
                Text = "How to find your ARL token:\n" +
                       "1. Log into deezer.com in your browser\n" +
                       "2. Open Developer Tools (F12)\n" +
                       "3. Go to Application → Cookies\n" +
                       "4. Copy the 'arl' cookie value",
                TextWrapping = TextWrapping.Wrap,
                FontSize = 12,
                Foreground = Brushes.Gray,
                Margin = new Thickness(0, 0, 0, 20)
            };
            Grid.SetRow(instructionsText, 3);

            var saveButton = new Button
            {
                Content = "Save and Continue",
                Padding = new Thickness(20, 10, 20, 10),
                FontSize = 14,
                HorizontalAlignment = HorizontalAlignment.Right
            };
            Grid.SetRow(saveButton, 4);
            
            LoggingService.Instance.LogInfo("Save button created");

            bool saved = false;
            saveButton.Click += async (s, e) =>
            {
                try
                {
                    LoggingService.Instance.LogInfo("=== Save button clicked ===");
                    System.Diagnostics.Debug.WriteLine("=== Save button clicked ===");
                    
                    var arl = arlTextBox.Text.Trim();
                    LoggingService.Instance.LogInfo($"ARL entered, length: {arl.Length}");
                    
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

                    // Save the ARL directly
                    LoggingService.Instance.LogInfo($"Setting ARL property: {arl.Substring(0, Math.Min(20, arl.Length))}...");
                    mainViewModel.SettingsViewModel.DeezerARL = arl;
                    LoggingService.Instance.LogInfo($"ARL property set, current value: {mainViewModel.SettingsViewModel.DeezerARL.Substring(0, Math.Min(20, mainViewModel.SettingsViewModel.DeezerARL.Length))}...");
                    LoggingService.Instance.LogInfo($"HasUnsavedChanges: {mainViewModel.SettingsViewModel.HasUnsavedChanges}");
                    
                    // Ensure download path is set
                    if (string.IsNullOrWhiteSpace(mainViewModel.SettingsViewModel.DownloadPath))
                    {
                        var defaultPath = System.IO.Path.Combine(
                            Environment.GetFolderPath(Environment.SpecialFolder.MyMusic),
                            "DeeMusic");
                        mainViewModel.SettingsViewModel.DownloadPath = defaultPath;
                        LoggingService.Instance.LogInfo($"Set default download path: {defaultPath}");
                    }

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
            grid.Children.Add(instructionsText);
            grid.Children.Add(saveButton);

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

        private void TopBar_MouseLeftButtonDown(object sender, MouseButtonEventArgs e)
        {
            if (e.ClickCount == 2)
            {
                WindowState = WindowState == WindowState.Maximized ? WindowState.Normal : WindowState.Maximized;
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

            // Create and show settings dialog
            var settingsWindow = new Window
            {
                Title = "Settings",
                Width = 700,
                Height = 750,
                WindowStartupLocation = WindowStartupLocation.CenterOwner,
                Owner = this,
                Content = new Views.SettingsView { DataContext = mainViewModel.SettingsViewModel },
                Background = Background,
                Foreground = Foreground,
                ResizeMode = ResizeMode.CanResize,
                MinWidth = 600,
                MinHeight = 600
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
