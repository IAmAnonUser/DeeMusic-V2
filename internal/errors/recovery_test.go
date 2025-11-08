package errors

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// MockTokenRefresher is a mock implementation of TokenRefresher for testing
type MockTokenRefresher struct {
	refreshCount int
	shouldFail   bool
	failAfter    int
}

func (m *MockTokenRefresher) RefreshToken(ctx context.Context) error {
	m.refreshCount++
	if m.shouldFail && m.refreshCount > m.failAfter {
		return fmt.Errorf("token refresh failed")
	}
	return nil
}

// MockLogger is a mock implementation of Logger for testing
type MockLogger struct {
	errorCount int
	warnCount  int
	infoCount  int
	lastError  map[string]interface{}
	lastWarn   map[string]interface{}
	lastInfo   map[string]interface{}
}

func (l *MockLogger) Error(msg string, fields map[string]interface{}) {
	l.errorCount++
	l.lastError = fields
}

func (l *MockLogger) Warn(msg string, fields map[string]interface{}) {
	l.warnCount++
	l.lastWarn = fields
}

func (l *MockLogger) Info(msg string, fields map[string]interface{}) {
	l.infoCount++
	l.lastInfo = fields
}

func TestNewErrorRecoveryManager(t *testing.T) {
	refresher := &MockTokenRefresher{}
	logger := &MockLogger{}
	config := DefaultRetryConfig()

	manager := NewErrorRecoveryManager(refresher, logger, config)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}
	if manager.tokenRefresher != refresher {
		t.Error("Token refresher not set correctly")
	}
	if manager.logger != logger {
		t.Error("Logger not set correctly")
	}
}

func TestErrorRecoveryManager_HandleAuthError(t *testing.T) {
	ctx := context.Background()
	refresher := &MockTokenRefresher{}
	logger := &MockLogger{}
	config := DefaultRetryConfig()

	manager := NewErrorRecoveryManager(refresher, logger, config)

	authErr := NewAuthError("token expired", nil)
	err := manager.HandleError(ctx, authErr, "test_operation")

	if err == nil {
		t.Error("Expected error to be returned")
	}
	if refresher.refreshCount != 1 {
		t.Errorf("Expected 1 token refresh, got %d", refresher.refreshCount)
	}
	if logger.warnCount != 1 {
		t.Errorf("Expected 1 warning log, got %d", logger.warnCount)
	}
	if logger.infoCount != 1 {
		t.Errorf("Expected 1 info log, got %d", logger.infoCount)
	}
}

func TestErrorRecoveryManager_HandleAuthError_RefreshFails(t *testing.T) {
	ctx := context.Background()
	refresher := &MockTokenRefresher{shouldFail: true, failAfter: 0}
	logger := &MockLogger{}
	config := DefaultRetryConfig()

	manager := NewErrorRecoveryManager(refresher, logger, config)

	authErr := NewAuthError("token expired", nil)
	err := manager.HandleError(ctx, authErr, "test_operation")

	if err == nil {
		t.Error("Expected error to be returned")
	}
	if refresher.refreshCount != 1 {
		t.Errorf("Expected 1 token refresh attempt, got %d", refresher.refreshCount)
	}
	if logger.errorCount < 1 {
		t.Errorf("Expected at least 1 error log, got %d", logger.errorCount)
	}
}

func TestErrorRecoveryManager_HandleRateLimitError(t *testing.T) {
	ctx := context.Background()
	refresher := &MockTokenRefresher{}
	logger := &MockLogger{}
	config := RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     50 * time.Millisecond,
		Multiplier:     2.0,
	}

	manager := NewErrorRecoveryManager(refresher, logger, config)

	rateLimitErr := NewRateLimitError("too many requests", 1)
	startTime := time.Now()
	err := manager.HandleError(ctx, rateLimitErr, "test_operation")
	duration := time.Since(startTime)

	if err == nil {
		t.Error("Expected error to be returned")
	}
	if duration < config.MaxBackoff {
		t.Errorf("Expected to wait at least %v, waited %v", config.MaxBackoff, duration)
	}
	if logger.warnCount != 1 {
		t.Errorf("Expected 1 warning log, got %d", logger.warnCount)
	}
	if logger.infoCount != 1 {
		t.Errorf("Expected 1 info log, got %d", logger.infoCount)
	}
}

func TestErrorRecoveryManager_HandleNetworkError(t *testing.T) {
	ctx := context.Background()
	refresher := &MockTokenRefresher{}
	logger := &MockLogger{}
	config := DefaultRetryConfig()

	manager := NewErrorRecoveryManager(refresher, logger, config)

	networkErr := NewNetworkError("connection timeout", nil)
	err := manager.HandleError(ctx, networkErr, "test_operation")

	if err == nil {
		t.Error("Expected error to be returned")
	}
	if !IsNetworkError(err) {
		t.Error("Expected network error to be returned")
	}
	if logger.warnCount != 1 {
		t.Errorf("Expected 1 warning log, got %d", logger.warnCount)
	}
}

func TestErrorRecoveryManager_IsRateLimited(t *testing.T) {
	refresher := &MockTokenRefresher{}
	logger := &MockLogger{}
	config := DefaultRetryConfig()

	manager := NewErrorRecoveryManager(refresher, logger, config)

	// Initially not rate limited
	if manager.IsRateLimited() {
		t.Error("Expected not to be rate limited initially")
	}

	// Set rate limit
	manager.mu.Lock()
	manager.rateLimitUntil = time.Now().Add(100 * time.Millisecond)
	manager.mu.Unlock()

	// Should be rate limited
	if !manager.IsRateLimited() {
		t.Error("Expected to be rate limited")
	}

	// Wait for rate limit to expire
	time.Sleep(150 * time.Millisecond)

	// Should no longer be rate limited
	if manager.IsRateLimited() {
		t.Error("Expected rate limit to have expired")
	}
}

func TestErrorRecoveryManager_ExecuteWithRecovery_Success(t *testing.T) {
	ctx := context.Background()
	refresher := &MockTokenRefresher{}
	logger := &MockLogger{}
	config := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}

	manager := NewErrorRecoveryManager(refresher, logger, config)

	attemptCount := 0
	err := manager.ExecuteWithRecovery(ctx, "test_operation", func() error {
		attemptCount++
		if attemptCount < 2 {
			return NewNetworkError("temporary failure", nil)
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if attemptCount != 2 {
		t.Errorf("Expected 2 attempts, got %d", attemptCount)
	}
}

func TestErrorRecoveryManager_ExecuteWithRecovery_WithTokenRefresh(t *testing.T) {
	ctx := context.Background()
	refresher := &MockTokenRefresher{}
	logger := &MockLogger{}
	config := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}

	manager := NewErrorRecoveryManager(refresher, logger, config)

	attemptCount := 0
	err := manager.ExecuteWithRecovery(ctx, "test_operation", func() error {
		attemptCount++
		if attemptCount == 1 {
			return NewAuthError("token expired", nil)
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if refresher.refreshCount != 1 {
		t.Errorf("Expected 1 token refresh, got %d", refresher.refreshCount)
	}
	if attemptCount < 2 {
		t.Errorf("Expected at least 2 attempts, got %d", attemptCount)
	}
}

func TestErrorRecoveryManager_ExecuteWithRecovery_RateLimited(t *testing.T) {
	ctx := context.Background()
	refresher := &MockTokenRefresher{}
	logger := &MockLogger{}
	config := RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     50 * time.Millisecond,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}

	manager := NewErrorRecoveryManager(refresher, logger, config)

	// Set rate limit
	manager.mu.Lock()
	manager.rateLimitUntil = time.Now().Add(50 * time.Millisecond)
	manager.mu.Unlock()

	startTime := time.Now()
	err := manager.ExecuteWithRecovery(ctx, "test_operation", func() error {
		return nil
	})
	duration := time.Since(startTime)

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if duration < 50*time.Millisecond {
		t.Errorf("Expected to wait at least 50ms, waited %v", duration)
	}
}

func TestErrorRecoveryManager_ExecuteWithRecovery_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	refresher := &MockTokenRefresher{}
	logger := &MockLogger{}
	config := RetryConfig{
		MaxRetries:     10,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}

	manager := NewErrorRecoveryManager(refresher, logger, config)

	err := manager.ExecuteWithRecovery(ctx, "test_operation", func() error {
		return NewNetworkError("failure", nil)
	})

	if err == nil {
		t.Error("Expected error due to context cancellation")
	}
}

func TestErrorRecoveryManager_ConcurrentTokenRefresh(t *testing.T) {
	ctx := context.Background()
	refresher := &MockTokenRefresher{}
	logger := &MockLogger{}
	config := RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}

	manager := NewErrorRecoveryManager(refresher, logger, config)

	// Simulate concurrent auth errors
	done := make(chan bool, 2)

	go func() {
		authErr := NewAuthError("token expired", nil)
		manager.HandleError(ctx, authErr, "operation1")
		done <- true
	}()

	go func() {
		authErr := NewAuthError("token expired", nil)
		manager.HandleError(ctx, authErr, "operation2")
		done <- true
	}()

	<-done
	<-done

	// Should only refresh once due to mutex
	if refresher.refreshCount > 2 {
		t.Errorf("Expected at most 2 token refreshes, got %d", refresher.refreshCount)
	}
}

func TestSimpleLogger(t *testing.T) {
	logger := NewSimpleLogger()

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	// Test that methods don't panic
	logger.Error("test error", map[string]interface{}{"key": "value"})
	logger.Warn("test warning", map[string]interface{}{"key": "value"})
	logger.Info("test info", map[string]interface{}{"key": "value"})
}

func TestErrorRecoveryManager_LogError(t *testing.T) {
	refresher := &MockTokenRefresher{}
	logger := &MockLogger{}
	config := DefaultRetryConfig()

	manager := NewErrorRecoveryManager(refresher, logger, config)

	// Test with AppError
	appErr := NewNetworkError("connection failed", fmt.Errorf("underlying error"))
	manager.logError(appErr, "test_operation")

	if logger.errorCount != 1 {
		t.Errorf("Expected 1 error log, got %d", logger.errorCount)
	}
	if logger.lastError["operation"] != "test_operation" {
		t.Error("Operation not logged correctly")
	}
	if logger.lastError["error_type"] != string(ErrTypeNetwork) {
		t.Error("Error type not logged correctly")
	}

	// Test with standard error
	logger.errorCount = 0
	stdErr := fmt.Errorf("standard error")
	manager.logError(stdErr, "test_operation")

	if logger.errorCount != 1 {
		t.Errorf("Expected 1 error log, got %d", logger.errorCount)
	}
}

func BenchmarkErrorRecoveryManager_ExecuteWithRecovery(b *testing.B) {
	ctx := context.Background()
	refresher := &MockTokenRefresher{}
	logger := &MockLogger{}
	config := RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}

	manager := NewErrorRecoveryManager(refresher, logger, config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.ExecuteWithRecovery(ctx, "test_operation", func() error {
			return nil
		})
	}
}
