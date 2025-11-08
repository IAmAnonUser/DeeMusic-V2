package errors

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// TokenRefresher is an interface for refreshing authentication tokens
type TokenRefresher interface {
	RefreshToken(ctx context.Context) error
}

// Logger is an interface for logging errors
type Logger interface {
	Error(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
}

// ErrorRecoveryManager handles centralized error recovery
type ErrorRecoveryManager struct {
	tokenRefresher TokenRefresher
	logger         Logger
	retryConfig    RetryConfig
	mu             sync.RWMutex
	rateLimitUntil time.Time
	refreshing     bool
}

// NewErrorRecoveryManager creates a new error recovery manager
func NewErrorRecoveryManager(tokenRefresher TokenRefresher, logger Logger, retryConfig RetryConfig) *ErrorRecoveryManager {
	return &ErrorRecoveryManager{
		tokenRefresher: tokenRefresher,
		logger:         logger,
		retryConfig:    retryConfig,
	}
}

// HandleError handles an error and attempts recovery
func (m *ErrorRecoveryManager) HandleError(ctx context.Context, err error, operation string) error {
	if err == nil {
		return nil
	}

	// Log the error with context
	m.logError(err, operation)

	// Determine error type and handle accordingly
	switch {
	case IsAuthError(err):
		return m.handleAuthError(ctx, err, operation)
	case IsRateLimitError(err):
		return m.handleRateLimitError(ctx, err, operation)
	case IsNetworkError(err):
		return m.handleNetworkError(err, operation)
	default:
		return err
	}
}

// handleAuthError handles authentication errors by refreshing the token
func (m *ErrorRecoveryManager) handleAuthError(ctx context.Context, err error, operation string) error {
	m.mu.Lock()
	
	// Check if already refreshing
	if m.refreshing {
		m.mu.Unlock()
		// Wait a bit and return the original error
		time.Sleep(2 * time.Second)
		return err
	}
	
	m.refreshing = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.refreshing = false
		m.mu.Unlock()
	}()

	// Log token refresh attempt
	if m.logger != nil {
		m.logger.Warn("Authentication error detected, attempting token refresh", map[string]interface{}{
			"operation": operation,
			"error":     err.Error(),
		})
	}

	// Attempt to refresh token
	refreshErr := m.tokenRefresher.RefreshToken(ctx)
	if refreshErr != nil {
		if m.logger != nil {
			m.logger.Error("Token refresh failed", map[string]interface{}{
				"operation": operation,
				"error":     refreshErr.Error(),
			})
		}
		return fmt.Errorf("token refresh failed: %w", refreshErr)
	}

	// Log successful refresh
	if m.logger != nil {
		m.logger.Info("Token refreshed successfully", map[string]interface{}{
			"operation": operation,
		})
	}

	// Return a retryable error to indicate the operation should be retried
	return NewAuthError("token refreshed, retry operation", nil)
}

// handleRateLimitError handles rate limit errors by waiting
func (m *ErrorRecoveryManager) handleRateLimitError(ctx context.Context, err error, operation string) error {
	m.mu.Lock()
	
	// Set rate limit until time
	waitDuration := m.retryConfig.MaxBackoff
	m.rateLimitUntil = time.Now().Add(waitDuration)
	
	m.mu.Unlock()

	// Log rate limit
	if m.logger != nil {
		m.logger.Warn("Rate limit detected, waiting before retry", map[string]interface{}{
			"operation":     operation,
			"wait_duration": waitDuration.String(),
		})
	}

	// Wait for the rate limit duration or context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("rate limit wait cancelled: %w", ctx.Err())
	case <-time.After(waitDuration):
		// Continue after waiting
	}

	// Log rate limit cleared
	if m.logger != nil {
		m.logger.Info("Rate limit wait completed", map[string]interface{}{
			"operation": operation,
		})
	}

	return err
}

// handleNetworkError handles network errors
func (m *ErrorRecoveryManager) handleNetworkError(err error, operation string) error {
	// Log network error
	if m.logger != nil {
		m.logger.Warn("Network error detected", map[string]interface{}{
			"operation": operation,
			"error":     err.Error(),
		})
	}

	// Network errors are retryable, return as-is
	return err
}

// logError logs an error with context
func (m *ErrorRecoveryManager) logError(err error, operation string) {
	if m.logger == nil {
		return
	}

	fields := map[string]interface{}{
		"operation": operation,
		"error":     err.Error(),
	}

	// Add error type if it's an AppError
	if appErr, ok := err.(*AppError); ok {
		fields["error_type"] = string(appErr.Type)
		fields["retryable"] = appErr.Retryable
		fields["status_code"] = appErr.StatusCode
		if appErr.Cause != nil {
			fields["cause"] = appErr.Cause.Error()
		}
	}

	m.logger.Error("Operation failed", fields)
}

// IsRateLimited checks if we're currently rate limited
func (m *ErrorRecoveryManager) IsRateLimited() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return time.Now().Before(m.rateLimitUntil)
}

// ExecuteWithRecovery executes a function with automatic error recovery
func (m *ErrorRecoveryManager) ExecuteWithRecovery(ctx context.Context, operation string, fn func() error) error {
	// Check if rate limited
	if m.IsRateLimited() {
		m.mu.RLock()
		waitTime := time.Until(m.rateLimitUntil)
		m.mu.RUnlock()

		if m.logger != nil {
			m.logger.Warn("Operation blocked by rate limit", map[string]interface{}{
				"operation":  operation,
				"wait_time": waitTime.String(),
			})
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			// Continue after rate limit
		}
	}

	// Execute with retry logic
	return RetryWithBackoff(ctx, m.retryConfig, func() error {
		err := fn()
		if err == nil {
			return nil
		}

		// Handle the error and attempt recovery
		recoveryErr := m.HandleError(ctx, err, operation)
		
		// If recovery was successful (token refreshed), return retryable error
		if IsAuthError(recoveryErr) && recoveryErr.Error() != err.Error() {
			return recoveryErr
		}

		return recoveryErr
	})
}

// SimpleLogger is a simple implementation of the Logger interface
type SimpleLogger struct{}

// Error logs an error message
func (l *SimpleLogger) Error(msg string, fields map[string]interface{}) {
	log.Printf("[ERROR] %s %v", msg, fields)
}

// Warn logs a warning message
func (l *SimpleLogger) Warn(msg string, fields map[string]interface{}) {
	log.Printf("[WARN] %s %v", msg, fields)
}

// Info logs an info message
func (l *SimpleLogger) Info(msg string, fields map[string]interface{}) {
	log.Printf("[INFO] %s %v", msg, fields)
}

// NewSimpleLogger creates a new simple logger
func NewSimpleLogger() *SimpleLogger {
	return &SimpleLogger{}
}
