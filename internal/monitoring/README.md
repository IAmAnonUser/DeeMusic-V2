# Monitoring Package

This package provides monitoring and observability features for DeeMusic Go, including Prometheus metrics, structured logging, and health checks.

## Features

### Prometheus Metrics

The application exposes Prometheus-compatible metrics at the `/metrics` endpoint:

#### Download Metrics
- `deemusic_downloads_total{status, quality}` - Total number of downloads (counter)
- `deemusic_download_duration_seconds{quality}` - Download duration histogram
- `deemusic_download_bytes_total` - Total bytes downloaded (counter)
- `deemusic_active_downloads` - Number of active downloads (gauge)
- `deemusic_queue_size` - Current queue size (gauge)

#### API Metrics
- `deemusic_api_requests_total{endpoint, status}` - Total API requests (counter)
- `deemusic_api_request_duration_seconds{endpoint}` - API request duration histogram

#### System Metrics
- `deemusic_decryption_duration_seconds` - Decryption duration histogram
- `deemusic_errors_total{type}` - Total errors by type (counter)

### Structured Logging

The package uses Zap for structured logging with the following features:
- Configurable log levels (debug, info, warn, error)
- Log rotation with configurable retention
- JSON and console output formats
- Contextual logging with fields

### Health Checks

The `/health` endpoint provides application health status:
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": 3600,
  "queue_size": 42,
  "active_downloads": 8,
  "memory_usage_mb": 128,
  "database_status": "connected"
}
```

## Usage

### Recording Metrics

```go
import "github.com/deemusic/deemusic-go/internal/monitoring"

// Record download start
monitoring.RecordDownloadStart("MP3_320")

// Record download completion
monitoring.RecordDownloadComplete("MP3_320", duration, bytes)

// Record download failure
monitoring.RecordDownloadFailed("MP3_320", "network_error")

// Update queue size
monitoring.UpdateQueueSize(42)

// Record API request
monitoring.RecordAPIRequest("/api/v1/search", "success", duration)
```

### Structured Logging

```go
import "github.com/deemusic/deemusic-go/internal/monitoring"

// Create logger
logger, err := monitoring.NewLogger(config)
if err != nil {
    log.Fatal(err)
}
defer logger.Sync()

// Log with context
logger.Info("Download started",
    zap.String("track_id", trackID),
    zap.String("quality", quality),
)

logger.Error("Download failed",
    zap.String("track_id", trackID),
    zap.Error(err),
)
```

## Configuration

Logging configuration in `config.json`:
```json
{
  "logging": {
    "level": "info",
    "format": "json",
    "output": "file",
    "file_path": "%APPDATA%/DeeMusic/logs/app.log",
    "max_size_mb": 100,
    "max_backups": 3,
    "max_age_days": 30,
    "compress": true
  }
}
```

## Monitoring Best Practices

1. **Use appropriate metric types**:
   - Counters for cumulative values (downloads, errors)
   - Gauges for current values (queue size, active downloads)
   - Histograms for distributions (duration, size)

2. **Add meaningful labels**:
   - Keep cardinality low (avoid user IDs, track IDs)
   - Use consistent label names
   - Document label values

3. **Log at appropriate levels**:
   - Debug: Detailed diagnostic information
   - Info: General informational messages
   - Warn: Warning messages for potential issues
   - Error: Error messages for failures

4. **Include context in logs**:
   - Add relevant fields (IDs, status, duration)
   - Use structured logging (not string concatenation)
   - Avoid logging sensitive information (tokens, passwords)
