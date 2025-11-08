package download

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/deemusic/deemusic-go/internal/api"
	"github.com/deemusic/deemusic-go/internal/config"
	"github.com/deemusic/deemusic-go/internal/store"
)

// ExampleBasicSetup demonstrates basic setup of the download manager
func ExampleBasicSetup() {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		log.Fatal(err)
	}

	// Initialize database
	db, err := store.InitDB(store.GetDefaultDBPath())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create queue store
	queueStore := store.NewQueueStore(db)

	// Create Deezer API client
	deezerAPI := api.NewDeezerClient(30 * time.Second)
	err = deezerAPI.Authenticate(context.Background(), cfg.Deezer.ARL)
	if err != nil {
		log.Fatal(err)
	}

	// Create progress notifier
	notifier := NewProgressNotifier()
	notifier.Start()

	// Create download manager
	manager := NewManager(cfg, queueStore, deezerAPI, notifier)

	// Start the manager
	ctx := context.Background()
	err = manager.Start(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer manager.Stop()

	fmt.Println("Download manager started successfully")
}

// ExampleDownloadTrack demonstrates downloading a single track
func ExampleDownloadTrack() {
	// Assume manager is already set up
	var manager *Manager
	ctx := context.Background()

	// Download a track
	trackID := "123456789"
	err := manager.DownloadTrack(ctx, trackID)
	if err != nil {
		log.Printf("Failed to queue track: %v", err)
		return
	}

	fmt.Printf("Track %s queued for download\n", trackID)
}

// ExampleDownloadAlbum demonstrates downloading an entire album
func ExampleDownloadAlbum() {
	var manager *Manager
	ctx := context.Background()

	// Download an album
	albumID := "987654321"
	err := manager.DownloadAlbum(ctx, albumID)
	if err != nil {
		log.Printf("Failed to queue album: %v", err)
		return
	}

	fmt.Printf("Album %s queued for download\n", albumID)
}

// ExampleDownloadPlaylist demonstrates downloading a playlist
func ExampleDownloadPlaylist() {
	var manager *Manager
	ctx := context.Background()

	// Download a playlist
	playlistID := "555555555"
	err := manager.DownloadPlaylist(ctx, playlistID)
	if err != nil {
		log.Printf("Failed to queue playlist: %v", err)
		return
	}

	fmt.Printf("Playlist %s queued for download\n", playlistID)
}

// ExampleQueueManagement demonstrates pause, resume, and cancel operations
func ExampleQueueManagement() {
	var manager *Manager

	itemID := "track_123456789"

	// Pause a download
	err := manager.PauseDownload(itemID)
	if err != nil {
		log.Printf("Failed to pause: %v", err)
	} else {
		fmt.Printf("Download %s paused\n", itemID)
	}

	// Wait a bit
	time.Sleep(5 * time.Second)

	// Resume the download
	err = manager.ResumeDownload(itemID)
	if err != nil {
		log.Printf("Failed to resume: %v", err)
	} else {
		fmt.Printf("Download %s resumed\n", itemID)
	}

	// Cancel a download
	err = manager.CancelDownload(itemID)
	if err != nil {
		log.Printf("Failed to cancel: %v", err)
	} else {
		fmt.Printf("Download %s cancelled\n", itemID)
	}
}

// ExampleStatistics demonstrates getting download statistics
func ExampleStatistics() {
	var manager *Manager
	var notifier *ProgressNotifier

	// Get manager statistics
	stats, err := manager.GetStats()
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
		return
	}

	fmt.Printf("Queue Statistics:\n")
	fmt.Printf("  Total: %d\n", stats["queue_total"])
	fmt.Printf("  Pending: %d\n", stats["queue_pending"])
	fmt.Printf("  Downloading: %d\n", stats["queue_downloading"])
	fmt.Printf("  Completed: %d\n", stats["queue_completed"])
	fmt.Printf("  Failed: %d\n", stats["queue_failed"])
	fmt.Printf("  Active Downloads: %d\n", stats["active_downloads"])
	fmt.Printf("  Max Workers: %d\n", stats["max_workers"])

	// Get notifier statistics
	notifierStats := notifier.GetStats()
	fmt.Printf("\nDownload Statistics:\n")
	fmt.Printf("  Total Downloads: %d\n", notifierStats["total_downloads"])
	fmt.Printf("  Success Count: %d\n", notifierStats["success_count"])
	fmt.Printf("  Failure Count: %d\n", notifierStats["failure_count"])
	fmt.Printf("  Success Rate: %.1f%%\n", notifierStats["success_rate"])
}

// ExampleProgressTracking demonstrates tracking download progress
func ExampleProgressTracking() {
	var notifier *ProgressNotifier

	// Get all active download stats
	allStats := notifier.GetAllDownloadStats()

	fmt.Printf("Active Downloads: %d\n\n", len(allStats))

	for _, stats := range allStats {
		fmt.Printf("Item: %s\n", stats.ItemID)
		fmt.Printf("  Progress: %d/%d bytes\n", stats.BytesProcessed, stats.TotalBytes)
		fmt.Printf("  Speed: %s\n", FormatSpeed(stats.Speed))
		fmt.Printf("  ETA: %s\n", FormatETA(stats.ETA))
		fmt.Printf("  Elapsed: %s\n", time.Since(stats.StartTime).Round(time.Second))
		fmt.Println()
	}
}

// ExampleWebSocketClient demonstrates WebSocket client integration
func ExampleWebSocketClient() {
	var notifier *ProgressNotifier

	// Create a new client
	client := NewClient("client-123")

	// Register the client
	notifier.Register(client)
	defer notifier.Unregister(client)

	// Listen for messages
	go func() {
		for data := range client.SendChan {
			// In a real application, you would send this to a WebSocket connection
			fmt.Printf("Received message: %s\n", string(data))
		}
	}()

	// Keep the example running for a bit
	time.Sleep(10 * time.Second)
}

// ExampleWorkerPool demonstrates direct worker pool usage
func ExampleWorkerPool() {
	// Create a job handler
	handler := func(ctx context.Context, job *Job) error {
		fmt.Printf("Processing job: %s (type: %s)\n", job.ID, job.Type)

		// Simulate work
		select {
		case <-time.After(2 * time.Second):
			fmt.Printf("Job %s completed\n", job.ID)
			return nil
		case <-ctx.Done():
			fmt.Printf("Job %s cancelled\n", job.ID)
			return ctx.Err()
		}
	}

	// Create worker pool
	pool := NewWorkerPool(4, handler)

	// Start the pool
	ctx := context.Background()
	err := pool.Start(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Stop()

	// Submit some jobs
	for i := 0; i < 10; i++ {
		job := &Job{
			ID:   fmt.Sprintf("job-%d", i),
			Type: JobTypeTrack,
		}

		err := pool.Submit(job)
		if err != nil {
			log.Printf("Failed to submit job: %v", err)
		}
	}

	// Process results
	go func() {
		for result := range pool.Results() {
			if result.Success {
				fmt.Printf("Job %s succeeded\n", result.JobID)
			} else {
				fmt.Printf("Job %s failed: %v\n", result.JobID, result.Error)
			}
		}
	}()

	// Wait for jobs to complete
	time.Sleep(30 * time.Second)
}

// ExampleCustomNotifications demonstrates custom notification messages
func ExampleCustomNotifications() {
	var notifier *ProgressNotifier

	// Broadcast a custom message
	notifier.BroadcastCustomMessage("system", map[string]interface{}{
		"message": "System maintenance scheduled",
		"time":    time.Now().Add(1 * time.Hour),
	})

	// Broadcast queue statistics
	stats := notifier.GetStats()
	notifier.BroadcastCustomMessage("stats", stats)
}

// ExampleErrorHandling demonstrates error handling and retry logic
func ExampleErrorHandling() {
	var manager *Manager
	ctx := context.Background()

	// Try to download a track
	trackID := "123456789"
	err := manager.DownloadTrack(ctx, trackID)
	if err != nil {
		log.Printf("Failed to queue track: %v", err)

		// Check if it's an authentication error
		if err.Error() == "authentication required or token expired" {
			// Try to refresh the token
			// In a real application, you would call deezerAPI.RefreshToken()
			fmt.Println("Attempting to refresh authentication token...")
		}

		return
	}

	// The download manager will automatically retry failed downloads
	// up to the configured max retries with exponential backoff
	fmt.Printf("Track %s queued (will auto-retry on failure)\n", trackID)
}

// ExampleConcurrentDownloads demonstrates managing concurrent downloads
func ExampleConcurrentDownloads() {
	var manager *Manager
	ctx := context.Background()

	// Queue multiple tracks
	trackIDs := []string{"111", "222", "333", "444", "555"}

	for _, trackID := range trackIDs {
		err := manager.DownloadTrack(ctx, trackID)
		if err != nil {
			log.Printf("Failed to queue track %s: %v", trackID, err)
			continue
		}
		fmt.Printf("Queued track %s\n", trackID)
	}

	// Monitor progress
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for i := 0; i < 10; i++ {
		<-ticker.C

		stats, err := manager.GetStats()
		if err != nil {
			continue
		}

		fmt.Printf("Active: %d, Pending: %d, Completed: %d\n",
			stats["active_downloads"],
			stats["queue_pending"],
			stats["queue_completed"])

		// Stop if all downloads are complete
		if stats["queue_pending"].(int) == 0 && stats["active_downloads"].(int) == 0 {
			break
		}
	}
}
