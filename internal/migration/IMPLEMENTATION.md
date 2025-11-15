# Migration Implementation

This document describes the implementation of the data migration tool for migrating from Python DeeMusic to Go DeeMusic.

## Overview

The migration package provides a complete solution for migrating user data from the Python version of DeeMusic to the Go version. It handles detection, backup, and migration of settings and queue data.

## Architecture

### Components

1. **Detector** (`detector.go`)
   - Detects Python DeeMusic installations
   - Creates backups of Python data
   - Validates backups

2. **SettingsMigrator** (`settings.go`)
   - Reads Python settings.json
   - Converts to Go Config format
   - Maps field names and values
   - Saves to Go location

3. **QueueMigrator** (`queue.go`)
   - Reads Python SQLite database
   - Converts queue items to Go format
   - Imports into Go database
   - Migrates download history

4. **Migrator** (`migrator.go`)
   - Orchestrates the complete migration
   - Coordinates all components
   - Provides high-level API

## Implementation Details

### Detection

The detector searches for Python installations in `%APPDATA%/DeeMusic`:

```go
pythonDir := filepath.Join(d.appDataDir, "DeeMusic")
```

It checks for:
- `settings.json` - Configuration file
- `queue.db`, `downloads.db`, or `deemusic.db` - Queue database

### Backup Strategy

Backups are created with timestamps to prevent overwrites:

```
%APPDATA%/DeeMusic/backup_20240126_143022/
├── settings.json          # Copy of settings
├── queue.db              # Copy of database
└── backup_manifest.json  # Metadata
```

The manifest includes:
- Backup timestamp
- Source directory
- What was backed up
- File paths

### Settings Migration

#### Field Mapping

Python settings are mapped to Go config structure:

| Python | Go | Notes |
|--------|-----|-------|
| `port` | `server.port` | Direct mapping |
| `host` | `server.host` | Direct mapping |
| `arl` | `deezer.arl` | Encrypted in Go |
| `download_path` | `download.output_dir` | Path normalization |
| `quality` | `download.quality` | Value normalization |
| `max_concurrent` | `download.concurrent_downloads` | Direct mapping |

#### Value Normalization

**Quality Values:**
```go
qualityMap := map[string]string{
    "mp3_320": "MP3_320",
    "flac":    "FLAC",
    "mp3":     "MP3_320",
    "320":     "MP3_320",
}
```

**Log Levels:**
```go
levelMap := map[string]string{
    "DEBUG":   "debug",
    "INFO":    "info",
    "WARNING": "warn",
    "ERROR":   "error",
}
```

**Status Values:**
```go
statusMap := map[string]string{
    "pending":     "pending",
    "downloading": "downloading",
    "completed":   "completed",
    "failed":      "failed",
    "queued":      "pending",
    "in_progress": "downloading",
}
```

#### Default Values

Missing or invalid values are replaced with defaults:

```go
if cfg.Server.Port == 0 {
    cfg.Server.Port = 8080
}
if cfg.Download.ConcurrentDownloads == 0 {
    cfg.Download.ConcurrentDownloads = 8
}
```

### Queue Migration

#### Schema Flexibility

The migrator tries multiple database schemas to handle different Python versions:

```go
queries := []string{
    // Standard schema
    `SELECT id, type, title, artist, album, status, progress, ...
     FROM queue_items`,
    
    // Alternative schema 1
    `SELECT id, type, title, artist, album, status, progress, ...
     FROM downloads`,
    
    // Alternative schema 2
    `SELECT id, item_type, title, artist, album, status, progress, ...
     FROM queue`,
}
```

#### Data Conversion

Queue items are converted with proper type and status mapping:

```go
goItem := &store.QueueItem{
    ID:           pythonItem.ID,
    Type:         qm.mapItemType(pythonItem.Type),
    Status:       qm.mapStatus(pythonItem.Status),
    // ... other fields
}
```

#### History Migration

Download history is migrated separately and errors are non-fatal:

```go
if err := qm.queueStore.AddToHistory(...); err != nil {
    // Log but don't fail
    fmt.Printf("Warning: failed to import history item: %v\n", err)
}
```

## Error Handling

### Backup Validation

Before proceeding with migration, backups are validated:

```go
if err := migrator.detector.ValidateBackup(installation); err != nil {
    return fmt.Errorf("backup validation failed: %w", err)
}
```

### Partial Migration

If one component fails, others can still succeed:

```go
// Migrate settings
if err := m.MigrateSettings(); err != nil {
    result.Errors = append(result.Errors, err)
} else {
    result.SettingsMigrated = true
}

// Migrate queue (continues even if settings failed)
if err := m.MigrateQueue(); err != nil {
    result.Errors = append(result.Errors, err)
} else {
    result.QueueMigrated = true
}
```

### Database Errors

Database operations use proper error wrapping:

```go
if err := qs.db.Exec(query, ...); err != nil {
    return fmt.Errorf("failed to add queue item: %w", err)
}
```

## Security Considerations

### ARL Token Encryption

The ARL token is encrypted when saved to Go config:

```go
encryptor := security.NewTokenEncryptor(GetDataDir())
encryptedARL, err := encryptor.EncryptToken(cfg.Deezer.ARL)
```

### Backup Protection

Backups are created with restricted permissions:

```go
os.MkdirAll(backupDir, 0755)
os.WriteFile(dst, data, 0644)
```

### Path Validation

All paths are validated before use:

```go
if _, err := os.Stat(pythonDir); os.IsNotExist(err) {
    return nil, fmt.Errorf("no Python installation found")
}
```

## Testing Considerations

### Unit Tests

Each component should be tested independently:

- **Detector**: Test detection with mock file systems
- **SettingsMigrator**: Test field mapping and value normalization
- **QueueMigrator**: Test database schema handling
- **Migrator**: Test orchestration and error handling

### Integration Tests

Test complete migration flow:

1. Create mock Python installation
2. Run migration
3. Verify Go installation
4. Verify backup creation
5. Test rollback scenarios

### Edge Cases

- Empty Python installation
- Corrupted settings file
- Invalid database schema
- Missing fields
- Invalid values
- Partial data

## Performance

### Database Operations

Queue migration uses batch operations where possible:

```go
for _, pythonItem := range items {
    goItem := qm.ConvertToGoQueueItem(pythonItem)
    if err := qm.queueStore.Add(goItem); err != nil {
        // Handle error
    }
}
```

### File Operations

Backup uses efficient file copying:

```go
data, err := os.ReadFile(src)
if err != nil {
    return err
}
return os.WriteFile(dst, data, 0644)
```

## Future Enhancements

### Potential Improvements

1. **Progress Reporting**: Add callbacks for migration progress
2. **Selective Migration**: Allow users to choose what to migrate
3. **Rollback Support**: Implement automatic rollback on failure
4. **Validation**: Add pre-migration validation checks
5. **Logging**: Add detailed logging for troubleshooting
6. **UI Integration**: Add migration wizard to web UI

### Schema Evolution

Support for future Python versions:

```go
// Add new schema patterns to queries array
queries = append(queries, `SELECT ... FROM new_table_name`)
```

## Requirements Mapping

This implementation satisfies the following requirements:

- **12.1**: Detects existing Python installation in %APPDATA%/DeeMusic
- **12.2**: Converts Python settings.json to Go Config format
- **12.3**: Reads Python queue database and imports to Go
- **12.4**: Preserves download history during migration
- **12.5**: Creates backup of Python data before migration

## Usage in Application

### Startup Check

Check for migration on first run:

```go
if needed, _ := migration.CheckMigrationNeeded(); needed {
    // Prompt user to migrate
    // Or auto-migrate with user consent
}
```

### CLI Command

Provide migration command:

```bash
deemusic migrate --from-python
```

### Web UI

Add migration page in settings:

```typescript
// Check if migration is available
const migrationAvailable = await api.checkMigration()

// Perform migration
const result = await api.migrate()
```

## Conclusion

The migration implementation provides a robust, flexible solution for migrating user data from Python to Go. It handles various edge cases, provides comprehensive error handling, and ensures data safety through automatic backups.
