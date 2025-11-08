package store

// Example usage of the store package
//
// This file demonstrates how to use the database and queue store components.
// It is not meant to be executed, but serves as documentation.

/*
Example 1: Initialize Database and Queue Store

	import (
		"github.com/deemusic/deemusic-go/internal/store"
	)

	func main() {
		// Initialize database with default path
		dbPath := store.GetDefaultDBPath()
		db, err := store.InitDB(dbPath)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		// Create queue store
		queueStore := store.NewQueueStore(db)

		// Use the queue store...
	}

Example 2: Add Items to Queue

	// Create a new queue item
	item := &store.QueueItem{
		ID:      "track-12345",
		Type:    "track",
		Title:   "My Favorite Song",
		Artist:  "Great Artist",
		Album:   "Amazing Album",
		Status:  "pending",
		Progress: 0,
	}

	// Add to queue
	if err := queueStore.Add(item); err != nil {
		log.Printf("Failed to add item: %v", err)
	}

Example 3: Update Queue Item Progress

	// Get item from queue
	item, err := queueStore.GetByID("track-12345")
	if err != nil {
		log.Printf("Failed to get item: %v", err)
		return
	}

	// Update progress
	item.Status = "downloading"
	item.Progress = 50

	if err := queueStore.Update(item); err != nil {
		log.Printf("Failed to update item: %v", err)
	}

Example 4: Get Pending Items

	// Get up to 10 pending items
	pending, err := queueStore.GetPending(10)
	if err != nil {
		log.Printf("Failed to get pending items: %v", err)
		return
	}

	for _, item := range pending {
		fmt.Printf("Pending: %s - %s\n", item.Artist, item.Title)
	}

Example 5: Get Queue Statistics

	stats, err := queueStore.GetStats()
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
		return
	}

	fmt.Printf("Total: %d, Pending: %d, Downloading: %d, Completed: %d, Failed: %d\n",
		stats.Total, stats.Pending, stats.Downloading, stats.Completed, stats.Failed)

Example 6: Clear Completed Downloads

	if err := queueStore.ClearCompleted(); err != nil {
		log.Printf("Failed to clear completed: %v", err)
	}

Example 7: Add to Download History

	err := queueStore.AddToHistory(
		"track-12345",
		"My Favorite Song",
		"Great Artist",
		"Amazing Album",
		"/path/to/file.mp3",
		"MP3_320",
		5242880, // 5MB
	)
	if err != nil {
		log.Printf("Failed to add to history: %v", err)
	}

Example 8: Use Config Cache

	// Set a cache value
	if err := queueStore.SetConfigCache("last_sync", "2024-01-01T00:00:00Z"); err != nil {
		log.Printf("Failed to set cache: %v", err)
	}

	// Get a cache value
	value, err := queueStore.GetConfigCache("last_sync")
	if err != nil {
		log.Printf("Failed to get cache: %v", err)
	}
	fmt.Printf("Last sync: %s\n", value)

Example 9: Store Metadata as JSON

	type TrackMetadata struct {
		ISRC        string
		ReleaseDate string
		Genre       string
	}

	metadata := TrackMetadata{
		ISRC:        "USRC12345678",
		ReleaseDate: "2024-01-01",
		Genre:       "Pop",
	}

	item := &store.QueueItem{
		ID:    "track-12345",
		Type:  "track",
		Title: "My Song",
	}

	// Set metadata
	if err := item.SetMetadata(metadata); err != nil {
		log.Printf("Failed to set metadata: %v", err)
	}

	// Later, retrieve metadata
	var retrievedMetadata TrackMetadata
	if err := item.GetMetadata(&retrievedMetadata); err != nil {
		log.Printf("Failed to get metadata: %v", err)
	}
*/
