package security

import (
	"testing"
)

func TestInputSanitization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Remove null bytes",
			input:    "test\x00data",
			expected: "testdata",
		},
		{
			name:     "Remove control characters",
			input:    "test\x01\x02data",
			expected: "testdata",
		},
		{
			name:     "Keep newlines and tabs",
			input:    "test\n\tdata",
			expected: "test\n\tdata",
		},
		{
			name:     "Normal string unchanged",
			input:    "normal string",
			expected: "normal string",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPathValidation(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Valid relative path",
			path:     "music/album/track.mp3",
			expected: true,
		},
		{
			name:     "Path traversal attempt",
			path:     "../../../etc/passwd",
			expected: false,
		},
		{
			name:     "Absolute Windows path",
			path:     "C:\\Windows\\System32",
			expected: false,
		},
		{
			name:     "Path with null byte",
			path:     "test\x00.txt",
			expected: false,
		},
		{
			name:     "Empty path",
			path:     "",
			expected: false,
		},
		{
			name:     "Current directory",
			path:     ".",
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidPath(tt.path)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for path %q", tt.expected, result, tt.path)
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name          string
		basePath      string
		requestedPath string
		shouldError   bool
	}{
		{
			name:          "Valid path within base",
			basePath:      "C:\\Users\\test\\music",
			requestedPath: "album/track.mp3",
			shouldError:   false,
		},
		{
			name:          "Path traversal attempt",
			basePath:      "C:\\Users\\test\\music",
			requestedPath: "../../etc/passwd",
			shouldError:   true,
		},
		{
			name:          "Absolute Windows path attempt",
			basePath:      "C:\\Users\\test\\music",
			requestedPath: "C:\\Windows\\System32",
			shouldError:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateFilePath(tt.basePath, tt.requestedPath)
			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
