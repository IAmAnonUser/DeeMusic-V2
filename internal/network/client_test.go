package network

import (
	"net/http"
	"testing"
	"time"
)

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.Timeout)
	}

	if config.MaxIdleConns != 100 {
		t.Errorf("Expected MaxIdleConns 100, got %d", config.MaxIdleConns)
	}

	if config.MaxIdleConnsPerHost != 20 {
		t.Errorf("Expected MaxIdleConnsPerHost 20, got %d", config.MaxIdleConnsPerHost)
	}

	if config.DisableKeepAlives {
		t.Error("Expected keep-alives to be enabled")
	}
}

func TestNewClient(t *testing.T) {
	config := &ClientConfig{
		Timeout:             10 * time.Second,
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     25,
		IdleConnTimeout:     60 * time.Second,
	}

	client := NewClient(config)

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", client.Timeout)
	}
}

func TestNewClientWithNilConfig(t *testing.T) {
	client := NewClient(nil)

	if client == nil {
		t.Fatal("Expected client to be created with default config")
	}

	// Should use default timeout
	if client.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", client.Timeout)
	}
}

func TestGetDefaultClient(t *testing.T) {
	client1 := GetDefaultClient()
	client2 := GetDefaultClient()

	if client1 == nil {
		t.Fatal("Expected default client to be created")
	}

	// Should return same instance (singleton)
	if client1 != client2 {
		t.Error("Expected GetDefaultClient to return same instance")
	}
}

func TestGetDownloadClient(t *testing.T) {
	timeout := 60 * time.Second
	client := GetDownloadClient(timeout)

	if client == nil {
		t.Fatal("Expected download client to be created")
	}

	if client.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, client.Timeout)
	}
}

func TestConnectionPoolingSettings(t *testing.T) {
	config := DefaultClientConfig()
	client := NewClient(config)

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected transport to be *http.Transport")
	}

	if transport.MaxIdleConns != config.MaxIdleConns {
		t.Errorf("Expected MaxIdleConns %d, got %d", config.MaxIdleConns, transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != config.MaxIdleConnsPerHost {
		t.Errorf("Expected MaxIdleConnsPerHost %d, got %d", config.MaxIdleConnsPerHost, transport.MaxIdleConnsPerHost)
	}

	if transport.MaxConnsPerHost != config.MaxConnsPerHost {
		t.Errorf("Expected MaxConnsPerHost %d, got %d", config.MaxConnsPerHost, transport.MaxConnsPerHost)
	}

	if transport.DisableKeepAlives != config.DisableKeepAlives {
		t.Errorf("Expected DisableKeepAlives %v, got %v", config.DisableKeepAlives, transport.DisableKeepAlives)
	}
}

func TestTimeoutSettings(t *testing.T) {
	config := DefaultClientConfig()
	client := NewClient(config)

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected transport to be *http.Transport")
	}

	if transport.TLSHandshakeTimeout != config.TLSHandshakeTimeout {
		t.Errorf("Expected TLSHandshakeTimeout %v, got %v", config.TLSHandshakeTimeout, transport.TLSHandshakeTimeout)
	}

	if transport.ResponseHeaderTimeout != config.ResponseHeaderTimeout {
		t.Errorf("Expected ResponseHeaderTimeout %v, got %v", config.ResponseHeaderTimeout, transport.ResponseHeaderTimeout)
	}

	if transport.ExpectContinueTimeout != config.ExpectContinueTimeout {
		t.Errorf("Expected ExpectContinueTimeout %v, got %v", config.ExpectContinueTimeout, transport.ExpectContinueTimeout)
	}
}
