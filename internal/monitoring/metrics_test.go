package monitoring

import (
	"testing"
	"time"
)

func TestRecordDownloadMetrics(t *testing.T) {
	// Test recording download start
	RecordDownloadStart("MP3_320")

	// Test recording download complete
	duration := 5 * time.Second
	bytes := int64(10 * 1024 * 1024) // 10 MB
	RecordDownloadComplete("MP3_320", duration, bytes)

	// Test recording download failed
	RecordDownloadFailed("FLAC", "network_error")
}

func TestUpdateQueueSize(t *testing.T) {
	// Test updating queue size
	UpdateQueueSize(42)
	UpdateQueueSize(0)
	UpdateQueueSize(10000)
}

func TestRecordAPIRequest(t *testing.T) {
	// Test recording API request
	duration := 100 * time.Millisecond
	RecordAPIRequest("/api/v1/search", "success", duration)
	RecordAPIRequest("/api/v1/album/123", "error", duration)
}

func TestRecordDecryption(t *testing.T) {
	// Test recording decryption
	duration := 2 * time.Second
	RecordDecryption(duration)
}

func TestRecordError(t *testing.T) {
	// Test recording errors
	RecordError("network_error")
	RecordError("auth_error")
	RecordError("decryption_error")
}
