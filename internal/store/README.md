# Store Package

This package provides database management and queue storage functionality for DeeMusic.

## Components

### Database Initialization (`db.go`)

- **InitDB**: Initializes SQLite database connection and runs migrations
- **GetDefaultDBPath**: Returns the default database path based on OS

### Migrations (`migrations.go`)

- Automatic schema versioning and migration system
- Creates tables: `queue_items`, `download_history`, `config_cache`, `schema_migrations`
- Adds indexes for performance optimization

### Queue Store (`queue.go`)

Main interface for managing download queue items.

#### Key Types

- **QueueItem**: Represents a download queue item (track, album, or playlist)
- **QueueStats**: Statistics about queue status

#### CRUD Operations

- `Add(item *QueueItem)`: Add new item to queue
- `Update(item *QueueItem)`: Update existing item
- `Delete(id string)`: Remove item from queue
- `GetByID(id string)`: Retrieve specific item
- `GetPending(limit int)`: Get pending items for processing
- `GetAll(offset, limit int)`: Get all items with pagination

#### Statistics

- `GetStats()`: Get queue statistics (total, pending, downloading, completed, failed)
- `ClearCompleted()`: Remove all completed items

#### History Management

- `AddToHistory()`: Record completed download
- `GetHistory(offset, limit int)`: Retrieve download history

#### Configuration Cache

- `SetConfigCache(key, value string)`: Store configuration cache
- `GetConfigCache(key string)`: Retrieve configuration cache

## Database Schema

### queue_items

Stores download queue items with status tracking.

```sql
CREATE TABLE queue_items (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,              -- track, album, playlist
    title TEXT NOT NULL,
    artist TEXT,
    album TEXT,
    status TEXT NOT NULL,            -- pending, downloading, completed, failed
    progress INTEGER DEFAULT 0,      -- 0-100
    download_url TEXT,
    output_path TEXT,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    metadata_json TEXT,              -- JSON metadata storage
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);
```

### download_history

Records completed downloads for tracking purposes.

```sql
CREATE TABLE download_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    track_id TEXT NOT NULL,
    title TEXT NOT NULL,
    artist TEXT,
    album TEXT,
    file_path TEXT,
    file_size INTEGER,
    quality TEXT,
    downloaded_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### config_cache

Stores temporary configuration and cache data.

```sql
CREATE TABLE config_cache (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Usage Example

```go
package main

import (
    "log"
    "github.com/deemusic/deemusic-go/internal/store"
)

func main() {
    // Initialize database
    db, err := store.InitDB(store.GetDefaultDBPath())
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create queue store
    queueStore := store.NewQueueStore(db)

    // Add item to queue
    item := &store.QueueItem{
        ID:     "track-123",
        Type:   "track",
        Title:  "My Song",
        Artist: "Artist Name",
        Status: "pending",
    }

    if err := queueStore.Add(item); err != nil {
        log.Fatal(err)
    }

    // Get queue statistics
    stats, err := queueStore.GetStats()
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Queue stats: %+v", stats)
}
```

## Requirements Satisfied

This implementation satisfies the following requirements from the design document:

- **Requirement 3.1**: SQLite database for persistent queue management
- **Requirement 8.1**: Queue persistence across application restarts
- **Requirement 3.2**: Pagination support for large queues
- **Requirement 8.2**: Queue management operations (pause, resume, cancel)
- **Requirement 8.4**: Queue statistics tracking

## Notes

- SQLite requires CGO to be enabled (`CGO_ENABLED=1`)
- On Windows, a C compiler (MinGW/GCC) is required for building
- The database uses WAL mode for better concurrent access
- All timestamps are stored in UTC
- Metadata can be stored as JSON for flexible data structures
