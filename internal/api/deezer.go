package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/deemusic/deemusic-go/internal/network"
	"golang.org/x/time/rate"
)

const (
	deezerAPIURL     = "https://api.deezer.com"
	deezerPrivateAPI = "https://www.deezer.com/ajax/gw-light.php"
	deezerMediaURL   = "https://media.deezer.com"
)

// DeezerClient handles all Deezer API interactions
type DeezerClient struct {
	httpClient   *http.Client
	arl          string
	apiToken     string
	licenseToken string
	userID       string
	rateLimiter  *rate.Limiter
	mu           sync.RWMutex
	authenticated bool
}

// NewDeezerClient creates a new Deezer API client with optimized connection pooling
func NewDeezerClient(timeout time.Duration) *DeezerClient {
	// Use shared client pool with custom timeout
	config := network.DefaultClientConfig()
	config.Timeout = timeout
	
	return &DeezerClient{
		httpClient:  network.NewClient(config),
		rateLimiter: rate.NewLimiter(rate.Every(100*time.Millisecond), 10), // 10 requests per second
	}
}

// Authenticate authenticates with Deezer using ARL token
func (c *DeezerClient) Authenticate(ctx context.Context, arl string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if arl == "" {
		return fmt.Errorf("ARL token cannot be empty")
	}

	c.arl = arl

	// Get API token and user info
	if err := c.getAPIToken(ctx); err != nil {
		return fmt.Errorf("failed to get API token: %w", err)
	}

	// Get license token
	if err := c.getLicenseToken(ctx); err != nil {
		return fmt.Errorf("failed to get license token: %w", err)
	}

	c.authenticated = true
	return nil
}

// getAPIToken retrieves the API token from Deezer
func (c *DeezerClient) getAPIToken(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", deezerPrivateAPI+"?method=deezer.getUserData&input=3&api_version=1.0&api_token=", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Cookie", "arl="+c.arl)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status: %d", resp.StatusCode)
	}

	var result struct {
		Results struct {
			CheckForm string `json:"checkForm"`
			User      struct {
				UserID int `json:"USER_ID"`
			} `json:"USER"`
		} `json:"results"`
		Error interface{} `json:"error"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to decode response: %w (body: %s)", err, string(body))
	}

	// Check if error is a non-empty array or non-null object
	// Empty arrays [] are considered success
	if result.Error != nil {
		// Check if it's an empty array by converting to string
		errorStr := fmt.Sprintf("%v", result.Error)
		if errorStr != "[]" && errorStr != "" {
			return fmt.Errorf("authentication error: %v", result.Error)
		}
	}

	if result.Results.User.UserID == 0 {
		return fmt.Errorf("invalid ARL token: user ID is 0")
	}

	c.apiToken = result.Results.CheckForm
	c.userID = fmt.Sprintf("%d", result.Results.User.UserID)

	return nil
}

// getLicenseToken retrieves the license token for downloads
func (c *DeezerClient) getLicenseToken(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", deezerPrivateAPI+"?method=deezer.getUserData&input=3&api_version=1.0&api_token="+c.apiToken, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Cookie", "arl="+c.arl)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Results struct {
			User struct {
				Options struct {
					License string `json:"license_token"`
				} `json:"OPTIONS"`
			} `json:"USER"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode license response: %w", err)
	}

	c.licenseToken = result.Results.User.Options.License
	return nil
}

// RefreshToken refreshes the authentication tokens
func (c *DeezerClient) RefreshToken(ctx context.Context) error {
	c.mu.Lock()
	arl := c.arl
	c.mu.Unlock()

	if arl == "" {
		return fmt.Errorf("no ARL token available for refresh")
	}

	return c.Authenticate(ctx, arl)
}

// IsAuthenticated returns whether the client is authenticated
func (c *DeezerClient) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.authenticated
}

// doRequest performs an HTTP request with rate limiting
func (c *DeezerClient) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Check for authentication errors
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		return nil, fmt.Errorf("authentication required or token expired")
	}

	return resp, nil
}

// doPrivateAPIRequest performs a request to Deezer's private API
func (c *DeezerClient) doPrivateAPIRequest(ctx context.Context, method string, params map[string]interface{}) (map[string]interface{}, error) {
	c.mu.RLock()
	if !c.authenticated {
		c.mu.RUnlock()
		return nil, fmt.Errorf("client not authenticated")
	}
	apiToken := c.apiToken
	arl := c.arl
	c.mu.RUnlock()

	// Build URL with client ID timestamp (like Python V1)
	cid := time.Now().Unix()
	apiURL := fmt.Sprintf("%s?method=%s&input=3&api_version=1.0&api_token=%s&cid=%d", deezerPrivateAPI, method, apiToken, cid)

	// Build request body
	var body io.Reader
	if params != nil {
		jsonData, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		body = strings.NewReader(string(jsonData))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Cookie", "arl="+arl)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for errors
	if errData, ok := result["error"].(map[string]interface{}); ok && errData != nil {
		if code, ok := errData["code"].(float64); ok && code != 0 {
			return nil, fmt.Errorf("API error: %v", errData)
		}
	}

	return result, nil
}

// doPublicAPIRequest performs a request to Deezer's public API with retry on quota errors
func (c *DeezerClient) doPublicAPIRequest(ctx context.Context, endpoint string, params url.Values) (map[string]interface{}, error) {
	maxRetries := 3
	baseDelay := 2 * time.Second
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		apiURL := deezerAPIURL + endpoint
		if len(params) > 0 {
			apiURL += "?" + params.Encode()
		}

		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

		resp, err := c.doRequest(ctx, req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		// Check for errors
		if errData, ok := result["error"].(map[string]interface{}); ok && errData != nil {
			// Check if it's a quota limit error (code 4)
			if code, ok := errData["code"].(float64); ok && code == 4 {
				if attempt < maxRetries {
					// Exponential backoff: 2s, 4s, 8s
					delay := baseDelay * time.Duration(1<<uint(attempt))
					fmt.Printf("Quota limit exceeded, retrying in %v (attempt %d/%d)\n", delay, attempt+1, maxRetries)
					
					select {
					case <-time.After(delay):
						continue // Retry
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}
			}
			return nil, fmt.Errorf("API error: %v", errData)
		}

		return result, nil
	}
	
	return nil, fmt.Errorf("max retries exceeded for quota limit")
}
