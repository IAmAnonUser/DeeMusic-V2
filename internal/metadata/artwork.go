package metadata

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nfnt/resize"
)

// ArtworkCache manages artwork caching
type ArtworkCache struct {
	cacheDir   string
	httpClient *http.Client
}

// NewArtworkCache creates a new artwork cache
func NewArtworkCache(cacheDir string) (*ArtworkCache, error) {
	if cacheDir == "" {
		return nil, fmt.Errorf("cache directory cannot be empty")
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &ArtworkCache{
		cacheDir: cacheDir,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// DownloadAndEmbedArtwork downloads artwork and embeds it in the audio file
func (m *Manager) DownloadAndEmbedArtwork(filePath string, artworkURL string, size int) error {
	if artworkURL == "" {
		return fmt.Errorf("artwork URL cannot be empty")
	}

	if size <= 0 {
		size = m.config.ArtworkSize
	}

	// Download artwork
	artworkData, mimeType, err := m.downloadArtwork(artworkURL, size)
	if err != nil {
		return fmt.Errorf("failed to download artwork: %w", err)
	}

	// Create metadata with artwork
	metadata := &TrackMetadata{
		ArtworkData: artworkData,
		ArtworkMIME: mimeType,
	}

	// Get existing metadata first
	existingMetadata, err := m.GetMetadata(filePath)
	if err == nil {
		// Preserve existing metadata
		metadata.Title = existingMetadata.Title
		metadata.Artist = existingMetadata.Artist
		metadata.Album = existingMetadata.Album
		metadata.AlbumArtist = existingMetadata.AlbumArtist
		metadata.TrackNumber = existingMetadata.TrackNumber
		metadata.DiscNumber = existingMetadata.DiscNumber
		metadata.Year = existingMetadata.Year
		metadata.Genre = existingMetadata.Genre
		metadata.ISRC = existingMetadata.ISRC
		metadata.Label = existingMetadata.Label
		metadata.Copyright = existingMetadata.Copyright
	}

	// Apply metadata with artwork
	return m.ApplyMetadata(filePath, metadata)
}

// downloadArtwork downloads and optionally resizes artwork
func (m *Manager) downloadArtwork(url string, targetSize int) ([]byte, string, error) {
	// Download image
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download artwork: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download artwork: status %d", resp.StatusCode)
	}

	// Read image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read artwork data: %w", err)
	}

	// Determine MIME type from Content-Type header
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	// Resize if needed
	if targetSize > 0 && targetSize != 1200 {
		resizedData, err := m.resizeImage(imageData, targetSize)
		if err != nil {
			// If resize fails, use original
			return imageData, mimeType, nil
		}
		return resizedData, mimeType, nil
	}

	return imageData, mimeType, nil
}

// resizeImage resizes an image to the target size
func (m *Manager) resizeImage(imageData []byte, targetSize int) ([]byte, error) {
	// Decode image
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Get current dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Skip resize if already at target size
	if width == targetSize && height == targetSize {
		return imageData, nil
	}

	// Resize image (maintain aspect ratio, use target as max dimension)
	var resized image.Image
	if width > height {
		resized = resize.Resize(uint(targetSize), 0, img, resize.Lanczos3)
	} else {
		resized = resize.Resize(0, uint(targetSize), img, resize.Lanczos3)
	}

	// Encode resized image
	var buf bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 95})
	case "png":
		err = png.Encode(&buf, resized)
	default:
		// Default to JPEG
		err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 95})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode resized image: %w", err)
	}

	return buf.Bytes(), nil
}

// DownloadArtwork downloads artwork from URL
func (ac *ArtworkCache) DownloadArtwork(url string, size int) ([]byte, string, error) {
	if url == "" {
		return nil, "", fmt.Errorf("artwork URL cannot be empty")
	}

	// Generate cache key from URL and size
	cacheKey := ac.generateCacheKey(url, size)
	cachePath := filepath.Join(ac.cacheDir, cacheKey)

	// Check if cached
	if data, mimeType, err := ac.loadFromCache(cachePath); err == nil {
		return data, mimeType, nil
	}

	// Download artwork
	resp, err := ac.httpClient.Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download artwork: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download artwork: status %d", resp.StatusCode)
	}

	// Read image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read artwork data: %w", err)
	}

	// Determine MIME type
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	// Resize if needed
	if size > 0 {
		resizedData, err := ac.resizeImage(imageData, size)
		if err == nil {
			imageData = resizedData
		}
	}

	// Save to cache
	if err := ac.saveToCache(cachePath, imageData, mimeType); err != nil {
		// Log error but don't fail
		fmt.Printf("Warning: failed to cache artwork: %v\n", err)
	}

	return imageData, mimeType, nil
}

// generateCacheKey generates a cache key from URL and size
func (ac *ArtworkCache) generateCacheKey(url string, size int) string {
	hash := md5.Sum([]byte(fmt.Sprintf("%s_%d", url, size)))
	return hex.EncodeToString(hash[:]) + ".jpg"
}

// loadFromCache loads artwork from cache
func (ac *ArtworkCache) loadFromCache(cachePath string) ([]byte, string, error) {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, "", err
	}

	// Determine MIME type from file extension
	ext := strings.ToLower(filepath.Ext(cachePath))
	mimeType := "image/jpeg"
	if ext == ".png" {
		mimeType = "image/png"
	}

	return data, mimeType, nil
}

// saveToCache saves artwork to cache
func (ac *ArtworkCache) saveToCache(cachePath string, data []byte, mimeType string) error {
	// Ensure cache directory exists
	dir := filepath.Dir(cachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write to temporary file first
	tempPath := cachePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// Rename to final path (atomic operation)
	if err := os.Rename(tempPath, cachePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename cache file: %w", err)
	}

	return nil
}

// resizeImage resizes an image to the target size
func (ac *ArtworkCache) resizeImage(imageData []byte, targetSize int) ([]byte, error) {
	// Decode image
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Get current dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Skip resize if already at target size
	if width == targetSize && height == targetSize {
		return imageData, nil
	}

	// Resize image (maintain aspect ratio, use target as max dimension)
	var resized image.Image
	if width > height {
		resized = resize.Resize(uint(targetSize), 0, img, resize.Lanczos3)
	} else {
		resized = resize.Resize(0, uint(targetSize), img, resize.Lanczos3)
	}

	// Encode resized image
	var buf bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 95})
	case "png":
		err = png.Encode(&buf, resized)
	default:
		// Default to JPEG
		err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 95})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode resized image: %w", err)
	}

	return buf.Bytes(), nil
}

// ClearCache clears all cached artwork
func (ac *ArtworkCache) ClearCache() error {
	entries, err := os.ReadDir(ac.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			path := filepath.Join(ac.cacheDir, entry.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove cache file: %w", err)
			}
		}
	}

	return nil
}

// GetCacheSize returns the total size of cached artwork in bytes
func (ac *ArtworkCache) GetCacheSize() (int64, error) {
	var totalSize int64

	entries, err := os.ReadDir(ac.cacheDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			totalSize += info.Size()
		}
	}

	return totalSize, nil
}

// CleanOldCache removes cached artwork older than the specified duration
func (ac *ArtworkCache) CleanOldCache(maxAge time.Duration) error {
	entries, err := os.ReadDir(ac.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	now := time.Now()
	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if now.Sub(info.ModTime()) > maxAge {
				path := filepath.Join(ac.cacheDir, entry.Name())
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("failed to remove old cache file: %w", err)
				}
			}
		}
	}

	return nil
}
