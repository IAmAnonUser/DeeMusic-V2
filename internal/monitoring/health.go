package monitoring

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"time"
)

// HealthStatus represents the overall health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a health check response
type HealthCheck struct {
	Status         HealthStatus       `json:"status"`
	Version        string             `json:"version"`
	Uptime         int64              `json:"uptime"`
	UptimeHuman    string             `json:"uptime_human"`
	QueueSize      int                `json:"queue_size"`
	ActiveDownloads int               `json:"active_downloads"`
	MemoryUsageMB  uint64             `json:"memory_usage_mb"`
	DatabaseStatus string             `json:"database_status"`
	Checks         map[string]Check   `json:"checks"`
	Timestamp      time.Time          `json:"timestamp"`
}

// Check represents an individual health check
type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HealthChecker performs health checks
type HealthChecker struct {
	version   string
	startTime time.Time
	db        *sql.DB
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(version string, db *sql.DB) *HealthChecker {
	return &HealthChecker{
		version:   version,
		startTime: time.Now(),
		db:        db,
	}
}

// Check performs all health checks and returns the result
func (h *HealthChecker) Check(queueSize, activeDownloads int) *HealthCheck {
	checks := make(map[string]Check)
	overallStatus := HealthStatusHealthy

	// Check database connectivity
	dbCheck := h.checkDatabase()
	checks["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		overallStatus = HealthStatusUnhealthy
	}

	// Check memory usage
	memCheck := h.checkMemory()
	checks["memory"] = memCheck
	if memCheck.Status == "unhealthy" {
		overallStatus = HealthStatusUnhealthy
	} else if memCheck.Status == "degraded" && overallStatus == HealthStatusHealthy {
		overallStatus = HealthStatusDegraded
	}

	// Check queue size
	queueCheck := h.checkQueue(queueSize)
	checks["queue"] = queueCheck
	if queueCheck.Status == "degraded" && overallStatus == HealthStatusHealthy {
		overallStatus = HealthStatusDegraded
	}

	// Calculate uptime
	uptime := time.Since(h.startTime)
	uptimeSeconds := int64(uptime.Seconds())

	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryMB := m.Alloc / 1024 / 1024

	// Determine database status
	dbStatus := "connected"
	if dbCheck.Status != "healthy" {
		dbStatus = "disconnected"
	}

	return &HealthCheck{
		Status:          overallStatus,
		Version:         h.version,
		Uptime:          uptimeSeconds,
		UptimeHuman:     formatDuration(uptime),
		QueueSize:       queueSize,
		ActiveDownloads: activeDownloads,
		MemoryUsageMB:   memoryMB,
		DatabaseStatus:  dbStatus,
		Checks:          checks,
		Timestamp:       time.Now(),
	}
}

// checkDatabase checks database connectivity
func (h *HealthChecker) checkDatabase() Check {
	if h.db == nil {
		return Check{
			Status:  "unhealthy",
			Message: "Database connection not initialized",
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		return Check{
			Status:  "unhealthy",
			Message: "Database ping failed: " + err.Error(),
		}
	}

	return Check{
		Status:  "healthy",
		Message: "Database connection is healthy",
	}
}

// checkMemory checks memory usage
func (h *HealthChecker) checkMemory() Check {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memoryMB := m.Alloc / 1024 / 1024

	// Thresholds
	const (
		warningThresholdMB  = 500  // 500 MB
		criticalThresholdMB = 1000 // 1 GB
	)

	if memoryMB > criticalThresholdMB {
		return Check{
			Status:  "unhealthy",
			Message: "Memory usage is critically high",
		}
	}

	if memoryMB > warningThresholdMB {
		return Check{
			Status:  "degraded",
			Message: "Memory usage is elevated",
		}
	}

	return Check{
		Status:  "healthy",
		Message: "Memory usage is normal",
	}
}

// checkQueue checks queue size
func (h *HealthChecker) checkQueue(queueSize int) Check {
	const warningThreshold = 10000

	if queueSize > warningThreshold {
		return Check{
			Status:  "degraded",
			Message: "Queue size is very large",
		}
	}

	return Check{
		Status:  "healthy",
		Message: "Queue size is normal",
	}
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
