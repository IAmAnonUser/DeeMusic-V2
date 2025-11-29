using System;
using System.Text.Json.Serialization;

namespace DeeMusic.Desktop.Models
{
    /// <summary>
    /// Represents a track that failed to download
    /// </summary>
    public class FailedTrack
    {
        [JsonPropertyName("id")]
        public int Id { get; set; }

        [JsonPropertyName("parent_id")]
        public string ParentId { get; set; } = string.Empty;

        [JsonPropertyName("track_id")]
        public string TrackId { get; set; } = string.Empty;

        [JsonPropertyName("track_title")]
        public string TrackTitle { get; set; } = string.Empty;

        [JsonPropertyName("track_artist")]
        public string TrackArtist { get; set; } = string.Empty;

        [JsonPropertyName("error_message")]
        public string ErrorMessage { get; set; } = string.Empty;

        [JsonPropertyName("retry_count")]
        public int RetryCount { get; set; }

        [JsonPropertyName("failed_at")]
        public DateTime FailedAt { get; set; }

        /// <summary>
        /// Display name for the failed track
        /// </summary>
        public string DisplayName => !string.IsNullOrEmpty(TrackArtist) 
            ? $"{TrackArtist} - {TrackTitle}" 
            : TrackTitle;
    }
}
