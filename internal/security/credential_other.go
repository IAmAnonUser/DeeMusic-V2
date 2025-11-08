//go:build !windows
// +build !windows

package security

import "fmt"

// storeInCredentialManager is not available on non-Windows platforms
func (te *TokenEncryptor) storeInCredentialManager(token string) error {
	return fmt.Errorf("credential manager not available on this platform")
}

// retrieveFromCredentialManager is not available on non-Windows platforms
func (te *TokenEncryptor) retrieveFromCredentialManager() (string, error) {
	return "", fmt.Errorf("credential manager not available on this platform")
}

// deleteFromCredentialManager is not available on non-Windows platforms
func (te *TokenEncryptor) deleteFromCredentialManager() error {
	return fmt.Errorf("credential manager not available on this platform")
}
