using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.ComponentModel.DataAnnotations;
using System.Linq;
using System.Runtime.CompilerServices;
using System.Threading.Tasks;
using System.Windows.Input;
using CommunityToolkit.Mvvm.Input;
using DeeMusic.Desktop.Models;
using DeeMusic.Desktop.Services;

namespace DeeMusic.Desktop.ViewModels
{
    /// <summary>
    /// ViewModel for application settings
    /// Manages settings loading, editing, and saving
    /// </summary>
    public class SettingsViewModel : INotifyPropertyChanged
    {
        private readonly DeeMusicService _service;
        private Settings _settings = new();
        private bool _isSaving;
        private bool _hasUnsavedChanges;
        private string? _validationError;
        private UpdateInfo? _availableUpdate;
        private bool _isCheckingForUpdates;
        private bool _isDownloadingUpdate;
        private int _downloadProgress;

        public event PropertyChangedEventHandler? PropertyChanged;
        public event EventHandler? SettingsSaved;

        #region Properties

        /// <summary>
        /// Gets or sets the application settings
        /// </summary>
        public Settings Settings
        {
            get => _settings;
            set
            {
                if (_settings != value)
                {
                    _settings = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(DeezerARL));
                    OnPropertyChanged(nameof(DownloadPath));
                    OnPropertyChanged(nameof(Quality));
                    OnPropertyChanged(nameof(ConcurrentDownloads));
                    OnPropertyChanged(nameof(EmbedArtwork));
                    OnPropertyChanged(nameof(ArtworkSize));
                    OnPropertyChanged(nameof(LyricsEnabled));
                    OnPropertyChanged(nameof(EmbedLyrics));
                    OnPropertyChanged(nameof(SaveLyricsFile));
                    OnPropertyChanged(nameof(Theme));
                    OnPropertyChanged(nameof(MinimizeToTray));
                    OnPropertyChanged(nameof(StartWithWindows));
                    OnPropertyChanged(nameof(StartMinimized));
                }
            }
        }

        /// <summary>
        /// Gets whether settings are being saved
        /// </summary>
        public bool IsSaving
        {
            get => _isSaving;
            private set
            {
                if (_isSaving != value)
                {
                    _isSaving = value;
                    OnPropertyChanged();
                }
            }
        }

        /// <summary>
        /// Gets whether there are unsaved changes
        /// </summary>
        public bool HasUnsavedChanges
        {
            get => _hasUnsavedChanges;
            private set
            {
                if (_hasUnsavedChanges != value)
                {
                    _hasUnsavedChanges = value;
                    OnPropertyChanged();
                    // Notify SaveCommand to re-evaluate CanExecute
                    if (SaveCommand is AsyncRelayCommand cmd)
                    {
                        cmd.NotifyCanExecuteChanged();
                    }
                }
            }
        }

        /// <summary>
        /// Gets or sets the validation error message
        /// </summary>
        public string? ValidationError
        {
            get => _validationError;
            private set
            {
                if (_validationError != value)
                {
                    _validationError = value;
                    OnPropertyChanged();
                }
            }
        }

        /// <summary>
        /// Gets whether an update check is in progress
        /// </summary>
        public bool IsCheckingForUpdates
        {
            get => _isCheckingForUpdates;
            private set
            {
                if (_isCheckingForUpdates != value)
                {
                    _isCheckingForUpdates = value;
                    OnPropertyChanged();
                }
            }
        }

        /// <summary>
        /// Gets whether an update is being downloaded
        /// </summary>
        public bool IsDownloadingUpdate
        {
            get => _isDownloadingUpdate;
            private set
            {
                if (_isDownloadingUpdate != value)
                {
                    _isDownloadingUpdate = value;
                    OnPropertyChanged();
                }
            }
        }

        /// <summary>
        /// Gets the download progress percentage
        /// </summary>
        public int DownloadProgress
        {
            get => _downloadProgress;
            private set
            {
                if (_downloadProgress != value)
                {
                    _downloadProgress = value;
                    OnPropertyChanged();
                }
            }
        }

        /// <summary>
        /// Gets whether an update is available
        /// </summary>
        public bool IsUpdateAvailable => _availableUpdate != null;

        /// <summary>
        /// Gets the available update version
        /// </summary>
        public string? UpdateVersion => _availableUpdate?.Version;

        /// <summary>
        /// Gets the available update release notes
        /// </summary>
        public string? UpdateReleaseNotes => _availableUpdate?.ReleaseNotes;

        #endregion

        #region Setting Properties (for easier binding)

        public string DeezerARL
        {
            get => Settings.Deezer.ARL;
            set
            {
                if (Settings.Deezer.ARL != value)
                {
                    System.Diagnostics.Debug.WriteLine($"DeezerARL changing from '{Settings.Deezer.ARL}' to '{value}'");
                    Settings.Deezer.ARL = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                    System.Diagnostics.Debug.WriteLine($"HasUnsavedChanges: {HasUnsavedChanges}");
                }
            }
        }

        public string SpotifyClientId
        {
            get => Settings.Spotify.ClientId;
            set
            {
                if (Settings.Spotify.ClientId != value)
                {
                    Settings.Spotify.ClientId = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                }
            }
        }

        public string SpotifyClientSecret
        {
            get => Settings.Spotify.ClientSecret;
            set
            {
                if (Settings.Spotify.ClientSecret != value)
                {
                    Settings.Spotify.ClientSecret = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                }
            }
        }

        public string DownloadPath
        {
            get => Settings.Download.OutputDir;
            set
            {
                if (Settings.Download.OutputDir != value)
                {
                    Settings.Download.OutputDir = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                    
                    // Save settings and update backend immediately when download path changes
                    _ = Task.Run(async () =>
                    {
                        await SaveSettingsAsync();
                        
                        // Notify Go backend of the new download path
                        try
                        {
                            var result = GoBackend.SetDownloadPath(value);
                            if (result == 0)
                            {
                                LoggingService.Instance.LogInfo($"Download directory changed to: {value}");
                            }
                            else
                            {
                                LoggingService.Instance.LogError($"Failed to update backend download path: {GoBackend.GetErrorMessage(result)}");
                            }
                        }
                        catch (Exception ex)
                        {
                            LoggingService.Instance.LogError($"Error updating backend download path: {ex.Message}");
                        }
                    });
                }
            }
        }

        public string Quality
        {
            get => Settings.Download.Quality;
            set
            {
                if (Settings.Download.Quality != value)
                {
                    Settings.Download.Quality = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                }
            }
        }

        public int ConcurrentDownloads
        {
            get => Settings.Download.ConcurrentDownloads;
            set
            {
                if (Settings.Download.ConcurrentDownloads != value)
                {
                    Settings.Download.ConcurrentDownloads = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                }
            }
        }

        public bool EmbedArtwork
        {
            get => Settings.Download.EmbedArtwork;
            set
            {
                if (Settings.Download.EmbedArtwork != value)
                {
                    Settings.Download.EmbedArtwork = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                }
            }
        }

        public int ArtworkSize
        {
            get => Settings.Download.ArtworkSize;
            set
            {
                if (Settings.Download.ArtworkSize != value)
                {
                    Settings.Download.ArtworkSize = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                }
            }
        }

        public bool LyricsEnabled
        {
            get => Settings.Lyrics.Enabled;
            set
            {
                if (Settings.Lyrics.Enabled != value)
                {
                    Settings.Lyrics.Enabled = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                }
            }
        }

        public bool EmbedLyrics
        {
            get => Settings.Lyrics.EmbedInFile;
            set
            {
                if (Settings.Lyrics.EmbedInFile != value)
                {
                    Settings.Lyrics.EmbedInFile = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                }
            }
        }

        public bool SaveLyricsFile
        {
            get => Settings.Lyrics.SaveSeparateFile;
            set
            {
                if (Settings.Lyrics.SaveSeparateFile != value)
                {
                    Settings.Lyrics.SaveSeparateFile = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                }
            }
        }

        public string Theme
        {
            get => Settings.System.Theme;
            set
            {
                if (Settings.System.Theme != value)
                {
                    LoggingService.Instance.LogInfo($"Theme property setter: changing from '{Settings.System.Theme}' to '{value}'");
                    Settings.System.Theme = value;
                    LoggingService.Instance.LogInfo($"Theme property setter: Settings.System.Theme is now '{Settings.System.Theme}'");
                    OnPropertyChanged();
                    MarkAsChanged();
                    
                    // Apply theme immediately
                    ThemeManager.Instance.ApplyTheme(value, animate: true);
                }
                else
                {
                    LoggingService.Instance.LogInfo($"Theme property setter: value '{value}' is same as current '{Settings.System.Theme}', skipping");
                }
            }
        }

        public bool MinimizeToTray
        {
            get => Settings.System.MinimizeToTray;
            set
            {
                if (Settings.System.MinimizeToTray != value)
                {
                    Settings.System.MinimizeToTray = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                }
            }
        }

        public bool StartWithWindows
        {
            get => Settings.System.RunOnStartup;
            set
            {
                if (Settings.System.RunOnStartup != value)
                {
                    Settings.System.RunOnStartup = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                    
                    // Update Windows startup registry
                    UpdateWindowsStartup();
                }
            }
        }

        public bool StartMinimized
        {
            get => Settings.System.StartMinimized;
            set
            {
                if (Settings.System.StartMinimized != value)
                {
                    Settings.System.StartMinimized = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                    
                    // Update Windows startup registry if startup is enabled
                    if (StartWithWindows)
                    {
                        UpdateWindowsStartup();
                    }
                }
            }
        }

        public bool CheckForUpdates
        {
            get => Settings.System.CheckForUpdates;
            set
            {
                if (Settings.System.CheckForUpdates != value)
                {
                    Settings.System.CheckForUpdates = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                }
            }
        }

        public bool AutoDownloadUpdates
        {
            get => Settings.System.AutoDownloadUpdates;
            set
            {
                if (Settings.System.AutoDownloadUpdates != value)
                {
                    Settings.System.AutoDownloadUpdates = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                }
            }
        }

        #endregion

        #region Available Options

        public List<string> AvailableQualities { get; } = new() { "MP3_320", "FLAC" };
        public List<string> AvailableThemes { get; } = new() { "dark", "light" };
        public List<int> AvailableArtworkSizes { get; } = new() { 500, 1000, 1200, 1500 };

        #endregion

        #region Commands

        /// <summary>
        /// Command to save settings
        /// </summary>
        public ICommand SaveCommand { get; }

        /// <summary>
        /// Command to reset settings to defaults
        /// </summary>
        public ICommand ResetCommand { get; }

        /// <summary>
        /// Command to browse for download folder
        /// </summary>
        public ICommand BrowseFolderCommand { get; }

        /// <summary>
        /// Command to open download folder
        /// </summary>
        public ICommand OpenDownloadFolderCommand { get; }

        /// <summary>
        /// Command to clear image cache
        /// </summary>
        public ICommand ClearImageCacheCommand { get; }
        
        /// <summary>
        /// Command to test Spotify connection
        /// </summary>
        public ICommand TestSpotifyConnectionCommand { get; }

        /// <summary>
        /// Command to check for updates
        /// </summary>
        public ICommand CheckForUpdatesCommand { get; }

        /// <summary>
        /// Command to download and install update
        /// </summary>
        public ICommand InstallUpdateCommand { get; }

        #endregion

        public SettingsViewModel(DeeMusicService service)
        {
            _service = service ?? throw new ArgumentNullException(nameof(service));

            // Initialize commands
            SaveCommand = new AsyncRelayCommand(SaveSettingsAsync); // Always enabled
            ResetCommand = new RelayCommand(ResetSettings);
            BrowseFolderCommand = new RelayCommand(BrowseForFolder);
            OpenDownloadFolderCommand = new RelayCommand(OpenDownloadFolder);
            ClearImageCacheCommand = new RelayCommand(ClearImageCache);
            TestSpotifyConnectionCommand = new AsyncRelayCommand(TestSpotifyConnectionAsync);
            CheckForUpdatesCommand = new AsyncRelayCommand(CheckForUpdatesAsync);
            InstallUpdateCommand = new AsyncRelayCommand(InstallUpdateAsync, () => _availableUpdate != null);
        }

        #region Settings Operations

        /// <summary>
        /// Load settings from the file (not from backend to avoid stale data)
        /// </summary>
        public async Task LoadSettingsAsync()
        {
            try
            {
                LoggingService.Instance.LogInfo("LoadSettingsAsync called");
                
                // ONLY load from local file - never from backend
                // The backend may have stale cached settings
                var settings = await LoadSettingsFromFileAsync();
                
                if (settings != null)
                {
                    LoggingService.Instance.LogInfo($"Settings loaded from file, ARL length: {settings.Deezer?.ARL?.Length ?? 0}");
                    Settings = settings;
                    HasUnsavedChanges = false;
                    
                    // Sync startup setting with Windows registry
                    SyncStartupWithRegistry();
                    
                    System.Diagnostics.Debug.WriteLine("Settings loaded successfully");
                }
                else
                {
                    LoggingService.Instance.LogWarning("No settings file found, using defaults");
                    // Use default settings if file doesn't exist
                    Settings = new Settings
                    {
                        Download = new DownloadSettings
                        {
                            OutputDir = GetDefaultDownloadPath(),
                            Quality = "MP3_320",
                            ConcurrentDownloads = 8
                        }
                    };
                }
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to load settings", ex);
                System.Diagnostics.Debug.WriteLine($"Failed to load settings: {ex.Message}");
            }
        }

        /// <summary>
        /// Load settings directly from file
        /// </summary>
        private async Task<Settings?> LoadSettingsFromFileAsync()
        {
            try
            {
                var appDataPath = Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData);
                var deeMusicPath = System.IO.Path.Combine(appDataPath, "DeeMusicV2");
                var settingsPath = System.IO.Path.Combine(deeMusicPath, "settings.json");

                if (!System.IO.File.Exists(settingsPath))
                {
                    System.Diagnostics.Debug.WriteLine($"Settings file not found: {settingsPath}");
                    return null;
                }

                var json = await System.IO.File.ReadAllTextAsync(settingsPath);
                var settings = System.Text.Json.JsonSerializer.Deserialize<Settings>(json, new System.Text.Json.JsonSerializerOptions
                {
                    PropertyNamingPolicy = System.Text.Json.JsonNamingPolicy.CamelCase,
                    PropertyNameCaseInsensitive = true
                });

                System.Diagnostics.Debug.WriteLine($"Settings loaded from: {settingsPath}");
                return settings;
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Failed to load settings from file: {ex.Message}");
                return null;
            }
        }

        /// <summary>
        /// Sync the startup setting with Windows registry
        /// </summary>
        private void SyncStartupWithRegistry()
        {
            try
            {
                var isRegistryEnabled = StartupManager.Instance.IsStartupEnabled();
                
                // If there's a mismatch, update the registry to match settings
                if (isRegistryEnabled != Settings.System.RunOnStartup)
                {
                    StartupManager.Instance.UpdateStartup(
                        Settings.System.RunOnStartup,
                        Settings.System.StartMinimized
                    );
                }
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Failed to sync startup with registry: {ex.Message}");
            }
        }

        /// <summary>
        /// Update Windows startup registry based on current settings
        /// </summary>
        private void UpdateWindowsStartup()
        {
            try
            {
                var success = StartupManager.Instance.UpdateStartup(
                    Settings.System.RunOnStartup,
                    Settings.System.StartMinimized
                );
                
                if (!success)
                {
                    System.Diagnostics.Debug.WriteLine("Failed to update Windows startup");
                }
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Failed to update Windows startup: {ex.Message}");
            }
        }

        /// <summary>
        /// Check if settings can be saved
        /// </summary>
        private bool CanSaveSettings()
        {
            return HasUnsavedChanges && !IsSaving && ValidateSettings();
        }

        /// <summary>
        /// Save settings to the backend
        /// </summary>
        private async Task SaveSettingsAsync()
        {
            System.Diagnostics.Debug.WriteLine("=== SaveSettingsAsync called ===");
            System.Diagnostics.Debug.WriteLine($"Current ARL: '{Settings.Deezer.ARL}'");
            System.Diagnostics.Debug.WriteLine($"Current DownloadPath: '{Settings.Download.OutputDir}'");
            System.Diagnostics.Debug.WriteLine($"HasUnsavedChanges: {HasUnsavedChanges}");
            
            if (!ValidateSettings())
            {
                System.Diagnostics.Debug.WriteLine($"Validation failed: {ValidationError}");
                return;
            }

            IsSaving = true;
            ValidationError = null;

            try
            {
                LoggingService.Instance.LogInfo("SaveSettingsAsync - saving to file");
                
                // Save settings locally to file
                // DO NOT update backend - the backend encrypts the ARL and will cause issues
                // The backend will load the new settings from file on next initialization
                await SaveSettingsToFileAsync();
                LoggingService.Instance.LogInfo("Settings saved to file successfully");
                
                HasUnsavedChanges = false;
                
                // Update Windows startup if needed
                UpdateWindowsStartup();
                
                // Notify that settings were saved
                SettingsSaved?.Invoke(this, EventArgs.Empty);
                
                System.Diagnostics.Debug.WriteLine("Settings saved successfully");
            }
            catch (Exception ex)
            {
                ValidationError = $"Failed to save settings: {ex.Message}";
                System.Diagnostics.Debug.WriteLine($"Failed to save settings: {ex.Message}");
                System.Diagnostics.Debug.WriteLine($"Stack trace: {ex.StackTrace}");
            }
            finally
            {
                IsSaving = false;
            }
        }

        /// <summary>
        /// Public method to force save settings (for use by MainWindow)
        /// </summary>
        public async Task ForceSaveAsync()
        {
            try
            {
                LoggingService.Instance.LogInfo($"ForceSaveAsync - ARL: {Settings.Deezer.ARL.Substring(0, Math.Min(20, Settings.Deezer.ARL.Length))}...");
                
                // Save to file
                // DO NOT update backend - the backend encrypts the ARL and will cause issues
                // The backend will load the new settings from file on next initialization
                LoggingService.Instance.LogInfo("Calling SaveSettingsToFileAsync...");
                await SaveSettingsToFileAsync();
                LoggingService.Instance.LogInfo("SaveSettingsToFileAsync completed - settings saved to file");
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("ForceSaveAsync failed", ex);
                throw;
            }
        }
        
        /// <summary>
        /// Save settings directly to file
        /// </summary>
        private async Task SaveSettingsToFileAsync()
        {
            var appDataPath = Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData);
            var deeMusicPath = System.IO.Path.Combine(appDataPath, "DeeMusicV2");
            var settingsPath = System.IO.Path.Combine(deeMusicPath, "settings.json");

            LoggingService.Instance.LogInfo($"SaveSettingsToFileAsync - path: {settingsPath}");
            LoggingService.Instance.LogInfo($"SaveSettingsToFileAsync - ARL: {Settings.Deezer.ARL?.Substring(0, Math.Min(20, Settings.Deezer.ARL?.Length ?? 0))}...");

            // Check if ARL is a placeholder value and warn user
            if (Settings.Deezer.ARL == "CREDENTIAL_MANAGER" || Settings.Deezer.ARL == "your_arl_token_here")
            {
                LoggingService.Instance.LogWarning("ARL is set to a placeholder value. Please enter your actual Deezer ARL token.");
                throw new InvalidOperationException("Please enter your actual Deezer ARL token in the Deezer tab before saving.");
            }

            // Ensure directory exists
            if (!System.IO.Directory.Exists(deeMusicPath))
            {
                System.IO.Directory.CreateDirectory(deeMusicPath);
                LoggingService.Instance.LogInfo($"Created directory: {deeMusicPath}");
            }

            // Log what we're about to save
            LoggingService.Instance.LogInfo($"About to serialize Settings.System.Theme: '{Settings.System.Theme}'");
            
            // Serialize and save
            var json = System.Text.Json.JsonSerializer.Serialize(Settings, new System.Text.Json.JsonSerializerOptions
            {
                WriteIndented = true,
                PropertyNamingPolicy = System.Text.Json.JsonNamingPolicy.CamelCase
            });

            LoggingService.Instance.LogInfo($"Serialized JSON length: {json.Length}");
            var jsonDoc = System.Text.Json.JsonDocument.Parse(json);
            var themeInJson = jsonDoc.RootElement.GetProperty("system").GetProperty("theme").GetString();
            LoggingService.Instance.LogInfo($"Theme in serialized JSON: '{themeInJson}'");
            LoggingService.Instance.LogInfo($"JSON ARL value: {jsonDoc.RootElement.GetProperty("deezer").GetProperty("arl").GetString()?.Substring(0, Math.Min(20, jsonDoc.RootElement.GetProperty("deezer").GetProperty("arl").GetString()?.Length ?? 0))}...");
            
            LoggingService.Instance.LogInfo($"About to write to file: {settingsPath}");
            await System.IO.File.WriteAllTextAsync(settingsPath, json);
            LoggingService.Instance.LogInfo($"File write completed");
            
            // Verify the file was written
            var verifyContent = await System.IO.File.ReadAllTextAsync(settingsPath);
            var verifyArl = System.Text.Json.JsonDocument.Parse(verifyContent).RootElement.GetProperty("deezer").GetProperty("arl").GetString();
            LoggingService.Instance.LogInfo($"Verified ARL in file: {verifyArl?.Substring(0, Math.Min(20, verifyArl?.Length ?? 0))}..., length: {verifyArl?.Length}");
        }

        /// <summary>
        /// Reset settings to defaults
        /// </summary>
        private void ResetSettings()
        {
            Settings = new Settings
            {
                Download = new DownloadSettings
                {
                    OutputDir = GetDefaultDownloadPath(),
                    Quality = "MP3_320",
                    ConcurrentDownloads = 8,
                    EmbedArtwork = true,
                    ArtworkSize = 1200,
                    FilenameTemplate = "{artist} - {title}",
                    FolderStructure = new Dictionary<string, string>
                    {
                        { "track", "{artist}/{album}" },
                        { "album", "{artist}/{album}" },
                        { "playlist", "Playlists/{playlist}" }
                    }
                },
                Lyrics = new LyricsSettings
                {
                    Enabled = true,
                    EmbedInFile = true,
                    SaveSeparateFile = false
                },
                System = new SystemSettings
                {
                    Theme = "dark",
                    MinimizeToTray = true,
                    RunOnStartup = false
                }
            };

            MarkAsChanged();
        }

        /// <summary>
        /// Browse for download folder
        /// </summary>
        private void BrowseForFolder()
        {
            try
            {
                // Use Windows Forms FolderBrowserDialog
                using var dialog = new System.Windows.Forms.FolderBrowserDialog
                {
                    Description = "Select Download Folder",
                    ShowNewFolderButton = true,
                    SelectedPath = !string.IsNullOrEmpty(DownloadPath) ? DownloadPath : 
                                   Environment.GetFolderPath(Environment.SpecialFolder.MyMusic)
                };

                var result = dialog.ShowDialog();
                if (result == System.Windows.Forms.DialogResult.OK && !string.IsNullOrWhiteSpace(dialog.SelectedPath))
                {
                    DownloadPath = dialog.SelectedPath;
                }
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Failed to browse for folder: {ex.Message}");
            }
        }

        /// <summary>
        /// Open the download folder in Windows Explorer
        /// </summary>
        private void OpenDownloadFolder()
        {
            try
            {
                if (System.IO.Directory.Exists(DownloadPath))
                {
                    System.Diagnostics.Process.Start("explorer.exe", DownloadPath);
                }
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"Failed to open folder: {ex.Message}");
            }
        }

        #endregion

        #region Validation

        /// <summary>
        /// Validate settings
        /// </summary>
        private bool ValidateSettings()
        {
            var validationResults = new List<ValidationResult>();
            var context = new ValidationContext(Settings);

            if (!Validator.TryValidateObject(Settings, context, validationResults, true))
            {
                ValidationError = string.Join(", ", validationResults.Select(r => r.ErrorMessage));
                return false;
            }

            // Additional custom validation
            if (string.IsNullOrWhiteSpace(DownloadPath))
            {
                ValidationError = "Download path is required";
                return false;
            }

            if (ConcurrentDownloads < 1 || ConcurrentDownloads > 32)
            {
                ValidationError = "Concurrent downloads must be between 1 and 32";
                return false;
            }

            ValidationError = null;
            return true;
        }

        #endregion

        #region Helper Methods

        /// <summary>
        /// Mark settings as changed
        /// </summary>
        private void MarkAsChanged()
        {
            HasUnsavedChanges = true;
        }

        /// <summary>
        /// Get the default download path
        /// </summary>
        private string GetDefaultDownloadPath()
        {
            var musicPath = Environment.GetFolderPath(Environment.SpecialFolder.UserProfile);
            return System.IO.Path.Combine(musicPath, "Music");
        }

        #endregion

        #region Cache Management

        /// <summary>
        /// Clear the image cache
        /// </summary>
        private void ClearImageCache()
        {
            try
            {
                ImageCacheService.Instance.ClearDiskCache();
                LoggingService.Instance.LogInfo("Image cache cleared successfully");
                
                // Show success message (you can implement a notification system)
                System.Windows.MessageBox.Show(
                    "Image cache has been cleared successfully.",
                    "Cache Cleared",
                    System.Windows.MessageBoxButton.OK,
                    System.Windows.MessageBoxImage.Information);
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to clear image cache", ex);
                System.Windows.MessageBox.Show(
                    $"Failed to clear image cache: {ex.Message}",
                    "Error",
                    System.Windows.MessageBoxButton.OK,
                    System.Windows.MessageBoxImage.Error);
            }
        }
        
        /// <summary>
        /// Test Spotify API connection
        /// </summary>
        private async Task TestSpotifyConnectionAsync()
        {
            if (string.IsNullOrWhiteSpace(SpotifyClientId) || string.IsNullOrWhiteSpace(SpotifyClientSecret))
            {
                NotificationService.Instance.ShowWarning("Please enter both Client ID and Client Secret");
                return;
            }

            try
            {
                LoggingService.Instance.LogInfo("Testing Spotify connection...");
                NotificationService.Instance.ShowInfo("Testing Spotify connection...");
                
                // Create a temporary Spotify service to test
                var spotifyService = new SpotifyService(_service);
                spotifyService.Configure(SpotifyClientId, SpotifyClientSecret);
                
                // Try to get an access token (this will authenticate)
                await Task.Run(async () =>
                {
                    // Use reflection to call the private GetAccessTokenAsync method
                    var method = spotifyService.GetType().GetMethod("GetAccessTokenAsync", 
                        System.Reflection.BindingFlags.NonPublic | System.Reflection.BindingFlags.Instance);
                    if (method != null)
                    {
                        await (Task<string>)method.Invoke(spotifyService, null)!;
                    }
                });
                
                LoggingService.Instance.LogInfo("Spotify connection successful");
                NotificationService.Instance.ShowSuccess("✓ Connected to Spotify successfully!");
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Spotify connection test failed", ex);
                
                if (ex.Message.Contains("401") || ex.Message.Contains("Unauthorized"))
                {
                    NotificationService.Instance.ShowError("Invalid Client ID or Client Secret");
                }
                else
                {
                    NotificationService.Instance.ShowError($"Connection failed: {ex.Message}");
                }
            }
        }

        #endregion

        #region Update Management

        /// <summary>
        /// Check for available updates
        /// </summary>
        private async Task CheckForUpdatesAsync()
        {
            IsCheckingForUpdates = true;
            
            try
            {
                LoggingService.Instance.LogInfo("Checking for updates...");
                NotificationService.Instance.ShowInfo("Checking for updates...");
                
                var updateInfo = await UpdateService.Instance.CheckForUpdatesAsync();
                
                if (updateInfo != null)
                {
                    _availableUpdate = updateInfo;
                    OnPropertyChanged(nameof(IsUpdateAvailable));
                    OnPropertyChanged(nameof(UpdateVersion));
                    OnPropertyChanged(nameof(UpdateReleaseNotes));
                    
                    // Notify command to re-evaluate CanExecute
                    if (InstallUpdateCommand is AsyncRelayCommand cmd)
                    {
                        cmd.NotifyCanExecuteChanged();
                    }
                    
                    LoggingService.Instance.LogInfo($"Update available: v{updateInfo.Version}");
                    NotificationService.Instance.ShowSuccess($"Update available: v{updateInfo.Version}");
                    
                    // Auto-download if enabled
                    if (AutoDownloadUpdates)
                    {
                        await DownloadUpdateAsync();
                    }
                }
                else
                {
                    LoggingService.Instance.LogInfo("No updates available");
                    NotificationService.Instance.ShowInfo("You're running the latest version!");
                }
                
                // Update last check time
                Settings.System.LastUpdateCheck = DateTime.Now;
                await SaveSettingsAsync();
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to check for updates", ex);
                NotificationService.Instance.ShowError($"Failed to check for updates: {ex.Message}");
            }
            finally
            {
                IsCheckingForUpdates = false;
            }
        }

        /// <summary>
        /// Download the available update
        /// </summary>
        private async Task<string?> DownloadUpdateAsync()
        {
            if (_availableUpdate == null) return null;
            
            IsDownloadingUpdate = true;
            DownloadProgress = 0;
            
            try
            {
                LoggingService.Instance.LogInfo("Downloading update...");
                NotificationService.Instance.ShowInfo("Downloading update...");
                
                var progress = new Progress<int>(percent =>
                {
                    DownloadProgress = percent;
                });
                
                var downloadPath = await UpdateService.Instance.DownloadUpdateAsync(_availableUpdate, progress);
                
                if (downloadPath != null)
                {
                    LoggingService.Instance.LogInfo($"Update downloaded: {downloadPath}");
                    NotificationService.Instance.ShowSuccess("Update downloaded successfully!");
                    return downloadPath;
                }
                else
                {
                    LoggingService.Instance.LogError("Failed to download update");
                    NotificationService.Instance.ShowError("Failed to download update");
                    return null;
                }
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to download update", ex);
                NotificationService.Instance.ShowError($"Failed to download update: {ex.Message}");
                return null;
            }
            finally
            {
                IsDownloadingUpdate = false;
                DownloadProgress = 0;
            }
        }

        /// <summary>
        /// Install the downloaded update
        /// </summary>
        private async Task InstallUpdateAsync()
        {
            if (_availableUpdate == null) return;
            
            try
            {
                // Download if not already downloaded
                var downloadPath = await DownloadUpdateAsync();
                
                if (downloadPath == null)
                {
                    NotificationService.Instance.ShowError("Failed to download update");
                    return;
                }
                
                // Confirm with user
                var result = System.Windows.MessageBox.Show(
                    $"Update v{_availableUpdate.Version} is ready to install.\n\n" +
                    "The application will restart to complete the update.\n\n" +
                    "Do you want to install now?",
                    "Install Update",
                    System.Windows.MessageBoxButton.YesNo,
                    System.Windows.MessageBoxImage.Question);
                
                if (result == System.Windows.MessageBoxResult.Yes)
                {
                    LoggingService.Instance.LogInfo("Installing update...");
                    NotificationService.Instance.ShowInfo("Installing update...");
                    
                    // Apply update (this will restart the app)
                    UpdateService.Instance.ApplyUpdate(downloadPath);
                }
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to install update", ex);
                NotificationService.Instance.ShowError($"Failed to install update: {ex.Message}");
            }
        }

        #endregion

        protected virtual void OnPropertyChanged([CallerMemberName] string? propertyName = null)
        {
            PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
        }
    }
}
