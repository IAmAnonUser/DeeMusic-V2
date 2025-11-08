package decryption

import (
	"crypto/cipher"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/deemusic/deemusic-go/internal/network"
	"golang.org/x/crypto/blowfish"
)

// StreamingProcessor handles memory-efficient streaming operations for downloading
// and decrypting audio files without loading entire files into memory.
type StreamingProcessor struct {
	// Fixed parameters for Deezer decryption - MUST NOT be changed
	encryptedChunkSize int    // 2048 bytes - first chunk in each segment (encrypted)
	plainChunkSize     int    // 4096 bytes - remaining data in each segment (plain)
	segmentSize        int    // 6144 bytes - total segment size (2048 + 4096)
	bfSecret           string // "g4el58wc0zvf9na1" - hardcoded Deezer secret
	iv                 []byte // Fixed IV for Blowfish CBC
	chunkSize          int    // Legacy chunk size for non-decryption operations
}

// NewStreamingProcessor creates a new StreamingProcessor with fixed Deezer decryption parameters.
// The chunkSize parameter is for backward compatibility and used for non-decryption operations only.
// Decryption uses fixed parameters regardless of this value.
func NewStreamingProcessor(chunkSize int) *StreamingProcessor {
	if chunkSize <= 0 {
		chunkSize = 8192 // Default 8KB
	}

	iv, _ := hex.DecodeString("0001020304050607")

	return &StreamingProcessor{
		encryptedChunkSize: 2048,
		plainChunkSize:     4096,
		segmentSize:        6144,
		bfSecret:           "g4el58wc0zvf9na1",
		iv:                 iv,
		chunkSize:          chunkSize,
	}
}

// GenerateDecryptionKey generates a decryption key for a given song ID.
// This implements the exact algorithm from the Python version:
// 1. MD5 hash of song ID
// 2. XOR with Blowfish secret
// 3. Return 16-byte key
func (sp *StreamingProcessor) GenerateDecryptionKey(songID string) ([]byte, error) {
	// Hash the song ID
	hasher := md5.New()
	hasher.Write([]byte(songID))
	hashedSongIDBytes := hasher.Sum(nil)
	hashedSongIDHex := hex.EncodeToString(hashedSongIDBytes)

	if len(hashedSongIDHex) != 32 {
		return nil, fmt.Errorf("invalid MD5 hash length for song ID %s", songID)
	}

	// Generate key using XOR with secret
	keyBytes := make([]byte, 16)
	for i := 0; i < 16; i++ {
		xorVal := hashedSongIDHex[i] ^ hashedSongIDHex[i+16] ^ sp.bfSecret[i]
		keyBytes[i] = byte(xorVal)
	}

	// Validate key length for Blowfish (4-56 bytes)
	if len(keyBytes) < 4 || len(keyBytes) > 56 {
		return nil, fmt.Errorf("generated key has invalid length for song ID %s", songID)
	}

	return keyBytes, nil
}

// DecryptFile decrypts a file using the exact Deezer CBC stripe method.
//
// Stripe Pattern:
// Segment (6144 bytes) = [Encrypted Chunk (2048)] + [Plain Data (4096)]
//
// The function processes the file in segments, decrypting only the first 2048 bytes
// of each 6144-byte segment, and writing the remaining 4096 bytes as-is.
// CRITICAL: A new cipher must be created for each encrypted chunk to prevent state corruption.
func (sp *StreamingProcessor) DecryptFile(encryptedPath, decryptedPath string, key []byte) error {
	// Open encrypted file for reading
	encFile, err := os.Open(encryptedPath)
	if err != nil {
		return fmt.Errorf("failed to open encrypted file: %w", err)
	}
	defer encFile.Close()

	// Create decrypted file for writing
	decFile, err := os.Create(decryptedPath)
	if err != nil {
		return fmt.Errorf("failed to create decrypted file: %w", err)
	}
	defer decFile.Close()

	// Process file in segments
	buffer := make([]byte, 0, sp.segmentSize)
	segmentBuffer := make([]byte, sp.segmentSize)

	for {
		// Read up to segmentSize bytes
		n, err := encFile.Read(segmentBuffer[len(buffer):])
		if n > 0 {
			buffer = append(buffer, segmentBuffer[len(buffer):len(buffer)+n]...)
		}

		// Check for EOF
		if err == io.EOF {
			// Process any remaining data in buffer
			if len(buffer) > 0 {
				if err := sp.processSegment(buffer, decFile, key); err != nil {
					return err
				}
			}
			break
		}

		if err != nil {
			return fmt.Errorf("error reading encrypted file: %w", err)
		}

		// Process complete segments
		for len(buffer) >= sp.segmentSize {
			segment := buffer[:sp.segmentSize]
			if err := sp.processSegment(segment, decFile, key); err != nil {
				return err
			}
			buffer = buffer[sp.segmentSize:]
		}
	}

	return nil
}

// processSegment processes a single segment (or partial segment) of data.
// For complete segments (6144 bytes), it decrypts the first 2048 bytes and writes
// the remaining 4096 bytes as-is.
// For partial segments, it handles them appropriately based on size.
// CRITICAL: Creates a NEW cipher for each encrypted chunk to prevent state corruption (as per Python version).
func (sp *StreamingProcessor) processSegment(segment []byte, writer io.Writer, key []byte) error {
	segmentLen := len(segment)

	if segmentLen >= sp.segmentSize {
		// Complete segment: decrypt first 2048 bytes, write remaining 4096 as-is
		encryptedChunk := segment[:sp.encryptedChunkSize]
		plainRemainder := segment[sp.encryptedChunkSize:sp.segmentSize]

		// Create NEW cipher for this chunk (critical for correct decryption)
		block, err := blowfish.NewCipher(key)
		if err != nil {
			return fmt.Errorf("failed to create Blowfish cipher: %w", err)
		}
		decrypter := cipher.NewCBCDecrypter(block, sp.iv)

		// Decrypt the encrypted chunk
		decryptedChunk := make([]byte, sp.encryptedChunkSize)
		decrypter.CryptBlocks(decryptedChunk, encryptedChunk)

		// Write decrypted chunk + plain remainder
		if _, err := writer.Write(decryptedChunk); err != nil {
			return fmt.Errorf("failed to write decrypted chunk: %w", err)
		}
		if _, err := writer.Write(plainRemainder); err != nil {
			return fmt.Errorf("failed to write plain remainder: %w", err)
		}
	} else if segmentLen >= sp.encryptedChunkSize {
		// Partial segment with enough data to decrypt
		encryptedChunk := segment[:sp.encryptedChunkSize]
		plainRemainder := segment[sp.encryptedChunkSize:]

		// Create NEW cipher for this chunk (critical for correct decryption)
		block, err := blowfish.NewCipher(key)
		if err != nil {
			return fmt.Errorf("failed to create Blowfish cipher: %w", err)
		}
		decrypter := cipher.NewCBCDecrypter(block, sp.iv)

		// Decrypt the encrypted chunk
		decryptedChunk := make([]byte, sp.encryptedChunkSize)
		decrypter.CryptBlocks(decryptedChunk, encryptedChunk)

		// Write decrypted chunk + plain remainder
		if _, err := writer.Write(decryptedChunk); err != nil {
			return fmt.Errorf("failed to write decrypted chunk: %w", err)
		}
		if _, err := writer.Write(plainRemainder); err != nil {
			return fmt.Errorf("failed to write plain remainder: %w", err)
		}
	} else {
		// Buffer too small to decrypt, write as-is
		if _, err := writer.Write(segment); err != nil {
			return fmt.Errorf("failed to write small segment: %w", err)
		}
	}

	return nil
}

// ProgressCallback is a function type for reporting progress during operations.
// It receives bytes processed and total bytes.
type ProgressCallback func(bytesProcessed, totalBytes int64)

// StreamDecrypt decrypts a file using the CBC stripe algorithm with integrity validation.
// It reads from an encrypted file and writes the decrypted output to the specified path.
func (sp *StreamingProcessor) StreamDecrypt(encryptedPath, outputPath string, key []byte, progressCallback ProgressCallback) error {
	// Get file size for progress reporting
	fileInfo, err := os.Stat(encryptedPath)
	if err != nil {
		return fmt.Errorf("failed to stat encrypted file: %w", err)
	}
	totalBytes := fileInfo.Size()

	// Decrypt the file
	if err := sp.DecryptFile(encryptedPath, outputPath, key); err != nil {
		// Clean up partial output file on error
		os.Remove(outputPath)
		return err
	}

	// Report progress completion if callback provided
	if progressCallback != nil {
		progressCallback(totalBytes, totalBytes)
	}

	return nil
}

// DownloadResult contains the result of a download and decrypt operation.
type DownloadResult struct {
	Success       bool
	ErrorMessage  string
	FileSize      int64
	DownloadTime  float64 // seconds
	DecryptTime   float64 // seconds
}

// StreamDownload downloads a file with streaming and integrated progress reporting.
// It downloads from the given URL and saves to the output path.
func (sp *StreamingProcessor) StreamDownload(url, outputPath string, progressCallback ProgressCallback, headers map[string]string, timeout int) error {
	// Use optimized download client with connection pooling
	client := network.GetDownloadClient(time.Duration(timeout) * time.Second)

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Get total size from headers
	totalSize := resp.ContentLength
	var bytesDownloaded int64

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Download with progress reporting
	buffer := make([]byte, sp.chunkSize)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := outFile.Write(buffer[:n]); writeErr != nil {
				return fmt.Errorf("failed to write to file: %w", writeErr)
			}
			bytesDownloaded += int64(n)

			// Report progress
			if progressCallback != nil {
				progressCallback(bytesDownloaded, totalSize)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			// Clean up partial download
			os.Remove(outputPath)
			return fmt.Errorf("error reading response: %w", err)
		}
	}

	// Verify download completed successfully
	if fileInfo, err := os.Stat(outputPath); err != nil || fileInfo.Size() == 0 {
		os.Remove(outputPath)
		return fmt.Errorf("download failed: output file is empty or missing")
	}

	return nil
}

// DownloadAndDecrypt downloads and decrypts a file in a single streaming operation.
// This is the main method that combines download and decryption with progress reporting.
func (sp *StreamingProcessor) DownloadAndDecrypt(url, songID, outputPath string, progressCallback ProgressCallback, headers map[string]string, timeout int) (*DownloadResult, error) {
	result := &DownloadResult{
		Success: false,
	}

	// Generate decryption key
	key, err := sp.GenerateDecryptionKey(songID)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to generate decryption key: %v", err)
		return result, fmt.Errorf("failed to generate decryption key: %w", err)
	}

	// Create temporary file for encrypted download
	tempFile, err := os.CreateTemp("", "deemusic-encrypted-*.tmp")
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create temp file: %v", err)
		return result, fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath) // Clean up temp file

	// Download encrypted file
	downloadStart := time.Now()
	downloadCallback := func(downloaded, total int64) {
		if progressCallback != nil {
			// Report download as first half of progress
			progressCallback(downloaded/2, total)
		}
	}

	if err := sp.StreamDownload(url, tempPath, downloadCallback, headers, timeout); err != nil {
		result.ErrorMessage = fmt.Sprintf("download failed: %v", err)
		return result, fmt.Errorf("download failed: %w", err)
	}
	result.DownloadTime = time.Since(downloadStart).Seconds()

	// Decrypt the downloaded file
	decryptStart := time.Now()
	decryptCallback := func(processed, total int64) {
		if progressCallback != nil {
			// Report decryption as second half of progress
			progressCallback(total/2+processed/2, total)
		}
	}

	if err := sp.StreamDecrypt(tempPath, outputPath, key, decryptCallback); err != nil {
		result.ErrorMessage = fmt.Sprintf("decryption failed: %v", err)
		return result, fmt.Errorf("decryption failed: %w", err)
	}
	result.DecryptTime = time.Since(decryptStart).Seconds()

	// Get final file size
	if fileInfo, err := os.Stat(outputPath); err == nil {
		result.FileSize = fileInfo.Size()
	}

	result.Success = true
	return result, nil
}

// DownloadAndDecryptResumable downloads and decrypts a file with resume capability.
// It supports resuming interrupted downloads using HTTP Range requests.
func (sp *StreamingProcessor) DownloadAndDecryptResumable(
	url, songID, outputPath, partialPath string,
	bytesDownloaded, totalBytes int64,
	progressCallback ProgressCallback,
	headers map[string]string,
	timeout int,
) (*DownloadResult, error) {
	result := &DownloadResult{
		Success: false,
	}

	// Generate decryption key
	key, err := sp.GenerateDecryptionKey(songID)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to generate decryption key: %v", err)
		return result, fmt.Errorf("failed to generate decryption key: %w", err)
	}

	// Use partial path as temp file if provided, otherwise create new
	tempPath := partialPath
	if tempPath == "" {
		tempFile, err := os.CreateTemp("", "deemusic-encrypted-*.tmp")
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("failed to create temp file: %v", err)
			return result, fmt.Errorf("failed to create temp file: %w", err)
		}
		tempPath = tempFile.Name()
		tempFile.Close()
	}
	defer func() {
		// Only remove temp file if download succeeded
		if result.Success {
			os.Remove(tempPath)
		}
	}()

	// Download encrypted file with resume support
	downloadStart := time.Now()
	downloadCallback := func(downloaded, total int64) {
		if progressCallback != nil {
			// Report download as first half of progress
			progressCallback(downloaded/2, total)
		}
	}

	downloadConfig := &network.ResumeDownloadConfig{
		URL:              url,
		OutputPath:       tempPath + ".complete",
		PartialPath:      tempPath,
		BytesDownloaded:  bytesDownloaded,
		TotalBytes:       totalBytes,
		Headers:          headers,
		Timeout:          time.Duration(timeout) * time.Second,
		ProgressCallback: downloadCallback,
	}

	_, err = network.ResumeDownload(downloadConfig)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("download failed: %v", err)
		return result, fmt.Errorf("download failed: %w", err)
	}
	result.DownloadTime = time.Since(downloadStart).Seconds()

	// Use the completed download file for decryption
	encryptedPath := downloadConfig.OutputPath

	// Decrypt the downloaded file
	decryptStart := time.Now()
	decryptCallback := func(processed, total int64) {
		if progressCallback != nil {
			// Report decryption as second half of progress
			progressCallback(total/2+processed/2, total)
		}
	}

	if err := sp.StreamDecrypt(encryptedPath, outputPath, key, decryptCallback); err != nil {
		result.ErrorMessage = fmt.Sprintf("decryption failed: %v", err)
		return result, fmt.Errorf("decryption failed: %w", err)
	}
	result.DecryptTime = time.Since(decryptStart).Seconds()

	// Clean up encrypted file
	os.Remove(encryptedPath)

	// Get final file size
	if fileInfo, err := os.Stat(outputPath); err == nil {
		result.FileSize = fileInfo.Size()
	}

	result.Success = true
	return result, nil
}
