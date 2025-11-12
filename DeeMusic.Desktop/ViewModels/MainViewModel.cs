using System;
using System.ComponentModel;
using System.Runtime.CompilerServices;
using System.Windows.Input;
using CommunityToolkit.Mvvm.Input;
using DeeMusic.Desktop.Services;

namespace DeeMusic.Desktop.ViewModels
{
    /// <summary>
    /// Main ViewModel for the application
    /// Manages navigation, application state, and window lifecycle
    /// </summary>
    public class MainViewModel : INotifyPropertyChanged
    {
        private readonly DeeMusicService _service;
        private object? _currentView;
        private string _currentPage = "Search";
        private bool _isInitialized;
        private readonly System.Collections.Generic.Stack<object> _navigationStack = new();

        public event PropertyChangedEventHandler? PropertyChanged;

        #region Properties

        /// <summary>
        /// Gets or sets the current view being displayed
        /// </summary>
        public object? CurrentView
        {
            get => _currentView;
            set
            {
                if (_currentView != value)
                {
                    _currentView = value;
                    OnPropertyChanged();
                }
            }
        }

        /// <summary>
        /// Gets or sets the current page name
        /// </summary>
        public string CurrentPage
        {
            get => _currentPage;
            set
            {
                if (_currentPage != value)
                {
                    _currentPage = value;
                    OnPropertyChanged();
                }
            }
        }

        /// <summary>
        /// Gets whether the application is initialized
        /// </summary>
        public bool IsInitialized
        {
            get => _isInitialized;
            private set
            {
                if (_isInitialized != value)
                {
                    _isInitialized = value;
                    OnPropertyChanged();
                }
            }
        }

        /// <summary>
        /// Gets the SearchViewModel instance
        /// </summary>
        public SearchViewModel SearchViewModel { get; }

        /// <summary>
        /// Gets the QueueViewModel instance
        /// </summary>
        public QueueViewModel QueueViewModel { get; }

        /// <summary>
        /// Gets the SettingsViewModel instance
        /// </summary>
        public SettingsViewModel SettingsViewModel { get; }

        #endregion

        #region Commands

        /// <summary>
        /// Command to navigate to a specific page
        /// </summary>
        public ICommand NavigateCommand { get; }

        /// <summary>
        /// Command to handle window closing
        /// </summary>
        public ICommand WindowClosingCommand { get; }

        /// <summary>
        /// Command to handle window loaded event
        /// </summary>
        public ICommand WindowLoadedCommand { get; }

        /// <summary>
        /// Command to toggle theme between dark and light
        /// </summary>
        public ICommand ToggleThemeCommand { get; }

        /// <summary>
        /// Command to navigate to settings
        /// </summary>
        public ICommand NavigateToSettingsCommand { get; }

        #endregion

        #region Events

        /// <summary>
        /// Event raised when settings dialog should be shown
        /// </summary>
        public event EventHandler? ShowSettingsRequested;

        #endregion

        public MainViewModel(DeeMusicService service)
        {
            _service = service ?? throw new ArgumentNullException(nameof(service));

            // Initialize child ViewModels
            SearchViewModel = new SearchViewModel(service);
            QueueViewModel = new QueueViewModel(service);
            SettingsViewModel = new SettingsViewModel(service);

            // Subscribe to events
            SearchViewModel.SettingsRequested += OnSettingsRequested;
            SearchViewModel.QueueRefreshRequested += OnQueueRefreshRequested;
            SearchViewModel.NavigateToAlbumRequested += OnNavigateToAlbumRequested;
            SearchViewModel.NavigateToArtistRequested += OnNavigateToArtistRequested;
            SearchViewModel.NavigateToPlaylistRequested += OnNavigateToPlaylistRequested;
            SettingsViewModel.SettingsSaved += OnSettingsSaved;

            // Initialize commands
            NavigateCommand = new RelayCommand<string>(NavigateTo);
            WindowClosingCommand = new RelayCommand(OnWindowClosing);
            WindowLoadedCommand = new RelayCommand(OnWindowLoaded);
            ToggleThemeCommand = new RelayCommand(ToggleTheme);
            NavigateToSettingsCommand = new RelayCommand(() => ShowSettingsRequested?.Invoke(this, EventArgs.Empty));

            // Set initial view
            CurrentView = SearchViewModel;
        }

        /// <summary>
        /// Set the tray service for notifications
        /// </summary>
        public void SetTrayService(Services.TrayService trayService)
        {
            QueueViewModel.SetTrayService(trayService);
        }

        #region Navigation

        /// <summary>
        /// Navigate to a specific page
        /// </summary>
        private async void NavigateTo(string? pageName)
        {
            if (string.IsNullOrEmpty(pageName))
                return;

            CurrentPage = pageName;

            // Clear navigation stack when navigating to main pages
            _navigationStack.Clear();

            CurrentView = pageName switch
            {
                "Search" => SearchViewModel,
                "Queue" => QueueViewModel,
                "Settings" => SettingsViewModel,
                _ => SearchViewModel
            };
            
            // Load queue data when navigating to Queue page
            if (pageName == "Queue" && IsInitialized)
            {
                await QueueViewModel.LoadQueueAsync();
            }
        }

        /// <summary>
        /// Navigate to the Queue page
        /// </summary>
        public void NavigateToQueue()
        {
            NavigateTo("Queue");
        }

        /// <summary>
        /// Navigate to the Settings page
        /// </summary>
        public void NavigateToSettings()
        {
            NavigateTo("Settings");
        }

        /// <summary>
        /// Handle settings requested from child ViewModels
        /// </summary>
        private void OnSettingsRequested(object? sender, EventArgs e)
        {
            ShowSettingsRequested?.Invoke(this, EventArgs.Empty);
        }
        
        /// <summary>
        /// Handle settings saved event - reconfigure services
        /// </summary>
        private void OnSettingsSaved(object? sender, EventArgs e)
        {
            ConfigureSpotifyService();
        }
        
        /// <summary>
        /// Handle queue refresh requested from child ViewModels
        /// </summary>
        private async void OnQueueRefreshRequested(object? sender, EventArgs e)
        {
            if (IsInitialized)
            {
                await QueueViewModel.LoadQueueAsync();
            }
        }
        
        /// <summary>
        /// Handle navigation to album detail requested
        /// </summary>
        private void OnNavigateToAlbumRequested(object? sender, Models.Album album)
        {
            if (album == null)
                return;
                
            LoggingService.Instance.LogInfo($"Navigating to album detail: {album.Title}");
            
            // Push current view to navigation stack
            if (CurrentView != null)
            {
                _navigationStack.Push(CurrentView);
            }
            
            // Create AlbumDetailViewModel and set as current view
            var albumDetailViewModel = new AlbumDetailViewModel(_service, album);
            
            // Subscribe to navigation events from album detail
            albumDetailViewModel.NavigateToArtistRequested += OnNavigateToArtistRequested;
            albumDetailViewModel.BackRequested += OnBackFromDetailRequested;
            albumDetailViewModel.QueueRefreshRequested += OnQueueRefreshRequested;
            
            CurrentView = albumDetailViewModel;
        }
        
        /// <summary>
        /// Handle navigation to artist detail requested
        /// </summary>
        private void OnNavigateToArtistRequested(object? sender, Models.Artist artist)
        {
            if (artist == null)
                return;
                
            LoggingService.Instance.LogInfo($"Navigating to artist detail: {artist.Name}");
            
            // Push current view to navigation stack
            if (CurrentView != null)
            {
                _navigationStack.Push(CurrentView);
            }
            
            // Create ArtistDetailViewModel and set as current view
            var artistDetailViewModel = new ArtistDetailViewModel(_service, artist);
            
            // Subscribe to events
            artistDetailViewModel.BackRequested += OnBackFromDetailRequested;
            artistDetailViewModel.NavigateToAlbum += OnNavigateToAlbumRequested;
            artistDetailViewModel.QueueRefreshRequested += OnQueueRefreshRequested;
            
            CurrentView = artistDetailViewModel;
        }
        
        /// <summary>
        /// Handle navigation to playlist detail requested
        /// </summary>
        private void OnNavigateToPlaylistRequested(object? sender, Models.Playlist playlist)
        {
            if (playlist == null)
                return;
                
            LoggingService.Instance.LogInfo($"Navigating to playlist detail: {playlist.Title}");
            
            // Push current view to navigation stack
            if (CurrentView != null)
            {
                _navigationStack.Push(CurrentView);
            }
            
            // Create PlaylistDetailViewModel and set as current view
            var playlistDetailViewModel = new PlaylistDetailViewModel(_service, playlist);
            
            // Subscribe to navigation events from playlist detail
            playlistDetailViewModel.NavigateToArtistRequested += OnNavigateToArtistRequested;
            playlistDetailViewModel.BackRequested += OnBackFromDetailRequested;
            playlistDetailViewModel.QueueRefreshRequested += OnQueueRefreshRequested;
            
            CurrentView = playlistDetailViewModel;
        }
        
        /// <summary>
        /// Handle back navigation from detail views
        /// </summary>
        private void OnBackFromDetailRequested(object? sender, EventArgs e)
        {
            LoggingService.Instance.LogInfo("Navigating back");
            
            // Pop from navigation stack if available
            if (_navigationStack.Count > 0)
            {
                var previousView = _navigationStack.Pop();
                CurrentView = previousView;
                LoggingService.Instance.LogInfo($"Navigated back to: {previousView.GetType().Name}");
            }
            else
            {
                // Default to search view if stack is empty
                LoggingService.Instance.LogInfo("Navigation stack empty, returning to search");
                CurrentView = SearchViewModel;
            }
        }

        #endregion

        #region Theme Management

        /// <summary>
        /// Toggle between dark and light themes
        /// </summary>
        private async void ToggleTheme()
        {
            try
            {
                // Toggle theme
                var newTheme = ThemeManager.Instance.ToggleTheme(animate: true);
                
                // Update settings
                SettingsViewModel.Theme = newTheme;
                
                // Save settings to persist theme preference
                if (SettingsViewModel.SaveCommand is ICommand saveCommand && saveCommand.CanExecute(null))
                {
                    await System.Threading.Tasks.Task.Run(() => saveCommand.Execute(null));
                }
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Error toggling theme: {ex.Message}");
            }
        }

        #endregion

        #region Window Lifecycle

        /// <summary>
        /// Handle window loaded event
        /// </summary>
        private async void OnWindowLoaded()
        {
            await InitializeAsync();
        }
        
        /// <summary>
        /// Initialize the application (public method that can be called from MainWindow)
        /// </summary>
        public async System.Threading.Tasks.Task InitializeAsync()
        {
            if (IsInitialized)
                return;

            try
            {
                // Initialize the backend service
                var configPath = GetConfigPath();
                await _service.InitializeAsync(configPath);
                
                IsInitialized = true;

                // Load initial data for ViewModels
                await SearchViewModel.InitializeAsync();
                await SettingsViewModel.LoadSettingsAsync();
                
                // Configure Spotify service with credentials from settings
                ConfigureSpotifyService();
                
                // Load queue in background (don't block initialization)
                _ = QueueViewModel.LoadQueueAsync();
            }
            catch (Exception ex)
            {
                // TODO: Show error dialog to user
                System.Diagnostics.Debug.WriteLine($"Failed to initialize: {ex.Message}");
                LoggingService.Instance.LogError("Failed to initialize application", ex);
            }
        }

        /// <summary>
        /// Handle window closing event
        /// </summary>
        private void OnWindowClosing()
        {
            try
            {
                // Cleanup ViewModels
                QueueViewModel.Dispose();
                
                // Shutdown backend service
                _service.Dispose();
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Error during shutdown: {ex.Message}");
            }
        }

        /// <summary>
        /// Get the configuration file path
        /// </summary>
        private string GetConfigPath()
        {
            var appDataPath = Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData);
            var configDir = System.IO.Path.Combine(appDataPath, "DeeMusicV2");
            
            // Ensure directory exists
            if (!System.IO.Directory.Exists(configDir))
            {
                System.IO.Directory.CreateDirectory(configDir);
            }

            return System.IO.Path.Combine(configDir, "settings.json");
        }
        
        /// <summary>
        /// Configure Spotify service with credentials from settings
        /// </summary>
        private void ConfigureSpotifyService()
        {
            try
            {
                var clientId = SettingsViewModel.SpotifyClientId;
                var clientSecret = SettingsViewModel.SpotifyClientSecret;
                
                if (!string.IsNullOrEmpty(clientId) && !string.IsNullOrEmpty(clientSecret))
                {
                    SearchViewModel.ConfigureSpotify(clientId, clientSecret);
                    LoggingService.Instance.LogInfo("Spotify service configured");
                }
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogWarning($"Failed to configure Spotify service: {ex.Message}");
            }
        }

        #endregion

        protected virtual void OnPropertyChanged([CallerMemberName] string? propertyName = null)
        {
            PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
        }
    }
}
