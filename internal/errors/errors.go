package errors

import (
	"fmt"
	"net/http"
)

// ErrorType represents the category of error
type ErrorType string

const (
	// ErrTypeNetwork represents network-related errors
	ErrTypeNetwork ErrorType = "network"
	// ErrTypeAuth represents authentication errors
	ErrTypeAuth ErrorType = "auth"
	// ErrTypeRateLimit represents rate limiting errors
	ErrTypeRateLimit ErrorType = "rate_limit"
	// ErrTypeNotFound represents resource not found errors
	ErrTypeNotFound ErrorType = "not_found"
	// ErrTypeDecryption represents decryption errors
	ErrTypeDecryption ErrorType = "decryption"
	// ErrTypeFileSystem represents file system errors
	ErrTypeFileSystem ErrorType = "filesystem"
	// ErrTypeValidation represents validation errors
	ErrTypeValidation ErrorType = "validation"
	// ErrTypeUnknown represents unknown errors
	ErrTypeUnknown ErrorType = "unknown"
)

// AppError represents an application error with context
type AppError struct {
	Type       ErrorType
	Message    string
	StatusCode int
	Retryable  bool
	Cause      error
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying cause
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewNetworkError creates a new network error
func NewNetworkError(message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeNetwork,
		Message:    message,
		StatusCode: http.StatusServiceUnavailable,
		Retryable:  true,
		Cause:      cause,
	}
}

// NewAuthError creates a new authentication error
func NewAuthError(message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeAuth,
		Message:    message,
		StatusCode: http.StatusUnauthorized,
		Retryable:  true, // Can retry after token refresh
		Cause:      cause,
	}
}

// NewRateLimitError creates a new rate limit error
func NewRateLimitError(message string, retryAfter int) *AppError {
	return &AppError{
		Type:       ErrTypeRateLimit,
		Message:    fmt.Sprintf("%s (retry after %d seconds)", message, retryAfter),
		StatusCode: http.StatusTooManyRequests,
		Retryable:  true,
		Cause:      nil,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message string) *AppError {
	return &AppError{
		Type:       ErrTypeNotFound,
		Message:    message,
		StatusCode: http.StatusNotFound,
		Retryable:  false,
		Cause:      nil,
	}
}

// NewDecryptionError creates a new decryption error
func NewDecryptionError(message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeDecryption,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Retryable:  false, // Decryption errors are not retryable
		Cause:      cause,
	}
}

// NewFileSystemError creates a new file system error
func NewFileSystemError(message string, cause error) *AppError {
	return &AppError{
		Type:       ErrTypeFileSystem,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Retryable:  true,
		Cause:      cause,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(message string) *AppError {
	return &AppError{
		Type:       ErrTypeValidation,
		Message:    message,
		StatusCode: http.StatusBadRequest,
		Retryable:  false,
		Cause:      nil,
	}
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Retryable
	}
	return false
}

// GetErrorType returns the error type from an error
func GetErrorType(err error) ErrorType {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type
	}
	return ErrTypeUnknown
}

// IsAuthError checks if an error is an authentication error
func IsAuthError(err error) bool {
	return GetErrorType(err) == ErrTypeAuth
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	return GetErrorType(err) == ErrTypeRateLimit
}

// IsNetworkError checks if an error is a network error
func IsNetworkError(err error) bool {
	return GetErrorType(err) == ErrTypeNetwork
}
