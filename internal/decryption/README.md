# Decryption Package

This package implements the Deezer audio file decryption system using Blowfish CBC stripe decryption. It provides memory-efficient streaming operations for downloading and decrypting audio files.

## Overview

The decryption algorithm uses a specific stripe pattern where files are processed in 6144-byte segments:
- First 2048 bytes: Encrypted with Blowfish CBC
- Next 4096 bytes: Plain (unencrypted)

This pattern repeats throughout the file.

## Key Components

### StreamingProcessor

The main struct that handles all decryption and download operations.

```go
processor := decryption.NewStreamingProcessor(8192) // 8KB chunk size for downloads
```

### Key Generation

Generates a decryption key from a song ID using MD5 hashing and XOR with a secret:

```go
key, err := processor.GenerateDecryptionKey(songID)
if err != nil {
    // Handle error
}
```

### File Decryption

Decrypts a file using the CBC stripe algorithm:

```go
err := processor.DecryptFile(encryptedPath, decryptedPath, key)
if err != nil {
    // Handle error
}
```

### Streaming Download and Decrypt

Downloads and decrypts a file in one operation:

```go
result, err := processor.DownloadAndDecrypt(
    url,
    songID,
    outputPath,
    progressCallback,
    headers,
    timeout,
)
if err != nil {
    // Handle error
}
```

## Progress Callbacks

All streaming operations support progress callbacks:

```go
progressCallback := func(bytesProcessed, totalBytes int64) {
    percentage := float64(bytesProcessed) / float64(totalBytes) * 100
    fmt.Printf("Progress: %.2f%%\n", percentage)
}
```

## Fixed Parameters

The following parameters are hardcoded and must not be changed to ensure compatibility with Deezer's encryption:

- **Encrypted Chunk Size**: 2048 bytes
- **Plain Chunk Size**: 4096 bytes
- **Segment Size**: 6144 bytes (2048 + 4096)
- **Blowfish Secret**: "g4el58wc0zvf9na1"
- **IV**: 0x0001020304050607

## Testing

Run the test suite:

```bash
go test -v ./internal/decryption/
```

The tests verify:
- Key generation algorithm matches Python implementation
- Decryption handles various file sizes correctly
- Small files (< 2048 bytes) are handled properly
- Complete segments (6144 bytes) are processed correctly
- Multiple segments are handled correctly
- Partial final segments are handled correctly
- Error cases are handled gracefully

## Example Usage

```go
package main

import (
    "fmt"
    "github.com/deemusic/deemusic-go/internal/decryption"
)

func main() {
    // Create processor
    processor := decryption.NewStreamingProcessor(8192)
    
    // Download and decrypt
    result, err := processor.DownloadAndDecrypt(
        "https://example.com/encrypted.mp3",
        "123456789",
        "/path/to/output.mp3",
        func(processed, total int64) {
            fmt.Printf("Progress: %d/%d bytes\n", processed, total)
        },
        map[string]string{
            "User-Agent": "DeeMusic/1.0",
        },
        30, // timeout in seconds
    )
    
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    fmt.Printf("Success! Downloaded %d bytes in %.2fs (download: %.2fs, decrypt: %.2fs)\n",
        result.FileSize,
        result.DownloadTime + result.DecryptTime,
        result.DownloadTime,
        result.DecryptTime,
    )
}
```

## Compatibility

This implementation is designed to be 100% compatible with the Python version's decryption algorithm. The key generation and decryption logic exactly match the Python implementation to ensure identical output.
