package monitoring

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewLogger(t *testing.T) {
	// Skip cleanup check on Windows due to file locking issues with lumberjack
	t.Cleanup(func() {
		// Allow time for file handles to be released
		time.Sleep(2 * time.Second)
	})

	// Create temp directory for logs
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")

	cfg := &LogConfig{
		Level:      "info",
		Format:     "json",
		Output:     "file",
		FilePath:   logPath,
		MaxSizeMB:  10,
		MaxBackups: 2,
		MaxAgeDays: 7,
		Compress:   false,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Test logging
	logger.Info("test message", zap.String("key", "value"))

	// Sync logger
	logger.Sync()

	// Verify log file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Log file was not created: %s", logPath)
	}
}

func TestNewLoggerConsole(t *testing.T) {
	cfg := &LogConfig{
		Level:  "debug",
		Format: "console",
		Output: "console",
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create console logger: %v", err)
	}
	defer logger.Sync()

	// Test logging at different levels
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
}

func TestNewLoggerBothOutputs(t *testing.T) {
	// Skip cleanup check on Windows due to file locking issues with lumberjack
	t.Cleanup(func() {
		time.Sleep(2 * time.Second)
	})

	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")

	cfg := &LogConfig{
		Level:      "info",
		Format:     "json",
		Output:     "both",
		FilePath:   logPath,
		MaxSizeMB:  10,
		MaxBackups: 2,
		MaxAgeDays: 7,
		Compress:   false,
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	logger.Info("test message to both outputs")

	// Sync logger
	logger.Sync()

	// Verify log file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("Log file was not created: %s", logPath)
	}
}

func TestNewDevelopmentLogger(t *testing.T) {
	logger, err := NewDevelopmentLogger()
	if err != nil {
		t.Fatalf("Failed to create development logger: %v", err)
	}
	defer logger.Sync()

	logger.Debug("development debug message")
	logger.Info("development info message")
}

func TestNewProductionLogger(t *testing.T) {
	// Skip cleanup check on Windows due to file locking issues with lumberjack
	t.Cleanup(func() {
		time.Sleep(2 * time.Second)
	})

	tempDir := t.TempDir()

	logger, err := NewProductionLogger(tempDir)
	if err != nil {
		t.Fatalf("Failed to create production logger: %v", err)
	}

	logger.Info("production info message")

	// Sync logger
	logger.Sync()
}

func TestLoggerWithContext(t *testing.T) {
	cfg := &LogConfig{
		Level:  "info",
		Format: "console",
		Output: "console",
	}

	logger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Add context fields
	contextLogger := LoggerWithContext(logger,
		zap.String("component", "test"),
		zap.String("session_id", "123"),
	)

	contextLogger.Info("message with context")
}

func TestInvalidLogLevel(t *testing.T) {
	cfg := &LogConfig{
		Level:  "invalid",
		Format: "json",
		Output: "console",
	}

	_, err := NewLogger(cfg)
	if err == nil {
		t.Error("Expected error for invalid log level, got nil")
	}
}
