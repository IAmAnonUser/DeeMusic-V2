package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// Encryption parameters
	keySize        = 32 // AES-256
	saltSize       = 32
	nonceSize      = 12 // GCM standard nonce size
	pbkdf2Iter     = 100000
	credentialName = "DeeMusic_ARL"
)

// TokenEncryptor handles ARL token encryption/decryption
type TokenEncryptor struct {
	keyPath string
}

// NewTokenEncryptor creates a new token encryptor
func NewTokenEncryptor(dataDir string) *TokenEncryptor {
	return &TokenEncryptor{
		keyPath: filepath.Join(dataDir, ".key"),
	}
}

// EncryptToken encrypts an ARL token
func (te *TokenEncryptor) EncryptToken(token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("token cannot be empty")
	}

	// Try Windows Credential Manager first on Windows
	if runtime.GOOS == "windows" {
		if err := te.storeInCredentialManager(token); err == nil {
			// Return a marker that indicates token is in credential manager
			return "CREDENTIAL_MANAGER", nil
		}
		// If credential manager fails, fall back to file encryption
	}

	// Generate or load encryption key
	key, err := te.getOrCreateKey()
	if err != nil {
		return "", fmt.Errorf("failed to get encryption key: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt token
	ciphertext := gcm.Seal(nonce, nonce, []byte(token), nil)

	// Encode to base64 for storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptToken decrypts an ARL token
func (te *TokenEncryptor) DecryptToken(encryptedToken string) (string, error) {
	if encryptedToken == "" {
		return "", fmt.Errorf("encrypted token cannot be empty")
	}

	// Check if token is in Windows Credential Manager
	if encryptedToken == "CREDENTIAL_MANAGER" {
		if runtime.GOOS == "windows" {
			token, err := te.retrieveFromCredentialManager()
			if err == nil {
				return token, nil
			}
			// If retrieval fails, return error
			return "", fmt.Errorf("failed to retrieve token from credential manager: %w", err)
		}
		return "", fmt.Errorf("credential manager marker found but not on Windows")
	}

	// Load encryption key
	key, err := te.loadKey()
	if err != nil {
		return "", fmt.Errorf("failed to load encryption key: %w", err)
	}

	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedToken)
	if err != nil {
		return "", fmt.Errorf("failed to decode token: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Check minimum size
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt token
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt token: %w", err)
	}

	return string(plaintext), nil
}

// getOrCreateKey gets or creates the encryption key
func (te *TokenEncryptor) getOrCreateKey() ([]byte, error) {
	// Try to load existing key
	key, err := te.loadKey()
	if err == nil {
		return key, nil
	}

	// Generate new key
	return te.generateAndSaveKey()
}

// loadKey loads the encryption key from file
func (te *TokenEncryptor) loadKey() ([]byte, error) {
	data, err := os.ReadFile(te.keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// Decode from base64
	keyData, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %w", err)
	}

	if len(keyData) < saltSize {
		return nil, fmt.Errorf("invalid key file format")
	}

	// Extract salt and derive key
	salt := keyData[:saltSize]
	
	// Use machine-specific data as password
	password := te.getMachineID()
	
	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iter, keySize, sha256.New)
	
	return key, nil
}

// generateAndSaveKey generates and saves a new encryption key
func (te *TokenEncryptor) generateAndSaveKey() ([]byte, error) {
	// Generate random salt
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Use machine-specific data as password
	password := te.getMachineID()

	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iter, keySize, sha256.New)

	// Encode salt to base64 for storage
	encoded := base64.StdEncoding.EncodeToString(salt)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(te.keyPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	// Save salt to file with restricted permissions
	if err := os.WriteFile(te.keyPath, []byte(encoded), 0600); err != nil {
		return nil, fmt.Errorf("failed to write key file: %w", err)
	}

	return key, nil
}

// getMachineID returns a machine-specific identifier
func (te *TokenEncryptor) getMachineID() string {
	// Use hostname as machine identifier
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "default-machine"
	}

	// Combine with username for additional uniqueness
	username := os.Getenv("USERNAME")
	if username == "" {
		username = os.Getenv("USER")
	}
	if username == "" {
		username = "default-user"
	}

	return hostname + ":" + username
}

// DeleteKey removes the encryption key file
func (te *TokenEncryptor) DeleteKey() error {
	if err := os.Remove(te.keyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete key file: %w", err)
	}
	return nil
}
