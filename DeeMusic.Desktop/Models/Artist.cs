using System.ComponentModel.DataAnnotations;
using System.Text.Json.Serialization;

namespace DeeMusic.Desktop.Models
{
    /// <summary>
    /// Represents a Deezer artist
    /// </summary>
    public class Artist
    {
        [JsonPropertyName("id")]
        public string Id { get; set; } = string.Empty;

        [JsonPropertyName("name")]
        [Required]
        public string Name { get; set; } = string.Empty;

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

        [JsonPropertyName("tracklist")]
        public string TrackList { get; set; } = string.Empty;

        [JsonPropertyName("type")]
        public string Type { get; set; } = "artist";

        [JsonPropertyName("role")]
        public string Role { get; set; } = string.Empty;
    }
}
