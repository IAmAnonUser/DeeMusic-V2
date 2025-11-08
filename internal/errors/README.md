# Error Handling and Recovery Package

This package provides comprehensive error handling and recovery mechanisms for the DeeMusic application.

## Features

- **Typed Errors**: Structured error types with context (network, auth, rate limit, etc.)
- **Retry Logic**: Exponential backoff with configurable parameters
- **Error Recovery**: Automatic token refresh and rate limit handling
- **Centralized Logging**: Structured error logging with context

## Components

### Error Types

The package defines several error types:

- `ErrTypeNetwork`: Network-related errors (retryable)
- `ErrTypeAuth`: Authentication errors (retryable after token refresh)
- `ErrTypeRateLimit`: Rate limiting errors (retryable with delay)
- `ErrTypeNotFound`: Resource not found errors (not retryable)
- `ErrTypeDecryption`: Decryption errors (not retryable)
- `ErrTypeFileSystem`: File system errors (retryable)
- `ErrTypeValidation`: Validation errors (not retryable)

### AppError

`AppError` is the main error type that wraps errors with additional context:

```go
type AppError struct {
    Type       ErrorType  // Category of error
    Message    string     // Human-readable message
    StatusCode int        // HTTP status code
    Retryable  bool       // Whether the error is retryable
    Cause      error      // Underlying cause
}
```

### Retry Logic

The package provides exponential backoff retry logic:

```go
config := errors.DefaultRetryConfig()
// MaxRetries: 5
// InitialBackoff: 1 second
// MaxBackoff: 30 seconds
// Multiplier: 2.0

err := errors.RetryWithBackoff(ctx, config, func() error {
    return someOperation()
})
```

Retry attempts follow this pattern:
- Attempt 1: Wait 1 second
- Attempt 2: Wait 2 seconds
- Attempt 3: Wait 4 seconds
- Attempt 4: Wait 8 seconds
- Attempt 5: Wait 16 seconds
- Attempt 6: Wait 30 seconds (capped at MaxBackoff)

### Error Recovery Manager

The `ErrorRecoveryManager` provides centralized error handling with automatic recovery:

```go
manager := errors.NewErrorRecoveryManager(
    tokenRefresher,
    logger,
    errors.DefaultRetryConfig(),
)

err := manager.ExecuteWithRecovery(ctx, "download_track", func() error {
    return downloadTrack()
})
```

The manager automatically:
- Refreshes authentication tokens on auth errors
- Waits for rate limits to clear
- Retries network errors with exponential backoff
- Logs all errors with context

## Usage Examples

### Creating Errors

```go
// Network error
err := errors.NewNetworkError("connection timeout", originalErr)

// Authentication error
err := errors.NewAuthError("invalid token", originalErr)

// Rate limit error
err := errors.NewRateLimitError("too many requests", 60)

// Validation error
err := errors.NewValidationError("invalid track ID")
```

### Checking Error Types

```go
if errors.IsRetryable(err) {
    // Retry the operation
}

if errors.IsAuthError(err) {
    // Handle authentication error
}

if errors.IsRateLimitError(err) {
    // Wait before retrying
}
```

### Using Retry Logic

```go
config := errors.RetryConfig{
    MaxRetries:     3,
    InitialBackoff: 500 * time.Millisecond,
    MaxBackoff:     10 * time.Second,
    Multiplier:     2.0,
    RetryableErrors: func(err error) bool {
        return errors.IsRetryable(err)
    },
}

err := errors.RetryWithBackoff(ctx, config, func() error {
    return apiClient.GetTrack(trackID)
})
```

### Using Error Recovery Manager

```go
// Create logger
logger := errors.NewSimpleLogger()

// Create recovery manager
manager := errors.NewErrorRecoveryManager(
    deezerClient,  // Implements TokenRefresher
    logger,
    errors.DefaultRetryConfig(),
)

// Execute with automatic recovery
err := manager.ExecuteWithRecovery(ctx, "fetch_album", func() error {
    album, err := deezerClient.GetAlbum(ctx, albumID)
    if err != nil {
        return errors.NewNetworkError("failed to fetch album", err)
    }
    return nil
})
```

## Integration with Existing Code

The error handling package integrates with existing components:

### Deezer API Client

The Deezer client should return typed errors:

```go
func (c *DeezerClient) GetTrack(ctx context.Context, trackID string) (*Track, error) {
    resp, err := c.doRequest(ctx, req)
    if err != nil {
        return nil, errors.NewNetworkError("request failed", err)
    }
    
    if resp.StatusCode == http.StatusUnauthorized {
        return nil, errors.NewAuthError("authentication required", nil)
    }
    
    if resp.StatusCode == http.StatusTooManyRequests {
        retryAfter := 60 // Parse from headers
        return nil, errors.NewRateLimitError("rate limit exceeded", retryAfter)
    }
    
    // ... rest of implementation
}
```

### Download Manager

The download manager should use the error recovery manager:

```go
func (m *Manager) downloadTrack(ctx context.Context, trackID string) error {
    return m.errorRecovery.ExecuteWithRecovery(ctx, "download_track", func() error {
        // Download logic here
        return nil
    })
}
```

## Requirements Satisfied

This implementation satisfies the following requirements:

- **Requirement 10.1**: Automatic retry logic for network failures with exponential backoff
- **Requirement 10.2**: Handling rate limiting from Deezer API with appropriate delays
- **Requirement 10.3**: Structured error logging with context
- **Requirement 10.4**: User-friendly error messages
- **Requirement 10.5**: Automatic ARL token refresh when authentication expires

## Testing

See `example_usage.go` for complete usage examples and `recovery_test.go` for unit tests.
