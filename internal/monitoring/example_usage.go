package monitoring

import (
	"time"

	"go.uber.org/zap"
)

// ExampleMetrics demonstrates how to use Prometheus metrics
func ExampleMetrics() {
	// Record download start
	RecordDownloadStart("MP3_320")

	// Simulate download
	startTime := time.Now()
	// ... download logic ...
	duration := time.Since(startTime)

	// Record successful download
	RecordDownloadComplete("MP3_320", duration, 10*1024*1024) // 10 MB

	// Or record failed download
	// RecordDownloadFailed("MP3_320", "network_error")

	// Update queue size
	UpdateQueueSize(42)

	// Record API request
	apiStart := time.Now()
	// ... API call ...
	apiDuration := time.Since(apiStart)
	RecordAPIRequest("/api/v1/search", "success", apiDuration)

	// Record decryption
	decryptStart := time.Now()
	// ... decryption logic ...
	decryptDuration := time.Since(decryptStart)
	RecordDecryption(decryptDuration)

	// Record error
	RecordError("network_error")
}

// ExampleLogger demonstrates how to use structured logging
func ExampleLogger() {
	// Create logger with default config
	cfg := &LogConfig{
		Level:      "info",
		Format:     "json",
		Output:     "both", // Log to both file and console
		FilePath:   "/path/to/logs/app.log",
		MaxSizeMB:  100,
		MaxBackups: 3,
		MaxAgeDays: 30,
		Compress:   true,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// Log at different levels
	logger.Debug("Debug message", zap.String("key", "value"))
	logger.Info("Info message", zap.Int("count", 42))
	logger.Warn("Warning message", zap.Duration("duration", time.Second))
	logger.Error("Error message", zap.Error(err))

	// Log with multiple fields
	logger.Info("Download started",
		zap.String("track_id", "123456"),
		zap.String("title", "Song Title"),
		zap.String("artist", "Artist Name"),
		zap.String("quality", "MP3_320"),
	)

	// Log with context
	contextLogger := logger.With(
		zap.String("component", "download_manager"),
		zap.String("session_id", "abc123"),
	)
	contextLogger.Info("Processing download")

	// Log errors with stack trace
	if err != nil {
		logger.Error("Download failed",
			zap.String("track_id", "123456"),
			zap.Error(err),
			zap.Stack("stacktrace"),
		)
	}
}

// ExampleHealthCheck demonstrates how to use health checks
func ExampleHealthCheck() {
	// Create health checker
	// db := ... // your database connection
	// healthChecker := NewHealthChecker("1.0.0", db)

	// Perform health check
	// queueSize := 42
	// activeDownloads := 8
	// healthCheck := healthChecker.Check(queueSize, activeDownloads)

	// Check overall status
	// if healthCheck.Status == HealthStatusHealthy {
	//     // System is healthy
	// } else if healthCheck.Status == HealthStatusDegraded {
	//     // System is degraded but operational
	// } else {
	//     // System is unhealthy
	// }

	// Check individual components
	// if dbCheck, ok := healthCheck.Checks["database"]; ok {
	//     if dbCheck.Status != "healthy" {
	//         // Database is not healthy
	//     }
	// }
}

// ExampleIntegration demonstrates full integration
func ExampleIntegration() {
	// 1. Create logger
	cfg := DefaultLogConfig("/path/to/data")
	logger, err := NewLogger(cfg)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// 2. Log application start
	logger.Info("Application starting",
		zap.String("version", "1.0.0"),
		zap.String("environment", "production"),
	)

	// 3. Record metrics during download
	trackID := "123456"
	quality := "MP3_320"

	logger.Info("Download started",
		zap.String("track_id", trackID),
		zap.String("quality", quality),
	)

	RecordDownloadStart(quality)
	startTime := time.Now()

	// ... download logic ...

	duration := time.Since(startTime)
	bytes := int64(10 * 1024 * 1024)

	RecordDownloadComplete(quality, duration, bytes)

	logger.Info("Download completed",
		zap.String("track_id", trackID),
		zap.Duration("duration", duration),
		zap.Int64("bytes", bytes),
	)

	// 4. Update queue metrics
	UpdateQueueSize(41) // One less after completion

	// 5. Handle errors
	if err != nil {
		RecordDownloadFailed(quality, "network_error")
		RecordError("network_error")

		logger.Error("Download failed",
			zap.String("track_id", trackID),
			zap.Error(err),
		)
	}
}
