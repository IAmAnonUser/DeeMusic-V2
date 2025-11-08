# Network Package

This package provides optimized HTTP client configuration with connection pooling and keep-alive support for efficient network operations.

## Features

- **Connection Pooling**: Reuses HTTP connections across requests to reduce overhead
- **Keep-Alive**: Maintains persistent connections to reduce latency
- **Configurable Timeouts**: Separate timeouts for different operations
- **Shared Client Pool**: Singleton pattern for efficient resource usage
- **Download Optimization**: Specialized client for large file downloads

## Usage

### Using the Default Client

```go
import "github.com/deemusic/deemusic-go/internal/network"

// Get shared default client (recommended for API calls)
client := network.GetDefaultClient()

req, _ := http.NewRequest("GET", "https://api.example.com/data", nil)
resp, err := client.Do(req)
```

### Using the Download Client

```go
// Get optimized client for large file downloads
client := network.GetDownloadClient(60 * time.Second)

req, _ := http.NewRequest("GET", "https://cdn.example.com/file.mp3", nil)
resp, err := client.Do(req)
```

### Custom Client Configuration

```go
config := &network.ClientConfig{
    Timeout:                30 * time.Second,
    MaxIdleConns:           100,
    MaxIdleConnsPerHost:    20,
    MaxConnsPerHost:        50,
    IdleConnTimeout:        90 * time.Second,
    TLSHandshakeTimeout:    10 * time.Second,
    ResponseHeaderTimeout:  30 * time.Second,
    ExpectContinueTimeout:  1 * time.Second,
    DisableKeepAlives:      false,
    MaxResponseHeaderBytes: 10 << 20, // 10 MB
}

client := network.NewClient(config)
```

## Configuration Parameters

### Connection Pooling

- **MaxIdleConns**: Maximum idle connections across all hosts (default: 100)
- **MaxIdleConnsPerHost**: Maximum idle connections per host (default: 20)
- **MaxConnsPerHost**: Maximum total connections per host (default: 50)
- **IdleConnTimeout**: How long idle connections stay open (default: 90s)

### Keep-Alive

- **DisableKeepAlives**: Whether to disable keep-alive (default: false)
- **MaxResponseHeaderBytes**: Maximum response header size (default: 10 MB)

### Timeouts

- **Timeout**: Overall request timeout (default: 30s)
- **TLSHandshakeTimeout**: TLS handshake timeout (default: 10s)
- **ResponseHeaderTimeout**: Response header timeout (default: 30s)
- **ExpectContinueTimeout**: Expect: 100-continue timeout (default: 1s)

## Benefits

### Performance

- Reduces connection establishment overhead by reusing connections
- Minimizes TLS handshake latency with persistent connections
- Improves throughput for multiple requests to the same host

### Resource Efficiency

- Limits total connections to prevent resource exhaustion
- Automatically closes idle connections after timeout
- Shared client pool reduces memory footprint

### Reliability

- Configurable timeouts prevent hanging requests
- Connection limits prevent overwhelming servers
- Automatic retry on connection failures (when combined with retry logic)

## Best Practices

1. **Use GetDefaultClient() for API calls**: The shared client is optimized for typical API usage
2. **Use GetDownloadClient() for large files**: Specialized configuration for file downloads
3. **Reuse clients**: Don't create new clients for each request
4. **Set appropriate timeouts**: Match timeout to expected operation duration
5. **Monitor connection usage**: Track active connections in production

## Implementation Details

### Connection Reuse

The HTTP client maintains a pool of idle connections that can be reused for subsequent requests to the same host. This eliminates the overhead of:

- TCP connection establishment (3-way handshake)
- TLS handshake (multiple round trips)
- DNS resolution (when using keep-alive)

### Keep-Alive

HTTP keep-alive allows multiple requests to be sent over a single TCP connection. Benefits include:

- Reduced latency (no connection setup time)
- Lower CPU usage (fewer handshakes)
- Better network utilization

### Thread Safety

All HTTP clients created by this package are safe for concurrent use by multiple goroutines. The connection pool is managed internally with proper synchronization.

## Download Resume Capability

The package includes support for resuming interrupted downloads using HTTP Range requests.

### Checking Resume Support

```go
supportsResume, totalSize, err := network.SupportsResume(
    "https://example.com/file.mp3",
    headers,
    30 * time.Second,
)

if supportsResume {
    fmt.Printf("Server supports resume, file size: %d bytes\n", totalSize)
}
```

### Resuming a Download

```go
config := &network.ResumeDownloadConfig{
    URL:              "https://example.com/file.mp3",
    OutputPath:       "/path/to/final/file.mp3",
    PartialPath:      "/path/to/partial/file.mp3.part",
    BytesDownloaded:  1024000, // Already downloaded 1MB
    TotalBytes:       5242880, // Total 5MB
    Headers:          headers,
    Timeout:          60 * time.Second,
    ProgressCallback: func(downloaded, total int64) {
        fmt.Printf("Progress: %d/%d bytes\n", downloaded, total)
    },
}

result, err := network.ResumeDownload(config)
if err != nil {
    log.Printf("Download failed: %v", err)
} else if result.Success {
    fmt.Printf("Download completed, resumed: %v\n", result.Resumed)
}
```

### Resume Features

- **Automatic Resume Detection**: Checks if partial file exists and matches expected size
- **Range Request Support**: Uses HTTP Range header to resume from last byte
- **Fallback to Full Download**: Automatically starts over if resume fails
- **Partial File Preservation**: Keeps partial file on error for future resume attempts
- **Progress Tracking**: Reports progress including resumed bytes

## Requirements

This package satisfies requirements from the design document:

- **14.1**: HTTP connection pooling with keep-alive
- **14.3**: Download resume capability for interrupted transfers

## Related Packages

- `internal/api`: Uses network clients for Deezer API calls
- `internal/decryption`: Uses download client for streaming file downloads with resume support
- `internal/download`: Coordinates downloads using optimized clients
- `internal/store`: Stores partial download state for resume capability
