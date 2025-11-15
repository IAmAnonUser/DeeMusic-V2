using System;
using System.ComponentModel.DataAnnotations;
using System.Text.Json.Serialization;

namespace DeeMusic.Desktop.Models
{
    /// <summary>
    /// Represents a Deezer playlist
    /// </summary>
    public class Playlist
    {
        [JsonPropertyName("id")]
        public string Id { get; set; } = string.Empty;

        [JsonPropertyName("title")]
        [Required]
        public string Title { get; set; } = string.Empty;

        [JsonPropertyName("description")]
        public string Description { get; set; } = string.Empty;

        [JsonPropertyName("duration")]
        public int Duration { get; set; }

        [JsonPropertyName("public")]
        public bool Public { get; set; }

        [JsonPropertyName("is_loved_track")]
        public bool IsLovedTrack { get; set; }

        [JsonPropertyName("collaborative")]
        public bool Collaborative { get; set; }

        [JsonPropertyName("nb_tracks")]
        public int TrackCount { get; set; }

        [JsonPropertyName("fans")]
        public int Fans { get; set; }

        [JsonPropertyName("link")]
        public string Link { get; set; } = string.Empty;

        [JsonPropertyName("picture")]
        public string Picture { get; set; } = string.Empty;

        [JsonPropertyName("picture_small")]
        public string PictureSmall { get; set; } = string.Empty;

        [JsonPropertyName("picture_medium")]
        public string PictureMedium { get; set; } = string.Empty;

        [JsonPropertyName("picture_big")]
        public string PictureBig { get; set; } = string.Empty;

        [JsonPropertyName("picture_xl")]
        public string PictureXL { get; set; } = string.Empty;

        [JsonPropertyName("checksum")]
        public string Checksum { get; set; } = string.Empty;

        [JsonPropertyName("creator")]
        public User? Creator { get; set; }

        [JsonPropertyName("type")]
        public string Type { get; set; } = "playlist";

        [JsonPropertyName("tracks")]
        public Tracks? Tracks { get; set; }

        [JsonPropertyName("creation_date")]
        public DateTime CreationDate { get; set; }

        [JsonPropertyName("explicit_content_lyrics")]
        public int ExplicitContentLyrics { get; set; }

        [JsonPropertyName("explicit_content_cover")]
        public int ExplicitContentCover { get; set; }

        /// <summary>
        /// Gets the display name for the playlist
        /// </summary>
        public string DisplayName => Creator != null ? $"{Title} by {Creator.Name}" : Title;

        /// <summary>
        /// Gets the formatted duration (HH:MM:SS or MM:SS)
        /// </summary>
        public string FormattedDuration
        {
            get
            {
                var hours = Duration / 3600;
                var minutes = (Duration % 3600) / 60;
                var seconds = Duration % 60;
                
                if (hours > 0)
                    return $"{hours}:{minutes:D2}:{seconds:D2}";
                
                return $"{minutes}:{seconds:D2}";
            }
        }
    }

    /// <summary>
    /// Represents a Deezer user
    /// </summary>
    public class User
    {
        [JsonPropertyName("id")]
        public string Id { get; set; } = string.Empty;

        [JsonPropertyName("name")]
        public string Name { get; set; } = string.Empty;

        [JsonPropertyName("link")]
        public string Link { get; set; } = string.Empty;

        [JsonPropertyName("picture")]
        public string Picture { get; set; } = string.Empty;

        [JsonPropertyName("type")]
        public string Type { get; set; } = "user";

        [JsonPropertyName("tracklist")]
        public string TrackList { get; set; } = string.Empty;
    }
}
