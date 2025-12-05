package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Deezer   DeezerConfig   `json:"deezer" mapstructure:"deezer"`
	Download DownloadConfig `json:"download" mapstructure:"download"`
	Spotify  SpotifyConfig  `json:"spotify" mapstructure:"spotify"`
	Lyrics   LyricsConfig   `json:"lyrics" mapstructure:"lyrics"`
	Network  NetworkConfig  `json:"network" mapstructure:"network"`
	System   SystemConfig   `json:"system" mapstructure:"system"`
	Logging  LoggingConfig  `json:"logging" mapstructure:"logging"`
}

// DeezerConfig contains Deezer API settings
type DeezerConfig struct {
	ARL string `json:"arl" mapstructure:"arl"`
}

// DownloadConfig contains download-related settings
type DownloadConfig struct {
	OutputDir                string            `json:"output_dir" mapstructure:"output_dir"`
	Quality                  string            `json:"quality" mapstructure:"quality"`
	ConcurrentDownloads      int               `json:"concurrent_downloads" mapstructure:"concurrent_downloads"`
	EmbedArtwork             bool              `json:"embed_artwork" mapstructure:"embed_artwork"`
	ArtworkSize              int               `json:"artwork_size" mapstructure:"artwork_size"`
	SaveAlbumCover           bool              `json:"save_album_cover" mapstructure:"save_album_cover"`
	AlbumCoverSize           int               `json:"album_cover_size" mapstructure:"album_cover_size"`
	AlbumCoverFilename       string            `json:"album_cover_filename" mapstructure:"album_cover_filename"`
	SaveArtistImage          bool              `json:"save_artist_image" mapstructure:"save_artist_image"`
	ArtistImageSize          int               `json:"artist_image_size" mapstructure:"artist_image_size"`
	ArtistImageFilename      string            `json:"artist_image_filename" mapstructure:"artist_image_filename"`
	SingleTrackTemplate      string            `json:"single_track_template" mapstructure:"single_track_template"`
	AlbumTrackTemplate       string            `json:"album_track_template" mapstructure:"album_track_template"`
	PlaylistTrackTemplate    string            `json:"playlist_track_template" mapstructure:"playlist_track_template"`
	CreatePlaylistFolder     bool              `json:"create_playlist_folder" mapstructure:"create_playlist_folder"`
	CreateArtistFolder       bool              `json:"create_artist_folder" mapstructure:"create_artist_folder"`
	CreateAlbumFolder        bool              `json:"create_album_folder" mapstructure:"create_album_folder"`
	CreateCDFolder           bool              `json:"create_cd_folder" mapstructure:"create_cd_folder"`
	PlaylistFolderStructure  bool              `json:"playlist_folder_structure" mapstructure:"playlist_folder_structure"`
	SinglesFolderStructure   bool              `json:"singles_folder_structure" mapstructure:"singles_folder_structure"`
	PlaylistFolderTemplate   string            `json:"playlist_folder_template" mapstructure:"playlist_folder_template"`
	ArtistFolderTemplate     string            `json:"artist_folder_template" mapstructure:"artist_folder_template"`
	AlbumFolderTemplate      string            `json:"album_folder_template" mapstructure:"album_folder_template"`
	CDFolderTemplate         string            `json:"cd_folder_template" mapstructure:"cd_folder_template"`
	FilenameTemplate         string            `json:"filename_template" mapstructure:"filename_template"`
	FolderStructure          map[string]string `json:"folder_structure" mapstructure:"folder_structure"`
}

// SpotifyConfig contains Spotify API settings
type SpotifyConfig struct {
	ClientID     string `json:"client_id" mapstructure:"client_id"`
	ClientSecret string `json:"client_secret" mapstructure:"client_secret"`
}

// LyricsConfig contains lyrics-related settings
type LyricsConfig struct {
	Enabled          bool   `json:"enabled" mapstructure:"enabled"`
	SyncedLyrics     bool   `json:"synced_lyrics" mapstructure:"synced_lyrics"`
	UnsyncedLyrics   bool   `json:"unsynced_lyrics" mapstructure:"unsynced_lyrics"`
	EmbedSynced      bool   `json:"embed_synced" mapstructure:"embed_synced"`
	EmbedUnsynced    bool   `json:"embed_unsynced" mapstructure:"embed_unsynced"`
	SaveSyncedFile   bool   `json:"save_synced_file" mapstructure:"save_synced_file"`
	SaveUnsyncedFile bool   `json:"save_unsynced_file" mapstructure:"save_unsynced_file"`
	EmbedInFile      bool   `json:"embed_in_file" mapstructure:"embed_in_file"`
	SaveSeparateFile bool   `json:"save_separate_file" mapstructure:"save_separate_file"`
	Language         string `json:"language" mapstructure:"language"`
}

// NetworkConfig contains network-related settings
type NetworkConfig struct {
	ProxyURL         string `json:"proxy_url" mapstructure:"proxy_url"`
	Timeout          int    `json:"timeout" mapstructure:"timeout"`
	MaxRetries       int    `json:"max_retries" mapstructure:"max_retries"`
	BandwidthLimit   int    `json:"bandwidth_limit" mapstructure:"bandwidth_limit"`
	ConnectionsPerDL int    `json:"connections_per_dl" mapstructure:"connections_per_dl"`
}

// SystemConfig contains system integration settings
type SystemConfig struct {
	RunOnStartup   bool   `json:"run_on_startup" mapstructure:"run_on_startup"`
	MinimizeToTray bool   `json:"minimize_to_tray" mapstructure:"minimize_to_tray"`
	StartMinimized bool   `json:"start_minimized" mapstructure:"start_minimized"`
	Theme          string `json:"theme" mapstructure:"theme"` // "dark" or "light"
	Language       string `json:"language" mapstructure:"language"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level      string `json:"level" mapstructure:"level"`
	Format     string `json:"format" mapstructure:"format"`
	Output     string `json:"output" mapstructure:"output"`
	FilePath   string `json:"file_path" mapstructure:"file_path"`
	MaxSizeMB  int    `json:"max_size_mb" mapstructure:"max_size_mb"`
	MaxBackups int    `json:"max_backups" mapstructure:"max_backups"`
	MaxAgeDays int    `json:"max_age_days" mapstructure:"max_age_days"`
	Compress   bool   `json:"compress" mapstructure:"compress"`
}

// Load loads configuration from file or creates default
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Determine config path
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	// Set config file
	v.SetConfigFile(configPath)
	v.SetConfigType("json")

	// Ensure config directory exists
	if err := ensureConfigDir(configPath); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Read config file if it exists
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, create with defaults
			if err := v.WriteConfigAs(configPath); err != nil {
				return nil, fmt.Errorf("failed to write default config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	// Allow environment variable overrides
	v.AutomaticEnv()
	v.SetEnvPrefix("DEEMUSIC")

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// No encryption/decryption needed - ARL is stored in plain text
	// The settings file is already in the user's AppData folder with appropriate permissions

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Download validation
	if c.Download.ConcurrentDownloads < 1 {
		return fmt.Errorf("concurrent downloads must be at least 1")
	}

	if c.Download.ConcurrentDownloads > 32 {
		return fmt.Errorf("concurrent downloads cannot exceed 32")
	}

	if c.Download.Quality != "MP3_320" && c.Download.Quality != "FLAC" {
		return fmt.Errorf("invalid quality: %s (must be MP3_320 or FLAC)", c.Download.Quality)
	}

	if c.Download.OutputDir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}

	if c.Download.ArtworkSize < 100 || c.Download.ArtworkSize > 5000 {
		return fmt.Errorf("artwork size must be between 100 and 5000 pixels")
	}

	// Network validation
	if c.Network.Timeout < 1 {
		return fmt.Errorf("network timeout must be at least 1 second")
	}

	if c.Network.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	if c.Network.ConnectionsPerDL < 1 {
		return fmt.Errorf("connections per download must be at least 1")
	}

	// Lyrics validation
	if c.Lyrics.Language == "" {
		c.Lyrics.Language = "en"
	}

	// System validation
	if c.System.Theme != "dark" && c.System.Theme != "light" {
		return fmt.Errorf("invalid theme: %s (must be dark or light)", c.System.Theme)
	}

	if c.System.Language == "" {
		c.System.Language = "en"
	}

	// Logging validation
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Logging.Level)
	}

	validFormats := map[string]bool{"json": true, "console": true}
	if !validFormats[c.Logging.Format] {
		return fmt.Errorf("invalid log format: %s (must be json or console)", c.Logging.Format)
	}

	validOutputs := map[string]bool{"file": true, "console": true, "both": true}
	if !validOutputs[c.Logging.Output] {
		return fmt.Errorf("invalid log output: %s (must be file, console, or both)", c.Logging.Output)
	}

	if c.Logging.MaxSizeMB < 1 {
		return fmt.Errorf("log max size must be at least 1 MB")
	}

	if c.Logging.MaxBackups < 0 {
		return fmt.Errorf("log max backups cannot be negative")
	}

	if c.Logging.MaxAgeDays < 0 {
		return fmt.Errorf("log max age cannot be negative")
	}

	return nil
}

// Save saves the configuration to file
func (c *Config) Save(path string) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("json")

	// Set all values directly without encryption
	v.Set("deezer", c.Deezer)
	v.Set("download", c.Download)
	v.Set("spotify", c.Spotify)
	v.Set("lyrics", c.Lyrics)
	v.Set("network", c.Network)
	v.Set("system", c.System)
	v.Set("logging", c.Logging)

	return v.WriteConfig()
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Download defaults
	v.SetDefault("download.output_dir", getDefaultDownloadDir())
	v.SetDefault("download.quality", "MP3_320")
	v.SetDefault("download.concurrent_downloads", 8)
	v.SetDefault("download.embed_artwork", true)
	v.SetDefault("download.artwork_size", 1200)
	v.SetDefault("download.filename_template", "{artist} - {title}")
	v.SetDefault("download.folder_structure", map[string]string{
		"track":    "{artist}/{album}",
		"album":    "{artist}/{album}",
		"playlist": "Playlists/{playlist}",
	})

	// Lyrics defaults
	v.SetDefault("lyrics.enabled", true)
	v.SetDefault("lyrics.embed_in_file", true)
	v.SetDefault("lyrics.save_synced_file", true)  // Save .lrc files
	v.SetDefault("lyrics.save_separate_file", false)
	v.SetDefault("lyrics.language", "en")

	// Network defaults
	v.SetDefault("network.timeout", 30)
	v.SetDefault("network.max_retries", 3)
	v.SetDefault("network.bandwidth_limit", 0)
	v.SetDefault("network.connections_per_dl", 1)

	// System defaults
	v.SetDefault("system.run_on_startup", false)
	v.SetDefault("system.minimize_to_tray", true)
	v.SetDefault("system.start_minimized", false)
	v.SetDefault("system.theme", "dark")
	v.SetDefault("system.language", "en")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "file")
	v.SetDefault("logging.file_path", filepath.Join(GetDataDir(), "logs", "app.log"))
	v.SetDefault("logging.max_size_mb", 100)
	v.SetDefault("logging.max_backups", 3)
	v.SetDefault("logging.max_age_days", 30)
	v.SetDefault("logging.compress", true)
}

// getDefaultConfigPath returns the default configuration file path
func getDefaultConfigPath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.Getenv("HOME")
	}
	return filepath.Join(appData, "DeeMusicV2", "settings.json")
}

// getDefaultDownloadDir returns the default download directory
func getDefaultDownloadDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.Getenv("HOME")
	}
	return filepath.Join(appData, "DeeMusicV2", "downloads")
}

// ensureConfigDir ensures the configuration directory exists
func ensureConfigDir(configPath string) error {
	dir := filepath.Dir(configPath)
	return os.MkdirAll(dir, 0755)
}

// Reload reloads the configuration from file
func (c *Config) Reload(configPath string) error {
	newConfig, err := Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	// Update current config
	*c = *newConfig
	return nil
}

// GetDataDir returns the application data directory
func GetDataDir() string {
	// Check if running in portable mode
	if IsPortableMode() {
		exePath, err := os.Executable()
		if err != nil {
			// Fallback to current directory
			return "."
		}
		return filepath.Dir(exePath)
	}
	
	// Standard installation mode
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.Getenv("HOME")
	}
	return filepath.Join(appData, "DeeMusicV2")
}

// IsPortableMode checks if the application is running in portable mode
func IsPortableMode() bool {
	exePath, err := os.Executable()
	if err != nil {
		return false
	}
	exeDir := filepath.Dir(exePath)
	portableMarker := filepath.Join(exeDir, ".portable")
	_, err = os.Stat(portableMarker)
	return err == nil
}

// GetConfigPath returns the configuration file path based on mode
func GetConfigPath() string {
	if IsPortableMode() {
		exePath, _ := os.Executable()
		exeDir := filepath.Dir(exePath)
		return filepath.Join(exeDir, "settings.json")
	}
	return getDefaultConfigPath()
}
