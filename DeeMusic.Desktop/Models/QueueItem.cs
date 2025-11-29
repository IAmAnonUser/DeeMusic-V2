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
                    _status = value;
                    
                    // Log when album is marked completed - DETAILED DEBUG
                    if (IsAlbumOrPlaylist && value == "completed")
                    {
                        var isPartial = CompletedTracks < TotalTracks;
                        var expectedColor = isPartial ? "ORANGE (partial)" : "GREEN (full success)";
                        Services.LoggingService.Instance.LogInfo($"Album '{Title}' Status->completed: CompletedTracks={CompletedTracks}, TotalTracks={TotalTracks}, IsPartialSuccess={isPartial}, ExpectedColor={expectedColor}");
                    }
                    
                    // Fire IsCompleted FIRST so IsPartialSuccess can use it
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
                    
                    // Fire HasFailedTracks and IsPartialSuccess BEFORE background color
                    OnPropertyChanged(nameof(HasFailedTracks));
                    OnPropertyChanged(nameof(IsPartialSuccess));
                    
                    // NOTE: Do NOT call UpdateComputedBackgroundColor here!
                    // During JSON deserialization, Status may be set before TotalTracks/CompletedTracks
                    // which would cause incorrect color calculation. Call it explicitly after all props are set.
                    
                    OnPropertyChanged(nameof(ItemBackgroundColor));
                    OnPropertyChanged(nameof(ItemBorderColor));
                    OnPropertyChanged(nameof(ShowErrorButton));
                    OnPropertyChanged(nameof(ErrorSummary));
                    OnPropertyChanged(nameof(StatusColor));
                    OnPropertyChanged(nameof(StatusBadgeText));
                    OnPropertyChanged(nameof(ShowStatusBadge));
                    OnPropertyChanged(nameof(TrackProgressText));
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
                    
                    // Log when album is marked completed with partial progress
                    if (IsAlbumOrPlaylist && value < 100 && Status == "completed")
                    {
                        Services.LoggingService.Instance.LogInfo($"Album '{Title}' marked completed with Progress={value}%, CompletedTracks={CompletedTracks}/{TotalTracks}");
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
                    OnPropertyChanged(nameof(HasFailedTracks));
                    OnPropertyChanged(nameof(ItemBackgroundColor));
                    OnPropertyChanged(nameof(ItemBorderColor));
                    OnPropertyChanged(nameof(ShowErrorButton));
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
                    _completedTracks = value;
                    OnPropertyChanged();
                    OnPropertyChanged(nameof(TrackProgressText));
                    OnPropertyChanged(nameof(DisplayTitle));
                    OnPropertyChanged(nameof(DisplayName));
                    OnPropertyChanged(nameof(StatusText));
                    OnPropertyChanged(nameof(HasFailedTracks));
                    OnPropertyChanged(nameof(IsPartialSuccess));
                    
                    // NOTE: Do NOT call UpdateComputedBackgroundColor here!
                    // Call it explicitly after all properties are set to avoid race conditions.
                    
                    OnPropertyChanged(nameof(ItemBackgroundColor));
                    OnPropertyChanged(nameof(ItemBorderColor));
                    OnPropertyChanged(nameof(ShowErrorButton));
                    OnPropertyChanged(nameof(ErrorSummary));
                    OnPropertyChanged(nameof(StatusColor));
                    OnPropertyChanged(nameof(StatusBadgeText));
                    OnPropertyChanged(nameof(ShowStatusBadge));
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
        public bool CanRetry => IsFailed || IsPartialSuccess;

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
                
                var suffix = IsPartialSuccess ? " ⚠️" : (IsCompleted ? " ✓" : "");
                return $"{CompletedTracks}/{TotalTracks} tracks{suffix}";
            }
        }
        
        /// <summary>
        /// Whether this item has failed tracks (for albums/playlists)
        /// Detects partial failures where some tracks completed but not all
        /// </summary>
        public bool HasFailedTracks
        {
            get
            {
                // Single track failure
                if (!IsAlbumOrPlaylist && IsFailed)
                    return true;
                    
                // Album/playlist with partial failure: completed but not all tracks succeeded
                if (IsAlbumOrPlaylist && IsCompleted && CompletedTracks < TotalTracks)
                    return true;
                    
                // Album/playlist with error message
                if (IsAlbumOrPlaylist && !string.IsNullOrEmpty(ErrorMessage))
                    return true;
                    
                return false;
            }
        }
        
        /// <summary>
        /// Whether this is a partial success (some tracks failed)
        /// </summary>
        public bool IsPartialSuccess => IsAlbumOrPlaylist && IsCompleted && CompletedTracks < TotalTracks;
        
        // Static frozen brushes for performance
        private static readonly System.Windows.Media.SolidColorBrush PartialSuccessBrush;
        private static readonly System.Windows.Media.SolidColorBrush FailedBrush;
        private static readonly System.Windows.Media.SolidColorBrush CompletedBrush;
        private static readonly System.Windows.Media.SolidColorBrush TransparentBrush;
        
        static QueueItem()
        {
            PartialSuccessBrush = new System.Windows.Media.SolidColorBrush(
                (System.Windows.Media.Color)System.Windows.Media.ColorConverter.ConvertFromString("#FEF3C7")!);
            FailedBrush = new System.Windows.Media.SolidColorBrush(
                (System.Windows.Media.Color)System.Windows.Media.ColorConverter.ConvertFromString("#FEE2E2")!);
            CompletedBrush = new System.Windows.Media.SolidColorBrush(
                (System.Windows.Media.Color)System.Windows.Media.ColorConverter.ConvertFromString("#E8F5E9")!);
            TransparentBrush = System.Windows.Media.Brushes.Transparent;
            
            PartialSuccessBrush.Freeze();
            FailedBrush.Freeze();
            CompletedBrush.Freeze();
        }
        
        // Backing field for background brush
        private System.Windows.Media.Brush _computedBackgroundBrush = System.Windows.Media.Brushes.Transparent;
        
        /// <summary>
        /// Background brush for WPF binding.
        /// </summary>
        public System.Windows.Media.Brush ComputedBackgroundColor
        {
            get => _computedBackgroundBrush;
            private set
            {
                if (_computedBackgroundBrush != value)
                {
                    _computedBackgroundBrush = value;
                    OnPropertyChanged();
                }
            }
        }
        
        /// <summary>
        /// Recalculates and updates the background color based on current state.
        /// </summary>
        public void UpdateComputedBackgroundColor()
        {
            if (IsPartialSuccess)
                ComputedBackgroundColor = PartialSuccessBrush;
            else if (IsFailed)
                ComputedBackgroundColor = FailedBrush;
            else if (IsCompleted)
                ComputedBackgroundColor = CompletedBrush;
            else
                ComputedBackgroundColor = TransparentBrush;
        }
        
        // Keep string version for compatibility
        public string ItemBackgroundColor => IsPartialSuccess ? "#FEF3C7" : (HasFailedTracks ? "#FEE2E2" : (IsCompleted ? "#E8F5E9" : "Transparent"));
        
        /// <summary>
        /// Border color for items with errors
        /// </summary>
        public string ItemBorderColor
        {
            get
            {
                if (HasFailedTracks && IsPartialSuccess)
                    return "#F59E0B"; // Orange border for partial success
                if (HasFailedTracks)
                    return "#EF4444"; // Red border for complete failure
                return "Transparent";
            }
        }
        
        /// <summary>
        /// Whether to show the error details button
        /// </summary>
        public bool ShowErrorButton => HasFailedTracks || IsFailed;
        
        /// <summary>
        /// Error summary for display
        /// </summary>
        public string ErrorSummary
        {
            get
            {
                if (IsPartialSuccess)
                {
                    int failedCount = TotalTracks - CompletedTracks;
                    return $"{failedCount} of {TotalTracks} tracks failed to download";
                }
                if (!string.IsNullOrEmpty(ErrorMessage))
                    return ErrorMessage;
                if (IsFailed)
                    return "Download failed";
                return string.Empty;
            }
        }
        
        public bool CanCancel => IsDownloading || IsPending;
        public bool ShowStatusBadge => IsFailed || IsCompleted || IsPartialSuccess;
        
        public string StatusColor
        {
            get
            {
                if (IsPartialSuccess)
                    return "#F59E0B"; // Orange for partial success
                return Status switch
                {
                    "completed" => "#10b981", // Green
                    "failed" => "#ef4444",    // Red
                    _ => "#6b7280"            // Gray
                };
            }
        }
        
        public string StatusBadgeText
        {
            get
            {
                if (IsPartialSuccess)
                    return "Partial";
                return Status switch
                {
                    "completed" => "Completed",
                    "failed" => "Failed",
                    _ => Status
                };
            }
        }
        
        public string CoverUrl { get; set; } = string.Empty;
        public string Speed { get; set; } = string.Empty;
        public string ETA { get; set; } = string.Empty;

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
