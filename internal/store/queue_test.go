package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) (*QueueStore, func()) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	store := NewQueueStore(db)

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

func TestQueueStore_AddAndGet(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	item := &QueueItem{
		ID:      "test-123",
		Type:    "track",
		Title:   "Test Track",
		Artist:  "Test Artist",
		Album:   "Test Album",
		Status:  "pending",
		Progress: 0,
	}

	// Test Add
	err := store.Add(item)
	if err != nil {
		t.Fatalf("Failed to add item: %v", err)
	}

	// Test GetByID
	retrieved, err := store.GetByID("test-123")
	if err != nil {
		t.Fatalf("Failed to get item: %v", err)
	}

	if retrieved.Title != item.Title {
		t.Errorf("Expected title %s, got %s", item.Title, retrieved.Title)
	}
	if retrieved.Artist != item.Artist {
		t.Errorf("Expected artist %s, got %s", item.Artist, retrieved.Artist)
	}
}

func TestQueueStore_Update(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	item := &QueueItem{
		ID:     "test-456",
		Type:   "track",
		Title:  "Test Track",
		Status: "pending",
	}

	err := store.Add(item)
	if err != nil {
		t.Fatalf("Failed to add item: %v", err)
	}

	// Update item
	item.Status = "downloading"
	item.Progress = 50

	err = store.Update(item)
	if err != nil {
		t.Fatalf("Failed to update item: %v", err)
	}

	// Verify update
	retrieved, err := store.GetByID("test-456")
	if err != nil {
		t.Fatalf("Failed to get item: %v", err)
	}

	if retrieved.Status != "downloading" {
		t.Errorf("Expected status downloading, got %s", retrieved.Status)
	}
	if retrieved.Progress != 50 {
		t.Errorf("Expected progress 50, got %d", retrieved.Progress)
	}
}

func TestQueueStore_Delete(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	item := &QueueItem{
		ID:     "test-789",
		Type:   "track",
		Title:  "Test Track",
		Status: "pending",
	}

	err := store.Add(item)
	if err != nil {
		t.Fatalf("Failed to add item: %v", err)
	}

	// Delete item
	err = store.Delete("test-789")
	if err != nil {
		t.Fatalf("Failed to delete item: %v", err)
	}

	// Verify deletion
	_, err = store.GetByID("test-789")
	if err == nil {
		t.Error("Expected error when getting deleted item")
	}
}

func TestQueueStore_GetPending(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Add multiple items
	items := []*QueueItem{
		{ID: "1", Type: "track", Title: "Track 1", Status: "pending"},
		{ID: "2", Type: "track", Title: "Track 2", Status: "pending"},
		{ID: "3", Type: "track", Title: "Track 3", Status: "completed"},
	}

	for _, item := range items {
		if err := store.Add(item); err != nil {
			t.Fatalf("Failed to add item: %v", err)
		}
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	// Get pending items
	pending, err := store.GetPending(10)
	if err != nil {
		t.Fatalf("Failed to get pending items: %v", err)
	}

	if len(pending) != 2 {
		t.Errorf("Expected 2 pending items, got %d", len(pending))
	}
}

func TestQueueStore_GetStats(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Add items with different statuses
	items := []*QueueItem{
		{ID: "1", Type: "track", Title: "Track 1", Status: "pending"},
		{ID: "2", Type: "track", Title: "Track 2", Status: "pending"},
		{ID: "3", Type: "track", Title: "Track 3", Status: "downloading"},
		{ID: "4", Type: "track", Title: "Track 4", Status: "completed"},
		{ID: "5", Type: "track", Title: "Track 5", Status: "failed"},
	}

	for _, item := range items {
		if err := store.Add(item); err != nil {
			t.Fatalf("Failed to add item: %v", err)
		}
	}

	// Get stats
	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Total != 5 {
		t.Errorf("Expected total 5, got %d", stats.Total)
	}
	if stats.Pending != 2 {
		t.Errorf("Expected pending 2, got %d", stats.Pending)
	}
	if stats.Downloading != 1 {
		t.Errorf("Expected downloading 1, got %d", stats.Downloading)
	}
	if stats.Completed != 1 {
		t.Errorf("Expected completed 1, got %d", stats.Completed)
	}
	if stats.Failed != 1 {
		t.Errorf("Expected failed 1, got %d", stats.Failed)
	}
}

func TestQueueStore_ClearCompleted(t *testing.T) {
	store, cleanup := setupTestDB(t)
	defer cleanup()

	// Add items
	items := []*QueueItem{
		{ID: "1", Type: "track", Title: "Track 1", Status: "completed"},
		{ID: "2", Type: "track", Title: "Track 2", Status: "completed"},
		{ID: "3", Type: "track", Title: "Track 3", Status: "pending"},
	}

	for _, item := range items {
		if err := store.Add(item); err != nil {
			t.Fatalf("Failed to add item: %v", err)
		}
	}

	// Clear completed
	err := store.ClearCompleted()
	if err != nil {
		t.Fatalf("Failed to clear completed: %v", err)
	}

	// Verify
	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Total != 1 {
		t.Errorf("Expected total 1, got %d", stats.Total)
	}
	if stats.Completed != 0 {
		t.Errorf("Expected completed 0, got %d", stats.Completed)
	}
}
