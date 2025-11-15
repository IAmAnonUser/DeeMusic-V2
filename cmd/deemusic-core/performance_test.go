package main

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/deemusic/deemusic-go/internal/store"
	_ "github.com/mattn/go-sqlite3"
)

// TestLargeQueuePerformance tests queue operations with 10,000+ items
func TestLargeQueuePerformance(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_queue.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := store.RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	queueStore := store.NewQueueStore(db)

	// Test 1: Insert 10,000 items
	t.Run("Insert10000Items", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 10000; i++ {
			item := &store.QueueItem{
				ID:       fmt.Sprintf("track_%d", i),
				Type:     "track",
				Title:    fmt.Sprintf("Test Track %d", i),
				Artist:   fmt.Sprintf("Test Artist %d", i%100),
				Album:    fmt.Sprintf("Test Album %d", i%50),
				Status:   getStatus(i),
				Progress: i % 101,
			}

			if err := queueStore.Add(item); err != nil {
				t.Fatalf("Failed to add item %d: %v", i, err)
			}
		}

		elapsed := time.Since(start)
		t.Logf("Inserted 10,000 items in %v (%.2f items/sec)", elapsed, 10000.0/elapsed.Seconds())

		// Should complete in reasonable time (< 5 seconds)
		if elapsed > 5*time.Second {
			t.Errorf("Insert took too long: %v", elapsed)
		}
	})

	// Test 2: Query with pagination (should be fast with indexes)
	t.Run("PaginatedQuery", func(t *testing.T) {
		start := time.Now()

		// Query first page
		items, err := queueStore.GetAll(0, 100)
		if err != nil {
			t.Fatalf("Failed to get first page: %v", err)
		}

		if len(items) != 100 {
			t.Errorf("Expected 100 items, got %d", len(items))
		}

		elapsed := time.Since(start)
		t.Logf("Queried first page (100 items) in %v", elapsed)

		// Should be very fast (< 50ms)
		if elapsed > 50*time.Millisecond {
			t.Errorf("Query took too long: %v", elapsed)
		}
	})

	// Test 3: Query middle page (tests offset performance)
	t.Run("MiddlePageQuery", func(t *testing.T) {
		start := time.Now()

		// Query page in the middle
		items, err := queueStore.GetAll(5000, 100)
		if err != nil {
			t.Fatalf("Failed to get middle page: %v", err)
		}

		if len(items) != 100 {
			t.Errorf("Expected 100 items, got %d", len(items))
		}

		elapsed := time.Since(start)
		t.Logf("Queried middle page (offset 5000) in %v", elapsed)

		// Should still be fast (< 100ms)
		if elapsed > 100*time.Millisecond {
			t.Errorf("Query took too long: %v", elapsed)
		}
	})

	// Test 4: Filter by status (tests index usage)
	t.Run("FilterByStatus", func(t *testing.T) {
		start := time.Now()

		items, err := queueStore.GetByStatus("pending", 0, 100)
		if err != nil {
			t.Fatalf("Failed to filter by status: %v", err)
		}

		elapsed := time.Since(start)
		t.Logf("Filtered by status (100 items) in %v", elapsed)

		// Should be fast with index (< 50ms)
		if elapsed > 50*time.Millisecond {
			t.Errorf("Filtered query took too long: %v", elapsed)
		}

		// Verify all items have correct status
		for _, item := range items {
			if item.Status != "pending" {
				t.Errorf("Expected status 'pending', got '%s'", item.Status)
			}
		}
	})

	// Test 5: Get statistics (should be fast with aggregation)
	t.Run("GetStatistics", func(t *testing.T) {
		start := time.Now()

		stats, err := queueStore.GetStats()
		if err != nil {
			t.Fatalf("Failed to get stats: %v", err)
		}

		elapsed := time.Since(start)
		t.Logf("Got statistics in %v", elapsed)

		// Should be fast (< 100ms)
		if elapsed > 100*time.Millisecond {
			t.Errorf("Stats query took too long: %v", elapsed)
		}

		// Verify stats
		if stats.Total != 10000 {
			t.Errorf("Expected total 10000, got %d", stats.Total)
		}

		t.Logf("Stats: Total=%d, Pending=%d, Downloading=%d, Completed=%d, Failed=%d",
			stats.Total, stats.Pending, stats.Downloading, stats.Completed, stats.Failed)
	})

	// Test 6: Get count (should use index)
	t.Run("GetCount", func(t *testing.T) {
		start := time.Now()

		count, err := queueStore.GetCount()
		if err != nil {
			t.Fatalf("Failed to get count: %v", err)
		}

		elapsed := time.Since(start)
		t.Logf("Got count in %v", elapsed)

		// Should be very fast (< 10ms)
		if elapsed > 10*time.Millisecond {
			t.Errorf("Count query took too long: %v", elapsed)
		}

		if count != 10000 {
			t.Errorf("Expected count 10000, got %d", count)
		}
	})

	// Test 7: Update items (tests update performance)
	t.Run("UpdateItems", func(t *testing.T) {
		start := time.Now()

		// Update 100 items
		for i := 0; i < 100; i++ {
			item, err := queueStore.GetByID(fmt.Sprintf("track_%d", i))
			if err != nil {
				t.Fatalf("Failed to get item: %v", err)
			}

			item.Progress = 100
			item.Status = "completed"
			now := time.Now()
			item.CompletedAt = &now

			if err := queueStore.Update(item); err != nil {
				t.Fatalf("Failed to update item: %v", err)
			}
		}

		elapsed := time.Since(start)
		t.Logf("Updated 100 items in %v (%.2f items/sec)", elapsed, 100.0/elapsed.Seconds())

		// Should be reasonably fast (< 1 second)
		if elapsed > 1*time.Second {
			t.Errorf("Update took too long: %v", elapsed)
		}
	})

	// Test 8: Memory usage with pagination (ensure we don't load all items)
	t.Run("MemoryUsageWithPagination", func(t *testing.T) {
		// Get memory stats before
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		// Load multiple pages (but not all at once)
		for page := 0; page < 10; page++ {
			_, err := queueStore.GetAll(page*100, 100)
			if err != nil {
				t.Fatalf("Failed to get page %d: %v", page, err)
			}
		}

		// Get memory stats after
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		allocatedMB := float64(m2.Alloc-m1.Alloc) / 1024 / 1024
		t.Logf("Memory allocated for 1000 items across 10 pages: %.2f MB", allocatedMB)

		// Should not use excessive memory (< 50MB for 1000 items)
		if allocatedMB > 50 {
			t.Errorf("Excessive memory usage: %.2f MB", allocatedMB)
		}
	})

	// Test 9: Clear completed (tests bulk delete performance)
	t.Run("ClearCompleted", func(t *testing.T) {
		start := time.Now()

		err := queueStore.ClearCompleted()
		if err != nil {
			t.Fatalf("Failed to clear completed: %v", err)
		}

		elapsed := time.Since(start)
		t.Logf("Cleared completed items in %v", elapsed)

		// Should be fast (< 500ms)
		if elapsed > 500*time.Millisecond {
			t.Errorf("Clear completed took too long: %v", elapsed)
		}

		// Verify completed items are gone
		stats, _ := queueStore.GetStats()
		if stats.Completed != 0 {
			t.Errorf("Expected 0 completed items, got %d", stats.Completed)
		}
	})
}

// TestDatabaseIndexes verifies that indexes are created correctly
func TestDatabaseIndexes(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_indexes.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := store.RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Query SQLite to check indexes
	rows, err := db.Query(`
		SELECT name, tbl_name 
		FROM sqlite_master 
		WHERE type = 'index' AND tbl_name = 'queue_items'
	`)
	if err != nil {
		t.Fatalf("Failed to query indexes: %v", err)
	}
	defer rows.Close()

	indexes := make(map[string]bool)
	for rows.Next() {
		var name, tblName string
		if err := rows.Scan(&name, &tblName); err != nil {
			t.Fatalf("Failed to scan index: %v", err)
		}
		indexes[name] = true
		t.Logf("Found index: %s on table %s", name, tblName)
	}

	// Verify required indexes exist
	requiredIndexes := []string{
		"idx_queue_status",
		"idx_queue_created",
		"idx_queue_resumable",
		"idx_queue_status_created",
		"idx_queue_updated",
		"idx_queue_status_progress",
	}

	for _, idx := range requiredIndexes {
		if !indexes[idx] {
			t.Errorf("Required index not found: %s", idx)
		}
	}
}

// Helper function to distribute statuses
func getStatus(i int) string {
	switch i % 4 {
	case 0:
		return "pending"
	case 1:
		return "downloading"
	case 2:
		return "completed"
	default:
		return "failed"
	}
}
