using System;
using System.Collections.Generic;
using System.Linq;
using System.Net.Http;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Threading.Tasks;
using DeeMusic.Desktop.Models;

namespace DeeMusic.Desktop.Services
{
    public class SpotifyService
    {
        private readonly HttpClient _httpClient;
        private readonly DeeMusicService _deezerService;
        private string? _accessToken;
        private DateTime _tokenExpiry = DateTime.MinValue;
        private string _clientId = string.Empty;
        private string _clientSecret = string.Empty;

        public SpotifyService(DeeMusicService deezerService)
        {
            _httpClient = new HttpClient();
            _deezerService = deezerService;
        }

        public void Configure(string clientId, string clientSecret)
        {
            _clientId = clientId;
            _clientSecret = clientSecret;
            _accessToken = null; // Force re-authentication
        }

        public bool IsConfigured => !string.IsNullOrEmpty(_clientId) && !string.IsNullOrEmpty(_clientSecret);

        private async Task<string> GetAccessTokenAsync()
        {
            // Return cached token if still valid
            if (!string.IsNullOrEmpty(_accessToken) && DateTime.UtcNow < _tokenExpiry)
            {
                return _accessToken;
            }

            if (!IsConfigured)
            {
                throw new InvalidOperationException("Spotify API credentials not configured");
            }

            // Request new token
            var authString = Convert.ToBase64String(Encoding.UTF8.GetBytes($"{_clientId}:{_clientSecret}"));
            
            var request = new HttpRequestMessage(HttpMethod.Post, "https://accounts.spotify.com/api/token");
            request.Headers.Add("Authorization", $"Basic {authString}");
            request.Content = new FormUrlEncodedContent(new[]
            {
                new KeyValuePair<string, string>("grant_type", "client_credentials")
            });

            var response = await _httpClient.SendAsync(request);
            response.EnsureSuccessStatusCode();

            var json = await response.Content.ReadAsStringAsync();
            var tokenResponse = JsonSerializer.Deserialize<SpotifyTokenResponse>(json);

            if (tokenResponse == null || string.IsNullOrEmpty(tokenResponse.AccessToken))
            {
                throw new Exception("Failed to get Spotify access token");
            }

            _accessToken = tokenResponse.AccessToken;
            _tokenExpiry = DateTime.UtcNow.AddSeconds(tokenResponse.ExpiresIn - 60); // Refresh 1 min early

            return _accessToken;
        }

        public async Task<Playlist?> ImportPlaylistAsync(string playlistUrl)
        {
            LoggingService.Instance.LogInfo($"Starting Spotify playlist import: {playlistUrl}");
            
            var playlistId = ExtractPlaylistId(playlistUrl);
            if (string.IsNullOrEmpty(playlistId))
            {
                LoggingService.Instance.LogError($"Invalid Spotify playlist URL: {playlistUrl}");
                throw new ArgumentException("Invalid Spotify playlist URL");
            }

            LoggingService.Instance.LogInfo($"Extracted playlist ID: {playlistId}");

            var token = await GetAccessTokenAsync();
            LoggingService.Instance.LogInfo("Got Spotify access token");
            
            // Fetch playlist metadata
            var request = new HttpRequestMessage(HttpMethod.Get, 
                $"https://api.spotify.com/v1/playlists/{playlistId}");
            request.Headers.Add("Authorization", $"Bearer {token}");

            LoggingService.Instance.LogInfo($"Fetching playlist from Spotify API...");
            var response = await _httpClient.SendAsync(request);
            
            if (!response.IsSuccessStatusCode)
            {
                var errorContent = await response.Content.ReadAsStringAsync();
                LoggingService.Instance.LogError($"Spotify API error ({response.StatusCode}): {errorContent}");
                throw new Exception($"Spotify API returned {response.StatusCode}: {errorContent}");
            }

            var json = await response.Content.ReadAsStringAsync();
            LoggingService.Instance.LogInfo($"Got playlist JSON, length: {json.Length}");
            
            var spotifyPlaylist = JsonSerializer.Deserialize<SpotifyPlaylist>(json);

            if (spotifyPlaylist == null)
            {
                LoggingService.Instance.LogError("Failed to deserialize Spotify playlist JSON");
                throw new Exception("Failed to parse Spotify playlist");
            }
            
            LoggingService.Instance.LogInfo($"Parsed playlist: {spotifyPlaylist.Name}, {spotifyPlaylist.Tracks?.Total ?? 0} tracks");

            // Convert to Deezer playlist format
            var deezerPlaylist = new Playlist
            {
                Id = $"spotify_{playlistId}",
                Title = spotifyPlaylist.Name ?? "Imported Playlist",
                Description = spotifyPlaylist.Description ?? "",
                TrackCount = spotifyPlaylist.Tracks?.Total ?? 0,
                Picture = spotifyPlaylist.Images?.FirstOrDefault()?.Url ?? "",
                PictureSmall = spotifyPlaylist.Images?.FirstOrDefault()?.Url ?? "",
                PictureMedium = spotifyPlaylist.Images?.FirstOrDefault()?.Url ?? "",
                PictureBig = spotifyPlaylist.Images?.FirstOrDefault()?.Url ?? "",
                PictureXL = spotifyPlaylist.Images?.FirstOrDefault()?.Url ?? "",
                Creator = new User
                {
                    Name = spotifyPlaylist.Owner?.DisplayName ?? "Spotify User"
                },
                Tracks = new Tracks
                {
                    Data = new List<Track>()
                }
            };

            // Match tracks with Deezer
            if (spotifyPlaylist.Tracks?.Items != null)
            {
                LoggingService.Instance.LogInfo($"Matching {spotifyPlaylist.Tracks.Items.Count} Spotify tracks with Deezer...");
                
                int matchedCount = 0;
                int failedCount = 0;
                
                foreach (var item in spotifyPlaylist.Tracks.Items)
                {
                    if (item?.Track == null)
                    {
                        LoggingService.Instance.LogWarning("Skipping null track item");
                        continue;
                    }

                    try
                    {
                        var artistName = item.Track.Artists?.FirstOrDefault()?.Name ?? "Unknown";
                        var trackName = item.Track.Name ?? "Unknown";
                        
                        LoggingService.Instance.LogInfo($"Searching for: {artistName} - {trackName}");
                        
                        var deezerTrack = await FindDeezerTrackAsync(item.Track);
                        if (deezerTrack != null)
                        {
                            deezerPlaylist.Tracks.Data.Add(deezerTrack);
                            matchedCount++;
                            LoggingService.Instance.LogInfo($"✓ Matched: {artistName} - {trackName}");
                        }
                        else
                        {
                            failedCount++;
                            LoggingService.Instance.LogWarning($"✗ No match found for: {artistName} - {trackName}");
                        }
                    }
                    catch (Exception ex)
                    {
                        failedCount++;
                        LoggingService.Instance.LogError($"Error matching track: {item.Track.Name}", ex);
                    }
                }

                LoggingService.Instance.LogInfo($"Matching complete: {matchedCount} matched, {failedCount} failed out of {spotifyPlaylist.Tracks.Items.Count} total");
            }
            else
            {
                LoggingService.Instance.LogWarning("Playlist has no tracks");
            }

            return deezerPlaylist;
        }

        private async Task<Track?> FindDeezerTrackAsync(SpotifyTrack spotifyTrack)
        {
            var artistName = spotifyTrack.Artists?.FirstOrDefault()?.Name ?? "";
            var trackName = spotifyTrack.Name ?? "";
            
            if (string.IsNullOrEmpty(artistName) || string.IsNullOrEmpty(trackName))
            {
                LoggingService.Instance.LogWarning($"Skipping track with missing artist or name");
                return null;
            }

            try
            {
                // Search Deezer for the track
                var searchQuery = $"{artistName} {trackName}";
                var searchResults = await _deezerService.SearchAsync<SearchResult<Track>>(searchQuery, "track");

                if (searchResults?.Data == null || searchResults.Data.Count == 0)
                {
                    LoggingService.Instance.LogWarning($"No Deezer results for: {searchQuery}");
                    return null;
                }

                // Find best match (first result is usually the best)
                var match = searchResults.Data.FirstOrDefault();
                if (match != null)
                {
                    LoggingService.Instance.LogInfo($"Found match: {match.Artist?.Name} - {match.Title} (ID: {match.Id})");
                }
                return match;
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Error searching Deezer for '{artistName} - {trackName}'", ex);
                return null;
            }
        }

        private string? ExtractPlaylistId(string url)
        {
            // Handle various Spotify URL formats:
            // https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M
            // spotify:playlist:37i9dQZF1DXcBWIGoYBM5M
            
            if (url.Contains("open.spotify.com/playlist/"))
            {
                var parts = url.Split('/');
                var id = parts[^1].Split('?')[0]; // Remove query params
                return id;
            }
            
            if (url.StartsWith("spotify:playlist:"))
            {
                return url.Replace("spotify:playlist:", "");
            }

            return null;
        }

        // Spotify API models
        private class SpotifyTokenResponse
        {
            [JsonPropertyName("access_token")]
            public string AccessToken { get; set; } = string.Empty;

            [JsonPropertyName("expires_in")]
            public int ExpiresIn { get; set; }
        }

        private class SpotifyPlaylist
        {
            [JsonPropertyName("id")]
            public string? Id { get; set; }

            [JsonPropertyName("name")]
            public string? Name { get; set; }

            [JsonPropertyName("description")]
            public string? Description { get; set; }

            [JsonPropertyName("images")]
            public List<SpotifyImage>? Images { get; set; }

            [JsonPropertyName("owner")]
            public SpotifyUser? Owner { get; set; }

            [JsonPropertyName("tracks")]
            public SpotifyTracks? Tracks { get; set; }
        }

        private class SpotifyTracks
        {
            [JsonPropertyName("total")]
            public int Total { get; set; }

            [JsonPropertyName("items")]
            public List<SpotifyTrackItem>? Items { get; set; }
        }

        private class SpotifyTrackItem
        {
            [JsonPropertyName("track")]
            public SpotifyTrack? Track { get; set; }
        }

        private class SpotifyTrack
        {
            [JsonPropertyName("id")]
            public string? Id { get; set; }

            [JsonPropertyName("name")]
            public string? Name { get; set; }

            [JsonPropertyName("artists")]
            public List<SpotifyArtist>? Artists { get; set; }

            [JsonPropertyName("duration_ms")]
            public int DurationMs { get; set; }
        }

        private class SpotifyArtist
        {
            [JsonPropertyName("name")]
            public string? Name { get; set; }
        }

        private class SpotifyUser
        {
            [JsonPropertyName("display_name")]
            public string? DisplayName { get; set; }
        }

        private class SpotifyImage
        {
            [JsonPropertyName("url")]
            public string? Url { get; set; }
        }
    }
}
