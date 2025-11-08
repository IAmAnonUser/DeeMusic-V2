# Monitoring and Observability Implementation

This document describes the implementation of monitoring and observability features for DeeMusic Go.

## Overview

The monitoring package provides three main features:
1. **Prometheus Metrics** - Performance and operational metrics
2. **Structured Logging** - Zap-based logging with rotation
3. **Health Checks** - Application health status endpoint

## Components

### 1. Prometheus Metrics (`metrics.go`)

Exposes metrics at `/metrics` endpoint in Prometheus format.

#### Available Metrics

**Download Metrics:**
- `deemusic_downloads_total{status, quality}` - Counter of total downloads
- `deemusic_download_duration_seconds{quality}` - Histogram of download durations
- `deemusic_download_bytes_total` - Counter of total bytes downloaded
- `deemusic_active_downloads` - Gauge of currently active downloads
- `deemusic_queue_size` - Gauge of current queue size

**API Metrics:**
- `deemusic_api_requests_total{endpoint, status}` - Counter of API requests
- `deemusic_api_request_duration_seconds{endpoint}` - Histogram of API request durations

**System Metrics:**
- `deemusic_decryption_duration_seconds` - Histogram of decryption durations
- `deemusic_errors_total{type}` - Counter of errors by type

#### Usage

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

### 2. Structured Logging (`logger.go`)

Uses Zap for high-performance structured logging with log rotation.

#### Features

- **Multiple log levels**: debug, info, warn, error
- **Multiple formats**: JSON (production), console (development)
- **Multiple outputs**: file, console, or both
- **Log rotation**: Automatic rotation based on size, age, and backup count
- **Compression**: Optional compression of rotated logs

#### Configuration

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

#### Usage

```go
import (
    "github.com/deemusic/deemusic-go/internal/monitoring"
    "go.uber.org/zap"
)

// Create logger
cfg := &monitoring.LogConfig{
    Level:      "info",
    Format:     "json",
    Output:     "file",
    FilePath:   "/path/to/logs/app.log",
    MaxSizeMB:  100,
    MaxBackups: 3,
    MaxAgeDays: 30,
    Compress:   true,
}

logger, err := monitoring.NewLogger(cfg)
if err != nil {
    log.Fatal(err)
}
defer logger.Sync()

// Log with structured fields
logger.Info("Download started",
    zap.String("track_id", trackID),
    zap.String("quality", quality),
)

logger.Error("Download failed",
    zap.String("track_id", trackID),
    zap.Error(err),
)
```

### 3. Health Checks (`health.go`)

Provides comprehensive health status at `/health` endpoint.

#### Health Check Response

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": 3600,
  "uptime_human": "1h 0m 0s",
  "queue_size": 42,
  "active_downloads": 8,
  "memory_usage_mb": 128,
  "database_status": "connected",
  "checks": {
    "database": {
      "status": "healthy",
      "message": "Database connection is healthy"
    },
    "memory": {
      "status": "healthy",
      "message": "Memory usage is normal"
    },
    "queue": {
      "status": "healthy",
      "message": "Queue size is normal"
    }
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### Health Status Levels

- **healthy** - All systems operational
- **degraded** - System operational but with warnings (high memory, large queue)
- **unhealthy** - Critical issues detected (database down, memory critical)

#### Individual Checks

1. **Database Check**
   - Pings database with 2-second timeout
   - Status: healthy/unhealthy

2. **Memory Check**
   - Monitors heap allocation
   - Warning threshold: 500 MB
   - Critical threshold: 1 GB
   - Status: healthy/degraded/unhealthy

3. **Queue Check**
   - Monitors queue size
   - Warning threshold: 10,000 items
   - Status: healthy/degraded

#### Usage

```go
import "github.com/deemusic/deemusic-go/internal/monitoring"

// Create health checker
healthChecker := monitoring.NewHealthChecker("1.0.0", db)

// Perform health check
queueSize := 42
activeDownloads := 8
healthCheck := healthChecker.Check(queueSize, activeDownloads)

// Check status
if healthCheck.Status == monitoring.HealthStatusHealthy {
    // System is healthy
}
```

## Integration

### Server Integration

The monitoring features are integrated into the HTTP server:

```go
// In server.go
import (
    "github.com/deemusic/deemusic-go/internal/monitoring"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// Add to setupRoutes()
s.router.GET("/health", s.handleHealth)
s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
```

### Download Manager Integration

Metrics are recorded during download operations:

```go
// Before download
monitoring.RecordDownloadStart(quality)

// After successful download
monitoring.RecordDownloadComplete(quality, duration, bytes)

// After failed download
monitoring.RecordDownloadFailed(quality, errorType)
```

### Logging Integration

Structured logging is used throughout the application:

```go
logger.Info("Download started",
    zap.String("track_id", trackID),
    zap.String("quality", quality),
)

logger.Error("Download failed",
    zap.String("track_id", trackID),
    zap.Error(err),
)
```

## Monitoring Best Practices

### Metrics

1. **Use appropriate metric types**:
   - Counters for cumulative values (downloads, errors)
   - Gauges for current values (queue size, active downloads)
   - Histograms for distributions (duration, size)

2. **Keep label cardinality low**:
   - Don't use user IDs or track IDs as labels
   - Use status categories instead of specific error messages
   - Limit label values to a small set

3. **Name metrics consistently**:
   - Use `deemusic_` prefix
   - Use snake_case
   - Include units in name (e.g., `_seconds`, `_bytes`)

### Logging

1. **Log at appropriate levels**:
   - Debug: Detailed diagnostic information
   - Info: General informational messages
   - Warn: Warning messages for potential issues
   - Error: Error messages for failures

2. **Include context in logs**:
   - Add relevant fields (IDs, status, duration)
   - Use structured logging (not string concatenation)
   - Avoid logging sensitive information

3. **Use consistent field names**:
   - `track_id`, `album_id`, `playlist_id`
   - `quality`, `status`, `error`
   - `duration`, `bytes`, `progress`

### Health Checks

1. **Set appropriate thresholds**:
   - Memory: 500 MB warning, 1 GB critical
   - Queue: 10,000 items warning
   - Database: 2-second timeout

2. **Return appropriate status codes**:
   - 200 OK for healthy/degraded
   - 503 Service Unavailable for unhealthy

3. **Include actionable information**:
   - Clear status messages
   - Specific component checks
   - Timestamp for correlation

## Testing

### Metrics Testing

```go
// Test metric recording
monitoring.RecordDownloadStart("MP3_320")
monitoring.RecordDownloadComplete("MP3_320", time.Second, 1024*1024)

// Verify metrics are exposed at /metrics endpoint
resp, err := http.Get("http://localhost:8080/metrics")
// Assert metrics are present
```

### Logging Testing

```go
// Create test logger
cfg := &monitoring.LogConfig{
    Level:  "debug",
    Format: "json",
    Output: "console",
}
logger, err := monitoring.NewLogger(cfg)

// Test logging
logger.Info("test message", zap.String("key", "value"))
```

### Health Check Testing

```go
// Create health checker
healthChecker := monitoring.NewHealthChecker("1.0.0", db)

// Test health check
healthCheck := healthChecker.Check(0, 0)
assert.Equal(t, monitoring.HealthStatusHealthy, healthCheck.Status)
```

## Performance Considerations

1. **Metrics are lock-free**: Prometheus client uses atomic operations
2. **Logging is buffered**: Zap uses buffered I/O for performance
3. **Health checks are cached**: Consider caching health check results for 5-10 seconds
4. **Log rotation is async**: Lumberjack rotates logs asynchronously

## Dependencies

- `github.com/prometheus/client_golang` - Prometheus client
- `go.uber.org/zap` - Structured logging
- `gopkg.in/natefinch/lumberjack.v2` - Log rotation

## Future Enhancements

1. **Distributed Tracing**: Add OpenTelemetry support
2. **Custom Dashboards**: Create Grafana dashboards
3. **Alerting**: Define Prometheus alerting rules
4. **Log Aggregation**: Support for log shipping to ELK/Loki
5. **Performance Profiling**: Add pprof endpoints for profiling
