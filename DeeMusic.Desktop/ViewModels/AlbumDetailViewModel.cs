using System;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Runtime.CompilerServices;
using System.Threading.Tasks;
using System.Windows.Input;
using CommunityToolkit.Mvvm.Input;
using DeeMusic.Desktop.Models;
using DeeMusic.Desktop.Services;

namespace DeeMusic.Desktop.ViewModels
{
    public class AlbumDetailViewModel : INotifyPropertyChanged
    {
        private readonly DeeMusicService _service;
        private Album? _album;
        private bool _isLoading;

        public event PropertyChangedEventHandler? PropertyChanged;
        public event EventHandler? BackRequested;
        public event EventHandler<Artist>? NavigateToArtistRequested;
        public event EventHandler? QueueRefreshRequested;

        public Album? Album
        {
            get => _album;
            set
            {
                if (_album != value)
                {
                    _album = value;
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

        public ObservableCollection<Track> Tracks { get; } = new();

        public ICommand BackCommand { get; }
        public ICommand DownloadAlbumCommand { get; }
        public ICommand DownloadTrackCommand { get; }
        public ICommand ViewArtistCommand { get; }

        public AlbumDetailViewModel(DeeMusicService service)
        {
            _service = service ?? throw new ArgumentNullException(nameof(service));

            BackCommand = new RelayCommand(() => BackRequested?.Invoke(this, EventArgs.Empty));
            DownloadAlbumCommand = new AsyncRelayCommand(DownloadAlbumAsync);
            DownloadTrackCommand = new AsyncRelayCommand<Track>(DownloadTrackAsync);
            ViewArtistCommand = new RelayCommand<Artist>(ViewArtist);
        }
        
        public AlbumDetailViewModel(DeeMusicService service, Album album) : this(service)
        {
            Album = album;
            
            // Load tracks if available
            if (album.Tracks?.Data != null)
            {
                foreach (var track in album.Tracks.Data)
                {
                    Tracks.Add(track);
                }
            }
        }
        
        private void ViewArtist(Artist? artist)
        {
            if (artist != null)
            {
                NavigateToArtistRequested?.Invoke(this, artist);
            }
        }

        public async Task LoadAlbumAsync(string albumId)
        {
            IsLoading = true;
            Tracks.Clear();

            try
            {
                LoggingService.Instance.LogInfo($"Loading album details: {albumId}");
                
                var album = await _service.GetAlbumAsync<Album>(albumId);
                
                if (album != null)
                {
                    Album = album;
                    
                    if (album.Tracks?.Data != null)
                    {
                        foreach (var track in album.Tracks.Data)
                        {
                            Tracks.Add(track);
                        }
                    }
                    
                    LoggingService.Instance.LogInfo($"Album loaded: {album.Title}, {Tracks.Count} tracks");
                }
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Failed to load album {albumId}", ex);
            }
            finally
            {
                IsLoading = false;
            }
        }

        private async Task DownloadAlbumAsync()
        {
            if (Album == null) return;

            try
            {
                await _service.DownloadAlbumAsync(Album.Id);
                NotificationService.Instance.ShowSuccess($"Downloading '{Album.Title}'");
                
                // Trigger queue refresh
                await Task.Delay(500);
                QueueRefreshRequested?.Invoke(this, EventArgs.Empty);
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Failed to download album {Album.Title}", ex);
                
                // Check if it's already in queue
                if (ex.Message.Contains("already in queue", StringComparison.OrdinalIgnoreCase))
                {
                    NotificationService.Instance.ShowWarning($"'{Album.Title}' is already in the queue");
                }
                else
                {
                    NotificationService.Instance.ShowError($"Failed to download '{Album.Title}'");
                }
            }
        }

        private async Task DownloadTrackAsync(Track? track)
        {
            if (track == null) return;

            try
            {
                await _service.DownloadTrackAsync(track.Id);
                NotificationService.Instance.ShowSuccess($"Downloading '{track.Title}'");
                
                // Trigger queue refresh
                await Task.Delay(500);
                QueueRefreshRequested?.Invoke(this, EventArgs.Empty);
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Failed to download track {track.Title}", ex);
                NotificationService.Instance.ShowError($"Failed to download '{track.Title}'");
            }
        }

        protected void OnPropertyChanged([CallerMemberName] string? propertyName = null)
        {
            PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
        }
    }
}
