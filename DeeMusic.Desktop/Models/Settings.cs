using System.Collections.Generic;
using System.ComponentModel.DataAnnotations;
using System.Text.Json.Serialization;

namespace DeeMusic.Desktop.Models
{
    /// <summary>
    /// Represents the application configuration
    /// </summary>
    public class Settings
    {
        [JsonPropertyName("deezer")]
        public DeezerSettings Deezer { get; set; } = new();

        [JsonPropertyName("download")]
        public DownloadSettings Download { get; set; } = new();

        [JsonPropertyName("spotify")]
        public SpotifySettings Spotify { get; set; } = new();

        [JsonPropertyName("lyrics")]
        public LyricsSettings Lyrics { get; set; } = new();

        [JsonPropertyName("network")]
        public NetworkSettings Network { get; set; } = new();

        [JsonPropertyName("system")]
        public SystemSettings System { get; set; } = new();

        [JsonPropertyName("logging")]
        public LoggingSettings Logging { get; set; } = new();
    }

    /// <summary>
    /// Deezer API settings
    /// </summary>
    public class DeezerSettings
    {
        [JsonPropertyName("arl")]
        public string ARL { get; set; } = string.Empty;
    }

    /// <summary>
    /// Download-related settings
    /// </summary>
    public class DownloadSettings
    {
        [JsonPropertyName("output_dir")]
        [Required]
        public string OutputDir { get; set; } = string.Empty;

        [JsonPropertyName("quality")]
        [Required]
        [RegularExpression("^(MP3_320|FLAC)$", ErrorMessage = "Quality must be MP3_320 or FLAC")]
        public string Quality { get; set; } = "MP3_320";

        [JsonPropertyName("concurrent_downloads")]
        [Range(1, 32, ErrorMessage = "Concurrent downloads must be between 1 and 32")]
        public int ConcurrentDownloads { get; set; } = 8;

        [JsonPropertyName("embed_artwork")]
        public bool EmbedArtwork { get; set; } = true;

        [JsonPropertyName("artwork_size")]
        [Range(100, 5000, ErrorMessage = "Artwork size must be between 100 and 5000 pixels")]
        public int ArtworkSize { get; set; } = 1200;

        [JsonPropertyName("save_album_cover")]
        public bool SaveAlbumCover { get; set; } = true;

        [JsonPropertyName("album_cover_size")]
        [Range(100, 5000, ErrorMessage = "Album cover size must be between 100 and 5000 pixels")]
        public int AlbumCoverSize { get; set; } = 1200;

        [JsonPropertyName("album_cover_filename")]
        public string AlbumCoverFilename { get; set; } = "cover";

        [JsonPropertyName("save_artist_image")]
        public bool SaveArtistImage { get; set; } = false;

        [JsonPropertyName("artist_image_size")]
        [Range(100, 5000, ErrorMessage = "Artist image size must be between 100 and 5000 pixels")]
        public int ArtistImageSize { get; set; } = 1000;

        [JsonPropertyName("artist_image_filename")]
        public string ArtistImageFilename { get; set; } = "artist";

        // Filename templates for different download types
        [JsonPropertyName("single_track_template")]
        public string SingleTrackTemplate { get; set; } = "{artist} - {title}";

        [JsonPropertyName("album_track_template")]
        public string AlbumTrackTemplate { get; set; } = "{track_number:02d} - {album_artist} - {title}";

        [JsonPropertyName("playlist_track_template")]
        public string PlaylistTrackTemplate { get; set; } = "{playlist_position:02d} - {album_artist} - {title}";

        // Folder structure options
        [JsonPropertyName("create_playlist_folder")]
        public bool CreatePlaylistFolder { get; set; } = true;

        [JsonPropertyName("create_artist_folder")]
        public bool CreateArtistFolder { get; set; } = true;

        [JsonPropertyName("create_album_folder")]
        public bool CreateAlbumFolder { get; set; } = true;

        [JsonPropertyName("create_cd_folder")]
        public bool CreateCDFolder { get; set; } = false;

        [JsonPropertyName("playlist_folder_structure")]
        public bool PlaylistFolderStructure { get; set; } = true;

        [JsonPropertyName("singles_folder_structure")]
        public bool SinglesFolderStructure { get; set; } = false;

        // Folder name templates
        [JsonPropertyName("playlist_folder_template")]
        public string PlaylistFolderTemplate { get; set; } = "{playlist}";

        [JsonPropertyName("artist_folder_template")]
        public string ArtistFolderTemplate { get; set; } = "{artist}";

        [JsonPropertyName("album_folder_template")]
        public string AlbumFolderTemplate { get; set; } = "{album}";

        [JsonPropertyName("cd_folder_template")]
        public string CDFolderTemplate { get; set; } = "CD {disc_number}";

        // Legacy property for backward compatibility
        [JsonPropertyName("filename_template")]
        public string FilenameTemplate { get; set; } = "{artist} - {title}";

        [JsonPropertyName("folder_structure")]
        public Dictionary<string, string> FolderStructure { get; set; } = new()
        {
            { "track", "{artist}/{album}" },
            { "album", "{artist}/{album}" },
            { "playlist", "Playlists/{playlist}" }
        };
    }

    /// <summary>
    /// Spotify API settings
    /// </summary>
    public class SpotifySettings
    {
        [JsonPropertyName("client_id")]
        public string ClientId { get; set; } = string.Empty;

        [JsonPropertyName("client_secret")]
        public string ClientSecret { get; set; } = string.Empty;
    }

    /// <summary>
    /// Lyrics-related settings
    /// </summary>
    public class LyricsSettings
    {
        [JsonPropertyName("enabled")]
        public bool Enabled { get; set; } = true;

        [JsonPropertyName("synced_lyrics")]
        public bool SyncedLyrics { get; set; } = true;

        [JsonPropertyName("unsynced_lyrics")]
        public bool UnsyncedLyrics { get; set; } = true;

        [JsonPropertyName("embed_synced")]
        public bool EmbedSynced { get; set; } = true;

        [JsonPropertyName("embed_unsynced")]
        public bool EmbedUnsynced { get; set; } = true;

        [JsonPropertyName("save_synced_file")]
        public bool SaveSyncedFile { get; set; } = false;

        [JsonPropertyName("save_unsynced_file")]
        public bool SaveUnsyncedFile { get; set; } = false;

        [JsonPropertyName("language")]
        public string Language { get; set; } = "en";

        // Note: Lyrics files will automatically use the same filename as the audio track
        // Synced lyrics: trackname.lrc
        // Unsynced lyrics: trackname.txt

        // Legacy properties for backward compatibility
        [JsonPropertyName("embed_in_file")]
        public bool EmbedInFile { get; set; } = true;

        [JsonPropertyName("save_separate_file")]
        public bool SaveSeparateFile { get; set; } = false;
    }

    /// <summary>
    /// Network-related settings
    /// </summary>
    public class NetworkSettings
    {
        [JsonPropertyName("proxy_url")]
        public string ProxyUrl { get; set; } = string.Empty;

        [JsonPropertyName("timeout")]
        [Range(1, 300, ErrorMessage = "Timeout must be between 1 and 300 seconds")]
        public int Timeout { get; set; } = 30;

        [JsonPropertyName("max_retries")]
        [Range(0, 10, ErrorMessage = "Max retries must be between 0 and 10")]
        public int MaxRetries { get; set; } = 3;

        [JsonPropertyName("bandwidth_limit")]
        [Range(0, int.MaxValue, ErrorMessage = "Bandwidth limit cannot be negative")]
        public int BandwidthLimit { get; set; } = 0;

        [JsonPropertyName("connections_per_dl")]
        [Range(1, 16, ErrorMessage = "Connections per download must be between 1 and 16")]
        public int ConnectionsPerDL { get; set; } = 1;
    }

    /// <summary>
    /// System integration settings
    /// </summary>
    public class SystemSettings
    {
        [JsonPropertyName("run_on_startup")]
        public bool RunOnStartup { get; set; } = false;

        [JsonPropertyName("minimize_to_tray")]
        public bool MinimizeToTray { get; set; } = true;

        [JsonPropertyName("start_minimized")]
        public bool StartMinimized { get; set; } = false;

        [JsonPropertyName("theme")]
        [RegularExpression("^(dark|light)$", ErrorMessage = "Theme must be 'dark' or 'light'")]
        public string Theme { get; set; } = "dark";

        [JsonPropertyName("language")]
        public string Language { get; set; } = "en";
    }

    /// <summary>
    /// Logging settings
    /// </summary>
    public class LoggingSettings
    {
        [JsonPropertyName("level")]
        [RegularExpression("^(debug|info|warn|error)$", ErrorMessage = "Log level must be debug, info, warn, or error")]
        public string Level { get; set; } = "info";

        [JsonPropertyName("format")]
        [RegularExpression("^(json|console)$", ErrorMessage = "Log format must be json or console")]
        public string Format { get; set; } = "json";

        [JsonPropertyName("output")]
        [RegularExpression("^(file|console|both)$", ErrorMessage = "Log output must be file, console, or both")]
        public string Output { get; set; } = "file";

        [JsonPropertyName("file_path")]
        public string FilePath { get; set; } = string.Empty;

        [JsonPropertyName("max_size_mb")]
        [Range(1, 1000, ErrorMessage = "Max size must be between 1 and 1000 MB")]
        public int MaxSizeMB { get; set; } = 100;

        [JsonPropertyName("max_backups")]
        [Range(0, 100, ErrorMessage = "Max backups must be between 0 and 100")]
        public int MaxBackups { get; set; } = 3;

        [JsonPropertyName("max_age_days")]
        [Range(0, 365, ErrorMessage = "Max age must be between 0 and 365 days")]
        public int MaxAgeDays { get; set; } = 30;

        [JsonPropertyName("compress")]
        public bool Compress { get; set; } = true;
    }
}
