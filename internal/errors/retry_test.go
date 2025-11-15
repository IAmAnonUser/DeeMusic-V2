package errors

import (
	"context"
	"testing"
	"time"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 5 {
		t.Errorf("MaxRetries = %v, want 5", config.MaxRetries)
	}
	if config.InitialBackoff != 1*time.Second {
		t.Errorf("InitialBackoff = %v, want 1s", config.InitialBackoff)
	}
	if config.MaxBackoff != 30*time.Second {
		t.Errorf("MaxBackoff = %v, want 30s", config.MaxBackoff)
	}
	if config.Multiplier != 2.0 {
		t.Errorf("Multiplier = %v, want 2.0", config.Multiplier)
	}
	if config.RetryableErrors == nil {
		t.Error("RetryableErrors function is nil")
	}
}

func TestRetryWithBackoff_Success(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}

	attemptCount := 0
	err := RetryWithBackoff(ctx, config, func() error {
		attemptCount++
		if attemptCount < 3 {
			return NewNetworkError("temporary failure", nil)
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestRetryWithBackoff_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}

	attemptCount := 0
	err := RetryWithBackoff(ctx, config, func() error {
		attemptCount++
		return NewNetworkError("persistent failure", nil)
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if attemptCount != 3 { // Initial attempt + 2 retries
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestRetryWithBackoff_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}

	attemptCount := 0
	err := RetryWithBackoff(ctx, config, func() error {
		attemptCount++
		return NewValidationError("invalid input")
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if attemptCount != 1 {
		t.Errorf("Expected 1 attempt (no retries), got %d", attemptCount)
	}
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	config := RetryConfig{
		MaxRetries:     10,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}

	err := RetryWithBackoff(ctx, config, func() error {
		return NewNetworkError("failure", nil)
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if ctx.Err() == nil {
		t.Error("Expected context to be cancelled")
	}
}

func TestRetryWithBackoff_ImmediateSuccess(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()

	attemptCount := 0
	err := RetryWithBackoff(ctx, config, func() error {
		attemptCount++
		return nil
	})

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if attemptCount != 1 {
		t.Errorf("Expected 1 attempt, got %d", attemptCount)
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name           string
		attempt        int
		initial        time.Duration
		max            time.Duration
		multiplier     float64
		expectedMin    time.Duration
		expectedMax    time.Duration
	}{
		{
			name:        "first retry",
			attempt:     0,
			initial:     1 * time.Second,
			max:         30 * time.Second,
			multiplier:  2.0,
			expectedMin: 1 * time.Second,
			expectedMax: 1 * time.Second,
		},
		{
			name:        "second retry",
			attempt:     1,
			initial:     1 * time.Second,
			max:         30 * time.Second,
			multiplier:  2.0,
			expectedMin: 2 * time.Second,
			expectedMax: 2 * time.Second,
		},
		{
			name:        "third retry",
			attempt:     2,
			initial:     1 * time.Second,
			max:         30 * time.Second,
			multiplier:  2.0,
			expectedMin: 4 * time.Second,
			expectedMax: 4 * time.Second,
		},
		{
			name:        "capped at max",
			attempt:     10,
			initial:     1 * time.Second,
			max:         30 * time.Second,
			multiplier:  2.0,
			expectedMin: 30 * time.Second,
			expectedMax: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backoff := calculateBackoff(tt.attempt, tt.initial, tt.max, tt.multiplier)
			if backoff < tt.expectedMin || backoff > tt.expectedMax {
				t.Errorf("calculateBackoff() = %v, want between %v and %v", backoff, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestRetryWithBackoff_RateLimitError(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}

	attemptCount := 0
	startTime := time.Now()

	err := RetryWithBackoff(ctx, config, func() error {
		attemptCount++
		if attemptCount == 1 {
			return NewRateLimitError("rate limited", 1)
		}
		return nil
	})

	duration := time.Since(startTime)

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if attemptCount != 2 {
		t.Errorf("Expected 2 attempts, got %d", attemptCount)
	}
	// Should wait for MaxBackoff on rate limit
	if duration < config.MaxBackoff {
		t.Errorf("Expected to wait at least %v, waited %v", config.MaxBackoff, duration)
	}
}

func TestNewRetryableOperation(t *testing.T) {
	config := DefaultRetryConfig()
	fn := func() error { return nil }

	op := NewRetryableOperation("test_operation", fn, config)

	if op.Name != "test_operation" {
		t.Errorf("Name = %v, want test_operation", op.Name)
	}
	if op.Fn == nil {
		t.Error("Fn is nil")
	}
	if op.Config.MaxRetries != config.MaxRetries {
		t.Errorf("Config.MaxRetries = %v, want %v", op.Config.MaxRetries, config.MaxRetries)
	}
}

func TestRetryableOperation_Execute(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}

	attemptCount := 0
	fn := func() error {
		attemptCount++
		if attemptCount < 2 {
			return NewNetworkError("temporary failure", nil)
		}
		return nil
	}

	op := NewRetryableOperation("test_op", fn, config)
	err := op.Execute(ctx)

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if attemptCount != 2 {
		t.Errorf("Expected 2 attempts, got %d", attemptCount)
	}
}

func TestRetryWithBackoffAndJitter(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}

	attemptCount := 0
	err := RetryWithBackoffAndJitter(ctx, config, func() error {
		attemptCount++
		if attemptCount < 3 {
			return NewNetworkError("temporary failure", nil)
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestRetryWithBackoff_CustomRetryableCheck(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			// Only retry network errors
			return IsNetworkError(err)
		},
	}

	// Test with network error (should retry)
	attemptCount := 0
	err := RetryWithBackoff(ctx, config, func() error {
		attemptCount++
		if attemptCount < 2 {
			return NewNetworkError("network failure", nil)
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if attemptCount != 2 {
		t.Errorf("Expected 2 attempts, got %d", attemptCount)
	}

	// Test with auth error (should not retry with this config)
	attemptCount = 0
	err = RetryWithBackoff(ctx, config, func() error {
		attemptCount++
		return NewAuthError("auth failure", nil)
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}
	if attemptCount != 1 {
		t.Errorf("Expected 1 attempt (no retries), got %d", attemptCount)
	}
}

func BenchmarkRetryWithBackoff(b *testing.B) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RetryWithBackoff(ctx, config, func() error {
			return nil
		})
	}
}

func BenchmarkRetryWithBackoff_WithRetries(b *testing.B) {
	ctx := context.Background()
	config := RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attemptCount := 0
		_ = RetryWithBackoff(ctx, config, func() error {
			attemptCount++
			if attemptCount < 2 {
				return NewNetworkError("failure", nil)
			}
			return nil
		})
	}
}
