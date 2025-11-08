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

        public event PropertyChangedEventHandler? PropertyChanged;

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
                    Settings.System.Theme = value;
                    OnPropertyChanged();
                    MarkAsChanged();
                    
                    // Apply theme immediately
                    ThemeManager.Instance.ApplyTheme(value, animate: true);
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
                
                // Save settings locally first
                await SaveSettingsToFileAsync();
                
                // Don't update backend - it will overwrite our changes with stale cached data
                // The backend will load the new settings from the file when needed
                LoggingService.Instance.LogInfo("Skipping backend update to prevent overwriting");
                
                HasUnsavedChanges = false;
                
                // Update Windows startup if needed
                UpdateWindowsStartup();
                
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
                LoggingService.Instance.LogInfo("Calling SaveSettingsToFileAsync...");
                await SaveSettingsToFileAsync();
                LoggingService.Instance.LogInfo("SaveSettingsToFileAsync completed");
                
                // Don't update backend here - it will overwrite our changes
                // The backend will load the new settings from the file on next initialization
                LoggingService.Instance.LogInfo("Skipping backend update to prevent overwriting");
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
            LoggingService.Instance.LogInfo($"SaveSettingsToFileAsync - ARL: {Settings.Deezer.ARL.Substring(0, Math.Min(20, Settings.Deezer.ARL.Length))}...");

            // Ensure directory exists
            if (!System.IO.Directory.Exists(deeMusicPath))
            {
                System.IO.Directory.CreateDirectory(deeMusicPath);
                LoggingService.Instance.LogInfo($"Created directory: {deeMusicPath}");
            }

            // Serialize and save
            var json = System.Text.Json.JsonSerializer.Serialize(Settings, new System.Text.Json.JsonSerializerOptions
            {
                WriteIndented = true,
                PropertyNamingPolicy = System.Text.Json.JsonNamingPolicy.CamelCase
            });

            LoggingService.Instance.LogInfo($"Serialized JSON length: {json.Length}");
            LoggingService.Instance.LogInfo($"JSON ARL value: {System.Text.Json.JsonDocument.Parse(json).RootElement.GetProperty("deezer").GetProperty("arl").GetString()?.Substring(0, Math.Min(20, System.Text.Json.JsonDocument.Parse(json).RootElement.GetProperty("deezer").GetProperty("arl").GetString()?.Length ?? 0))}...");
            
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
            var documentsPath = Environment.GetFolderPath(Environment.SpecialFolder.MyDocuments);
            return System.IO.Path.Combine(documentsPath, "DeeMusic", "Downloads");
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

        #endregion

        protected virtual void OnPropertyChanged([CallerMemberName] string? propertyName = null)
        {
            PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
        }
    }
}
