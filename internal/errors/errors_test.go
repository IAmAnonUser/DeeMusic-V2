package errors

import (
	"fmt"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		expected string
	}{
		{
			name: "error without cause",
			err: &AppError{
				Type:    ErrTypeNetwork,
				Message: "connection failed",
			},
			expected: "network: connection failed",
		},
		{
			name: "error with cause",
			err: &AppError{
				Type:    ErrTypeNetwork,
				Message: "connection failed",
				Cause:   fmt.Errorf("dial tcp: timeout"),
			},
			expected: "network: connection failed (caused by: dial tcp: timeout)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := &AppError{
		Type:  ErrTypeNetwork,
		Cause: cause,
	}

	if unwrapped := err.Unwrap(); unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestNewNetworkError(t *testing.T) {
	cause := fmt.Errorf("connection timeout")
	err := NewNetworkError("network failed", cause)

	if err.Type != ErrTypeNetwork {
		t.Errorf("Type = %v, want %v", err.Type, ErrTypeNetwork)
	}
	if err.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusServiceUnavailable)
	}
	if !err.Retryable {
		t.Error("Expected network error to be retryable")
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
}

func TestNewAuthError(t *testing.T) {
	err := NewAuthError("invalid token", nil)

	if err.Type != ErrTypeAuth {
		t.Errorf("Type = %v, want %v", err.Type, ErrTypeAuth)
	}
	if err.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusUnauthorized)
	}
	if !err.Retryable {
		t.Error("Expected auth error to be retryable")
	}
}

func TestNewRateLimitError(t *testing.T) {
	err := NewRateLimitError("too many requests", 60)

	if err.Type != ErrTypeRateLimit {
		t.Errorf("Type = %v, want %v", err.Type, ErrTypeRateLimit)
	}
	if err.StatusCode != http.StatusTooManyRequests {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusTooManyRequests)
	}
	if !err.Retryable {
		t.Error("Expected rate limit error to be retryable")
	}
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("track not found")

	if err.Type != ErrTypeNotFound {
		t.Errorf("Type = %v, want %v", err.Type, ErrTypeNotFound)
	}
	if err.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusNotFound)
	}
	if err.Retryable {
		t.Error("Expected not found error to be non-retryable")
	}
}

func TestNewDecryptionError(t *testing.T) {
	cause := fmt.Errorf("invalid key")
	err := NewDecryptionError("decryption failed", cause)

	if err.Type != ErrTypeDecryption {
		t.Errorf("Type = %v, want %v", err.Type, ErrTypeDecryption)
	}
	if err.Retryable {
		t.Error("Expected decryption error to be non-retryable")
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
}

func TestNewFileSystemError(t *testing.T) {
	cause := fmt.Errorf("permission denied")
	err := NewFileSystemError("file write failed", cause)

	if err.Type != ErrTypeFileSystem {
		t.Errorf("Type = %v, want %v", err.Type, ErrTypeFileSystem)
	}
	if !err.Retryable {
		t.Error("Expected filesystem error to be retryable")
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("invalid input")

	if err.Type != ErrTypeValidation {
		t.Errorf("Type = %v, want %v", err.Type, ErrTypeValidation)
	}
	if err.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, http.StatusBadRequest)
	}
	if err.Retryable {
		t.Error("Expected validation error to be non-retryable")
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "retryable network error",
			err:      NewNetworkError("connection failed", nil),
			expected: true,
		},
		{
			name:     "retryable auth error",
			err:      NewAuthError("token expired", nil),
			expected: true,
		},
		{
			name:     "non-retryable decryption error",
			err:      NewDecryptionError("invalid key", nil),
			expected: false,
		},
		{
			name:     "non-retryable validation error",
			err:      NewValidationError("invalid input"),
			expected: false,
		},
		{
			name:     "standard error",
			err:      fmt.Errorf("standard error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetErrorType(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorType
	}{
		{
			name:     "network error",
			err:      NewNetworkError("connection failed", nil),
			expected: ErrTypeNetwork,
		},
		{
			name:     "auth error",
			err:      NewAuthError("invalid token", nil),
			expected: ErrTypeAuth,
		},
		{
			name:     "rate limit error",
			err:      NewRateLimitError("too many requests", 60),
			expected: ErrTypeRateLimit,
		},
		{
			name:     "standard error",
			err:      fmt.Errorf("standard error"),
			expected: ErrTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetErrorType(tt.err); got != tt.expected {
				t.Errorf("GetErrorType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "auth error",
			err:      NewAuthError("invalid token", nil),
			expected: true,
		},
		{
			name:     "network error",
			err:      NewNetworkError("connection failed", nil),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAuthError(tt.err); got != tt.expected {
				t.Errorf("IsAuthError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "rate limit error",
			err:      NewRateLimitError("too many requests", 60),
			expected: true,
		},
		{
			name:     "network error",
			err:      NewNetworkError("connection failed", nil),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRateLimitError(tt.err); got != tt.expected {
				t.Errorf("IsRateLimitError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "network error",
			err:      NewNetworkError("connection failed", nil),
			expected: true,
		},
		{
			name:     "auth error",
			err:      NewAuthError("invalid token", nil),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNetworkError(tt.err); got != tt.expected {
				t.Errorf("IsNetworkError() = %v, want %v", got, tt.expected)
			}
		})
	}
}
