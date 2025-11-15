package monitoring

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// DownloadsTotal tracks total number of downloads by status and quality
	DownloadsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "deemusic_downloads_total",
			Help: "Total number of downloads",
		},
		[]string{"status", "quality"},
	)

	// DownloadDuration tracks download duration in seconds by quality
	DownloadDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "deemusic_download_duration_seconds",
			Help:    "Download duration in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~17min
		},
		[]string{"quality"},
	)

	// QueueSize tracks current queue size
	QueueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "deemusic_queue_size",
			Help: "Current queue size",
		},
	)

	// ActiveDownloads tracks number of active downloads
	ActiveDownloads = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "deemusic_active_downloads",
			Help: "Number of active downloads",
		},
	)

	// DownloadBytesTotal tracks total bytes downloaded
	DownloadBytesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "deemusic_download_bytes_total",
			Help: "Total bytes downloaded",
		},
	)

	// APIRequestsTotal tracks API requests by endpoint and status
	APIRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "deemusic_api_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"endpoint", "status"},
	)

	// APIRequestDuration tracks API request duration
	APIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "deemusic_api_request_duration_seconds",
			Help:    "API request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)

	// DecryptionDuration tracks decryption duration
	DecryptionDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "deemusic_decryption_duration_seconds",
			Help:    "Decryption duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// ErrorsTotal tracks errors by type
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "deemusic_errors_total",
			Help: "Total number of errors",
		},
		[]string{"type"},
	)
)

// RecordDownloadStart records the start of a download
func RecordDownloadStart(quality string) {
	ActiveDownloads.Inc()
}

// RecordDownloadComplete records a completed download
func RecordDownloadComplete(quality string, duration time.Duration, bytes int64) {
	DownloadsTotal.WithLabelValues("completed", quality).Inc()
	DownloadDuration.WithLabelValues(quality).Observe(duration.Seconds())
	DownloadBytesTotal.Add(float64(bytes))
	ActiveDownloads.Dec()
}

// RecordDownloadFailed records a failed download
func RecordDownloadFailed(quality string, errorType string) {
	DownloadsTotal.WithLabelValues("failed", quality).Inc()
	ErrorsTotal.WithLabelValues(errorType).Inc()
	ActiveDownloads.Dec()
}

// UpdateQueueSize updates the queue size metric
func UpdateQueueSize(size int) {
	QueueSize.Set(float64(size))
}

// RecordAPIRequest records an API request
func RecordAPIRequest(endpoint string, status string, duration time.Duration) {
	APIRequestsTotal.WithLabelValues(endpoint, status).Inc()
	APIRequestDuration.WithLabelValues(endpoint).Observe(duration.Seconds())
}

// RecordDecryption records a decryption operation
func RecordDecryption(duration time.Duration) {
	DecryptionDuration.Observe(duration.Seconds())
}

// RecordError records an error
func RecordError(errorType string) {
	ErrorsTotal.WithLabelValues(errorType).Inc()
}
