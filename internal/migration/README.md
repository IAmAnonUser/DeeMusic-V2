# Migration Package

The migration package handles migrating data from the Python version of DeeMusic to the Go version.

## Features

- **Python Installation Detection**: Automatically detects existing Python DeeMusic installations in `%APPDATA%/DeeMusic`
- **Automatic Backup**: Creates timestamped backups of Python data before migration
- **Settings Migration**: Converts Python `settings.json` to Go configuration format
- **Queue Migration**: Migrates download queue and history from Python SQLite database
- **Flexible Schema Support**: Handles different Python database schemas

## Usage

### Basic Migration

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/deemusic/deemusic-go/internal/migration"
)

func main() {
    // Create migrator
    migrator := migration.NewMigrator()
    
    // Perform migration
    result := migrator.Migrate()
    
    // Check results
    if len(result.Errors) > 0 {
        log.Printf("Migration completed with errors:")
        for _, err := range result.Errors {
            log.Printf("  - %v", err)
        }
    }
    
    if result.SettingsMigrated {
        log.Println("✓ Settings migrated successfully")
    }
    
    if result.QueueMigrated {
        log.Println("✓ Queue migrated successfully")
    }
    
    if result.HistoryMigrated {
        log.Println("✓ History migrated successfully")
    }
    
    log.Printf("Backup created at: %s", result.BackupPath)
}
```

### Check if Migration is Needed

```go
needed, err := migration.CheckMigrationNeeded()
if err != nil {
    log.Fatal(err)
}

if needed {
    fmt.Println("Python installation detected. Migration recommended.")
} else {
    fmt.Println("No migration needed.")
}
```

### Get Migration Info

```go
info, err := migration.GetMigrationInfo()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Python directory: %s\n", info["python_dir"])
fmt.Printf("Has settings: %v\n", info["has_settings"])
fmt.Printf("Has queue: %v\n", info["has_queue"])
```

### Step-by-Step Migration

```go
migrator := migration.NewMigrator()

// 1. Detect Python installation
installation, err := migrator.DetectPythonInstallation()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Found Python installation at: %s\n", installation.DataDir)
fmt.Printf("Has settings: %v\n", installation.HasSettings)
fmt.Printf("Has queue: %v\n", installation.HasQueue)

// 2. Create backup
if err := migrator.CreateBackup(); err != nil {
    log.Fatal(err)
}

fmt.Printf("Backup created at: %s\n", installation.BackupPath)

// 3. Migrate settings
if installation.HasSettings {
    if err := migrator.MigrateSettings(); err != nil {
        log.Printf("Settings migration failed: %v", err)
    } else {
        fmt.Println("✓ Settings migrated")
    }
}

// 4. Migrate queue
if installation.HasQueue {
    if err := migrator.MigrateQueue(); err != nil {
        log.Printf("Queue migration failed: %v", err)
    } else {
        fmt.Println("✓ Queue migrated")
    }
}
```

## Migration Process

### 1. Detection

The migrator looks for Python DeeMusic installation in:
- Windows: `%APPDATA%/DeeMusic`
- Other: `$HOME/DeeMusic`

It checks for:
- `settings.json` - Configuration file
- `queue.db`, `downloads.db`, or `deemusic.db` - Queue database

### 2. Backup

Before migration, a timestamped backup is created:
```
%APPDATA%/DeeMusic/backup_20240126_143022/
├── settings.json
├── queue.db
└── backup_manifest.json
```

The backup manifest contains metadata about the backup.

### 3. Settings Migration

Python settings are converted to Go format with field mapping:

| Python Field | Go Field |
|-------------|----------|
| `port` | `server.port` |
| `host` | `server.host` |
| `arl` | `deezer.arl` |
| `download_path` | `download.output_dir` |
| `quality` | `download.quality` |
| `max_concurrent` | `download.concurrent_downloads` |

Quality values are normalized:
- `mp3_320`, `mp3`, `320` → `MP3_320`
- `flac` → `FLAC`

### 4. Queue Migration

Queue items are migrated with status mapping:

| Python Status | Go Status |
|--------------|-----------|
| `pending`, `queued` | `pending` |
| `downloading`, `in_progress` | `downloading` |
| `completed`, `done` | `completed` |
| `failed`, `error` | `failed` |

Download history is also migrated to preserve completed downloads.

## Error Handling

The migration process is designed to be resilient:

- **Backup Validation**: Ensures backup was created successfully before proceeding
- **Partial Migration**: If settings migration fails, queue migration can still succeed
- **Schema Flexibility**: Tries multiple database schemas to handle different Python versions
- **Non-Fatal History**: History migration errors don't fail the entire process

## Data Locations

### Python (Old)
- Config: `%APPDATA%/DeeMusic/settings.json`
- Database: `%APPDATA%/DeeMusic/queue.db`
- Backups: `%APPDATA%/DeeMusic/backup_*/`

### Go (New)
- Config: `%APPDATA%/DeeMusicV2/settings.json`
- Database: `%APPDATA%/DeeMusicV2/deemusic.db`
- Logs: `%APPDATA%/DeeMusicV2/logs/`

## Requirements

- Python installation must be in `%APPDATA%/DeeMusic`
- Python database must be SQLite format
- Go installation directory must be writable

## Limitations

- Only SQLite queue databases are supported
- Custom Python database schemas may require manual migration
- Partial downloads are not migrated (only completed and pending items)
- Python-specific settings that don't exist in Go are ignored
