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
    public class ArtistDetailViewModel : INotifyPropertyChanged
    {
        private readonly DeeMusicService _service;
        private Artist? _artist;
        private bool _isLoading;
        private bool _isBulkDownloading;
        private string _selectedTab = "toptracks";
        private string _sortOrder = "date"; // "date" or "alphabetical"
        
        // Store original unsorted collections
        private readonly List<Album> _allAlbums = new();
        private readonly List<Album> _allSingles = new();
        private readonly List<Album> _allEPs = new();

        public event PropertyChangedEventHandler? PropertyChanged;
        public event EventHandler? BackRequested;
        public event EventHandler<Album>? NavigateToAlbum;
        public event EventHandler? QueueRefreshRequested;

        public Artist? Artist
        {
            get => _artist;
            set
            {
                if (_artist != value)
                {
                    _artist = value;
                    OnPropertyChanged();
                }
            }
        }

        public bool IsLoading
        {
            get => _isLoading;
            set
            {
                if (_isLoading != value)
                {
                    _isLoading = value;
                    OnPropertyChanged();
                }
            }
        }

        public string SelectedTab
        {
            get => _selectedTab;
            set
            {
                if (_selectedTab != value)
                {
                    _selectedTab = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(IsTopTracksTabSelected));
                    OnPropertyChanged(nameof(IsAlbumsTabSelected));
                    OnPropertyChanged(nameof(IsSinglesTabSelected));
                    OnPropertyChanged(nameof(IsEPsTabSelected));
                    OnPropertyChanged(nameof(IsFeaturedInTabSelected));
                }
            }
        }

        public bool IsTopTracksTabSelected => SelectedTab == "toptracks";
        public bool IsAlbumsTabSelected => SelectedTab == "albums";
        public bool IsSinglesTabSelected => SelectedTab == "singles";
        public bool IsEPsTabSelected => SelectedTab == "eps";
        public bool IsFeaturedInTabSelected => SelectedTab == "featuredin";

        public string SortOrder
        {
            get => _sortOrder;
            set
            {
                if (_sortOrder != value)
                {
                    _sortOrder = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(IsSortByDateSelected));
                    OnPropertyChanged(nameof(IsSortByAlphabeticalSelected));
                    ApplySorting();
                }
            }
        }

        public bool IsSortByDateSelected => SortOrder == "date";
        public bool IsSortByAlphabeticalSelected => SortOrder == "alphabetical";

        public ObservableCollection<Track> TopTracks { get; } = new();
        public ObservableCollection<Album> Albums { get; } = new();
        public ObservableCollection<Album> Singles { get; } = new();
        public ObservableCollection<Album> EPs { get; } = new();
        public ObservableCollection<Playlist> FeaturedIn { get; } = new();

        public ICommand BackCommand { get; }
        public ICommand ChangeTabCommand { get; }
        public ICommand DownloadTrackCommand { get; }
        public ICommand DownloadAlbumCommand { get; }
        public ICommand ViewAlbumCommand { get; }
        public ICommand ChangeSortOrderCommand { get; }
        public ICommand DownloadAllAlbumsCommand { get; }
        public ICommand DownloadAllSinglesCommand { get; }
        public ICommand DownloadAllEPsCommand { get; }

        public ArtistDetailViewModel(DeeMusicService service)
        {
            _service = service ?? throw new ArgumentNullException(nameof(service));

            BackCommand = new RelayCommand(() => BackRequested?.Invoke(this, EventArgs.Empty));
            ChangeTabCommand = new RelayCommand<string>(ChangeTab);
            DownloadTrackCommand = new AsyncRelayCommand<Track>(DownloadTrackAsync);
            DownloadAlbumCommand = new AsyncRelayCommand<Album>(DownloadAlbumAsync);
            ViewAlbumCommand = new RelayCommand<Album>(album => NavigateToAlbum?.Invoke(this, album!));
            ChangeSortOrderCommand = new RelayCommand<string>(ChangeSortOrder);
            DownloadAllAlbumsCommand = new AsyncRelayCommand(DownloadAllAlbumsAsync);
            DownloadAllSinglesCommand = new AsyncRelayCommand(DownloadAllSinglesAsync);
            DownloadAllEPsCommand = new AsyncRelayCommand(DownloadAllEPsAsync);
        }
        
        public ArtistDetailViewModel(DeeMusicService service, Artist artist) : this(service)
        {
            Artist = artist;
            
            // Load artist data asynchronously
            _ = LoadArtistAsync(artist.Id);
        }

        public async Task LoadArtistAsync(string artistId)
        {
            IsLoading = true;
            TopTracks.Clear();
            Albums.Clear();
            Singles.Clear();
            EPs.Clear();
            FeaturedIn.Clear();

            try
            {
                LoggingService.Instance.LogInfo($"Loading artist details: {artistId}");
                
                // Get artist details
                var artist = await _service.GetArtistAsync<Artist>(artistId);
                
                if (artist != null)
                {
                    Artist = artist;
                    LoggingService.Instance.LogInfo($"Artist loaded: {artist.Name}");
                    
                    // Search for artist's content
                    var searchQuery = artist.Name;
                    
                    // Load top tracks
                    var trackResults = await _service.SearchAsync<SearchResponse<Track>>(searchQuery, "track", 25);
                    if (trackResults?.Data != null)
                    {
                        // Filter to only this artist's tracks
                        var artistTracks = trackResults.Data.Where(t => t.Artist?.Id == artistId).Take(10);
                        foreach (var track in artistTracks)
                        {
                            TopTracks.Add(track);
                        }
                        LoggingService.Instance.LogInfo($"Loaded {TopTracks.Count} top tracks");
                    }
                    
                    // Load albums using dedicated artist albums API (gets ALL albums)
                    var albums = await _service.GetArtistAlbumsAsync<List<Album>>(artistId, 500);
                    if (albums != null)
                    {
                        LoggingService.Instance.LogInfo($"GetArtistAlbums returned {albums.Count} total albums");
                        
                        // Clear backing lists
                        _allAlbums.Clear();
                        _allSingles.Clear();
                        _allEPs.Clear();
                        
                        // Categorize albums into backing lists
                        foreach (var album in albums)
                        {
                            LoggingService.Instance.LogInfo($"Album: {album.Title}, RecordType: '{album.RecordType}'");
                            
                            if (album.RecordType == "single")
                            {
                                _allSingles.Add(album);
                            }
                            else if (album.RecordType == "ep")
                            {
                                _allEPs.Add(album);
                            }
                            else
                            {
                                _allAlbums.Add(album);
                            }
                        }
                        
                        // Apply sorting to populate observable collections
                        ApplySorting();
                        
                        LoggingService.Instance.LogInfo($"Loaded {Albums.Count} albums, {Singles.Count} singles, {EPs.Count} EPs");
                        LoggingService.Instance.LogInfo($"Backing lists: {_allAlbums.Count} albums, {_allSingles.Count} singles, {_allEPs.Count} EPs");
                    }
                    else
                    {
                        LoggingService.Instance.LogWarning("GetArtistAlbums returned null");
                    }
                    
                    // Load playlists featuring this artist
                    var playlistResults = await _service.SearchAsync<SearchResponse<Playlist>>(searchQuery, "playlist", 25);
                    if (playlistResults?.Data != null)
                    {
                        foreach (var playlist in playlistResults.Data.Take(10))
                        {
                            FeaturedIn.Add(playlist);
                        }
                        LoggingService.Instance.LogInfo($"Loaded {FeaturedIn.Count} playlists");
                    }
                }
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Failed to load artist {artistId}", ex);
            }
            finally
            {
                IsLoading = false;
            }
        }

        private void ChangeTab(string? tab)
        {
            if (!string.IsNullOrEmpty(tab))
            {
                SelectedTab = tab;
            }
        }

        private async Task DownloadTrackAsync(Track? track)
        {
            if (track == null) return;

            try
            {
                await _service.DownloadTrackAsync(track.Id);
                NotificationService.Instance.ShowSuccess($"Downloading '{track.Title}'");
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Failed to download track {track.Title}", ex);
                NotificationService.Instance.ShowError($"Failed to download '{track.Title}'");
            }
        }
        
        private async Task DownloadAlbumAsync(Album? album)
        {
            if (album == null) return;

            try
            {
                await _service.DownloadAlbumAsync(album.Id);
                NotificationService.Instance.ShowSuccess($"Downloading '{album.Title}'");
                
                // Trigger queue refresh
                await Task.Delay(500);
                QueueRefreshRequested?.Invoke(this, EventArgs.Empty);
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Failed to download album {album.Title}", ex);
                NotificationService.Instance.ShowError($"Failed to download '{album.Title}'");
            }
        }

        private void ChangeSortOrder(string? sortOrder)
        {
            if (!string.IsNullOrEmpty(sortOrder))
            {
                SortOrder = sortOrder;
            }
        }

        private void ApplySorting()
        {
            // Sort Albums
            Albums.Clear();
            var sortedAlbums = SortOrder == "alphabetical"
                ? _allAlbums.OrderBy(a => a.Title).ToList()
                : _allAlbums.OrderByDescending(a => a.ReleaseDate).ToList();
            foreach (var album in sortedAlbums)
            {
                Albums.Add(album);
            }

            // Sort Singles
            Singles.Clear();
            var sortedSingles = SortOrder == "alphabetical"
                ? _allSingles.OrderBy(a => a.Title).ToList()
                : _allSingles.OrderByDescending(a => a.ReleaseDate).ToList();
            foreach (var single in sortedSingles)
            {
                Singles.Add(single);
            }

            // Sort EPs
            EPs.Clear();
            var sortedEPs = SortOrder == "alphabetical"
                ? _allEPs.OrderBy(a => a.Title).ToList()
                : _allEPs.OrderByDescending(a => a.ReleaseDate).ToList();
            foreach (var ep in sortedEPs)
            {
                EPs.Add(ep);
            }
        }

        private async Task DownloadAllAlbumsAsync()
        {
            LoggingService.Instance.LogInfo($"DownloadAllAlbumsAsync called. Album count: {_allAlbums.Count}");
            
            if (_allAlbums.Count == 0)
            {
                NotificationService.Instance.ShowWarning("No albums to download");
                LoggingService.Instance.LogWarning("No albums available to download");
                return;
            }

            try
            {
                LoggingService.Instance.LogInfo($"Starting bulk download of {_allAlbums.Count} albums");
                NotificationService.Instance.ShowInfo($"Adding {_allAlbums.Count} albums to queue...");
                
                foreach (var album in _allAlbums)
                {
                    await _service.DownloadAlbumAsync(album.Id);
                }
                
                NotificationService.Instance.ShowSuccess($"Added {_allAlbums.Count} albums to queue");
                
                // Trigger queue refresh
                await Task.Delay(500);
                QueueRefreshRequested?.Invoke(this, EventArgs.Empty);
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to download all albums", ex);
                NotificationService.Instance.ShowError("Failed to download all albums");
            }
        }

        private async Task DownloadAllSinglesAsync()
        {
            LoggingService.Instance.LogInfo($"DownloadAllSinglesAsync called. Singles count: {_allSingles.Count}");
            
            if (_allSingles.Count == 0)
            {
                NotificationService.Instance.ShowWarning("No singles to download");
                LoggingService.Instance.LogWarning("No singles available to download");
                return;
            }

            try
            {
                LoggingService.Instance.LogInfo($"Starting bulk download of {_allSingles.Count} singles");
                NotificationService.Instance.ShowInfo($"Adding {_allSingles.Count} singles to queue...");
                
                foreach (var single in _allSingles)
                {
                    await _service.DownloadAlbumAsync(single.Id);
                }
                
                NotificationService.Instance.ShowSuccess($"Added {_allSingles.Count} singles to queue");
                
                // Trigger queue refresh
                await Task.Delay(500);
                QueueRefreshRequested?.Invoke(this, EventArgs.Empty);
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to download all singles", ex);
                NotificationService.Instance.ShowError("Failed to download all singles");
            }
        }

        private async Task DownloadAllEPsAsync()
        {
            LoggingService.Instance.LogInfo($"DownloadAllEPsAsync called. EPs count: {_allEPs.Count}");
            
            if (_allEPs.Count == 0)
            {
                NotificationService.Instance.ShowWarning("No EPs to download");
                LoggingService.Instance.LogWarning("No EPs available to download");
                return;
            }

            try
            {
                LoggingService.Instance.LogInfo($"Starting bulk download of {_allEPs.Count} EPs");
                NotificationService.Instance.ShowInfo($"Adding {_allEPs.Count} EPs to queue...");
                
                foreach (var ep in _allEPs)
                {
                    await _service.DownloadAlbumAsync(ep.Id);
                }
                
                NotificationService.Instance.ShowSuccess($"Added {_allEPs.Count} EPs to queue");
                
                // Trigger queue refresh
                await Task.Delay(500);
                QueueRefreshRequested?.Invoke(this, EventArgs.Empty);
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to download all EPs", ex);
                NotificationService.Instance.ShowError("Failed to download all EPs");
            }
        }

        protected void OnPropertyChanged([CallerMemberName] string? propertyName = null)
        {
            PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
        }
    }
}
