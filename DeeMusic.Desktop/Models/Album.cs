using System.Collections.Generic;
using System.ComponentModel.DataAnnotations;
using System.Text.Json.Serialization;

namespace DeeMusic.Desktop.Models
{
    /// <summary>
    /// Represents a Deezer album
    /// </summary>
    public class Album
    {
        [JsonPropertyName("id")]
        public string Id { get; set; } = string.Empty;

        [JsonPropertyName("title")]
        [Required]
        public string Title { get; set; } = string.Empty;

        [JsonPropertyName("upc")]
        public string UPC { get; set; } = string.Empty;

        [JsonPropertyName("link")]
        public string Link { get; set; } = string.Empty;

        [JsonPropertyName("cover")]
        public string Cover { get; set; } = string.Empty;

        [JsonPropertyName("cover_small")]
        public string CoverSmall { get; set; } = string.Empty;

        [JsonPropertyName("cover_medium")]
        public string CoverMedium { get; set; } = string.Empty;

        [JsonPropertyName("cover_big")]
        public string CoverBig { get; set; } = string.Empty;

        [JsonPropertyName("cover_xl")]
        public string CoverXL { get; set; } = string.Empty;

        [JsonPropertyName("md5_image")]
        public string MD5Image { get; set; } = string.Empty;

        [JsonPropertyName("genre_id")]
        public int GenreId { get; set; }

        [JsonPropertyName("genres")]
        public Genres? Genres { get; set; }

        [JsonPropertyName("label")]
        public string Label { get; set; } = string.Empty;

        [JsonPropertyName("nb_tracks")]
        public int TrackCount { get; set; }

        [JsonPropertyName("duration")]
        public int Duration { get; set; }

        [JsonPropertyName("fans")]
        public int Fans { get; set; }

        [JsonPropertyName("release_date")]
        public string ReleaseDate { get; set; } = string.Empty;

        [JsonPropertyName("record_type")]
        public string RecordType { get; set; } = string.Empty;

        [JsonPropertyName("available")]
        public bool Available { get; set; }

        [JsonPropertyName("explicit_lyrics")]
        public bool ExplicitLyrics { get; set; }

        [JsonPropertyName("explicit_content_lyrics")]
        public int ExplicitContent { get; set; }

        [JsonPropertyName("contributors")]
        public List<Artist> Contributors { get; set; } = new();

        [JsonPropertyName("artist")]
        public Artist? Artist { get; set; }

        [JsonPropertyName("type")]
        public string Type { get; set; } = "album";

        [JsonPropertyName("tracks")]
        public Tracks? Tracks { get; set; }

        /// <summary>
        /// Gets the display name for the album (Artist - Album)
        /// </summary>
        public string DisplayName => Artist != null ? $"{Artist.Name} - {Title}" : Title;

        /// <summary>
        /// Gets the release year from the release date
        /// </summary>
        public string Year
        {
            get
            {
                if (string.IsNullOrEmpty(ReleaseDate))
                    return string.Empty;
                    
                // ReleaseDate format is typically "YYYY-MM-DD"
                if (System.DateTime.TryParse(ReleaseDate, out var date))
                    return date.Year.ToString();
                    
                // If parsing fails, try to extract first 4 characters as year
                if (ReleaseDate.Length >= 4)
                    return ReleaseDate.Substring(0, 4);
                    
                return ReleaseDate;
            }
        }

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
    /// Represents a collection of tracks
    /// </summary>
    public class Tracks
    {
        [JsonPropertyName("data")]
        public List<Track> Data { get; set; } = new();
    }

    /// <summary>
    /// Represents a collection of genres
    /// </summary>
    public class Genres
    {
        [JsonPropertyName("data")]
        public List<Genre> Data { get; set; } = new();
    }

    /// <summary>
    /// Represents a music genre
    /// </summary>
    public class Genre
    {
        [JsonPropertyName("id")]
        public int Id { get; set; }

        [JsonPropertyName("name")]
        public string Name { get; set; } = string.Empty;

        [JsonPropertyName("picture")]
        public string Picture { get; set; } = string.Empty;

        [JsonPropertyName("type")]
        public string Type { get; set; } = "genre";
    }
}
