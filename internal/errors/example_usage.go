package errors

import (
	"context"
	"fmt"
	"time"
)

// ExampleTokenRefresher is an example implementation of TokenRefresher
type ExampleTokenRefresher struct {
	refreshCount int
}

// RefreshToken refreshes the authentication token
func (r *ExampleTokenRefresher) RefreshToken(ctx context.Context) error {
	r.refreshCount++
	fmt.Printf("Refreshing token (attempt %d)...\n", r.refreshCount)
	
	// Simulate token refresh
	time.Sleep(100 * time.Millisecond)
	
	if r.refreshCount > 3 {
		return fmt.Errorf("token refresh failed after %d attempts", r.refreshCount)
	}
	
	return nil
}

// ExampleBasicRetry demonstrates basic retry with exponential backoff
func ExampleBasicRetry() {
	ctx := context.Background()
	
	// Configure retry behavior
	config := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}
	
	attemptCount := 0
	
	// Execute with retry
	err := RetryWithBackoff(ctx, config, func() error {
		attemptCount++
		fmt.Printf("Attempt %d\n", attemptCount)
		
		if attemptCount < 3 {
			return NewNetworkError("connection timeout", nil)
		}
		
		return nil
	})
	
	if err != nil {
		fmt.Printf("Operation failed: %v\n", err)
	} else {
		fmt.Printf("Operation succeeded after %d attempts\n", attemptCount)
	}
}

// ExampleErrorRecoveryManager demonstrates the error recovery manager
func ExampleErrorRecoveryManager() {
	ctx := context.Background()
	
	// Create components
	tokenRefresher := &ExampleTokenRefresher{}
	logger := NewSimpleLogger()
	config := DefaultRetryConfig()
	
	// Create recovery manager
	manager := NewErrorRecoveryManager(tokenRefresher, logger, config)
	
	// Execute operation with automatic recovery
	err := manager.ExecuteWithRecovery(ctx, "fetch_track", func() error {
		// Simulate an auth error on first attempt
		if tokenRefresher.refreshCount == 0 {
			return NewAuthError("token expired", nil)
		}
		
		// Success after token refresh
		return nil
	})
	
	if err != nil {
		fmt.Printf("Operation failed: %v\n", err)
	} else {
		fmt.Println("Operation succeeded with automatic token refresh")
	}
}

// ExampleRateLimitHandling demonstrates rate limit handling
func ExampleRateLimitHandling() {
	ctx := context.Background()
	
	tokenRefresher := &ExampleTokenRefresher{}
	logger := NewSimpleLogger()
	config := RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     500 * time.Millisecond,
		Multiplier:     2.0,
		RetryableErrors: func(err error) bool {
			return IsRetryable(err)
		},
	}
	
	manager := NewErrorRecoveryManager(tokenRefresher, logger, config)
	
	attemptCount := 0
	
	err := manager.ExecuteWithRecovery(ctx, "api_request", func() error {
		attemptCount++
		
		if attemptCount == 1 {
			// First attempt hits rate limit
			return NewRateLimitError("too many requests", 1)
		}
		
		// Success after waiting
		return nil
	})
	
	if err != nil {
		fmt.Printf("Operation failed: %v\n", err)
	} else {
		fmt.Println("Operation succeeded after rate limit")
	}
}

// ExampleErrorTypes demonstrates different error types
func ExampleErrorTypes() {
	// Network error (retryable)
	netErr := NewNetworkError("connection timeout", fmt.Errorf("dial tcp: timeout"))
	fmt.Printf("Network error - Retryable: %v, Type: %s\n", netErr.Retryable, netErr.Type)
	
	// Auth error (retryable after token refresh)
	authErr := NewAuthError("invalid token", nil)
	fmt.Printf("Auth error - Retryable: %v, Type: %s\n", authErr.Retryable, authErr.Type)
	
	// Rate limit error (retryable with delay)
	rateLimitErr := NewRateLimitError("too many requests", 60)
	fmt.Printf("Rate limit error - Retryable: %v, Type: %s\n", rateLimitErr.Retryable, rateLimitErr.Type)
	
	// Decryption error (not retryable)
	decryptErr := NewDecryptionError("invalid key", nil)
	fmt.Printf("Decryption error - Retryable: %v, Type: %s\n", decryptErr.Retryable, decryptErr.Type)
	
	// Validation error (not retryable)
	validationErr := NewValidationError("invalid track ID format")
	fmt.Printf("Validation error - Retryable: %v, Type: %s\n", validationErr.Retryable, validationErr.Type)
}

// ExampleCustomRetryConfig demonstrates custom retry configuration
func ExampleCustomRetryConfig() {
	ctx := context.Background()
	
	// Custom configuration for aggressive retries
	config := RetryConfig{
		MaxRetries:     10,
		InitialBackoff: 50 * time.Millisecond,
		MaxBackoff:     5 * time.Second,
		Multiplier:     1.5,
		RetryableErrors: func(err error) bool {
			// Only retry network errors
			return IsNetworkError(err)
		},
	}
	
	attemptCount := 0
	
	err := RetryWithBackoff(ctx, config, func() error {
		attemptCount++
		
		if attemptCount < 5 {
			return NewNetworkError("temporary network issue", nil)
		}
		
		return nil
	})
	
	if err != nil {
		fmt.Printf("Failed after %d attempts: %v\n", attemptCount, err)
	} else {
		fmt.Printf("Succeeded after %d attempts\n", attemptCount)
	}
}

// ExampleContextCancellation demonstrates context cancellation during retry
func ExampleContextCancellation() {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	
	config := DefaultRetryConfig()
	
	err := RetryWithBackoff(ctx, config, func() error {
		// Always fail to trigger retries
		return NewNetworkError("connection failed", nil)
	})
	
	if err != nil {
		fmt.Printf("Operation cancelled: %v\n", err)
	}
}

// ExampleIntegrationWithAPI demonstrates integration with API client
func ExampleIntegrationWithAPI() {
	ctx := context.Background()
	
	// Setup
	tokenRefresher := &ExampleTokenRefresher{}
	logger := NewSimpleLogger()
	manager := NewErrorRecoveryManager(tokenRefresher, logger, DefaultRetryConfig())
	
	// Simulate API call with automatic error handling
	var track interface{}
	err := manager.ExecuteWithRecovery(ctx, "get_track", func() error {
		// Simulate API call
		// In real code, this would be: track, err := apiClient.GetTrack(ctx, trackID)
		
		// Simulate auth error on first call
		if tokenRefresher.refreshCount == 0 {
			return NewAuthError("token expired", nil)
		}
		
		// Success after token refresh
		track = map[string]string{"id": "123", "title": "Example Track"}
		return nil
	})
	
	if err != nil {
		fmt.Printf("Failed to get track: %v\n", err)
	} else {
		fmt.Printf("Successfully retrieved track: %v\n", track)
	}
}
