using System;
using System.ComponentModel;
using System.ComponentModel.DataAnnotations;
using System.Runtime.CompilerServices;
using System.Text.Json.Serialization;

namespace DeeMusic.Desktop.Models
{
    /// <summary>
    /// Represents a download queue item
    /// </summary>
    public class QueueItem : INotifyPropertyChanged
    {
        private string _status = "pending";
        private int _progress;
        private long _bytesDownloaded;
        private long _totalBytes;
        private string _errorMessage = string.Empty;
        private int _completedTracks;
        private int _totalTracks;

        [JsonPropertyName("id")]
        public string Id { get; set; } = string.Empty;

        [JsonPropertyName("type")]
        [Required]
        public string Type { get; set; } = "track"; // track, album, playlist

        private string _title = string.Empty;
        
        [JsonPropertyName("title")]
        [Required]
        public string Title
        {
            get => _title;
            set
            {
                if (_title != value)
                {
                    _title = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(DisplayTitle));
                    OnPropertyChanged(nameof(DisplayName));
                }
            }
        }
        
        /// <summary>
        /// Title with track progress for albums/playlists - shown in UI
        /// </summary>
        public string DisplayTitle
        {
            get
            {
                // Don't modify title - track progress will show separately
                return Title;
            }
        }

        private string _artist = string.Empty;
        
        [JsonPropertyName("artist")]
        public string Artist
        {
            get => _artist;
            set
            {
                if (_artist != value)
                {
                    _artist = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(DisplayName));
                }
            }
        }

        [JsonPropertyName("album")]
        public string Album { get; set; } = string.Empty;

        [JsonPropertyName("status")]
        [Required]
        public string Status
        {
            get => _status;
            set
            {
                if (_status != value)
                {
                    Services.LoggingService.Instance.LogInfo($"Status changing for {Title}: {_status} -> {value}");
                    _status = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(IsPending));
                    OnPropertyChanged(nameof(IsDownloading));
                    OnPropertyChanged(nameof(IsCompleted));
                    OnPropertyChanged(nameof(IsFailed));
                    OnPropertyChanged(nameof(IsPaused));
                    OnPropertyChanged(nameof(CanPause));
                    OnPropertyChanged(nameof(CanResume));
                    OnPropertyChanged(nameof(CanRetry));
                    OnPropertyChanged(nameof(StatusText));
                    OnPropertyChanged(nameof(DisplayName));
                    OnPropertyChanged(nameof(DisplayTitle));
                }
            }
        }

        [JsonPropertyName("progress")]
        [Range(0, 100)]
        public int Progress
        {
            get => _progress;
            set
            {
                if (_progress != value)
                {
                    // Log progress changes to track what's resetting it
                    if (value < _progress)
                    {
                        var stackTrace = new System.Diagnostics.StackTrace(true);
                        var caller = stackTrace.GetFrame(1)?.GetMethod()?.DeclaringType?.Name + "." + stackTrace.GetFrame(1)?.GetMethod()?.Name;
                        Services.LoggingService.Instance.LogWarning($"Progress DECREASED for {Title}: {_progress}% -> {value}% (called from {caller})");
                    }
                    
                    _progress = value;
                    OnPropertyChanged();
                }
            }
        }

        [JsonPropertyName("output_path")]
        public string OutputPath { get; set; } = string.Empty;

        [JsonPropertyName("error_message")]
        public string ErrorMessage
        {
            get => _errorMessage;
            set
            {
                if (_errorMessage != value)
                {
                    _errorMessage = value;
                    OnPropertyChanged();
                }
            }
        }

        [JsonPropertyName("retry_count")]
        public int RetryCount { get; set; }

        [JsonPropertyName("bytes_downloaded")]
        public long BytesDownloaded
        {
            get => _bytesDownloaded;
            set
            {
                if (_bytesDownloaded != value)
                {
                    _bytesDownloaded = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(FormattedBytesDownloaded));
                    OnPropertyChanged(nameof(DownloadSpeed));
                }
            }
        }

        [JsonPropertyName("total_bytes")]
        public long TotalBytes
        {
            get => _totalBytes;
            set
            {
                if (_totalBytes != value)
                {
                    _totalBytes = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(FormattedTotalBytes));
                }
            }
        }

        [JsonPropertyName("created_at")]
        public DateTime CreatedAt { get; set; }

        [JsonPropertyName("updated_at")]
        public DateTime UpdatedAt { get; set; }

        [JsonPropertyName("completed_at")]
        public DateTime? CompletedAt { get; set; }

        [JsonPropertyName("parent_id")]
        public string? ParentId { get; set; }

        [JsonPropertyName("total_tracks")]
        public int TotalTracks
        {
            get => _totalTracks;
            set
            {
                if (_totalTracks != value)
                {
                    Services.LoggingService.Instance.LogInfo($"TotalTracks changing for {Title}: {_totalTracks} -> {value}");
                    _totalTracks = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(IsAlbumOrPlaylist));
                    OnPropertyChanged(nameof(TrackProgressText));
                    OnPropertyChanged(nameof(DisplayTitle));
                    OnPropertyChanged(nameof(DisplayName));
                    OnPropertyChanged(nameof(StatusText));
                }
            }
        }

        [JsonPropertyName("completed_tracks")]
        public int CompletedTracks
        {
            get => _completedTracks;
            set
            {
                if (_completedTracks != value)
                {
                    Services.LoggingService.Instance.LogInfo($"CompletedTracks changing for {Title}: {_completedTracks} -> {value}");
                    _completedTracks = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(TrackProgressText));
                    OnPropertyChanged(nameof(DisplayTitle));
                    OnPropertyChanged(nameof(DisplayName));
                    OnPropertyChanged(nameof(StatusText));
                }
            }
        }

        // UI Helper Properties
        public bool IsPending => Status == "pending";
        public bool IsDownloading => Status == "downloading";
        public bool IsCompleted => Status == "completed";
        public bool IsFailed => Status == "failed";
        public bool IsPaused => Status == "paused";

        public bool CanPause => IsDownloading;
        public bool CanResume => IsPaused || IsFailed;
        public bool CanRetry => IsFailed;

        /// <summary>
        /// Full display name with artist and track progress
        /// </summary>
        public string DisplayName
        {
            get
            {
                var baseName = !string.IsNullOrEmpty(Artist) ? $"{Artist} - {Title}" : Title;
                
                // Add track progress for albums/playlists
                if ((Type == "album" || Type == "playlist") && TotalTracks > 0)
                {
                    return $"{baseName} [{CompletedTracks}/{TotalTracks}]";
                }
                
                return baseName;
            }
        }

        public string FormattedBytesDownloaded => FormatBytes(BytesDownloaded);
        public string FormattedTotalBytes => FormatBytes(TotalBytes);

        public string DownloadSpeed { get; set; } = string.Empty;

        public string StatusText
        {
            get
            {
                // For albums/playlists, show track progress in status
                if ((Type == "album" || Type == "playlist") && TotalTracks > 0)
                {
                    return Status switch
                    {
                        "pending" => $"Pending (0/{TotalTracks} tracks)",
                        "downloading" => $"Downloading {CompletedTracks}/{TotalTracks} tracks ({Progress}%)",
                        "completed" => $"✓ Completed ({TotalTracks}/{TotalTracks} tracks)",
                        "failed" => $"Failed ({CompletedTracks}/{TotalTracks} tracks)",
                        "paused" => $"Paused ({CompletedTracks}/{TotalTracks} tracks)",
                        _ => Status
                    };
                }
                
                // For single tracks
                return Status switch
                {
                    "pending" => "Pending",
                    "downloading" => $"Downloading {Progress}%",
                    "completed" => "✓ Completed",
                    "failed" => "Failed",
                    "paused" => "Paused",
                    _ => Status
                };
            }
        }

        public bool IsAlbumOrPlaylist => (Type == "album" || Type == "playlist") && TotalTracks > 0;

        public string TrackProgressText
        {
            get
            {
                if (!IsAlbumOrPlaylist)
                    return string.Empty;
                    
                return $"\n{CompletedTracks:00}/{TotalTracks:00}";
            }
        }

        private static string FormatBytes(long bytes)
        {
            string[] sizes = { "B", "KB", "MB", "GB", "TB" };
            double len = bytes;
            int order = 0;
            
            while (len >= 1024 && order < sizes.Length - 1)
            {
                order++;
                len = len / 1024;
            }
            
            return $"{len:0.##} {sizes[order]}";
        }

        public event PropertyChangedEventHandler? PropertyChanged;

        public virtual void OnPropertyChanged([CallerMemberName] string? propertyName = null)
        {
            PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
        }
    }
}
