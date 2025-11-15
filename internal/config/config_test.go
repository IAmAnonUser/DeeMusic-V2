package config

import (
	"path/filepath"
	"testing"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Download: DownloadConfig{
					Quality:             "MP3_320",
					ConcurrentDownloads: 8,
					OutputDir:           "/tmp/downloads",
					ArtworkSize:         1200,
				},
				Network: NetworkConfig{
					Timeout:          30,
					MaxRetries:       3,
					ConnectionsPerDL: 1,
				},
				System: SystemConfig{
					Theme:    "dark",
					Language: "en",
				},
				Logging: LoggingConfig{
					Level:      "info",
					Format:     "json",
					Output:     "console",
					MaxSizeMB:  10,
					MaxBackups: 3,
					MaxAgeDays: 7,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid quality",
			config: Config{
				Download: DownloadConfig{
					Quality:             "INVALID",
					ConcurrentDownloads: 8,
					OutputDir:           "/tmp/downloads",
					ArtworkSize:         1200,
				},
				Network: NetworkConfig{
					Timeout:          30,
					ConnectionsPerDL: 1,
				},
				System: SystemConfig{
					Theme:    "dark",
					Language: "en",
				},
				Logging: LoggingConfig{
					Level:      "info",
					Format:     "json",
					Output:     "console",
					MaxSizeMB:  10,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid concurrent downloads",
			config: Config{
				Download: DownloadConfig{
					Quality:             "MP3_320",
					ConcurrentDownloads: 0,
					OutputDir:           "/tmp/downloads",
					ArtworkSize:         1200,
				},
				Network: NetworkConfig{
					Timeout:          30,
					ConnectionsPerDL: 1,
				},
				System: SystemConfig{
					Theme:    "dark",
					Language: "en",
				},
				Logging: LoggingConfig{
					Level:      "info",
					Format:     "json",
					Output:     "console",
					MaxSizeMB:  10,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid theme",
			config: Config{
				Download: DownloadConfig{
					Quality:             "MP3_320",
					ConcurrentDownloads: 8,
					OutputDir:           "/tmp/downloads",
					ArtworkSize:         1200,
				},
				Network: NetworkConfig{
					Timeout:          30,
					ConnectionsPerDL: 1,
				},
				System: SystemConfig{
					Theme:    "invalid",
					Language: "en",
				},
				Logging: LoggingConfig{
					Level:      "info",
					Format:     "json",
					Output:     "console",
					MaxSizeMB:  10,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create temporary config directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "settings.json")

	// Create a valid config file first
	validConfig := &Config{
		Download: DownloadConfig{
			Quality:             "MP3_320",
			ConcurrentDownloads: 8,
			OutputDir:           tmpDir,
			ArtworkSize:         1200,
		},
		Network: NetworkConfig{
			Timeout:          30,
			MaxRetries:       3,
			ConnectionsPerDL: 1,
		},
		System: SystemConfig{
			Theme:          "dark",
			Language:       "en",
			MinimizeToTray: true,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "console",
			MaxSizeMB:  10,
			MaxBackups: 3,
			MaxAgeDays: 7,
		},
	}

	// Save the config first
	if err := validConfig.Save(configPath); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Now test loading it
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Download.Quality != "MP3_320" {
		t.Errorf("Expected quality MP3_320, got %s", cfg.Download.Quality)
	}

	if cfg.System.Theme != "dark" {
		t.Errorf("Expected theme dark, got %s", cfg.System.Theme)
	}

	if cfg.System.Language != "en" {
		t.Errorf("Expected language en, got %s", cfg.System.Language)
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "settings.json")

	cfg := &Config{
		Download: DownloadConfig{
			Quality:             "FLAC",
			ConcurrentDownloads: 4,
			OutputDir:           tmpDir,
			ArtworkSize:         1200,
		},
		Network: NetworkConfig{
			Timeout:          30,
			MaxRetries:       3,
			ConnectionsPerDL: 1,
		},
		System: SystemConfig{
			Theme:          "light",
			Language:       "en",
			RunOnStartup:   true,
			MinimizeToTray: false,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "console",
			MaxSizeMB:  10,
			MaxBackups: 3,
			MaxAgeDays: 7,
		},
	}

	// Save config
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load config back
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loadedCfg.Download.Quality != "FLAC" {
		t.Errorf("Expected quality FLAC, got %s", loadedCfg.Download.Quality)
	}

	if loadedCfg.System.Theme != "light" {
		t.Errorf("Expected theme light, got %s", loadedCfg.System.Theme)
	}

	if !loadedCfg.System.RunOnStartup {
		t.Error("Expected RunOnStartup to be true")
	}
}
