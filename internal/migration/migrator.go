package migration

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/deemusic/deemusic-go/internal/config"
)

// Migrator orchestrates the complete migration process
type Migrator struct {
	detector          *Detector
	settingsMigrator  *SettingsMigrator
	queueMigrator     *QueueMigrator
	installation      *PythonInstallation
	goConfigPath      string
	goDBPath          string
}

// MigrationResult contains the results of the migration
type MigrationResult struct {
	SettingsMigrated bool
	QueueMigrated    bool
	HistoryMigrated  bool
	BackupPath       string
	Errors           []error
}

// NewMigrator creates a new Migrator
func NewMigrator() *Migrator {
	detector := NewDetector()
	
	// Determine Go paths
	goConfigPath := config.GetConfigPath()
	goDBPath := filepath.Join(config.GetDataDir(), "deemusic.db")

	return &Migrator{
		detector:     detector,
		goConfigPath: goConfigPath,
		goDBPath:     goDBPath,
	}
}

// DetectPythonInstallation detects Python DeeMusic installation
func (m *Migrator) DetectPythonInstallation() (*PythonInstallation, error) {
	installation, err := m.detector.DetectPythonInstallation()
	if err != nil {
		return nil, err
	}

	m.installation = installation
	return installation, nil
}

// CreateBackup creates a backup of Python data
func (m *Migrator) CreateBackup() error {
	if m.installation == nil {
		return fmt.Errorf("no Python installation detected")
	}

	return m.detector.CreateBackup(m.installation)
}

// MigrateSettings migrates settings from Python to Go
func (m *Migrator) MigrateSettings() error {
	if m.installation == nil || !m.installation.HasSettings {
		return fmt.Errorf("no Python settings found to migrate")
	}

	m.settingsMigrator = NewSettingsMigrator(m.installation.SettingsPath, m.goConfigPath)
	return m.settingsMigrator.Migrate()
}

// MigrateQueue migrates queue and history from Python to Go
func (m *Migrator) MigrateQueue() error {
	if m.installation == nil || !m.installation.HasQueue {
		return fmt.Errorf("no Python queue database found to migrate")
	}

	m.queueMigrator = NewQueueMigrator(m.installation.QueueDBPath, m.goDBPath)
	return m.queueMigrator.Migrate()
}

// Migrate performs the complete migration process
func (m *Migrator) Migrate() *MigrationResult {
	result := &MigrationResult{
		Errors: []error{},
	}

	// Detect Python installation
	installation, err := m.DetectPythonInstallation()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("detection failed: %w", err))
		return result
	}

	// Create backup
	if err := m.CreateBackup(); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("backup failed: %w", err))
		return result
	}
	result.BackupPath = installation.BackupPath

	// Validate backup
	if err := m.detector.ValidateBackup(installation); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("backup validation failed: %w", err))
		return result
	}

	// Migrate settings
	if installation.HasSettings {
		if err := m.MigrateSettings(); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("settings migration failed: %w", err))
		} else {
			result.SettingsMigrated = true
		}
	}

	// Migrate queue
	if installation.HasQueue {
		if err := m.MigrateQueue(); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("queue migration failed: %w", err))
		} else {
			result.QueueMigrated = true
			result.HistoryMigrated = true
		}
	}

	return result
}

// CheckMigrationNeeded checks if migration is needed
func CheckMigrationNeeded() (bool, error) {
	detector := NewDetector()
	installation, err := detector.DetectPythonInstallation()
	if err != nil {
		return false, nil // No Python installation found
	}

	// Check if Go installation already exists
	goConfigPath := config.GetConfigPath()
	if _, err := os.Stat(goConfigPath); err == nil {
		// Go config exists, check if it was migrated
		cfg, err := config.Load(goConfigPath)
		if err == nil && cfg.Deezer.ARL != "" {
			// Go installation appears to be configured, migration may not be needed
			return false, nil
		}
	}

	// Python installation exists and Go is not configured
	return installation.HasSettings || installation.HasQueue, nil
}

// GetMigrationInfo returns information about what can be migrated
func GetMigrationInfo() (map[string]interface{}, error) {
	detector := NewDetector()
	installation, err := detector.DetectPythonInstallation()
	if err != nil {
		return nil, err
	}

	info := map[string]interface{}{
		"python_dir":    installation.DataDir,
		"has_settings":  installation.HasSettings,
		"has_queue":     installation.HasQueue,
		"settings_path": installation.SettingsPath,
		"queue_path":    installation.QueueDBPath,
		"detected_at":   installation.DetectedAt,
	}

	return info, nil
}
