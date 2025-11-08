package errors

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RetryConfig defines retry behavior configuration
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int
	// InitialBackoff is the initial backoff duration
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration
	// Multiplier is the backoff multiplier for exponential backoff
	Multiplier float64
	// RetryableErrors is a function to determine if an error is retryable
	RetryableErrors func(error) bool
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}
}

// RetryWithBackoff executes a function with exponential backoff retry logic
func RetryWithBackoff(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if config.RetryableErrors != nil && !config.RetryableErrors(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// Don't sleep after the last attempt
		if attempt == config.MaxRetries {
			break
		}

		// Calculate backoff duration with exponential increase
		backoff := calculateBackoff(attempt, config.InitialBackoff, config.MaxBackoff, config.Multiplier)

		// Handle rate limit errors with specific retry-after duration
		if IsRateLimitError(err) {
			// For rate limit errors, use a longer backoff
			backoff = config.MaxBackoff
		}

		// Wait for backoff duration or context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("max retries (%d) exceeded: %w", config.MaxRetries, lastErr)
}

// calculateBackoff calculates the backoff duration for a given attempt
func calculateBackoff(attempt int, initial, max time.Duration, multiplier float64) time.Duration {
	// Calculate exponential backoff: initial * (multiplier ^ attempt)
	backoff := float64(initial) * math.Pow(multiplier, float64(attempt))

	// Cap at maximum backoff
	if backoff > float64(max) {
		backoff = float64(max)
	}

	return time.Duration(backoff)
}

// RetryWithBackoffAndJitter executes a function with exponential backoff and jitter
func RetryWithBackoffAndJitter(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the function
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if config.RetryableErrors != nil && !config.RetryableErrors(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// Don't sleep after the last attempt
		if attempt == config.MaxRetries {
			break
		}

		// Calculate backoff duration with jitter
		backoff := calculateBackoff(attempt, config.InitialBackoff, config.MaxBackoff, config.Multiplier)
		
		// Add jitter (±25% randomness)
		jitter := time.Duration(float64(backoff) * 0.25 * (2.0*float64(time.Now().UnixNano()%100)/100.0 - 1.0))
		backoff += jitter

		// Ensure backoff is positive and within bounds
		if backoff < 0 {
			backoff = config.InitialBackoff
		}
		if backoff > config.MaxBackoff {
			backoff = config.MaxBackoff
		}

		// Wait for backoff duration or context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry cancelled: %w", ctx.Err())
		case <-time.After(backoff):
			// Continue to next attempt
		}
	}

	return fmt.Errorf("max retries (%d) exceeded: %w", config.MaxRetries, lastErr)
}

// RetryableFunc is a function that can be retried
type RetryableFunc func() error

// RetryableOperation represents an operation that can be retried
type RetryableOperation struct {
	Name   string
	Fn     RetryableFunc
	Config RetryConfig
}

// Execute executes the retryable operation
func (r *RetryableOperation) Execute(ctx context.Context) error {
	return RetryWithBackoff(ctx, r.Config, r.Fn)
}

// NewRetryableOperation creates a new retryable operation
func NewRetryableOperation(name string, fn RetryableFunc, config RetryConfig) *RetryableOperation {
	return &RetryableOperation{
		Name:   name,
		Fn:     fn,
		Config: config,
	}
}
