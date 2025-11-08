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

// BenchmarkQueueInsert benchmarks inserting items into the queue
func BenchmarkQueueInsert(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_queue.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := store.RunMigrations(db); err != nil {
		b.Fatalf("Failed to run migrations: %v", err)
	}

	queueStore := store.NewQueueStore(db)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := &store.QueueItem{
			ID:     fmt.Sprintf("track_%d", i),
			Type:   "track",
			Title:  fmt.Sprintf("Test Track %d", i),
			Artist: fmt.Sprintf("Test Artist %d", i%100),
			Album:  fmt.Sprintf("Test Album %d", i%50),
			Status: "pending",
		}

		if err := queueStore.Add(item); err != nil {
			b.Fatalf("Failed to add item: %v", err)
		}
	}
}

// BenchmarkQueueQuery benchmarks querying items from the queue
func BenchmarkQueueQuery(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_queue.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := store.RunMigrations(db); err != nil {
		b.Fatalf("Failed to run migrations: %v", err)
	}

	queueStore := store.NewQueueStore(db)

	// Pre-populate with 10,000 items
	for i := 0; i < 10000; i++ {
		item := &store.QueueItem{
			ID:     fmt.Sprintf("track_%d", i),
			Type:   "track",
			Title:  fmt.Sprintf("Test Track %d", i),
			Artist: fmt.Sprintf("Test Artist %d", i%100),
			Status: "pending",
		}
		queueStore.Add(item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := queueStore.GetAll(0, 100)
		if err != nil {
			b.Fatalf("Failed to query: %v", err)
		}
	}
}

// BenchmarkQueueQueryByStatus benchmarks filtered queries
func BenchmarkQueueQueryByStatus(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_queue.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := store.RunMigrations(db); err != nil {
		b.Fatalf("Failed to run migrations: %v", err)
	}

	queueStore := store.NewQueueStore(db)

	// Pre-populate with 10,000 items
	for i := 0; i < 10000; i++ {
		status := "pending"
		if i%4 == 0 {
			status = "completed"
		}
		item := &store.QueueItem{
			ID:     fmt.Sprintf("track_%d", i),
			Type:   "track",
			Title:  fmt.Sprintf("Test Track %d", i),
			Status: status,
		}
		queueStore.Add(item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := queueStore.GetByStatus("pending", 0, 100)
		if err != nil {
			b.Fatalf("Failed to query: %v", err)
		}
	}
}

// BenchmarkQueueStats benchmarks statistics queries
func BenchmarkQueueStats(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_queue.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := store.RunMigrations(db); err != nil {
		b.Fatalf("Failed to run migrations: %v", err)
	}

	queueStore := store.NewQueueStore(db)

	// Pre-populate with 10,000 items
	for i := 0; i < 10000; i++ {
		item := &store.QueueItem{
			ID:     fmt.Sprintf("track_%d", i),
			Type:   "track",
			Status: getStatus(i),
		}
		queueStore.Add(item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := queueStore.GetStats()
		if err != nil {
			b.Fatalf("Failed to get stats: %v", err)
		}
	}
}

// BenchmarkQueueUpdate benchmarks updating items
func BenchmarkQueueUpdate(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_queue.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := store.RunMigrations(db); err != nil {
		b.Fatalf("Failed to run migrations: %v", err)
	}

	queueStore := store.NewQueueStore(db)

	// Pre-populate with items
	for i := 0; i < 1000; i++ {
		item := &store.QueueItem{
			ID:     fmt.Sprintf("track_%d", i),
			Type:   "track",
			Status: "pending",
		}
		queueStore.Add(item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item, _ := queueStore.GetByID(fmt.Sprintf("track_%d", i%1000))
		item.Progress = i % 100
		item.Status = "downloading"
		queueStore.Update(item)
	}
}

// BenchmarkMemoryUsage tests memory usage with large queue
func BenchmarkMemoryUsage(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_memory.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		b.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := store.RunMigrations(db); err != nil {
		b.Fatalf("Failed to run migrations: %v", err)
	}

	queueStore := store.NewQueueStore(db)

	// Pre-populate with items
	for i := 0; i < 1000; i++ {
		item := &store.QueueItem{
			ID:     fmt.Sprintf("track_%d", i),
			Type:   "track",
			Title:  fmt.Sprintf("Test Track %d", i),
			Status: "pending",
		}
		queueStore.Add(item)
	}

	// Force GC before measuring
	runtime.GC()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Load 1000 items
		items, err := queueStore.GetAll(0, 1000)
		if err != nil {
			b.Fatalf("Failed to query: %v", err)
		}

		// Process items (simulate UI binding)
		for _, item := range items {
			_ = item.Title
			_ = item.Artist
			_ = item.Status
		}
	}
}

// TestStartupPerformance measures application startup time
func TestStartupPerformance(t *testing.T) {
	start := time.Now()

	// Simulate startup operations
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "startup_test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run migrations (part of startup)
	if err := store.RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize stores
	_ = store.NewQueueStore(db)

	elapsed := time.Since(start)
	t.Logf("Startup time: %v", elapsed)

	// Target: < 500ms for backend initialization
	if elapsed > 500*time.Millisecond {
		t.Errorf("Startup took too long: %v (target: < 500ms)", elapsed)
	}
}

// TestConcurrentOperations tests performance under concurrent load
func TestConcurrentOperations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "concurrent_test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := store.RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	queueStore := store.NewQueueStore(db)

	// Pre-populate
	for i := 0; i < 1000; i++ {
		item := &store.QueueItem{
			ID:     fmt.Sprintf("track_%d", i),
			Type:   "track",
			Status: "pending",
		}
		queueStore.Add(item)
	}

	start := time.Now()

	// Simulate 5 concurrent operations
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			// Each goroutine performs 100 operations
			for j := 0; j < 100; j++ {
				// Mix of reads and writes
				if j%2 == 0 {
					queueStore.GetAll(0, 100)
				} else {
					item, _ := queueStore.GetByID(fmt.Sprintf("track_%d", (id*100+j)%1000))
					if item != nil {
						item.Progress = j
						queueStore.Update(item)
					}
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	elapsed := time.Since(start)
	t.Logf("Concurrent operations (5 workers, 100 ops each) completed in: %v", elapsed)

	// Should complete in reasonable time (< 5 seconds)
	if elapsed > 5*time.Second {
		t.Errorf("Concurrent operations took too long: %v", elapsed)
	}
}

// TestMemoryLeaks checks for memory leaks over repeated operations
func TestMemoryLeaks(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "leak_test.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := store.RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	queueStore := store.NewQueueStore(db)

	// Pre-populate
	for i := 0; i < 1000; i++ {
		item := &store.QueueItem{
			ID:     fmt.Sprintf("track_%d", i),
			Type:   "track",
			Status: "pending",
		}
		queueStore.Add(item)
	}

	// Get baseline memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	baselineAlloc := m1.Alloc

	// Perform 1000 query cycles
	for i := 0; i < 1000; i++ {
		items, _ := queueStore.GetAll(0, 100)
		// Process items
		for _, item := range items {
			_ = item.Title
		}
	}

	// Force GC and check memory
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	finalAlloc := m2.Alloc

	// Calculate increase (handle case where GC reduced memory)
	var allocatedMB float64
	if finalAlloc > baselineAlloc {
		allocatedMB = float64(finalAlloc-baselineAlloc) / 1024 / 1024
	} else {
		allocatedMB = 0
	}

	t.Logf("Memory increase after 1000 query cycles: %.2f MB", allocatedMB)
	t.Logf("Baseline: %.2f MB, Final: %.2f MB", float64(baselineAlloc)/1024/1024, float64(finalAlloc)/1024/1024)

	// Should not leak significant memory (< 10MB increase)
	if allocatedMB > 10 {
		t.Errorf("Possible memory leak detected: %.2f MB increase", allocatedMB)
	}
}
