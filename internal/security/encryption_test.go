package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTokenEncryption(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	
	encryptor := NewTokenEncryptor(tempDir)
	
	testToken := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	
	// Test encryption
	encrypted, err := encryptor.EncryptToken(testToken)
	if err != nil {
		t.Fatalf("Failed to encrypt token: %v", err)
	}
	
	if encrypted == "" {
		t.Fatal("Encrypted token is empty")
	}
	
	if encrypted == testToken {
		t.Fatal("Encrypted token is same as plaintext")
	}
	
	// Test decryption
	decrypted, err := encryptor.DecryptToken(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt token: %v", err)
	}
	
	if decrypted != testToken {
		t.Fatalf("Decrypted token doesn't match original. Got: %s, Want: %s", decrypted, testToken)
	}
}

func TestTokenEncryptionEmptyToken(t *testing.T) {
	tempDir := t.TempDir()
	encryptor := NewTokenEncryptor(tempDir)
	
	_, err := encryptor.EncryptToken("")
	if err == nil {
		t.Fatal("Expected error for empty token, got nil")
	}
}

func TestTokenDecryptionEmptyToken(t *testing.T) {
	tempDir := t.TempDir()
	encryptor := NewTokenEncryptor(tempDir)
	
	_, err := encryptor.DecryptToken("")
	if err == nil {
		t.Fatal("Expected error for empty encrypted token, got nil")
	}
}

func TestTokenDecryptionInvalidData(t *testing.T) {
	tempDir := t.TempDir()
	encryptor := NewTokenEncryptor(tempDir)
	
	// First encrypt a token to create the key
	_, err := encryptor.EncryptToken("test")
	if err != nil {
		t.Fatalf("Failed to encrypt token: %v", err)
	}
	
	// Try to decrypt invalid data
	_, err = encryptor.DecryptToken("invalid-base64-data!!!")
	if err == nil {
		t.Fatal("Expected error for invalid encrypted token, got nil")
	}
}

func TestKeyPersistence(t *testing.T) {
	tempDir := t.TempDir()
	
	encryptor1 := NewTokenEncryptor(tempDir)
	testToken := "test-token-123"
	
	// Encrypt with first encryptor
	encrypted, err := encryptor1.EncryptToken(testToken)
	if err != nil {
		t.Fatalf("Failed to encrypt token: %v", err)
	}
	
	// Create new encryptor with same directory
	encryptor2 := NewTokenEncryptor(tempDir)
	
	// Decrypt with second encryptor (should use same key)
	decrypted, err := encryptor2.DecryptToken(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt token with second encryptor: %v", err)
	}
	
	if decrypted != testToken {
		t.Fatalf("Decrypted token doesn't match. Got: %s, Want: %s", decrypted, testToken)
	}
}

func TestDeleteKey(t *testing.T) {
	tempDir := t.TempDir()
	encryptor := NewTokenEncryptor(tempDir)
	
	// Encrypt a token (creates key) - skip credential manager on Windows
	encrypted, err := encryptor.EncryptToken("test")
	if err != nil {
		t.Fatalf("Failed to encrypt token: %v", err)
	}
	
	// Skip test if using credential manager
	if encrypted == "CREDENTIAL_MANAGER" {
		t.Skip("Skipping test when using credential manager")
	}
	
	// Verify key file exists
	keyPath := filepath.Join(tempDir, ".key")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Fatal("Key file was not created")
	}
	
	// Delete key
	if err := encryptor.DeleteKey(); err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}
	
	// Verify key file is deleted
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Fatal("Key file still exists after deletion")
	}
}

func TestMultipleEncryptions(t *testing.T) {
	tempDir := t.TempDir()
	encryptor := NewTokenEncryptor(tempDir)
	
	testToken := "test-token-456"
	
	// Encrypt same token multiple times
	encrypted1, err := encryptor.EncryptToken(testToken)
	if err != nil {
		t.Fatalf("Failed first encryption: %v", err)
	}
	
	encrypted2, err := encryptor.EncryptToken(testToken)
	if err != nil {
		t.Fatalf("Failed second encryption: %v", err)
	}
	
	// Skip nonce check if using credential manager (returns same marker)
	if encrypted1 != "CREDENTIAL_MANAGER" && encrypted2 != "CREDENTIAL_MANAGER" {
		// Encrypted values should be different (due to random nonce)
		if encrypted1 == encrypted2 {
			t.Fatal("Multiple encryptions of same token produced identical ciphertext")
		}
	}
	
	// Both should decrypt to original
	decrypted1, err := encryptor.DecryptToken(encrypted1)
	if err != nil {
		t.Fatalf("Failed to decrypt first ciphertext: %v", err)
	}
	
	decrypted2, err := encryptor.DecryptToken(encrypted2)
	if err != nil {
		t.Fatalf("Failed to decrypt second ciphertext: %v", err)
	}
	
	if decrypted1 != testToken || decrypted2 != testToken {
		t.Fatal("Decrypted tokens don't match original")
	}
}
