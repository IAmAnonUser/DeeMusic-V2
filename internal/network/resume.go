package network

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ResumeDownloadConfig holds configuration for resumable downloads
type ResumeDownloadConfig struct {
	URL              string
	OutputPath       string
	PartialPath      string
	BytesDownloaded  int64
	TotalBytes       int64
	Headers          map[string]string
	Timeout          time.Duration
	ProgressCallback func(downloaded, total int64)
}

// ResumeDownloadResult contains the result of a resumable download
type ResumeDownloadResult struct {
	Success         bool
	BytesDownloaded int64
	TotalBytes      int64
	Resumed         bool
	ErrorMessage    string
}

// SupportsResume checks if a URL supports HTTP Range requests
func SupportsResume(url string, headers map[string]string, timeout time.Duration) (bool, int64, error) {
	client := GetDownloadClient(timeout)

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, 0, fmt.Errorf("failed to create HEAD request: %w", err)
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, 0, fmt.Errorf("HEAD request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check if server supports range requests
	acceptRanges := resp.Header.Get("Accept-Ranges")
	supportsRange := acceptRanges == "bytes"

	// Get content length
	contentLength := resp.ContentLength

	return supportsRange, contentLength, nil
}

// ResumeDownload downloads a file with resume capability using HTTP Range requests
func ResumeDownload(config *ResumeDownloadConfig) (*ResumeDownloadResult, error) {
	result := &ResumeDownloadResult{
		Success:         false,
		BytesDownloaded: config.BytesDownloaded,
		TotalBytes:      config.TotalBytes,
		Resumed:         false,
	}

	// Check if we should resume
	var startByte int64 = 0
	var outputFile *os.File
	var err error

	if config.PartialPath != "" && config.BytesDownloaded > 0 {
		// Verify partial file exists
		if fileInfo, err := os.Stat(config.PartialPath); err == nil {
			actualSize := fileInfo.Size()
			if actualSize == config.BytesDownloaded {
				// Resume from where we left off
				startByte = config.BytesDownloaded
				result.Resumed = true

				// Open file in append mode
				outputFile, err = os.OpenFile(config.PartialPath, os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					result.ErrorMessage = fmt.Sprintf("failed to open partial file: %v", err)
					return result, fmt.Errorf("failed to open partial file: %w", err)
				}
			} else {
				// Partial file size mismatch, start over
				os.Remove(config.PartialPath)
			}
		}
	}

	// If not resuming, create new file
	if outputFile == nil {
		// Ensure output directory exists
		if err := os.MkdirAll(filepath.Dir(config.PartialPath), 0755); err != nil {
			result.ErrorMessage = fmt.Sprintf("failed to create output directory: %v", err)
			return result, fmt.Errorf("failed to create output directory: %w", err)
		}

		outputFile, err = os.Create(config.PartialPath)
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("failed to create output file: %v", err)
			return result, fmt.Errorf("failed to create output file: %w", err)
		}
	}
	defer outputFile.Close()

	// Create HTTP client
	client := GetDownloadClient(config.Timeout)

	// Create request
	req, err := http.NewRequest("GET", config.URL, nil)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to create request: %v", err)
		return result, fmt.Errorf("failed to create request: %w", err)
	}

	// Add custom headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// Add Range header if resuming
	if startByte > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startByte))
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("download request failed: %v", err)
		return result, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if startByte > 0 {
		// When resuming, expect 206 Partial Content
		if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
			result.ErrorMessage = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
			return result, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
	} else {
		// When starting fresh, expect 200 OK
		if resp.StatusCode != http.StatusOK {
			result.ErrorMessage = fmt.Sprintf("download failed with status: %d", resp.StatusCode)
			return result, fmt.Errorf("download failed with status: %d", resp.StatusCode)
		}
	}

	// Get total size
	if result.TotalBytes == 0 {
		result.TotalBytes = resp.ContentLength + startByte
	}

	// Use buffered writer for better I/O performance (256KB buffer)
	bufferedWriter := bufio.NewWriterSize(outputFile, 256*1024)

	// Download with progress reporting
	buffer := make([]byte, 256*1024) // 256KB buffer for better throughput
	bytesDownloaded := startByte

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := bufferedWriter.Write(buffer[:n]); writeErr != nil {
				result.ErrorMessage = fmt.Sprintf("failed to write to file: %v", writeErr)
				return result, fmt.Errorf("failed to write to file: %w", writeErr)
			}
			bytesDownloaded += int64(n)
			result.BytesDownloaded = bytesDownloaded

			// Report progress
			if config.ProgressCallback != nil {
				config.ProgressCallback(bytesDownloaded, result.TotalBytes)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			// Flush buffer before returning error
			bufferedWriter.Flush()
			result.ErrorMessage = fmt.Sprintf("error reading response: %v", err)
			// Don't delete partial file on error - allow resume
			return result, fmt.Errorf("error reading response: %w", err)
		}
	}
	
	// Flush buffered writer
	if err := bufferedWriter.Flush(); err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to flush buffer: %v", err)
		return result, fmt.Errorf("failed to flush buffer: %w", err)
	}

	// Verify download completed successfully
	if result.TotalBytes > 0 && bytesDownloaded < result.TotalBytes {
		result.ErrorMessage = "download incomplete"
		return result, fmt.Errorf("download incomplete")
	}

	// Move partial file to final location
	if err := os.Rename(config.PartialPath, config.OutputPath); err != nil {
		// If rename fails, try copy
		if copyErr := copyFile(config.PartialPath, config.OutputPath); copyErr != nil {
			result.ErrorMessage = fmt.Sprintf("failed to move file to final location: %v", err)
			return result, fmt.Errorf("failed to move file to final location: %w", err)
		}
		os.Remove(config.PartialPath)
	}

	result.Success = true
	return result, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
