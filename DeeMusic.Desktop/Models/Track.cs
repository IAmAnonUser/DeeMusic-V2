using System.Collections.Generic;
using System.ComponentModel.DataAnnotations;
using System.Text.Json.Serialization;

namespace DeeMusic.Desktop.Models
{
    /// <summary>
    /// Represents a Deezer track
    /// </summary>
    public class Track
    {
        [JsonPropertyName("id")]
        public string Id { get; set; } = string.Empty;

        [JsonPropertyName("title")]
        [Required]
        public string Title { get; set; } = string.Empty;

        [JsonPropertyName("title_short")]
        public string TitleShort { get; set; } = string.Empty;

        [JsonPropertyName("title_version")]
        public string TitleVersion { get; set; } = string.Empty;

        [JsonPropertyName("isrc")]
        public string ISRC { get; set; } = string.Empty;

        [JsonPropertyName("link")]
        public string Link { get; set; } = string.Empty;

        [JsonPropertyName("duration")]
        public int Duration { get; set; }

        [JsonPropertyName("track_number")]
        public int TrackNumber { get; set; }

        [JsonPropertyName("disk_number")]
        public int DiscNumber { get; set; }

        [JsonPropertyName("rank")]
        public int Rank { get; set; }

        [JsonPropertyName("explicit_lyrics")]
        public bool ExplicitLyrics { get; set; }

        [JsonPropertyName("explicit_content_lyrics")]
        public int ExplicitContent { get; set; }

        [JsonPropertyName("preview")]
        public string PreviewURL { get; set; } = string.Empty;

        [JsonPropertyName("md5_image")]
        public string MD5Image { get; set; } = string.Empty;

        [JsonPropertyName("artist")]
        public Artist? Artist { get; set; }

        [JsonPropertyName("album")]
        public Album? Album { get; set; }

        [JsonPropertyName("type")]
        public string Type { get; set; } = "track";

        [JsonPropertyName("release_date")]
        public string ReleaseDate { get; set; } = string.Empty;

        [JsonPropertyName("readable")]
        public bool Available { get; set; }

        [JsonPropertyName("contributors")]
        public List<Artist> Contributors { get; set; } = new();

        /// <summary>
        /// Gets the display name for the track (Artist - Title)
        /// </summary>
        public string DisplayName => Artist != null ? $"{Artist.Name} - {Title}" : Title;

        /// <summary>
        /// Gets the formatted duration (MM:SS)
        /// </summary>
        public string FormattedDuration
        {
            get
            {
                var minutes = Duration / 60;
                var seconds = Duration % 60;
                return $"{minutes}:{seconds:D2}";
            }
        }

        /// <summary>
        /// Gets the cover image URL from the album (for UI binding compatibility)
        /// </summary>
        public string? CoverMedium => Album?.CoverMedium;

        /// <summary>
        /// Gets the cover image URL from the album (for UI binding compatibility)
        /// </summary>
        public string? CoverSmall => Album?.CoverSmall;

        /// <summary>
        /// Gets the cover image URL from the album (for UI binding compatibility)
        /// </summary>
        public string? CoverBig => Album?.CoverBig;

        /// <summary>
        /// Gets the cover image URL from the album (for UI binding compatibility)
        /// </summary>
        public string? CoverXL => Album?.CoverXL;
    }
}
