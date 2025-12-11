package network

import (
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"
)

var (
	// defaultClient is a shared HTTP client with optimized connection pooling
	defaultClient     *http.Client
	defaultClientOnce sync.Once
)

// ClientConfig holds configuration for HTTP client
type ClientConfig struct {
	Timeout                time.Duration
	MaxIdleConns           int
	MaxIdleConnsPerHost    int
	MaxConnsPerHost        int
	IdleConnTimeout        time.Duration
	TLSHandshakeTimeout    time.Duration
	ResponseHeaderTimeout  time.Duration
	ExpectContinueTimeout  time.Duration
	DisableKeepAlives      bool
	MaxResponseHeaderBytes int64
}

// DefaultClientConfig returns the default client configuration
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
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
}

// NewClient creates a new HTTP client with optimized connection pooling
func NewClient(config *ClientConfig) *http.Client {
	if config == nil {
		config = DefaultClientConfig()
	}

	transport := &http.Transport{
		// Connection pooling settings
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
		MaxConnsPerHost:     config.MaxConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,

		// Keep-alive settings
		DisableKeepAlives:      config.DisableKeepAlives,
		MaxResponseHeaderBytes: config.MaxResponseHeaderBytes,

		// Timeout settings
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		ExpectContinueTimeout: config.ExpectContinueTimeout,
	}

	// Add cookie jar for automatic cookie handling
	jar, _ := cookiejar.New(nil)

	return &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
		Jar:       jar,
	}
}

// GetDefaultClient returns a shared HTTP client with optimized settings
// This client is safe for concurrent use and reuses connections efficiently
func GetDefaultClient() *http.Client {
	defaultClientOnce.Do(func() {
		defaultClient = NewClient(DefaultClientConfig())
	})
	return defaultClient
}

// GetDownloadClient returns an HTTP client optimized for large file downloads
func GetDownloadClient(timeout time.Duration) *http.Client {
	config := DefaultClientConfig()
	config.Timeout = timeout
	config.MaxIdleConns = 200                        // More idle connections for reuse
	config.MaxIdleConnsPerHost = 50                  // More connections per host for parallel downloads
	config.MaxConnsPerHost = 100                     // Allow more concurrent connections to Deezer
	config.IdleConnTimeout = 120 * time.Second       // Keep connections alive longer
	config.ResponseHeaderTimeout = 60 * time.Second  // Longer timeout for large files
	config.DisableKeepAlives = false                 // Ensure keep-alives are enabled
	
	return NewClient(config)
}
