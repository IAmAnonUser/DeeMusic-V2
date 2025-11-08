package security

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SanitizeInput sanitizes a string input by removing dangerous characters
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	
	// Remove control characters except newline and tab
	var result strings.Builder
	for _, r := range input {
		if r >= 32 || r == '\n' || r == '\t' {
			result.WriteRune(r)
		}
	}
	
	return result.String()
}

// IsValidPath checks if a path is valid and doesn't contain traversal attempts
func IsValidPath(path string) bool {
	// Reject empty paths
	if path == "" {
		return false
	}
	
	// Reject paths with null bytes
	if strings.Contains(path, "\x00") {
		return false
	}
	
	// Clean the path
	cleaned := filepath.Clean(path)
	
	// Reject paths that try to go up directories
	if strings.Contains(cleaned, "..") {
		return false
	}
	
	// Reject absolute paths on Windows (C:\, D:\, etc.)
	if len(cleaned) >= 2 && cleaned[1] == ':' {
		return false
	}
	
	// Reject absolute paths (both Unix and Windows style)
	if filepath.IsAbs(cleaned) {
		return false
	}
	
	return true
}

// ValidateFilePath validates a file path for download operations
func ValidateFilePath(basePath, requestedPath string) (string, error) {
	// Reject absolute paths in requested path
	if filepath.IsAbs(requestedPath) {
		return "", fmt.Errorf("absolute paths not allowed")
	}
	
	// Clean both paths
	cleanBase := filepath.Clean(basePath)
	cleanRequested := filepath.Clean(requestedPath)
	
	// Check for path traversal in cleaned path
	if strings.Contains(cleanRequested, "..") {
		return "", fmt.Errorf("path traversal attempt detected")
	}
	
	// Join paths
	fullPath := filepath.Join(cleanBase, cleanRequested)
	
	// Ensure the full path is within the base path
	// Use filepath.Rel to check if the path is relative to base
	relPath, err := filepath.Rel(cleanBase, fullPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path traversal attempt detected")
	}
	
	return fullPath, nil
}


