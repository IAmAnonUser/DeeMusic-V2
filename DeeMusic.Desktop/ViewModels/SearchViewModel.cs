using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
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
    /// ViewModel for search functionality
    /// Manages search queries, results, and filtering
    /// </summary>
    public class SearchViewModel : INotifyPropertyChanged
    {
        private readonly DeeMusicService _service;
        private readonly SpotifyService _spotifyService;
        private string _searchQuery = string.Empty;
        private string _selectedSearchType = "all";
        private bool _isSearching;
        private object? _selectedResult;
        private string? _currentViewAllCategory;
        private bool _isViewingAll;

        public event PropertyChangedEventHandler? PropertyChanged;

        #region Properties

        /// <summary>
        /// Gets or sets the search query
        /// </summary>
        public string SearchQuery
        {
            get => _searchQuery;
            set
            {
                if (_searchQuery != value)
                {
                    _searchQuery = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(ShowSearchResults));
                    
                    // If user starts typing while in ViewAll mode, exit ViewAll and show search
                    // But preserve the search query they're typing
                    if (IsViewingAll && !string.IsNullOrWhiteSpace(value))
                    {
                        // Clear view all state but keep the search query
                        IsViewingAll = false;
                        CurrentViewAllCategory = null;
                        ViewAllItems.Clear();
                        // Don't clear SearchQuery or results - user is typing a new search
                    }
                }
            }
        }

        /// <summary>
        /// Gets or sets the selected search type
        /// </summary>
        public string SelectedSearchType
        {
            get => _selectedSearchType;
            set
            {
                if (_selectedSearchType != value)
                {
                    _selectedSearchType = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(IsAllTabSelected));
                    OnPropertyChanged(nameof(IsTracksTabSelected));
                    OnPropertyChanged(nameof(IsAlbumsTabSelected));
                    OnPropertyChanged(nameof(IsArtistsTabSelected));
                    OnPropertyChanged(nameof(IsPlaylistsTabSelected));
                    OnPropertyChanged(nameof(ShowSections));
                    OnPropertyChanged(nameof(ShowFilteredResults));
                    
                    // If we have a search query, re-search with new type
                    if (!string.IsNullOrWhiteSpace(SearchQuery))
                    {
                        _ = ExecuteSearchAsync();
                    }
                }
            }
        }

        /// <summary>
        /// Gets whether the "All" tab is selected
        /// </summary>
        public bool IsAllTabSelected => SelectedSearchType == "all";

        /// <summary>
        /// Gets whether the "Tracks" tab is selected
        /// </summary>
        public bool IsTracksTabSelected => SelectedSearchType == "track";

        /// <summary>
        /// Gets whether the "Albums" tab is selected
        /// </summary>
        public bool IsAlbumsTabSelected => SelectedSearchType == "album";

        /// <summary>
        /// Gets whether the "Artists" tab is selected
        /// </summary>
        public bool IsArtistsTabSelected => SelectedSearchType == "artist";

        /// <summary>
        /// Gets whether the "Playlists" tab is selected
        /// </summary>
        public bool IsPlaylistsTabSelected => SelectedSearchType == "playlist";

        /// <summary>
        /// Gets whether to show sections (when "All" is selected)
        /// </summary>
        public bool ShowSections => IsAllTabSelected && HasResults;

        /// <summary>
        /// Gets whether to show filtered results (when specific tab is selected)
        /// </summary>
        public bool ShowFilteredResults => !IsAllTabSelected && HasResults;

        /// <summary>
        /// Gets whether a search is in progress
        /// </summary>
        public bool IsSearching
        {
            get => _isSearching;
            private set
            {
                if (_isSearching != value)
                {
                    _isSearching = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(IsLoading));
                    OnPropertyChanged(nameof(ShowWelcome));
                    OnPropertyChanged(nameof(ShowNoResults));
                    OnPropertyChanged(nameof(ShowSearchResults));
                }
            }
        }

        /// <summary>
        /// Gets or sets the selected search result
        /// </summary>
        public object? SelectedResult
        {
            get => _selectedResult;
            set
            {
                if (_selectedResult != value)
                {
                    _selectedResult = value;
                    OnPropertyChanged();
                }
            }
        }

        /// <summary>
        /// Gets the collection of track search results
        /// </summary>
        public ObservableCollection<Track> TrackResults { get; } = new();

        /// <summary>
        /// Gets the collection of album search results
        /// </summary>
        public ObservableCollection<Album> AlbumResults { get; } = new();

        /// <summary>
        /// Gets the collection of artist search results
        /// </summary>
        public ObservableCollection<Artist> ArtistResults { get; } = new();

        /// <summary>
        /// Gets the collection of playlist search results
        /// </summary>
        public ObservableCollection<Playlist> PlaylistResults { get; } = new();

        /// <summary>
        /// Gets the available search types
        /// </summary>
        public List<string> SearchTypes { get; } = new() { "track", "album", "artist", "playlist" };

        /// <summary>
        /// Gets the collection of new releases for the home page
        /// </summary>
        public ObservableCollection<Album> NewReleases { get; } = new();

        /// <summary>
        /// Gets the collection of popular playlists for the home page
        /// </summary>
        public ObservableCollection<Playlist> PopularPlaylists { get; } = new();

        /// <summary>
        /// Gets the collection of most streamed artists for the home page
        /// </summary>
        public ObservableCollection<Artist> MostStreamedArtists { get; } = new();

        /// <summary>
        /// Gets the collection of top albums for the home page
        /// </summary>
        public ObservableCollection<Album> TopAlbums { get; } = new();

        /// <summary>
        /// Gets whether currently viewing all items in a category
        /// </summary>
        public bool IsViewingAll
        {
            get => _isViewingAll;
            private set
            {
                if (_isViewingAll != value)
                {
                    _isViewingAll = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(ShowWelcome));
                    OnPropertyChanged(nameof(ShowViewAllPage));
                }
            }
        }

        /// <summary>
        /// Gets the current view all category
        /// </summary>
        public string? CurrentViewAllCategory
        {
            get => _currentViewAllCategory;
            private set
            {
                if (_currentViewAllCategory != value)
                {
                    _currentViewAllCategory = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(ViewAllTitle));
                }
            }
        }

        /// <summary>
        /// Gets the title for the view all page
        /// </summary>
        public string ViewAllTitle => CurrentViewAllCategory switch
        {
            "NewReleases" => "New Releases",
            "TopAlbums" => "Top Albums",
            "PopularPlaylists" => "Popular Playlists",
            "MostStreamedArtists" => "Most Streamed Artists",
            _ => "View All"
        };

        /// <summary>
        /// Gets the collection for the current view all category
        /// </summary>
        public ObservableCollection<object> ViewAllItems { get; } = new();

        /// <summary>
        /// Gets whether to show the view all page
        /// </summary>
        public bool ShowViewAllPage => IsViewingAll && !ShowConfigurationNeeded;

        /// <summary>
        /// Gets whether to show the welcome message
        /// </summary>
        public bool ShowWelcome => !IsSearching && !HasResults && string.IsNullOrWhiteSpace(SearchQuery) && !ShowConfigurationNeeded && !IsViewingAll;

        /// <summary>
        /// Gets whether to show configuration needed message
        /// </summary>
        public bool ShowConfigurationNeeded { get; private set; }

        private async void CheckConfiguration()
        {
            // Add a small delay to ensure file is fully written
            await Task.Delay(100);
            
            // Check if ARL is configured
            var settingsPath = System.IO.Path.Combine(
                Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData),
                "DeeMusicV2", "settings.json");
            
            if (System.IO.File.Exists(settingsPath))
            {
                try
                {
                    // Retry logic in case file is still being written
                    string json = null;
                    for (int i = 0; i < 3; i++)
                    {
                        try
                        {
                            json = System.IO.File.ReadAllText(settingsPath);
                            break;
                        }
                        catch (System.IO.IOException)
                        {
                            if (i < 2)
                            {
                                await Task.Delay(100);
                                continue;
                            }
                            throw;
                        }
                    }
                    
                    if (string.IsNullOrWhiteSpace(json))
                    {
                        LoggingService.Instance.LogError("Settings file is empty");
                        ShowConfigurationNeeded = true;
                        return;
                    }
                    
                    LoggingService.Instance.LogInfo($"Settings JSON length: {json.Length}");
                    
                    var settings = System.Text.Json.JsonSerializer.Deserialize<Models.Settings>(json, 
                        new System.Text.Json.JsonSerializerOptions 
                        { 
                            PropertyNameCaseInsensitive = true 
                        });
                    
                    if (settings == null)
                    {
                        LoggingService.Instance.LogError("Settings deserialization returned null");
                        ShowConfigurationNeeded = true;
                        return;
                    }
                    
                    if (settings.Deezer == null)
                    {
                        LoggingService.Instance.LogError("Settings.Deezer is null");
                        ShowConfigurationNeeded = true;
                        return;
                    }
                    
                    var arl = settings.Deezer.ARL;
                    LoggingService.Instance.LogInfo($"ARL value: IsNull={arl == null}, IsEmpty={string.IsNullOrEmpty(arl)}, Length={arl?.Length ?? 0}");
                    
                    // Check if ARL is empty, null, or a placeholder value
                    ShowConfigurationNeeded = string.IsNullOrWhiteSpace(arl) || 
                                             arl == "CREDENTIAL_MANAGER" ||
                                             arl == "your_arl_token_here" ||
                                             arl.Length < 100; // Real ARL tokens are typically 192 characters
                    
                    if (!ShowConfigurationNeeded && arl != null)
                    {
                        LoggingService.Instance.LogInfo($"Configuration check: ARL configured = TRUE, ARL length = {arl.Length}, ARL preview = {arl.Substring(0, Math.Min(20, arl.Length))}...");
                    }
                    else
                    {
                        LoggingService.Instance.LogWarning($"Configuration check: ARL NOT configured properly");
                    }
                }
                catch (Exception ex)
                {
                    LoggingService.Instance.LogError("Failed to check configuration", ex);
                    ShowConfigurationNeeded = true;
                }
            }
            else
            {
                LoggingService.Instance.LogInfo("Settings file not found - configuration needed");
                ShowConfigurationNeeded = true;
            }
            
            OnPropertyChanged(nameof(ShowConfigurationNeeded));
            OnPropertyChanged(nameof(ShowWelcome));
            
            LoggingService.Instance.LogInfo($"ShowConfigurationNeeded = {ShowConfigurationNeeded}, ShowWelcome = {ShowWelcome}");
        }

        /// <summary>
        /// Gets whether to show no results message
        /// </summary>
        public bool ShowNoResults => !IsSearching && !HasResults && !string.IsNullOrWhiteSpace(SearchQuery);

        /// <summary>
        /// Gets whether to show search results UI (tabs and back button)
        /// </summary>
        public bool ShowSearchResults => !string.IsNullOrWhiteSpace(SearchQuery) && !IsSearching && !ShowConfigurationNeeded && !IsViewingAll;

        /// <summary>
        /// Gets whether there are any search results
        /// </summary>
        public bool HasResults => TrackResults.Count > 0 || AlbumResults.Count > 0 || 
                                  ArtistResults.Count > 0 || PlaylistResults.Count > 0;

        /// <summary>
        /// Gets whether there are track results
        /// </summary>
        public bool HasTrackResults => TrackResults.Count > 0;

        /// <summary>
        /// Gets whether there are album results
        /// </summary>
        public bool HasAlbumResults => AlbumResults.Count > 0;

        /// <summary>
        /// Gets whether there are artist results
        /// </summary>
        public bool HasArtistResults => ArtistResults.Count > 0;

        /// <summary>
        /// Gets whether there are playlist results
        /// </summary>
        public bool HasPlaylistResults => PlaylistResults.Count > 0;

        /// <summary>
        /// Gets whether currently loading
        /// </summary>
        public bool IsLoading => IsSearching;

        /// <summary>
        /// Gets the search results based on the selected search type
        /// </summary>
        public System.Collections.IEnumerable SearchResults
        {
            get
            {
                return SelectedSearchType switch
                {
                    "track" => TrackResults,
                    "album" => AlbumResults,
                    "artist" => ArtistResults,
                    "playlist" => PlaylistResults,
                    _ => TrackResults
                };
            }
        }

        #endregion

        #region Events

        /// <summary>
        /// Event raised when settings dialog is requested
        /// </summary>
        public event EventHandler? SettingsRequested;
        
        /// <summary>
        /// Event raised when queue should be refreshed
        /// </summary>
        public event EventHandler? QueueRefreshRequested;
        
        /// <summary>
        /// Event raised when navigation to album detail is requested
        /// </summary>
        public event EventHandler<Album>? NavigateToAlbumRequested;
        
        /// <summary>
        /// Event raised when navigation to artist detail is requested
        /// </summary>
        public event EventHandler<Artist>? NavigateToArtistRequested;
        
        /// <summary>
        /// Event raised when navigation to playlist detail is requested
        /// </summary>
        public event EventHandler<Playlist>? NavigateToPlaylistRequested;

        #endregion

        #region Commands

        /// <summary>
        /// Command to execute a search
        /// </summary>
        public ICommand SearchCommand { get; }

        /// <summary>
        /// Command to download a track
        /// </summary>
        public ICommand DownloadTrackCommand { get; }

        /// <summary>
        /// Command to download an album
        /// </summary>
        public ICommand DownloadAlbumCommand { get; }

        /// <summary>
        /// Command to download a playlist
        /// </summary>
        public ICommand DownloadPlaylistCommand { get; }

        /// <summary>
        /// Command to view details of a result
        /// </summary>
        public ICommand ViewDetailsCommand { get; }

        /// <summary>
        /// Command to open settings
        /// </summary>
        public ICommand OpenSettingsCommand { get; }

        /// <summary>
        /// Command to view all items in a category
        /// </summary>
        public ICommand ViewAllCommand { get; }

        /// <summary>
        /// Command to go back to home view
        /// </summary>
        public ICommand BackToHomeCommand { get; }

        /// <summary>
        /// Command to change search type/tab
        /// </summary>
        public ICommand ChangeSearchTypeCommand { get; }

        /// <summary>
        /// Command to view album details
        /// </summary>
        public ICommand ViewAlbumCommand { get; }
        
        /// <summary>
        /// Command to view artist details
        /// </summary>
        public ICommand ViewArtistCommand { get; }
        
        /// <summary>
        /// Command to view playlist details
        /// </summary>
        public ICommand ViewPlaylistCommand { get; }

        #endregion

        public SearchViewModel(DeeMusicService service)
        {
            _service = service ?? throw new ArgumentNullException(nameof(service));
            _spotifyService = new SpotifyService(service);

            // Initialize commands
            SearchCommand = new AsyncRelayCommand(ExecuteSearchAsync, CanExecuteSearch);
            DownloadTrackCommand = new AsyncRelayCommand<Track>(DownloadTrackAsync);
            DownloadAlbumCommand = new AsyncRelayCommand<Album>(DownloadAlbumAsync);
            DownloadPlaylistCommand = new AsyncRelayCommand<Playlist>(DownloadPlaylistAsync);
            ViewDetailsCommand = new RelayCommand<object>(ViewDetails);
            OpenSettingsCommand = new RelayCommand(() => 
            {
                // Raise event to request settings dialog
                SettingsRequested?.Invoke(this, EventArgs.Empty);
            });
            ViewAllCommand = new RelayCommand<string>(ViewAllCategory);
            BackToHomeCommand = new RelayCommand(BackToHome);
            ChangeSearchTypeCommand = new RelayCommand<string>(ChangeSearchType);
            ViewAlbumCommand = new AsyncRelayCommand<Album>(ViewAlbumAsync);
            ViewArtistCommand = new AsyncRelayCommand<Artist>(ViewArtistAsync);
            ViewPlaylistCommand = new AsyncRelayCommand<Playlist>(ViewPlaylistAsync);

            // Check configuration status
            CheckConfiguration();
        }

        /// <summary>
        /// Change the search type/tab
        /// </summary>
        private void ChangeSearchType(string? searchType)
        {
            if (!string.IsNullOrEmpty(searchType))
            {
                SelectedSearchType = searchType;
            }
        }

        /// <summary>
        /// View album details
        /// </summary>
        private async Task ViewAlbumAsync(Album? album)
        {
            if (album == null)
                return;

            try
            {
                LoggingService.Instance.LogInfo($"Viewing album details: {album.Title} (ID: {album.Id})");
                
                // Get full album details with tracks from the API
                var fullAlbum = await _service.GetAlbumAsync<Album>(album.Id);
                
                if (fullAlbum != null)
                {
                    LoggingService.Instance.LogInfo($"Album loaded: {fullAlbum.Title}, Tracks: {fullAlbum.Tracks?.Data?.Count ?? 0}");
                    
                    // Raise navigation event
                    NavigateToAlbumRequested?.Invoke(this, fullAlbum);
                }
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Failed to load album details for {album.Title}", ex);
            }
        }
        
        /// <summary>
        /// View artist details
        /// </summary>
        private async Task ViewArtistAsync(Artist? artist)
        {
            if (artist == null)
                return;

            try
            {
                LoggingService.Instance.LogInfo($"Viewing artist details: {artist.Name} (ID: {artist.Id})");
                
                // Get full artist details from the API
                var fullArtist = await _service.GetArtistAsync<Artist>(artist.Id);
                
                if (fullArtist != null)
                {
                    LoggingService.Instance.LogInfo($"Artist loaded: {fullArtist.Name}");
                    
                    // Raise navigation event
                    NavigateToArtistRequested?.Invoke(this, fullArtist);
                }
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Failed to load artist details for {artist.Name}", ex);
            }
        }
        
        /// <summary>
        /// View playlist details
        /// </summary>
        private Task ViewPlaylistAsync(Playlist? playlist)
        {
            if (playlist == null)
                return Task.CompletedTask;

            try
            {
                LoggingService.Instance.LogInfo($"Viewing playlist details: {playlist.Title} (ID: {playlist.Id})");
                
                // Navigate to playlist detail view
                NavigateToPlaylistRequested?.Invoke(this, playlist);
                return Task.CompletedTask;
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Failed to handle playlist view for {playlist.Title}", ex);
                return Task.CompletedTask;
            }
        }
        
        /// <summary>
        /// Import a Spotify playlist
        /// </summary>
        private async Task ImportSpotifyPlaylistAsync(string spotifyUrl)
        {
            IsSearching = true;
            
            try
            {
                LoggingService.Instance.LogInfo($"Importing Spotify playlist: {spotifyUrl}");
                NotificationService.Instance.ShowInfo("Importing Spotify playlist...");
                
                var playlist = await _spotifyService.ImportPlaylistAsync(spotifyUrl);
                
                if (playlist != null)
                {
                    LoggingService.Instance.LogInfo($"Successfully imported playlist: {playlist.Title} with {playlist.Tracks?.Data?.Count ?? 0} tracks");
                    NotificationService.Instance.ShowSuccess($"Imported '{playlist.Title}' ({playlist.Tracks?.Data?.Count ?? 0} tracks matched)");
                    
                    // Clear search query and results before navigating
                    // This ensures when user presses back, they return to a clean search view
                    SearchQuery = string.Empty;
                    ClearResults();
                    
                    // Navigate to playlist detail view
                    NavigateToPlaylistRequested?.Invoke(this, playlist);
                }
                else
                {
                    NotificationService.Instance.ShowError("Failed to import Spotify playlist");
                }
            }
            catch (InvalidOperationException ex) when (ex.Message.Contains("not configured"))
            {
                LoggingService.Instance.LogWarning("Spotify API not configured");
                NotificationService.Instance.ShowError("Spotify API credentials not configured. Please add them in Settings.");
                
                // Open settings
                SettingsRequested?.Invoke(this, EventArgs.Empty);
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Failed to import Spotify playlist: {spotifyUrl}", ex);
                NotificationService.Instance.ShowError($"Failed to import Spotify playlist: {ex.Message}");
            }
            finally
            {
                IsSearching = false;
            }
        }
        
        /// <summary>
        /// Configure Spotify service with credentials
        /// </summary>
        public void ConfigureSpotify(string clientId, string clientSecret)
        {
            _spotifyService.Configure(clientId, clientSecret);
        }

        /// <summary>
        /// Initialize the view model (call after backend is initialized)
        /// </summary>
        public async Task InitializeAsync()
        {
            CheckConfiguration();
            
            if (!ShowConfigurationNeeded)
            {
                await LoadFeaturedContentAsync();
            }
            else
            {
                LoggingService.Instance.LogWarning("Skipping featured content load - ARL not configured");
            }
        }



        /// <summary>
        /// Load featured content for the home page
        /// </summary>
        private async Task LoadFeaturedContentAsync()
        {
            try
            {
                LoggingService.Instance.LogInfo("Loading featured content for home page");
                
                // Clear existing content
                NewReleases.Clear();
                TopAlbums.Clear();
                PopularPlaylists.Clear();
                MostStreamedArtists.Clear();
                
                // Load editorial releases for New Releases section
                var editorialReleasesTask = _service.GetEditorialReleasesAsync<AlbumList>(25);
                
                // Load chart data from Deezer for other sections
                var chartDataTask = _service.GetChartsAsync<ChartData>(25);
                
                await Task.WhenAll(editorialReleasesTask, chartDataTask);
                
                var editorialReleases = await editorialReleasesTask;
                var chartData = await chartDataTask;
                
                if (chartData == null)
                {
                    LoggingService.Instance.LogWarning("Failed to load chart data");
                    LoadPlaceholderContent();
                    return;
                }

                // Populate New Releases (using editorial releases) - show 25 items
                if (editorialReleases?.Data != null)
                {
                    foreach (var album in editorialReleases.Data.Take(25))
                    {
                        NewReleases.Add(album);
                    }
                    LoggingService.Instance.LogInfo($"Loaded {NewReleases.Count} new releases");
                }

                // Populate Top Albums (same as new releases for now) - show 25 items
                if (chartData.Albums?.Data != null)
                {
                    foreach (var album in chartData.Albums.Data.Take(25))
                    {
                        TopAlbums.Add(album);
                    }
                    LoggingService.Instance.LogInfo($"Loaded {TopAlbums.Count} top albums");
                }

                // Populate Popular Playlists - show 25 items
                if (chartData.Playlists?.Data != null)
                {
                    foreach (var playlist in chartData.Playlists.Data.Take(25))
                    {
                        PopularPlaylists.Add(playlist);
                    }
                    LoggingService.Instance.LogInfo($"Loaded {PopularPlaylists.Count} popular playlists");
                }

                // Populate Most Streamed Artists - show 25 items
                if (chartData.Artists?.Data != null)
                {
                    foreach (var artist in chartData.Artists.Data.Take(25))
                    {
                        MostStreamedArtists.Add(artist);
                    }
                    LoggingService.Instance.LogInfo($"Loaded {MostStreamedArtists.Count} most streamed artists");
                }
                
                LoggingService.Instance.LogInfo("Featured content loaded successfully");
                System.Diagnostics.Debug.WriteLine($"Featured content counts - NewReleases: {NewReleases.Count}, TopAlbums: {TopAlbums.Count}, Playlists: {PopularPlaylists.Count}, Artists: {MostStreamedArtists.Count}");
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to load featured content", ex);
                System.Diagnostics.Debug.WriteLine($"Failed to load featured content: {ex.Message}");
                System.Diagnostics.Debug.WriteLine($"Stack trace: {ex.StackTrace}");
                
                // Load placeholder content if API fails
                LoadPlaceholderContent();
            }
        }

        /// <summary>
        /// Load placeholder content when API is unavailable
        /// </summary>
        private void LoadPlaceholderContent()
        {
            LoggingService.Instance.LogInfo("Loading placeholder content");
            
            // This will show empty sections with proper headers
            // In a real implementation, you might want to show demo/cached content
        }

        #region Search Operations

        /// <summary>
        /// Check if search can be executed
        /// </summary>
        private bool CanExecuteSearch()
        {
            return !string.IsNullOrWhiteSpace(SearchQuery) && !IsSearching;
        }

        /// <summary>
        /// Execute a search
        /// </summary>
        private async Task ExecuteSearchAsync()
        {
            if (string.IsNullOrWhiteSpace(SearchQuery))
                return;

            LoggingService.Instance.LogInfo($"ExecuteSearchAsync called: query='{SearchQuery}', type='{SelectedSearchType}'");
            
            // Check if it's a Spotify playlist URL
            if (SearchQuery.Contains("spotify.com/playlist") || SearchQuery.StartsWith("spotify:playlist:"))
            {
                await ImportSpotifyPlaylistAsync(SearchQuery);
                return;
            }
            
            IsSearching = true;
            ClearResults();

            try
            {
                if (SelectedSearchType == "all")
                {
                    // Search all types in parallel
                    await Task.WhenAll(
                        SearchTracksAsync(),
                        SearchAlbumsAsync(),
                        SearchArtistsAsync(),
                        SearchPlaylistsAsync()
                    );
                }
                else
                {
                    switch (SelectedSearchType)
                    {
                        case "track":
                            await SearchTracksAsync();
                            break;
                        case "album":
                            await SearchAlbumsAsync();
                            break;
                        case "artist":
                            await SearchArtistsAsync();
                            break;
                        case "playlist":
                            await SearchPlaylistsAsync();
                            break;
                    }
                }
                
                LoggingService.Instance.LogInfo($"Search completed: {AlbumResults.Count} albums, {TrackResults.Count} tracks, {ArtistResults.Count} artists, {PlaylistResults.Count} playlists");
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Search failed for query '{SearchQuery}'", ex);
                System.Diagnostics.Debug.WriteLine($"Search failed: {ex.Message}");
            }
            finally
            {
                IsSearching = false;
                OnPropertyChanged(nameof(ShowSections));
                OnPropertyChanged(nameof(ShowFilteredResults));
                OnPropertyChanged(nameof(ShowSearchResults));
                
                // Debug logging
                LoggingService.Instance.LogInfo($"Search UI State: IsAllTabSelected={IsAllTabSelected}, HasResults={HasResults}, ShowSections={ShowSections}, ShowSearchResults={ShowSearchResults}, TrackResults={TrackResults.Count}, AlbumResults={AlbumResults.Count}, ArtistResults={ArtistResults.Count}, PlaylistResults={PlaylistResults.Count}");
            }
        }

        /// <summary>
        /// Search for tracks
        /// </summary>
        private async Task SearchTracksAsync()
        {
            LoggingService.Instance.LogInfo("SearchTracksAsync: Starting track search");
            var response = await _service.SearchAsync<SearchResponse<Track>>(
                SearchQuery, "track", 50);

            LoggingService.Instance.LogInfo($"SearchTracksAsync: Response received - IsNull={response == null}, DataIsNull={response?.Data == null}, DataCount={response?.Data?.Count ?? -1}");

            if (response?.Data != null)
            {
                LoggingService.Instance.LogInfo($"SearchTracksAsync: Adding {response.Data.Count} tracks to TrackResults");
                foreach (var track in response.Data)
                {
                    TrackResults.Add(track);
                }
                
                LoggingService.Instance.LogInfo($"SearchTracksAsync: TrackResults.Count = {TrackResults.Count}");
                
                OnPropertyChanged(nameof(HasResults));
                OnPropertyChanged(nameof(HasTrackResults));
                OnPropertyChanged(nameof(ShowWelcome));
                OnPropertyChanged(nameof(ShowNoResults));
                OnPropertyChanged(nameof(SearchResults));
            }
            else
            {
                LoggingService.Instance.LogWarning("SearchTracksAsync: Response or Data is null!");
            }
        }

        /// <summary>
        /// Search for albums
        /// </summary>
        private async Task SearchAlbumsAsync()
        {
            var response = await _service.SearchAsync<SearchResponse<Album>>(
                SearchQuery, "album", 50);

            if (response?.Data != null)
            {
                foreach (var album in response.Data)
                {
                    AlbumResults.Add(album);
                }
                
                OnPropertyChanged(nameof(HasResults));
                OnPropertyChanged(nameof(HasAlbumResults));
                OnPropertyChanged(nameof(ShowWelcome));
                OnPropertyChanged(nameof(ShowNoResults));
                OnPropertyChanged(nameof(SearchResults));
            }
        }

        /// <summary>
        /// Search for artists
        /// </summary>
        private async Task SearchArtistsAsync()
        {
            var response = await _service.SearchAsync<SearchResponse<Artist>>(
                SearchQuery, "artist", 50);

            if (response?.Data != null)
            {
                foreach (var artist in response.Data)
                {
                    ArtistResults.Add(artist);
                }
                
                OnPropertyChanged(nameof(HasResults));
                OnPropertyChanged(nameof(HasArtistResults));
                OnPropertyChanged(nameof(ShowWelcome));
                OnPropertyChanged(nameof(ShowNoResults));
                OnPropertyChanged(nameof(SearchResults));
            }
        }

        /// <summary>
        /// Search for playlists
        /// </summary>
        private async Task SearchPlaylistsAsync()
        {
            var response = await _service.SearchAsync<SearchResponse<Playlist>>(
                SearchQuery, "playlist", 50);

            if (response?.Data != null)
            {
                foreach (var playlist in response.Data)
                {
                    PlaylistResults.Add(playlist);
                }
                
                OnPropertyChanged(nameof(HasResults));
                OnPropertyChanged(nameof(HasPlaylistResults));
                OnPropertyChanged(nameof(ShowWelcome));
                OnPropertyChanged(nameof(ShowNoResults));
                OnPropertyChanged(nameof(SearchResults));
            }
        }

        /// <summary>
        /// Clear all search results
        /// </summary>
        private void ClearResults()
        {
            TrackResults.Clear();
            AlbumResults.Clear();
            ArtistResults.Clear();
            PlaylistResults.Clear();
            
            OnPropertyChanged(nameof(HasResults));
            OnPropertyChanged(nameof(ShowWelcome));
            OnPropertyChanged(nameof(ShowNoResults));
            OnPropertyChanged(nameof(SearchResults));
            OnPropertyChanged(nameof(ShowSections));
            OnPropertyChanged(nameof(ShowFilteredResults));
        }

        #endregion

        #region Download Operations

        /// <summary>
        /// Download a track
        /// </summary>
        private async Task DownloadTrackAsync(Track? track)
        {
            if (track == null)
                return;

            try
            {
                await _service.DownloadTrackAsync(track.Id);
                // TODO: Show success notification
            }
            catch (Exception ex)
            {
                // TODO: Show error to user
                System.Diagnostics.Debug.WriteLine($"Download failed: {ex.Message}");
            }
        }

        /// <summary>
        /// Download an album
        /// </summary>
        private async Task DownloadAlbumAsync(Album? album)
        {
            if (album == null)
            {
                LoggingService.Instance.LogWarning("DownloadAlbumAsync called with null album");
                return;
            }

            try
            {
                LoggingService.Instance.LogInfo($"Downloading album: ID={album.Id}, Title={album.Title}");
                
                if (string.IsNullOrWhiteSpace(album.Id))
                {
                    LoggingService.Instance.LogError("Album ID is empty or null");
                    System.Windows.MessageBox.Show(
                        "Album ID is invalid.",
                        "Download Error",
                        System.Windows.MessageBoxButton.OK,
                        System.Windows.MessageBoxImage.Error);
                    return;
                }
                
                // Fire and forget - don't wait for the backend
                _ = Task.Run(async () =>
                {
                    try
                    {
                        await _service.DownloadAlbumAsync(album.Id);
                        LoggingService.Instance.LogInfo($"Album download initiated successfully: {album.Title}");
                        
                        // Trigger queue refresh
                        await TriggerQueueRefresh();
                    }
                    catch (BackendException ex) when (ex.ErrorCode == -15 || ex.Message.Contains("already in queue"))
                    {
                        LoggingService.Instance.LogWarning($"Album already in queue: {album.Title}");
                        System.Windows.Application.Current.Dispatcher.Invoke(() =>
                        {
                            NotificationService.Instance.ShowInfo($"'{album.Title}' is already in the download queue");
                        });
                    }
                    catch (Exception ex)
                    {
                        LoggingService.Instance.LogError($"Failed to add album to queue: {album.Title}", ex);
                        System.Windows.Application.Current.Dispatcher.Invoke(() =>
                        {
                            NotificationService.Instance.ShowError($"Failed to add '{album.Title}' to queue");
                        });
                    }
                });
                
                // Show immediate feedback
                NotificationService.Instance.ShowSuccess($"Adding '{album.Title}' to download queue...");
            }
            catch (BackendException ex) when (ex.ErrorCode == -15 || ex.Message.Contains("already in queue") || ex.Message.Contains("UNIQUE constraint") || ex.Message.Contains("already exists"))
            {
                LoggingService.Instance.LogWarning($"Album already in queue: {album.Title}");
                NotificationService.Instance.ShowInfo($"'{album.Title}' is already in the download queue");
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Download failed for album {album.Title}", ex);
                NotificationService.Instance.ShowError($"Failed to download '{album.Title}'");
            }
        }

        /// <summary>
        /// Download a playlist
        /// </summary>
        private async Task DownloadPlaylistAsync(Playlist? playlist)
        {
            if (playlist == null)
            {
                LoggingService.Instance.LogWarning("DownloadPlaylistAsync called with null playlist");
                return;
            }

            try
            {
                LoggingService.Instance.LogInfo($"Downloading playlist: ID={playlist.Id}, Title={playlist.Title}");
                
                if (string.IsNullOrWhiteSpace(playlist.Id))
                {
                    LoggingService.Instance.LogError("Playlist ID is empty or null");
                    System.Windows.MessageBox.Show(
                        "Playlist ID is invalid.",
                        "Download Error",
                        System.Windows.MessageBoxButton.OK,
                        System.Windows.MessageBoxImage.Error);
                    return;
                }
                
                // Fire and forget - don't wait for the backend
                _ = Task.Run(async () =>
                {
                    try
                    {
                        await _service.DownloadPlaylistAsync(playlist.Id);
                        LoggingService.Instance.LogInfo($"Playlist download initiated successfully: {playlist.Title}");
                        
                        // Show success notification
                        System.Windows.Application.Current.Dispatcher.Invoke(() =>
                        {
                            NotificationService.Instance.ShowSuccess($"'{playlist.Title}' added to download queue");
                        });
                        
                        // Trigger queue refresh
                        await TriggerQueueRefresh();
                    }
                    catch (BackendException ex) when (ex.ErrorCode == -15 || ex.Message.Contains("already in queue"))
                    {
                        LoggingService.Instance.LogWarning($"Playlist already in queue: {playlist.Title}");
                        System.Windows.Application.Current.Dispatcher.Invoke(() =>
                        {
                            NotificationService.Instance.ShowInfo($"'{playlist.Title}' is already in the download queue");
                        });
                    }
                    catch (Exception ex)
                    {
                        LoggingService.Instance.LogError($"Failed to add playlist to queue: {playlist.Title}", ex);
                        System.Windows.Application.Current.Dispatcher.Invoke(() =>
                        {
                            NotificationService.Instance.ShowError($"Failed to add '{playlist.Title}' to queue");
                        });
                    }
                });
            }
            catch (BackendException ex) when (ex.ErrorCode == -15 || ex.Message.Contains("already in queue") || ex.Message.Contains("UNIQUE constraint") || ex.Message.Contains("already exists"))
            {
                LoggingService.Instance.LogWarning($"Playlist already in queue: {playlist.Title}");
                NotificationService.Instance.ShowInfo($"'{playlist.Title}' is already in the download queue");
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Download failed for playlist {playlist.Title}", ex);
                NotificationService.Instance.ShowError($"Failed to download '{playlist.Title}'");
            }
        }

        #endregion

        #region Detail View

        /// <summary>
        /// View details of a selected result
        /// </summary>
        private void ViewDetails(object? result)
        {
            if (result == null)
                return;

            SelectedResult = result;
            // TODO: Navigate to detail view or show detail panel
        }

        /// <summary>
        /// View all items in a specific category
        /// </summary>
        private async void ViewAllCategory(string? category)
        {
            if (string.IsNullOrEmpty(category))
                return;

            LoggingService.Instance.LogInfo($"ViewAllCategory called for: {category}");
            
            CurrentViewAllCategory = category;
            IsViewingAll = true;

            LoggingService.Instance.LogInfo($"IsViewingAll set to true, ShowViewAllPage={ShowViewAllPage}");

            // Clear and load fresh data with 100 items (max from API)
            ViewAllItems.Clear();
            
            try
            {
                switch (category)
                {
                    case "NewReleases":
                        // Load editorial releases for New Releases
                        var editorialReleases = await _service.GetEditorialReleasesAsync<AlbumList>(100);
                        LoggingService.Instance.LogInfo($"Editorial releases response: IsNull={editorialReleases == null}, DataIsNull={editorialReleases?.Data == null}, Count={editorialReleases?.Data?.Count ?? 0}");
                        if (editorialReleases?.Data != null)
                        {
                            foreach (var item in editorialReleases.Data)
                                ViewAllItems.Add(item);
                        }
                        break;
                    case "TopAlbums":
                        // Load chart albums for Top Albums
                        var chartData = await _service.GetChartsAsync<ChartData>(100);
                        LoggingService.Instance.LogInfo($"Chart data response: IsNull={chartData == null}, AlbumsIsNull={chartData?.Albums == null}, AlbumsDataIsNull={chartData?.Albums?.Data == null}, Count={chartData?.Albums?.Data?.Count ?? 0}");
                        if (chartData?.Albums?.Data != null)
                        {
                            foreach (var item in chartData.Albums.Data)
                                ViewAllItems.Add(item);
                        }
                        break;
                    case "PopularPlaylists":
                        var playlistChartData = await _service.GetChartsAsync<ChartData>(100);
                        LoggingService.Instance.LogInfo($"Playlist chart data response: IsNull={playlistChartData == null}, PlaylistsIsNull={playlistChartData?.Playlists == null}, PlaylistsDataIsNull={playlistChartData?.Playlists?.Data == null}, Count={playlistChartData?.Playlists?.Data?.Count ?? 0}");
                        if (playlistChartData?.Playlists?.Data != null)
                        {
                            foreach (var item in playlistChartData.Playlists.Data)
                                ViewAllItems.Add(item);
                        }
                        break;
                    case "MostStreamedArtists":
                        var artistChartData = await _service.GetChartsAsync<ChartData>(100);
                        LoggingService.Instance.LogInfo($"Artist chart data response: IsNull={artistChartData == null}, ArtistsIsNull={artistChartData?.Artists == null}, ArtistsDataIsNull={artistChartData?.Artists?.Data == null}, Count={artistChartData?.Artists?.Data?.Count ?? 0}");
                        if (artistChartData?.Artists?.Data != null)
                        {
                            foreach (var item in artistChartData.Artists.Data)
                                ViewAllItems.Add(item);
                        }
                        break;
                }
                
                LoggingService.Instance.LogInfo($"Loaded {ViewAllItems.Count} items for {category}");
                System.Diagnostics.Debug.WriteLine($"ViewAllItems.Count = {ViewAllItems.Count}");
                
                // Force UI update
                OnPropertyChanged(nameof(ViewAllItems));
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Failed to load view all for {category}", ex);
                System.Diagnostics.Debug.WriteLine($"ViewAllCategory error: {ex.Message}");
            }
        }

        /// <summary>
        /// Go back to home view
        /// </summary>
        private void BackToHome()
        {
            // Clear view all state
            IsViewingAll = false;
            CurrentViewAllCategory = null;
            ViewAllItems.Clear();
            
            // Clear search state
            SearchQuery = string.Empty;
            TrackResults.Clear();
            AlbumResults.Clear();
            ArtistResults.Clear();
            PlaylistResults.Clear();
            
            // Reset to "all" tab
            SelectedSearchType = "all";
            
            // Update UI state
            OnPropertyChanged(nameof(HasResults));
            OnPropertyChanged(nameof(ShowWelcome));
            OnPropertyChanged(nameof(ShowNoResults));
            OnPropertyChanged(nameof(ShowSections));
            OnPropertyChanged(nameof(ShowFilteredResults));
        }

        #endregion
        
        #region Helper Methods
        
        /// <summary>
        /// Trigger queue refresh after adding items
        /// </summary>
        private async Task TriggerQueueRefresh()
        {
            // Wait a bit for the backend to process the album
            await Task.Delay(500);
            
            // Ensure event is raised on UI thread
            await System.Windows.Application.Current.Dispatcher.InvokeAsync(() =>
            {
                QueueRefreshRequested?.Invoke(this, EventArgs.Empty);
            });
        }
        
        #endregion

        protected virtual void OnPropertyChanged([CallerMemberName] string? propertyName = null)
        {
            PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
        }
    }

    /// <summary>
    /// Generic search response wrapper
    /// </summary>
    public class SearchResponse<T>
    {
        public List<T>? Data { get; set; }
        public int Total { get; set; }
        public string? Next { get; set; }
    }
}
