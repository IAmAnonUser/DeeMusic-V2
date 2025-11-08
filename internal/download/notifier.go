package download

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ProgressUpdate represents a progress update message
type ProgressUpdate struct {
	ItemID         string    `json:"item_id"`
	Progress       int       `json:"progress"`
	BytesProcessed int64     `json:"bytes_processed"`
	TotalBytes     int64     `json:"total_bytes"`
	Speed          float64   `json:"speed"` // bytes per second
	ETA            int       `json:"eta"`   // seconds remaining
	Timestamp      time.Time `json:"timestamp"`
}

// StatusUpdate represents a status change message
type StatusUpdate struct {
	ItemID    string    `json:"item_id"`
	Status    string    `json:"status"` // started, completed, failed
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Message represents a notification message
type Message struct {
	Type    string      `json:"type"` // progress, status
	Payload interface{} `json:"payload"`
}

// Client represents a connected WebSocket client
type Client struct {
	ID       string
	SendChan chan []byte
	mu       sync.Mutex
}

// NewClient creates a new client
func NewClient(id string) *Client {
	return &Client{
		ID:       id,
		SendChan: make(chan []byte, 256),
	}
}

// Send sends a message to the client
func (c *Client) Send(data []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case c.SendChan <- data:
		return true
	default:
		// Channel full, drop message
		return false
	}
}

// Close closes the client's send channel
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	close(c.SendChan)
}

// ProgressNotifier handles progress tracking and WebSocket broadcasting
type ProgressNotifier struct {
	clients       map[string]*Client
	broadcast     chan *Message
	register      chan *Client
	unregister    chan *Client
	mu            sync.RWMutex
	stats         map[string]*DownloadStats
	statsMu       sync.RWMutex
	successCount  int
	failureCount  int
	totalDownloads int
}

// DownloadStats tracks statistics for a download
type DownloadStats struct {
	ItemID         string
	StartTime      time.Time
	LastUpdate     time.Time
	BytesProcessed int64
	TotalBytes     int64
	Speed          float64 // bytes per second
	ETA            int     // seconds remaining
}

// NewProgressNotifier creates a new progress notifier
func NewProgressNotifier() *ProgressNotifier {
	return &ProgressNotifier{
		clients:    make(map[string]*Client),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stats:      make(map[string]*DownloadStats),
	}
}

// Start starts the notifier
func (pn *ProgressNotifier) Start() {
	go pn.run()
}

// run is the main event loop for the notifier
func (pn *ProgressNotifier) run() {
	for {
		select {
		case client := <-pn.register:
			pn.registerClient(client)

		case client := <-pn.unregister:
			pn.unregisterClient(client)

		case message := <-pn.broadcast:
			pn.broadcastMessage(message)
		}
	}
}

// registerClient registers a new client
func (pn *ProgressNotifier) registerClient(client *Client) {
	pn.mu.Lock()
	pn.clients[client.ID] = client
	pn.mu.Unlock()
}

// unregisterClient unregisters a client
func (pn *ProgressNotifier) unregisterClient(client *Client) {
	pn.mu.Lock()
	if _, ok := pn.clients[client.ID]; ok {
		delete(pn.clients, client.ID)
		client.Close()
	}
	pn.mu.Unlock()
}

// broadcastMessage broadcasts a message to all clients
func (pn *ProgressNotifier) broadcastMessage(message *Message) {
	// Serialize message
	data, err := json.Marshal(message)
	if err != nil {
		return
	}

	// Send to all clients
	pn.mu.RLock()
	for _, client := range pn.clients {
		client.Send(data)
	}
	pn.mu.RUnlock()
}

// Register registers a new client
func (pn *ProgressNotifier) Register(client *Client) {
	pn.register <- client
}

// Unregister unregisters a client
func (pn *ProgressNotifier) Unregister(client *Client) {
	pn.unregister <- client
}

// NotifyProgress notifies progress for a download
func (pn *ProgressNotifier) NotifyProgress(itemID string, progress int, bytesProcessed, totalBytes int64) {
	now := time.Now()

	// Update stats
	pn.statsMu.Lock()
	stats, exists := pn.stats[itemID]
	if !exists {
		stats = &DownloadStats{
			ItemID:    itemID,
			StartTime: now,
		}
		pn.stats[itemID] = stats
	}

	// Calculate speed and ETA
	elapsed := now.Sub(stats.LastUpdate).Seconds()
	if elapsed > 0 && stats.LastUpdate.After(stats.StartTime) {
		bytesDelta := bytesProcessed - stats.BytesProcessed
		stats.Speed = float64(bytesDelta) / elapsed
	}

	stats.BytesProcessed = bytesProcessed
	stats.TotalBytes = totalBytes
	stats.LastUpdate = now

	// Calculate ETA
	if stats.Speed > 0 && totalBytes > 0 {
		remaining := totalBytes - bytesProcessed
		stats.ETA = int(float64(remaining) / stats.Speed)
	}

	speed := stats.Speed
	eta := stats.ETA
	pn.statsMu.Unlock()

	// Create progress update
	update := &ProgressUpdate{
		ItemID:         itemID,
		Progress:       progress,
		BytesProcessed: bytesProcessed,
		TotalBytes:     totalBytes,
		Speed:          speed,
		ETA:            eta,
		Timestamp:      now,
	}

	// Broadcast
	message := &Message{
		Type:    "progress",
		Payload: update,
	}

	select {
	case pn.broadcast <- message:
	default:
		// Broadcast channel full, drop message
	}
}

// NotifyStarted notifies that a download has started
func (pn *ProgressNotifier) NotifyStarted(itemID string) {
	now := time.Now()

	// Initialize stats
	pn.statsMu.Lock()
	pn.stats[itemID] = &DownloadStats{
		ItemID:    itemID,
		StartTime: now,
		LastUpdate: now,
	}
	pn.totalDownloads++
	pn.statsMu.Unlock()

	// Create status update
	update := &StatusUpdate{
		ItemID:    itemID,
		Status:    "started",
		Timestamp: now,
	}

	// Broadcast
	message := &Message{
		Type:    "status",
		Payload: update,
	}

	select {
	case pn.broadcast <- message:
	default:
	}
}

// NotifyCompleted notifies that a download has completed
func (pn *ProgressNotifier) NotifyCompleted(itemID string) {
	now := time.Now()

	// Update stats
	pn.statsMu.Lock()
	delete(pn.stats, itemID)
	pn.successCount++
	pn.statsMu.Unlock()

	// Create status update
	update := &StatusUpdate{
		ItemID:    itemID,
		Status:    "completed",
		Timestamp: now,
	}

	// Broadcast
	message := &Message{
		Type:    "status",
		Payload: update,
	}

	select {
	case pn.broadcast <- message:
	default:
	}
}

// NotifyFailed notifies that a download has failed
func (pn *ProgressNotifier) NotifyFailed(itemID string, err error) {
	now := time.Now()

	// Update stats
	pn.statsMu.Lock()
	delete(pn.stats, itemID)
	pn.failureCount++
	pn.statsMu.Unlock()

	// Create status update
	update := &StatusUpdate{
		ItemID:    itemID,
		Status:    "failed",
		Error:     err.Error(),
		Timestamp: now,
	}

	// Broadcast
	message := &Message{
		Type:    "status",
		Payload: update,
	}

	select {
	case pn.broadcast <- message:
	default:
	}
}

// GetStats returns overall download statistics
func (pn *ProgressNotifier) GetStats() map[string]interface{} {
	pn.statsMu.RLock()
	defer pn.statsMu.RUnlock()

	activeDownloads := len(pn.stats)
	successRate := 0.0
	if pn.totalDownloads > 0 {
		successRate = float64(pn.successCount) / float64(pn.totalDownloads) * 100
	}

	return map[string]interface{}{
		"active_downloads": activeDownloads,
		"total_downloads":  pn.totalDownloads,
		"success_count":    pn.successCount,
		"failure_count":    pn.failureCount,
		"success_rate":     successRate,
	}
}

// GetDownloadStats returns statistics for a specific download
func (pn *ProgressNotifier) GetDownloadStats(itemID string) *DownloadStats {
	pn.statsMu.RLock()
	defer pn.statsMu.RUnlock()

	if stats, ok := pn.stats[itemID]; ok {
		// Return a copy
		return &DownloadStats{
			ItemID:         stats.ItemID,
			StartTime:      stats.StartTime,
			LastUpdate:     stats.LastUpdate,
			BytesProcessed: stats.BytesProcessed,
			TotalBytes:     stats.TotalBytes,
			Speed:          stats.Speed,
			ETA:            stats.ETA,
		}
	}

	return nil
}

// GetAllDownloadStats returns statistics for all active downloads
func (pn *ProgressNotifier) GetAllDownloadStats() []*DownloadStats {
	pn.statsMu.RLock()
	defer pn.statsMu.RUnlock()

	stats := make([]*DownloadStats, 0, len(pn.stats))
	for _, s := range pn.stats {
		stats = append(stats, &DownloadStats{
			ItemID:         s.ItemID,
			StartTime:      s.StartTime,
			LastUpdate:     s.LastUpdate,
			BytesProcessed: s.BytesProcessed,
			TotalBytes:     s.TotalBytes,
			Speed:          s.Speed,
			ETA:            s.ETA,
		})
	}

	return stats
}

// GetClientCount returns the number of connected clients
func (pn *ProgressNotifier) GetClientCount() int {
	pn.mu.RLock()
	defer pn.mu.RUnlock()
	return len(pn.clients)
}

// BroadcastCustomMessage broadcasts a custom message to all clients
func (pn *ProgressNotifier) BroadcastCustomMessage(messageType string, payload interface{}) {
	message := &Message{
		Type:    messageType,
		Payload: payload,
	}

	select {
	case pn.broadcast <- message:
	default:
	}
}

// ResetStats resets all statistics
func (pn *ProgressNotifier) ResetStats() {
	pn.statsMu.Lock()
	defer pn.statsMu.Unlock()

	pn.stats = make(map[string]*DownloadStats)
	pn.successCount = 0
	pn.failureCount = 0
	pn.totalDownloads = 0
}

// FormatSpeed formats speed in human-readable format
func FormatSpeed(bytesPerSecond float64) string {
	if bytesPerSecond < 1024 {
		return "< 1 KB/s"
	} else if bytesPerSecond < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", bytesPerSecond/1024)
	} else {
		return fmt.Sprintf("%.1f MB/s", bytesPerSecond/(1024*1024))
	}
}

// FormatETA formats ETA in human-readable format
func FormatETA(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	} else if seconds < 3600 {
		minutes := seconds / 60
		secs := seconds % 60
		return fmt.Sprintf("%dm %ds", minutes, secs)
	} else {
		hours := seconds / 3600
		minutes := (seconds % 3600) / 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}

// CallbackNotifier implements the Notifier interface using direct callbacks
// This is used for the C# WPF frontend integration via P/Invoke
type CallbackNotifier struct {
	progressCallback func(itemID string, progress int, speed string, eta string)
	statusCallback   func(itemID string, status string, errorMsg string)
	mu               sync.RWMutex
	stats            map[string]*DownloadStats
	statsMu          sync.RWMutex
}

// NewCallbackNotifier creates a new callback-based notifier
func NewCallbackNotifier() *CallbackNotifier {
	return &CallbackNotifier{
		stats: make(map[string]*DownloadStats),
	}
}

// SetProgressCallback sets the callback function for progress updates
func (cn *CallbackNotifier) SetProgressCallback(callback func(itemID string, progress int, speed string, eta string)) {
	cn.mu.Lock()
	defer cn.mu.Unlock()
	cn.progressCallback = callback
}

// SetStatusCallback sets the callback function for status updates
func (cn *CallbackNotifier) SetStatusCallback(callback func(itemID string, status string, errorMsg string)) {
	cn.mu.Lock()
	defer cn.mu.Unlock()
	cn.statusCallback = callback
}

// NotifyProgress notifies progress for a download via callback
func (cn *CallbackNotifier) NotifyProgress(itemID string, progress int, bytesProcessed, totalBytes int64) {
	now := time.Now()

	// Update stats
	cn.statsMu.Lock()
	stats, exists := cn.stats[itemID]
	if !exists {
		stats = &DownloadStats{
			ItemID:    itemID,
			StartTime: now,
		}
		cn.stats[itemID] = stats
	}

	// Calculate speed and ETA
	elapsed := now.Sub(stats.LastUpdate).Seconds()
	if elapsed > 0 && stats.LastUpdate.After(stats.StartTime) {
		bytesDelta := bytesProcessed - stats.BytesProcessed
		stats.Speed = float64(bytesDelta) / elapsed
	}

	stats.BytesProcessed = bytesProcessed
	stats.TotalBytes = totalBytes
	stats.LastUpdate = now

	// Calculate ETA
	if stats.Speed > 0 && totalBytes > 0 {
		remaining := totalBytes - bytesProcessed
		stats.ETA = int(float64(remaining) / stats.Speed)
	}

	speed := FormatSpeed(stats.Speed)
	eta := FormatETA(stats.ETA)
	cn.statsMu.Unlock()

	// Invoke callback if set
	cn.mu.RLock()
	callback := cn.progressCallback
	cn.mu.RUnlock()

	if callback != nil {
		// Call in a goroutine to avoid blocking the download
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// Callback panicked, log but don't crash
					fmt.Printf("Progress callback panicked: %v\n", r)
				}
			}()
			callback(itemID, progress, speed, eta)
		}()
	}
}

// NotifyStarted notifies that a download has started via callback
func (cn *CallbackNotifier) NotifyStarted(itemID string) {
	now := time.Now()

	// Initialize stats
	cn.statsMu.Lock()
	cn.stats[itemID] = &DownloadStats{
		ItemID:     itemID,
		StartTime:  now,
		LastUpdate: now,
	}
	cn.statsMu.Unlock()

	// Invoke callback if set
	cn.mu.RLock()
	callback := cn.statusCallback
	cn.mu.RUnlock()

	if callback != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Status callback panicked: %v\n", r)
				}
			}()
			callback(itemID, "started", "")
		}()
	}
}

// NotifyCompleted notifies that a download has completed via callback
func (cn *CallbackNotifier) NotifyCompleted(itemID string) {
	// Clean up stats
	cn.statsMu.Lock()
	delete(cn.stats, itemID)
	cn.statsMu.Unlock()

	// Invoke callback if set
	cn.mu.RLock()
	callback := cn.statusCallback
	cn.mu.RUnlock()

	if callback != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Status callback panicked: %v\n", r)
				}
			}()
			callback(itemID, "completed", "")
		}()
	}
}

// NotifyFailed notifies that a download has failed via callback
func (cn *CallbackNotifier) NotifyFailed(itemID string, err error) {
	// Clean up stats
	cn.statsMu.Lock()
	delete(cn.stats, itemID)
	cn.statsMu.Unlock()

	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	// Invoke callback if set
	cn.mu.RLock()
	callback := cn.statusCallback
	cn.mu.RUnlock()

	if callback != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Status callback panicked: %v\n", r)
				}
			}()
			callback(itemID, "failed", errorMsg)
		}()
	}
}
