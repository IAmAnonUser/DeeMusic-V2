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
    public class PlaylistDetailViewModel : INotifyPropertyChanged
    {
        private readonly DeeMusicService _service;
        private Playlist? _playlist;
        private bool _isLoading;

        public event PropertyChangedEventHandler? PropertyChanged;
        public event EventHandler? BackRequested;
        public event EventHandler<Artist>? NavigateToArtistRequested;
        public event EventHandler? QueueRefreshRequested;

        public Playlist? Playlist
        {
            get => _playlist;
            set
            {
                if (_playlist != value)
                {
                    _playlist = value;
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
        public ICommand DownloadPlaylistCommand { get; }
        public ICommand DownloadTrackCommand { get; }
        public ICommand ViewArtistCommand { get; }

        public PlaylistDetailViewModel(DeeMusicService service)
        {
            _service = service ?? throw new ArgumentNullException(nameof(service));

            BackCommand = new RelayCommand(() => BackRequested?.Invoke(this, EventArgs.Empty));
            DownloadPlaylistCommand = new AsyncRelayCommand(DownloadPlaylistAsync);
            DownloadTrackCommand = new AsyncRelayCommand<Track>(DownloadTrackAsync);
            ViewArtistCommand = new RelayCommand<Artist>(ViewArtist);
        }
        
        public PlaylistDetailViewModel(DeeMusicService service, Playlist playlist) : this(service)
        {
            Playlist = playlist;
            
            // Load tracks if available
            if (playlist.Tracks?.Data != null && playlist.Tracks.Data.Count > 0)
            {
                foreach (var track in playlist.Tracks.Data)
                {
                    Tracks.Add(track);
                }
            }
            else
            {
                // If tracks aren't loaded, fetch the full playlist details
                _ = LoadPlaylistAsync(playlist.Id);
            }
        }
        
        private void ViewArtist(Artist? artist)
        {
            if (artist != null)
            {
                NavigateToArtistRequested?.Invoke(this, artist);
            }
        }

        public async Task LoadPlaylistAsync(string playlistId)
        {
            IsLoading = true;
            Tracks.Clear();

            try
            {
                LoggingService.Instance.LogInfo($"Loading playlist details: {playlistId}");
                
                var playlist = await _service.GetPlaylistAsync<Playlist>(playlistId);
                
                if (playlist != null)
                {
                    Playlist = playlist;
                    
                    if (playlist.Tracks?.Data != null)
                    {
                        foreach (var track in playlist.Tracks.Data)
                        {
                            Tracks.Add(track);
                        }
                    }
                    
                    LoggingService.Instance.LogInfo($"Playlist loaded: {playlist.Title}, {Tracks.Count} tracks");
                }
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Failed to load playlist {playlistId}", ex);
            }
            finally
            {
                IsLoading = false;
            }
        }

        private async Task DownloadPlaylistAsync()
        {
            if (Playlist == null) return;

            try
            {
                // Check if this is a Spotify imported playlist
                if (Playlist.Id.StartsWith("spotify_"))
                {
                    // For Spotify playlists, download as a custom playlist
                    LoggingService.Instance.LogInfo($"Downloading Spotify playlist '{Playlist.Title}' as custom playlist");
                    
                    var trackIDs = Tracks.Select(t => t.Id).ToList();
                    await _service.DownloadCustomPlaylistAsync(
                        Playlist.Id,
                        Playlist.Title,
                        Playlist.Creator?.Name ?? "Spotify User",
                        trackIDs,
                        Playlist.PictureBig ?? Playlist.Picture ?? ""
                    );
                    
                    NotificationService.Instance.ShowSuccess($"Downloading '{Playlist.Title}' ({Tracks.Count} tracks)");
                }
                else
                {
                    // Regular Deezer playlist
                    await _service.DownloadPlaylistAsync(Playlist.Id);
                    NotificationService.Instance.ShowSuccess($"Downloading '{Playlist.Title}'");
                }
                
                // Trigger queue refresh
                await Task.Delay(500);
                QueueRefreshRequested?.Invoke(this, EventArgs.Empty);
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Failed to download playlist {Playlist.Title}", ex);
                
                // Check if it's already in queue
                if (ex.Message.Contains("already in queue", StringComparison.OrdinalIgnoreCase))
                {
                    NotificationService.Instance.ShowWarning($"'{Playlist.Title}' is already in the queue");
                }
                else
                {
                    NotificationService.Instance.ShowError($"Failed to download '{Playlist.Title}'");
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
