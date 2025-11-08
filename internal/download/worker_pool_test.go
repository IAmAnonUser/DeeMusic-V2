package download

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWorkerPoolCreation(t *testing.T) {
	handler := func(ctx context.Context, job *Job) error {
		return nil
	}

	pool := NewWorkerPool(4, handler)

	if pool == nil {
		t.Fatal("Expected non-nil worker pool")
	}

	if pool.GetMaxWorkers() != 4 {
		t.Errorf("Expected 4 workers, got %d", pool.GetMaxWorkers())
	}
}

func TestWorkerPoolStartStop(t *testing.T) {
	handler := func(ctx context.Context, job *Job) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	pool := NewWorkerPool(2, handler)
	ctx := context.Background()

	err := pool.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}

	// Try to start again (should fail)
	err = pool.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already started pool")
	}

	pool.Stop()
}

func TestWorkerPoolJobProcessing(t *testing.T) {
	processed := make(chan string, 10)

	handler := func(ctx context.Context, job *Job) error {
		processed <- job.ID
		return nil
	}

	pool := NewWorkerPool(2, handler)
	ctx := context.Background()

	err := pool.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop()

	// Submit jobs
	jobCount := 5
	for i := 0; i < jobCount; i++ {
		job := &Job{
			ID:   string(rune('A' + i)),
			Type: JobTypeTrack,
		}
		err := pool.Submit(job)
		if err != nil {
			t.Errorf("Failed to submit job: %v", err)
		}
	}

	// Collect results
	timeout := time.After(5 * time.Second)
	resultCount := 0

	for resultCount < jobCount {
		select {
		case <-processed:
			resultCount++
		case <-timeout:
			t.Fatalf("Timeout waiting for results, got %d/%d", resultCount, jobCount)
		}
	}
}

func TestWorkerPoolJobCancellation(t *testing.T) {
	handler := func(ctx context.Context, job *Job) error {
		select {
		case <-time.After(5 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	pool := NewWorkerPool(2, handler)
	ctx := context.Background()

	err := pool.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop()

	// Submit a job
	job := &Job{
		ID:   "test-job",
		Type: JobTypeTrack,
	}

	err = pool.Submit(job)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Wait a bit for job to start
	time.Sleep(100 * time.Millisecond)

	// Cancel the job
	err = pool.CancelJob("test-job")
	if err != nil {
		t.Errorf("Failed to cancel job: %v", err)
	}

	// Check result
	select {
	case result := <-pool.Results():
		if result.Success {
			t.Error("Expected job to fail after cancellation")
		}
		if !errors.Is(result.Error, context.Canceled) {
			t.Errorf("Expected context.Canceled error, got: %v", result.Error)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for cancelled job result")
	}
}

func TestWorkerPoolActiveJobCount(t *testing.T) {
	handler := func(ctx context.Context, job *Job) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	pool := NewWorkerPool(2, handler)
	ctx := context.Background()

	err := pool.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop()

	// Submit multiple jobs
	for i := 0; i < 4; i++ {
		job := &Job{
			ID:   string(rune('A' + i)),
			Type: JobTypeTrack,
		}
		pool.Submit(job)
	}

	// Check active job count
	time.Sleep(50 * time.Millisecond)
	activeCount := pool.GetActiveJobCount()

	if activeCount == 0 {
		t.Error("Expected some active jobs")
	}

	if activeCount > 2 {
		t.Errorf("Expected at most 2 active jobs (worker count), got %d", activeCount)
	}
}

func TestWorkerPoolErrorHandling(t *testing.T) {
	expectedError := errors.New("test error")

	handler := func(ctx context.Context, job *Job) error {
		if job.ID == "error-job" {
			return expectedError
		}
		return nil
	}

	pool := NewWorkerPool(2, handler)
	ctx := context.Background()

	err := pool.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start pool: %v", err)
	}
	defer pool.Stop()

	// Submit a job that will fail
	job := &Job{
		ID:   "error-job",
		Type: JobTypeTrack,
	}

	err = pool.Submit(job)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Check result
	select {
	case result := <-pool.Results():
		if result.Success {
			t.Error("Expected job to fail")
		}
		if result.Error != expectedError {
			t.Errorf("Expected error %v, got %v", expectedError, result.Error)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for error job result")
	}
}
