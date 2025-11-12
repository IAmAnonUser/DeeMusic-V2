package download

import (
	"context"
	"fmt"
	"os"
	"sync"
	
	"github.com/deemusic/deemusic-go/internal/store"
)

// JobType represents the type of download job
type JobType string

const (
	JobTypeTrack    JobType = "track"
	JobTypeAlbum    JobType = "album"
	JobTypePlaylist JobType = "playlist"
)

// Job represents a download job
type Job struct {
	ID           string
	Type         JobType
	TrackID      string
	AlbumID      string
	PlaylistID   string
	RetryCount   int
	ctx          context.Context
	cancel       context.CancelFunc
	QueueItem    *store.QueueItem
	IsCustom     bool
	CustomTracks []string
}

// Result represents the result of a job execution
type Result struct {
	JobID   string
	Success bool
	Error   error
}

// WorkerPool manages a pool of worker goroutines for concurrent downloads
type WorkerPool struct {
	maxWorkers int
	jobs       chan *Job
	results    chan *Result
	activeJobs sync.Map // map[string]*Job
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	handler    JobHandler
	mu         sync.RWMutex
	started    bool
}

// JobHandler is a function that processes a job
type JobHandler func(ctx context.Context, job *Job) error

// NewWorkerPool creates a new worker pool
func NewWorkerPool(maxWorkers int, handler JobHandler) *WorkerPool {
	if maxWorkers <= 0 {
		maxWorkers = 8 // Default to 8 workers
	}

	return &WorkerPool{
		maxWorkers: maxWorkers,
		jobs:       make(chan *Job, 10000), // Very large buffer to handle thousands of albums/tracks
		results:    make(chan *Result, maxWorkers*10),
		handler:    handler,
		started:    false,
	}
}

// Start spawns worker goroutines and begins processing jobs
func (wp *WorkerPool) Start(ctx context.Context) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.started {
		return fmt.Errorf("worker pool already started")
	}

	if wp.handler == nil {
		return fmt.Errorf("job handler not set")
	}

	// Use the provided context instead of creating a new one
	wp.ctx, wp.cancel = context.WithCancel(ctx)

	// Spawn worker goroutines
	for i := 0; i < wp.maxWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}

	wp.started = true
	return nil
}

// worker is the main worker goroutine that processes jobs
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	
	fmt.Fprintf(os.Stderr, "[DEBUG] Worker %d started\n", id)

	for {
		select {
		case <-wp.ctx.Done():
			// Worker pool is shutting down
			fmt.Fprintf(os.Stderr, "[WARN] Worker %d shutting down due to context cancellation: %v\n", id, wp.ctx.Err())
			return

		case job, ok := <-wp.jobs:
			if !ok {
				// Jobs channel closed
				fmt.Fprintf(os.Stderr, "[WARN] Worker %d shutting down due to closed jobs channel\n", id)
				return
			}

			// Process the job
			wp.processJob(job)
		}
	}
}

// processJob processes a single job
func (wp *WorkerPool) processJob(job *Job) {
	// Store active job
	wp.activeJobs.Store(job.ID, job)
	defer wp.activeJobs.Delete(job.ID)

	// Create job context if not set
	if job.ctx == nil {
		job.ctx, job.cancel = context.WithCancel(wp.ctx)
	}

	// Execute job handler
	err := wp.handler(job.ctx, job)

	// Send result
	result := &Result{
		JobID:   job.ID,
		Success: err == nil,
		Error:   err,
	}

	select {
	case wp.results <- result:
		// Result sent successfully
	case <-wp.ctx.Done():
		// Worker pool shutting down, discard result
	}
}

// Submit submits a job to the worker pool
func (wp *WorkerPool) Submit(job *Job) error {
	wp.mu.RLock()
	if !wp.started {
		wp.mu.RUnlock()
		return fmt.Errorf("worker pool not started")
	}
	wp.mu.RUnlock()

	// Create job context
	job.ctx, job.cancel = context.WithCancel(wp.ctx)

	select {
	case wp.jobs <- job:
		return nil
	case <-wp.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	}
}

// Stop gracefully stops the worker pool
func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	if !wp.started {
		wp.mu.Unlock()
		return
	}
	wp.mu.Unlock()

	// Cancel all active jobs
	wp.activeJobs.Range(func(key, value interface{}) bool {
		if job, ok := value.(*Job); ok && job.cancel != nil {
			job.cancel()
		}
		return true
	})

	// Signal shutdown
	wp.cancel()

	// Close jobs channel
	close(wp.jobs)

	// Wait for all workers to finish
	wp.wg.Wait()

	// Close results channel
	close(wp.results)

	wp.mu.Lock()
	wp.started = false
	wp.mu.Unlock()
}

// Results returns the results channel
func (wp *WorkerPool) Results() <-chan *Result {
	return wp.results
}

// CancelJob cancels a specific job by ID
func (wp *WorkerPool) CancelJob(jobID string) error {
	value, ok := wp.activeJobs.Load(jobID)
	if !ok {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job, ok := value.(*Job)
	if !ok {
		return fmt.Errorf("invalid job type for ID: %s", jobID)
	}

	if job.cancel != nil {
		job.cancel()
	}

	return nil
}

// CancelAll cancels all active jobs and drains the job queue
func (wp *WorkerPool) CancelAll() {
	// Cancel all active jobs
	wp.activeJobs.Range(func(key, value interface{}) bool {
		job, ok := value.(*Job)
		if ok && job.cancel != nil {
			job.cancel()
		}
		return true
	})
	
	// Clear the active jobs map
	wp.activeJobs = sync.Map{}
	
	// Drain the job queue (non-blocking)
	drained := 0
	for {
		select {
		case <-wp.jobs:
			drained++
		default:
			// Queue is empty
			if drained > 0 {
				fmt.Fprintf(os.Stderr, "[INFO] Drained %d pending jobs from queue\n", drained)
			}
			return
		}
	}
}

// GetActiveJobCount returns the number of currently active jobs
func (wp *WorkerPool) GetActiveJobCount() int {
	count := 0
	wp.activeJobs.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// IsJobActive checks if a job is currently active
func (wp *WorkerPool) IsJobActive(jobID string) bool {
	_, ok := wp.activeJobs.Load(jobID)
	return ok
}

// SetMaxWorkers updates the maximum number of workers (requires restart)
func (wp *WorkerPool) SetMaxWorkers(maxWorkers int) error {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.started {
		return fmt.Errorf("cannot change max workers while pool is running")
	}

	if maxWorkers <= 0 {
		return fmt.Errorf("max workers must be greater than 0")
	}

	wp.maxWorkers = maxWorkers
	return nil
}

// GetMaxWorkers returns the maximum number of workers
func (wp *WorkerPool) GetMaxWorkers() int {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.maxWorkers
}
