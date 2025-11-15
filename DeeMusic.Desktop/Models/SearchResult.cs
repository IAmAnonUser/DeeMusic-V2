using System.Collections.Generic;
using System.Text.Json.Serialization;

namespace DeeMusic.Desktop.Models
{
    /// <summary>
    /// Represents search results from Deezer API
    /// </summary>
    /// <typeparam name="T">The type of items in the search results (Track, Album, Artist, or Playlist)</typeparam>
    public class SearchResult<T>
    {
        [JsonPropertyName("data")]
        public List<T> Data { get; set; } = new();

        [JsonPropertyName("total")]
        public int Total { get; set; }

        [JsonPropertyName("next")]
        public string Next { get; set; } = string.Empty;
    }

    /// <summary>
    /// Represents a generic error response from the backend
    /// </summary>
    public class ErrorResponse
    {
        [JsonPropertyName("error")]
        public string Error { get; set; } = string.Empty;

        public bool HasError => !string.IsNullOrEmpty(Error);
    }
}
