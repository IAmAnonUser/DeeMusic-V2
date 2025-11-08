package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PythonInstallation represents a detected Python DeeMusic installation
type PythonInstallation struct {
	DataDir         string
	SettingsPath    string
	QueueDBPath     string
	HasSettings     bool
	HasQueue        bool
	BackupPath      string
	DetectedAt      time.Time
}

// Detector handles detection of Python DeeMusic installations
type Detector struct {
	appDataDir string
}

// NewDetector creates a new Detector
func NewDetector() *Detector {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = os.Getenv("HOME")
	}
	return &Detector{
		appDataDir: appData,
	}
}

// DetectPythonInstallation detects existing Python DeeMusic installation
func (d *Detector) DetectPythonInstallation() (*PythonInstallation, error) {
	pythonDir := filepath.Join(d.appDataDir, "DeeMusic")
	
	// Check if Python installation directory exists
	if _, err := os.Stat(pythonDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no Python DeeMusic installation found at %s", pythonDir)
	}

	installation := &PythonInstallation{
		DataDir:    pythonDir,
		DetectedAt: time.Now(),
	}

	// Check for settings.json
	settingsPath := filepath.Join(pythonDir, "settings.json")
	if _, err := os.Stat(settingsPath); err == nil {
		installation.SettingsPath = settingsPath
		installation.HasSettings = true
	}

	// Check for queue database (common SQLite names)
	queuePaths := []string{
		filepath.Join(pythonDir, "queue.db"),
		filepath.Join(pythonDir, "downloads.db"),
		filepath.Join(pythonDir, "deemusic.db"),
	}

	for _, queuePath := range queuePaths {
		if _, err := os.Stat(queuePath); err == nil {
			installation.QueueDBPath = queuePath
			installation.HasQueue = true
			break
		}
	}

	return installation, nil
}

// CreateBackup creates a backup of Python data before migration
func (d *Detector) CreateBackup(installation *PythonInstallation) error {
	if installation == nil {
		return fmt.Errorf("no installation to backup")
	}

	// Create backup directory with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupDir := filepath.Join(installation.DataDir, fmt.Sprintf("backup_%s", timestamp))
	
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	installation.BackupPath = backupDir

	// Backup settings.json if exists
	if installation.HasSettings {
		if err := d.copyFile(installation.SettingsPath, filepath.Join(backupDir, "settings.json")); err != nil {
			return fmt.Errorf("failed to backup settings: %w", err)
		}
	}

	// Backup queue database if exists
	if installation.HasQueue {
		dbFileName := filepath.Base(installation.QueueDBPath)
		if err := d.copyFile(installation.QueueDBPath, filepath.Join(backupDir, dbFileName)); err != nil {
			return fmt.Errorf("failed to backup queue database: %w", err)
		}
	}

	// Create backup manifest
	manifest := map[string]interface{}{
		"backup_date":    installation.DetectedAt,
		"source_dir":     installation.DataDir,
		"has_settings":   installation.HasSettings,
		"has_queue":      installation.HasQueue,
		"settings_path":  installation.SettingsPath,
		"queue_db_path":  installation.QueueDBPath,
	}

	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to create backup manifest: %w", err)
	}

	manifestPath := filepath.Join(backupDir, "backup_manifest.json")
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write backup manifest: %w", err)
	}

	return nil
}

// copyFile copies a file from src to dst
func (d *Detector) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}

// ValidateBackup validates that backup was created successfully
func (d *Detector) ValidateBackup(installation *PythonInstallation) error {
	if installation.BackupPath == "" {
		return fmt.Errorf("no backup path set")
	}

	// Check backup directory exists
	if _, err := os.Stat(installation.BackupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup directory does not exist: %s", installation.BackupPath)
	}

	// Verify manifest exists
	manifestPath := filepath.Join(installation.BackupPath, "backup_manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("backup manifest not found")
	}

	// Verify backed up files exist
	if installation.HasSettings {
		backupSettings := filepath.Join(installation.BackupPath, "settings.json")
		if _, err := os.Stat(backupSettings); os.IsNotExist(err) {
			return fmt.Errorf("settings backup not found")
		}
	}

	if installation.HasQueue {
		dbFileName := filepath.Base(installation.QueueDBPath)
		backupDB := filepath.Join(installation.BackupPath, dbFileName)
		if _, err := os.Stat(backupDB); os.IsNotExist(err) {
			return fmt.Errorf("queue database backup not found")
		}
	}

	return nil
}
