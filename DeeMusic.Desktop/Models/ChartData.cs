using System.Collections.Generic;
using System.Text.Json.Serialization;

namespace DeeMusic.Desktop.Models
{
    /// <summary>
    /// Represents chart data from Deezer
    /// </summary>
    public class ChartData
    {
        [JsonPropertyName("tracks")]
        public TrackList? Tracks { get; set; }

        [JsonPropertyName("albums")]
        public AlbumList? Albums { get; set; }

        [JsonPropertyName("artists")]
        public ArtistList? Artists { get; set; }

        [JsonPropertyName("playlists")]
        public PlaylistList? Playlists { get; set; }
    }

    public class TrackList
    {
        [JsonPropertyName("data")]
        public List<Track>? Data { get; set; }
    }

    public class AlbumList
    {
        [JsonPropertyName("data")]
        public List<Album>? Data { get; set; }
    }

    public class ArtistList
    {
        [JsonPropertyName("data")]
        public List<Artist>? Data { get; set; }
    }

    public class PlaylistList
    {
        [JsonPropertyName("data")]
        public List<Playlist>? Data { get; set; }
    }
}
