package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/deemusic/deemusic-go/internal/config"
)

// PythonSettings represents the Python version settings structure
type PythonSettings struct {
	// Server settings
	Port int    `json:"port"`
	Host string `json:"host"`
	
	// Deezer settings
	ARL string `json:"arl"`
	
	// Download settings
	DownloadPath        string            `json:"download_path"`
	Quality             string            `json:"quality"`
	MaxConcurrent       int               `json:"max_concurrent"`
	EmbedArtwork        bool              `json:"embed_artwork"`
	ArtworkSize         int               `json:"artwork_size"`
	FilenameTemplate    string            `json:"filename_template"`
	FolderStructure     map[string]string `json:"folder_structure"`
	
	// Spotify settings
	SpotifyClientID     string `json:"spotify_client_id"`
	SpotifyClientSecret string `json:"spotify_client_secret"`
	
	// Lyrics settings
	LyricsEnabled       bool   `json:"lyrics_enabled"`
	EmbedLyrics         bool   `json:"embed_lyrics"`
	SaveLyricsFile      bool   `json:"save_lyrics_file"`
	LyricsLanguage      string `json:"lyrics_language"`
	
	// Network settings
	ProxyURL            string `json:"proxy_url"`
	Timeout             int    `json:"timeout"`
	MaxRetries          int    `json:"max_retries"`
	BandwidthLimit      int    `json:"bandwidth_limit"`
	ConnectionsPerDL    int    `json:"connections_per_dl"`
	
	// Desktop settings
	RunOnStartup        bool `json:"run_on_startup"`
	MinimizeToTray      bool `json:"minimize_to_tray"`
	AutoOpenBrowser     bool `json:"auto_open_browser"`
	
	// Logging settings
	LogLevel            string `json:"log_level"`
	LogFormat           string `json:"log_format"`
	LogOutput           string `json:"log_output"`
}

// SettingsMigrator handles migration of settings from Python to Go
type SettingsMigrator struct {
	pythonSettingsPath string
	goConfigPath       string
}

// NewSettingsMigrator creates a new SettingsMigrator
func NewSettingsMigrator(pythonSettingsPath, goConfigPath string) *SettingsMigrator {
	return &SettingsMigrator{
		pythonSettingsPath: pythonSettingsPath,
		goConfigPath:       goConfigPath,
	}
}

// ReadPythonSettings reads and parses Python settings.json
func (sm *SettingsMigrator) ReadPythonSettings() (*PythonSettings, error) {
	data, err := os.ReadFile(sm.pythonSettingsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Python settings: %w", err)
	}

	var settings PythonSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse Python settings: %w", err)
	}

	return &settings, nil
}

// ConvertToGoConfig converts Python settings to Go Config struct
func (sm *SettingsMigrator) ConvertToGoConfig(pythonSettings *PythonSettings) *config.Config {
	cfg := &config.Config{
		Deezer: config.DeezerConfig{
			ARL: pythonSettings.ARL,
		},
		Download: config.DownloadConfig{
			OutputDir:           pythonSettings.DownloadPath,
			Quality:             sm.mapQuality(pythonSettings.Quality),
			ConcurrentDownloads: pythonSettings.MaxConcurrent,
			EmbedArtwork:        pythonSettings.EmbedArtwork,
			ArtworkSize:         pythonSettings.ArtworkSize,
			FilenameTemplate:    pythonSettings.FilenameTemplate,
			FolderStructure:     pythonSettings.FolderStructure,
		},
		Spotify: config.SpotifyConfig{
			ClientID:     pythonSettings.SpotifyClientID,
			ClientSecret: pythonSettings.SpotifyClientSecret,
		},
		Lyrics: config.LyricsConfig{
			Enabled:          pythonSettings.LyricsEnabled,
			EmbedInFile:      pythonSettings.EmbedLyrics,
			SaveSeparateFile: pythonSettings.SaveLyricsFile,
			Language:         pythonSettings.LyricsLanguage,
		},
		Network: config.NetworkConfig{
			ProxyURL:         pythonSettings.ProxyURL,
			Timeout:          pythonSettings.Timeout,
			MaxRetries:       pythonSettings.MaxRetries,
			BandwidthLimit:   pythonSettings.BandwidthLimit,
			ConnectionsPerDL: pythonSettings.ConnectionsPerDL,
		},
		System: config.SystemConfig{
			RunOnStartup:   pythonSettings.RunOnStartup,
			MinimizeToTray: pythonSettings.MinimizeToTray,
			StartMinimized: false, // Default for new field
			Theme:          "dark", // Default theme
			Language:       "en",   // Default language
		},
		Logging: config.LoggingConfig{
			Level:      sm.mapLogLevel(pythonSettings.LogLevel),
			Format:     sm.mapLogFormat(pythonSettings.LogFormat),
			Output:     sm.mapLogOutput(pythonSettings.LogOutput),
			FilePath:   filepath.Join(config.GetDataDir(), "logs", "app.log"),
			MaxSizeMB:  100,
			MaxBackups: 3,
			MaxAgeDays: 30,
			Compress:   true,
		},
	}

	// Apply defaults for missing values
	sm.applyDefaults(cfg)

	return cfg
}

// mapQuality maps Python quality values to Go quality values
func (sm *SettingsMigrator) mapQuality(pythonQuality string) string {
	qualityMap := map[string]string{
		"mp3_320": "MP3_320",
		"MP3_320": "MP3_320",
		"flac":    "FLAC",
		"FLAC":    "FLAC",
		"mp3":     "MP3_320",
		"320":     "MP3_320",
	}

	if mapped, ok := qualityMap[pythonQuality]; ok {
		return mapped
	}

	return "MP3_320" // Default
}

// mapLogLevel maps Python log levels to Go log levels
func (sm *SettingsMigrator) mapLogLevel(pythonLevel string) string {
	levelMap := map[string]string{
		"DEBUG":   "debug",
		"INFO":    "info",
		"WARNING": "warn",
		"WARN":    "warn",
		"ERROR":   "error",
		"debug":   "debug",
		"info":    "info",
		"warn":    "warn",
		"error":   "error",
	}

	if mapped, ok := levelMap[pythonLevel]; ok {
		return mapped
	}

	return "info" // Default
}

// mapLogFormat maps Python log formats to Go log formats
func (sm *SettingsMigrator) mapLogFormat(pythonFormat string) string {
	formatMap := map[string]string{
		"json":    "json",
		"text":    "console",
		"console": "console",
		"plain":   "console",
	}

	if mapped, ok := formatMap[pythonFormat]; ok {
		return mapped
	}

	return "json" // Default
}

// mapLogOutput maps Python log outputs to Go log outputs
func (sm *SettingsMigrator) mapLogOutput(pythonOutput string) string {
	outputMap := map[string]string{
		"file":    "file",
		"console": "console",
		"both":    "both",
		"stdout":  "console",
	}

	if mapped, ok := outputMap[pythonOutput]; ok {
		return mapped
	}

	return "file" // Default
}

// applyDefaults applies default values for missing or invalid settings
func (sm *SettingsMigrator) applyDefaults(cfg *config.Config) {
	// Download defaults
	if cfg.Download.OutputDir == "" {
		cfg.Download.OutputDir = filepath.Join(config.GetDataDir(), "downloads")
	}
	if cfg.Download.ConcurrentDownloads == 0 {
		cfg.Download.ConcurrentDownloads = 8
	}
	if cfg.Download.ArtworkSize == 0 {
		cfg.Download.ArtworkSize = 1200
	}
	if cfg.Download.FilenameTemplate == "" {
		cfg.Download.FilenameTemplate = "{artist} - {title}"
	}
	if cfg.Download.FolderStructure == nil {
		cfg.Download.FolderStructure = map[string]string{
			"track":    "{artist}/{album}",
			"album":    "{artist}/{album}",
			"playlist": "Playlists/{playlist}",
		}
	}

	// Network defaults
	if cfg.Network.Timeout == 0 {
		cfg.Network.Timeout = 30
	}
	if cfg.Network.MaxRetries == 0 {
		cfg.Network.MaxRetries = 3
	}
	if cfg.Network.ConnectionsPerDL == 0 {
		cfg.Network.ConnectionsPerDL = 1
	}

	// Lyrics defaults
	if cfg.Lyrics.Language == "" {
		cfg.Lyrics.Language = "en"
	}
}

// SaveGoConfig saves the converted config to Go location
func (sm *SettingsMigrator) SaveGoConfig(cfg *config.Config) error {
	// Ensure config directory exists
	configDir := filepath.Dir(sm.goConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save config
	if err := cfg.Save(sm.goConfigPath); err != nil {
		return fmt.Errorf("failed to save Go config: %w", err)
	}

	return nil
}

// Migrate performs the complete settings migration
func (sm *SettingsMigrator) Migrate() error {
	// Read Python settings
	pythonSettings, err := sm.ReadPythonSettings()
	if err != nil {
		return fmt.Errorf("failed to read Python settings: %w", err)
	}

	// Convert to Go config
	goConfig := sm.ConvertToGoConfig(pythonSettings)

	// Validate Go config
	if err := goConfig.Validate(); err != nil {
		return fmt.Errorf("converted config validation failed: %w", err)
	}

	// Save Go config
	if err := sm.SaveGoConfig(goConfig); err != nil {
		return fmt.Errorf("failed to save Go config: %w", err)
	}

	return nil
}
