package monitoring

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestNewHealthChecker(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	healthChecker := NewHealthChecker("1.0.0", db)
	if healthChecker == nil {
		t.Fatal("Expected health checker, got nil")
	}

	if healthChecker.version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", healthChecker.version)
	}
}

func TestHealthCheckHealthy(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	healthChecker := NewHealthChecker("1.0.0", db)

	// Perform health check with normal values
	healthCheck := healthChecker.Check(100, 5)

	if healthCheck.Status != HealthStatusHealthy {
		t.Errorf("Expected status healthy, got %s", healthCheck.Status)
	}

	if healthCheck.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", healthCheck.Version)
	}

	if healthCheck.QueueSize != 100 {
		t.Errorf("Expected queue size 100, got %d", healthCheck.QueueSize)
	}

	if healthCheck.ActiveDownloads != 5 {
		t.Errorf("Expected active downloads 5, got %d", healthCheck.ActiveDownloads)
	}

	if healthCheck.DatabaseStatus != "connected" {
		t.Errorf("Expected database status connected, got %s", healthCheck.DatabaseStatus)
	}
}

func TestHealthCheckDegraded(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	healthChecker := NewHealthChecker("1.0.0", db)

	// Perform health check with large queue
	healthCheck := healthChecker.Check(15000, 5)

	if healthCheck.Status != HealthStatusDegraded {
		t.Errorf("Expected status degraded, got %s", healthCheck.Status)
	}

	// Check queue check status
	if queueCheck, ok := healthCheck.Checks["queue"]; ok {
		if queueCheck.Status != "degraded" {
			t.Errorf("Expected queue check degraded, got %s", queueCheck.Status)
		}
	} else {
		t.Error("Queue check not found")
	}
}

func TestHealthCheckUnhealthy(t *testing.T) {
	// Create health checker with nil database
	healthChecker := NewHealthChecker("1.0.0", nil)

	// Perform health check
	healthCheck := healthChecker.Check(100, 5)

	if healthCheck.Status != HealthStatusUnhealthy {
		t.Errorf("Expected status unhealthy, got %s", healthCheck.Status)
	}

	if healthCheck.DatabaseStatus != "disconnected" {
		t.Errorf("Expected database status disconnected, got %s", healthCheck.DatabaseStatus)
	}

	// Check database check status
	if dbCheck, ok := healthCheck.Checks["database"]; ok {
		if dbCheck.Status != "unhealthy" {
			t.Errorf("Expected database check unhealthy, got %s", dbCheck.Status)
		}
	} else {
		t.Error("Database check not found")
	}
}

func TestHealthCheckUptime(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	healthChecker := NewHealthChecker("1.0.0", db)

	// Wait a bit to accumulate uptime
	time.Sleep(1 * time.Second)

	healthCheck := healthChecker.Check(0, 0)

	if healthCheck.Uptime < 1 {
		t.Errorf("Expected uptime >= 1, got %d", healthCheck.Uptime)
	}

	if healthCheck.UptimeHuman == "" {
		t.Error("Expected non-empty uptime human string")
	}
}

func TestHealthCheckMemoryUsage(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	healthChecker := NewHealthChecker("1.0.0", db)

	healthCheck := healthChecker.Check(0, 0)

	// Memory usage should be reported (can be 0 in some test environments)
	// Just check that the field exists
	_ = healthCheck.MemoryUsageMB

	// Check memory check exists
	if _, ok := healthCheck.Checks["memory"]; !ok {
		t.Error("Memory check not found")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m 30s"},
		{3661 * time.Second, "1h 1m 1s"},
		{86400 * time.Second, "1d 0h 0m 0s"},
		{90061 * time.Second, "1d 1h 1m 1s"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", tt.duration, result, tt.expected)
		}
	}
}

func TestHealthCheckTimestamp(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	healthChecker := NewHealthChecker("1.0.0", db)

	before := time.Now()
	healthCheck := healthChecker.Check(0, 0)
	after := time.Now()

	if healthCheck.Timestamp.Before(before) || healthCheck.Timestamp.After(after) {
		t.Error("Health check timestamp is not within expected range")
	}
}
