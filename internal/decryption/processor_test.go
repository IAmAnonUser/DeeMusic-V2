package decryption

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// TestGenerateDecryptionKey tests the key generation algorithm matches Python implementation
func TestGenerateDecryptionKey(t *testing.T) {
	sp := NewStreamingProcessor(8192)

	tests := []struct {
		name     string
		songID   string
		wantErr  bool
	}{
		{
			name:    "valid song ID",
			songID:  "123456789",
			wantErr: false,
		},
		{
			name:    "another valid song ID",
			songID:  "987654321",
			wantErr: false,
		},
		{
			name:    "empty song ID",
			songID:  "",
			wantErr: false, // Should still generate a key
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := sp.GenerateDecryptionKey(tt.songID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateDecryptionKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(key) != 16 {
					t.Errorf("GenerateDecryptionKey() key length = %d, want 16", len(key))
				}
				// Verify key is within Blowfish valid range
				if len(key) < 4 || len(key) > 56 {
					t.Errorf("GenerateDecryptionKey() key length %d outside Blowfish range [4, 56]", len(key))
				}
			}
		})
	}
}

// TestGenerateDecryptionKeyConsistency verifies the key generation is deterministic
func TestGenerateDecryptionKeyConsistency(t *testing.T) {
	sp := NewStreamingProcessor(8192)
	songID := "123456789"

	key1, err1 := sp.GenerateDecryptionKey(songID)
	if err1 != nil {
		t.Fatalf("First key generation failed: %v", err1)
	}

	key2, err2 := sp.GenerateDecryptionKey(songID)
	if err2 != nil {
		t.Fatalf("Second key generation failed: %v", err2)
	}

	if !bytes.Equal(key1, key2) {
		t.Errorf("Key generation is not deterministic: %x != %x", key1, key2)
	}
}

// TestGenerateDecryptionKeyAlgorithm verifies the exact algorithm matches Python
func TestGenerateDecryptionKeyAlgorithm(t *testing.T) {
	sp := NewStreamingProcessor(8192)
	songID := "123456789"

	// Generate key
	key, err := sp.GenerateDecryptionKey(songID)
	if err != nil {
		t.Fatalf("Key generation failed: %v", err)
	}

	// Manually verify the algorithm
	hasher := md5.New()
	hasher.Write([]byte(songID))
	hashedBytes := hasher.Sum(nil)
	hashedHex := hex.EncodeToString(hashedBytes)

	// Verify MD5 hash length
	if len(hashedHex) != 32 {
		t.Fatalf("MD5 hash has wrong length: %d", len(hashedHex))
	}

	// Manually compute expected key
	expectedKey := make([]byte, 16)
	bfSecret := "g4el58wc0zvf9na1"
	for i := 0; i < 16; i++ {
		xorVal := hashedHex[i] ^ hashedHex[i+16] ^ bfSecret[i]
		expectedKey[i] = byte(xorVal)
	}

	if !bytes.Equal(key, expectedKey) {
		t.Errorf("Key algorithm mismatch:\ngot:  %x\nwant: %x", key, expectedKey)
	}
}

// TestDecryptFileWithSmallFile tests decryption with a file smaller than one segment
func TestDecryptFileWithSmallFile(t *testing.T) {
	sp := NewStreamingProcessor(8192)

	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a small encrypted file (less than 2048 bytes)
	encryptedPath := filepath.Join(tempDir, "small_encrypted.bin")
	smallData := bytes.Repeat([]byte("test"), 100) // 400 bytes
	if err := os.WriteFile(encryptedPath, smallData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Generate a test key
	key, err := sp.GenerateDecryptionKey("test123")
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Decrypt the file
	decryptedPath := filepath.Join(tempDir, "small_decrypted.bin")
	err = sp.DecryptFile(encryptedPath, decryptedPath, key)
	if err != nil {
		t.Fatalf("DecryptFile failed: %v", err)
	}

	// Verify decrypted file exists
	if _, err := os.Stat(decryptedPath); os.IsNotExist(err) {
		t.Fatal("Decrypted file was not created")
	}

	// For small files (< 2048 bytes), they should be written as-is
	decryptedData, err := os.ReadFile(decryptedPath)
	if err != nil {
		t.Fatalf("Failed to read decrypted file: %v", err)
	}

	if !bytes.Equal(decryptedData, smallData) {
		t.Error("Small file was not written as-is")
	}
}

// TestDecryptFileWithExactSegment tests decryption with exactly one segment (6144 bytes)
func TestDecryptFileWithExactSegment(t *testing.T) {
	sp := NewStreamingProcessor(8192)

	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create an encrypted file with exactly one segment (6144 bytes)
	encryptedPath := filepath.Join(tempDir, "segment_encrypted.bin")
	segmentData := make([]byte, 6144)
	for i := range segmentData {
		segmentData[i] = byte(i % 256)
	}
	if err := os.WriteFile(encryptedPath, segmentData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Generate a test key
	key, err := sp.GenerateDecryptionKey("test456")
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Decrypt the file
	decryptedPath := filepath.Join(tempDir, "segment_decrypted.bin")
	err = sp.DecryptFile(encryptedPath, decryptedPath, key)
	if err != nil {
		t.Fatalf("DecryptFile failed: %v", err)
	}

	// Verify decrypted file exists and has correct size
	fileInfo, err := os.Stat(decryptedPath)
	if err != nil {
		t.Fatal("Decrypted file was not created")
	}

	if fileInfo.Size() != 6144 {
		t.Errorf("Decrypted file size = %d, want 6144", fileInfo.Size())
	}
}

// TestDecryptFileWithMultipleSegments tests decryption with multiple complete segments
func TestDecryptFileWithMultipleSegments(t *testing.T) {
	sp := NewStreamingProcessor(8192)

	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create an encrypted file with multiple segments (3 * 6144 = 18432 bytes)
	encryptedPath := filepath.Join(tempDir, "multi_encrypted.bin")
	multiData := make([]byte, 18432)
	for i := range multiData {
		multiData[i] = byte(i % 256)
	}
	if err := os.WriteFile(encryptedPath, multiData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Generate a test key
	key, err := sp.GenerateDecryptionKey("test789")
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Decrypt the file
	decryptedPath := filepath.Join(tempDir, "multi_decrypted.bin")
	err = sp.DecryptFile(encryptedPath, decryptedPath, key)
	if err != nil {
		t.Fatalf("DecryptFile failed: %v", err)
	}

	// Verify decrypted file exists and has correct size
	fileInfo, err := os.Stat(decryptedPath)
	if err != nil {
		t.Fatal("Decrypted file was not created")
	}

	if fileInfo.Size() != 18432 {
		t.Errorf("Decrypted file size = %d, want 18432", fileInfo.Size())
	}
}

// TestDecryptFileWithPartialSegment tests decryption with partial final segment
func TestDecryptFileWithPartialSegment(t *testing.T) {
	sp := NewStreamingProcessor(8192)

	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create an encrypted file with one complete segment + partial (6144 + 3000 = 9144 bytes)
	encryptedPath := filepath.Join(tempDir, "partial_encrypted.bin")
	partialData := make([]byte, 9144)
	for i := range partialData {
		partialData[i] = byte(i % 256)
	}
	if err := os.WriteFile(encryptedPath, partialData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Generate a test key
	key, err := sp.GenerateDecryptionKey("test999")
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	// Decrypt the file
	decryptedPath := filepath.Join(tempDir, "partial_decrypted.bin")
	err = sp.DecryptFile(encryptedPath, decryptedPath, key)
	if err != nil {
		t.Fatalf("DecryptFile failed: %v", err)
	}

	// Verify decrypted file exists and has correct size
	fileInfo, err := os.Stat(decryptedPath)
	if err != nil {
		t.Fatal("Decrypted file was not created")
	}

	if fileInfo.Size() != 9144 {
		t.Errorf("Decrypted file size = %d, want 9144", fileInfo.Size())
	}
}

// TestStreamingProcessorParameters verifies the fixed decryption parameters
func TestStreamingProcessorParameters(t *testing.T) {
	sp := NewStreamingProcessor(8192)

	if sp.encryptedChunkSize != 2048 {
		t.Errorf("encryptedChunkSize = %d, want 2048", sp.encryptedChunkSize)
	}

	if sp.plainChunkSize != 4096 {
		t.Errorf("plainChunkSize = %d, want 4096", sp.plainChunkSize)
	}

	if sp.segmentSize != 6144 {
		t.Errorf("segmentSize = %d, want 6144", sp.segmentSize)
	}

	if sp.bfSecret != "g4el58wc0zvf9na1" {
		t.Errorf("bfSecret = %s, want g4el58wc0zvf9na1", sp.bfSecret)
	}

	expectedIV, _ := hex.DecodeString("0001020304050607")
	if !bytes.Equal(sp.iv, expectedIV) {
		t.Errorf("iv = %x, want %x", sp.iv, expectedIV)
	}
}

// TestDecryptFileErrorHandling tests error cases
func TestDecryptFileErrorHandling(t *testing.T) {
	sp := NewStreamingProcessor(8192)
	tempDir := t.TempDir()

	tests := []struct {
		name          string
		setupFunc     func() (string, string, []byte)
		wantErr       bool
	}{
		{
			name: "non-existent input file",
			setupFunc: func() (string, string, []byte) {
				key, _ := sp.GenerateDecryptionKey("test")
				return filepath.Join(tempDir, "nonexistent.bin"),
					filepath.Join(tempDir, "output.bin"),
					key
			},
			wantErr: true,
		},
		{
			name: "invalid output directory",
			setupFunc: func() (string, string, []byte) {
				inputPath := filepath.Join(tempDir, "input.bin")
				os.WriteFile(inputPath, []byte("test"), 0644)
				key, _ := sp.GenerateDecryptionKey("test")
				return inputPath,
					"/invalid/path/that/does/not/exist/output.bin",
					key
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encPath, decPath, key := tt.setupFunc()
			err := sp.DecryptFile(encPath, decPath, key)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecryptFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
