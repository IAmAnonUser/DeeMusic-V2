# Monitoring Quick Start Guide

## Quick Setup

### 1. Create Logger

```go
import (
    "github.com/deemusic/deemusic-go/internal/monitoring"
    "go.uber.org/zap"
)

// Use production logger
logger, err := monitoring.NewProductionLogger("/path/to/data")
if err != nil {
    log.Fatal(err)
}
defer logger.Sync()

// Or use development logger
logger, err := monitoring.NewDevelopmentLogger()
```

### 2. Record Metrics

```go
import "github.com/deemusic/deemusic-go/internal/monitoring"

// Download lifecycle
monitoring.RecordDownloadStart("MP3_320")
// ... download ...
monitoring.RecordDownloadComplete("MP3_320", duration, bytes)

// Or on failure
monitoring.RecordDownloadFailed("MP3_320", "network_error")

// Update queue
monitoring.UpdateQueueSize(42)
```

### 3. Check Health

```go
// Create health checker (once at startup)
healthChecker := monitoring.NewHealthChecker("1.0.0", db)

// Perform check (in HTTP handler)
healthCheck := healthChecker.Check(queueSize, activeDownloads)
```

## Endpoints

- **Metrics:** `http://localhost:8080/metrics`
- **Health:** `http://localhost:8080/health`

## Configuration

Add to `settings.json`:

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

## Common Patterns

### Log with Context

```go
logger.Info("Download started",
    zap.String("track_id", "123"),
    zap.String("quality", "MP3_320"),
)
```

### Log Errors

```go
logger.Error("Download failed",
    zap.String("track_id", "123"),
    zap.Error(err),
)
```

### Record API Metrics

```go
start := time.Now()
// ... API call ...
duration := time.Since(start)
monitoring.RecordAPIRequest("/api/v1/search", "success", duration)
```

### Check System Health

```go
if healthCheck.Status == monitoring.HealthStatusUnhealthy {
    // System has critical issues
    return http.StatusServiceUnavailable
}
```

## Prometheus Queries

### Download Rate
```promql
rate(deemusic_downloads_total[5m])
```

### Average Download Duration
```promql
rate(deemusic_download_duration_seconds_sum[5m]) / 
rate(deemusic_download_duration_seconds_count[5m])
```

### Error Rate
```promql
rate(deemusic_errors_total[5m])
```

### Queue Size
```promql
deemusic_queue_size
```

## Grafana Dashboard

Example dashboard panels:

1. **Download Rate** - Graph of downloads per second
2. **Download Duration** - Heatmap of download times
3. **Queue Size** - Gauge showing current queue
4. **Error Rate** - Graph of errors per second
5. **Active Downloads** - Gauge showing concurrent downloads

## Troubleshooting

### Logs Not Appearing

Check:
- Log directory exists and is writable
- Log level is appropriate (debug vs info)
- Logger is synced before exit

### Metrics Not Updating

Check:
- Metrics are being recorded in code
- `/metrics` endpoint is accessible
- Prometheus is scraping correctly

### Health Check Always Unhealthy

Check:
- Database connection is valid
- Database is accessible
- Health checker has correct DB reference

## Performance Tips

1. **Batch Metrics:** Record metrics in batches when possible
2. **Cache Health:** Cache health check results for 5-10 seconds
3. **Async Logging:** Zap already uses buffered I/O
4. **Low Cardinality:** Keep metric labels to a small set of values
