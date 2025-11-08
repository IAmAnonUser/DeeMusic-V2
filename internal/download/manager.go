package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/deemusic/deemusic-go/internal/api"
	"github.com/deemusic/deemusic-go/internal/config"
	"github.com/deemusic/deemusic-go/internal/decryption"
	"github.com/deemusic/deemusic-go/internal/metadata"
	"github.com/deemusic/deemusic-go/internal/store"
)

// Manager coordinates all download operations
type Manager struct {
	config      *config.Config
	workerPool  *WorkerPool
	queueStore  *store.QueueStore
	deezerAPI   *api.DeezerClient
	processor   *decryption.StreamingProcessor
	notifier    Notifier
	mu          sync.RWMutex
	pausedJobs  map[string]bool
	started     bool
}

// Notifier interface for progress notifications
type Notifier interface {
	NotifyProgress(itemID string, progress int, bytesProcessed, totalBytes int64)
	NotifyStarted(itemID string)
	NotifyCompleted(itemID string)
	NotifyFailed(itemID string, err error)
}

// NewManager creates a new download manager
func NewManager(
	cfg *config.Config,
	queueStore *store.QueueStore,
	deezerAPI *api.DeezerClient,
	notifier Notifier,
) *Manager {
	processor := decryption.NewStreamingProcessor(8192)

	mgr := &Manager{
		config:     cfg,
		queueStore: queueStore,
		deezerAPI:  deezerAPI,
		processor:  processor,
		notifier:   notifier,
		pausedJobs: make(map[string]bool),
		started:    false,
	}

	// Create worker pool with job handler
	mgr.workerPool = NewWorkerPool(cfg.Download.ConcurrentDownloads, mgr.handleJob)

	return mgr
}

// Start starts the download manager
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fmt.Fprintf(os.Stderr, "[DEBUG] Manager.Start() called, started=%v\n", m.started)

	if m.started {
		return fmt.Errorf("download manager already started")
	}

	// Start worker pool
	fmt.Fprintf(os.Stderr, "[DEBUG] Starting worker pool...\n")
	if err := m.workerPool.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Worker pool start failed: %v\n", err)
		return fmt.Errorf("failed to start worker pool: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] Worker pool started\n")

	// Start result processor
	fmt.Fprintf(os.Stderr, "[DEBUG] Starting result processor goroutine...\n")
	go m.processResults()

	// Start queue processor
	fmt.Fprintf(os.Stderr, "[DEBUG] Starting queue processor goroutine...\n")
	go m.processQueue(ctx)

	m.started = true
	fmt.Fprintf(os.Stderr, "[DEBUG] Manager.Start() completed successfully\n")
	return nil
}

// Stop stops the download manager
func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()

	m.workerPool.Stop()

	m.mu.Lock()
	m.started = false
	m.mu.Unlock()
}

// handleJob processes a single download job
func (m *Manager) handleJob(ctx context.Context, job *Job) error {
	switch job.Type {
	case JobTypeTrack:
		return m.downloadTrackJob(ctx, job)
	case JobTypeAlbum:
		return m.downloadAlbumJob(ctx, job)
	case JobTypePlaylist:
		return m.downloadPlaylistJob(ctx, job)
	default:
		return fmt.Errorf("unknown job type: %s", job.Type)
	}
}

// downloadTrackJob downloads a single track
func (m *Manager) downloadTrackJob(ctx context.Context, job *Job) error {
	// Log to temp file
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] downloadTrackJob started for track %s (ID: %s)\n", time.Now().Format("2006-01-02 15:04:05"), job.TrackID, job.ID)
		logFile.Close()
	}

	// Get queue item
	item, err := m.queueStore.GetByID(job.ID)
	if err != nil {
		if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
			fmt.Fprintf(logFile, "[%s] ERROR getting queue item: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
			logFile.Close()
		}
		return fmt.Errorf("failed to get queue item: %w", err)
	}

	// Check if paused
	if m.isJobPaused(job.ID) {
		return fmt.Errorf("job is paused")
	}

	// Update status to downloading
	item.Status = "downloading"
	item.Progress = 0
	if err := m.queueStore.Update(item); err != nil {
		if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
			fmt.Fprintf(logFile, "[%s] ERROR updating status to downloading: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
			logFile.Close()
		}
		return fmt.Errorf("failed to update queue item: %w", err)
	}

	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] Track status updated to downloading\n", time.Now().Format("2006-01-02 15:04:05"))
		logFile.Close()
	}

	// Notify started
	if m.notifier != nil {
		m.notifier.NotifyStarted(job.ID)
	}

	// Get track details
	track, err := m.deezerAPI.GetTrack(ctx, job.TrackID)
	if err != nil {
		if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
			fmt.Fprintf(logFile, "[%s] ERROR getting track details: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
			logFile.Close()
		}
		return fmt.Errorf("failed to get track details: %w", err)
	}

	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] Got track details: %s\n", time.Now().Format("2006-01-02 15:04:05"), track.Title)
		logFile.Close()
	}

	// Check if this track is part of an album or playlist download (has ParentID)
	if item.ParentID != "" {
		// Get parent item to determine if it's an album or playlist
		parentItem, err := m.queueStore.GetByID(item.ParentID)
		if err == nil && parentItem != nil {
			if parentItem.Type == "playlist" {
				// This is part of a playlist download
				// Get playlist details to attach to track
				playlistID := strings.TrimPrefix(item.ParentID, "playlist_")
				playlist, err := m.deezerAPI.GetPlaylist(ctx, playlistID)
				if err == nil {
					track.Playlist = playlist
					// Find position in playlist
					for i, plTrack := range playlist.Tracks.Data {
						if plTrack.ID.String() == track.ID.String() {
							track.PlaylistPosition = i + 1
							break
						}
					}
					
					if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
						fmt.Fprintf(logFile, "[%s] Track is part of playlist download. PlaylistID=%s, Position=%d\n", 
							time.Now().Format("2006-01-02 15:04:05"), playlistID, track.PlaylistPosition)
						logFile.Close()
					}
				}
			} else if parentItem.Type == "album" {
				// This is part of an album download
				// Check the cache to see if this album is multi-disc
				albumID := track.Album.ID.String()
		
		// Check cache first
		multiDiscCacheMu.RLock()
		discInfo, cached := multiDiscCache[albumID]
		multiDiscCacheMu.RUnlock()
		
		// If this track has disc_number > 1, the album is definitely multi-disc
		// Update the cache if needed (upgradeable cache)
		if track.DiscNumber > 1 && (!cached || !discInfo.IsMultiDisc) {
			totalDiscs := track.DiscNumber // At minimum, we know there are this many discs
			if cached && discInfo.TotalDiscs > totalDiscs {
				totalDiscs = discInfo.TotalDiscs
			}
			
			multiDiscCacheMu.Lock()
			multiDiscCache[albumID] = &DiscInfo{
				IsMultiDisc: true,
				TotalDiscs:  totalDiscs,
			}
			multiDiscCacheMu.Unlock()
			
			track.IsMultiDiscAlbum = true
			track.TotalDiscs = totalDiscs
			
			if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				fmt.Fprintf(logFile, "[%s] Album %s upgraded to multi-disc (track has DiscNumber=%d, TotalDiscs=%d)\n", 
					time.Now().Format("2006-01-02 15:04:05"), albumID, track.DiscNumber, totalDiscs)
				logFile.Close()
			}
		} else if !cached {
			// First track from this album and it's disc 1 - assume single disc for now
			// Will be upgraded if we see a disc 2+ track later
			multiDiscCacheMu.Lock()
			multiDiscCache[albumID] = &DiscInfo{
				IsMultiDisc: false,
				TotalDiscs:  1,
			}
			multiDiscCacheMu.Unlock()
			
			track.IsMultiDiscAlbum = false
			track.TotalDiscs = 1
			
			if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				fmt.Fprintf(logFile, "[%s] Album %s initially cached as single-disc (track has DiscNumber=%d)\n", 
					time.Now().Format("2006-01-02 15:04:05"), albumID, track.DiscNumber)
				logFile.Close()
			}
		} else {
			// Use cached info
			track.IsMultiDiscAlbum = discInfo.IsMultiDisc
			track.TotalDiscs = discInfo.TotalDiscs
		}
		
				if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
					fmt.Fprintf(logFile, "[%s] Track is part of album download. AlbumID=%s, DiscNumber=%d, TotalDiscs=%d, IsMultiDisc=%v\n", 
						time.Now().Format("2006-01-02 15:04:05"), albumID, track.DiscNumber, track.TotalDiscs, track.IsMultiDiscAlbum)
					logFile.Close()
				}
			}
		}
	} else {
		// Single track download - never create CD folders
		track.IsMultiDiscAlbum = false
		track.TotalDiscs = 0
		
		if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			fmt.Fprintf(logFile, "[%s] Single track download, IsMultiDiscAlbum=false\n", time.Now().Format("2006-01-02 15:04:05"))
			logFile.Close()
		}
	}

	// Get download URL
	downloadURLInfo, err := m.deezerAPI.GetTrackDownloadURL(ctx, job.TrackID, m.config.Download.Quality)
	if err != nil {
		if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
			fmt.Fprintf(logFile, "[%s] ERROR getting download URL: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
			logFile.Close()
		}
		return fmt.Errorf("failed to get download URL: %w", err)
	}

	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] Got download URL, starting download...\n", time.Now().Format("2006-01-02 15:04:05"))
		logFile.Close()
	}

	// Determine album artist for folder structure
	// Default to track artist
	track.AlbumArtist = track.Artist.Name
	
	// For playlist downloads, use "Various Artists"
	if track.Playlist != nil {
		track.AlbumArtist = "Various Artists"
	} else if track.Album.RecordType != "single" && track.Album.RecordType != "ep" &&
	          (track.Album.RecordType == "compilation" || 
	           strings.Contains(strings.ToLower(track.Album.Title), "soundtrack") ||
	           strings.Contains(strings.ToLower(track.Album.Title), "original score") ||
	           strings.Contains(strings.ToLower(track.Album.Title), "original motion picture")) {
		// For compilations and soundtracks, use "Various Artists"
		track.AlbumArtist = "Various Artists"
		
		if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			fmt.Fprintf(logFile, "[%s] Compilation/Soundtrack detected for folder structure: Album='%s', RecordType='%s', using AlbumArtist=Various Artists\n", 
				time.Now().Format("2006-01-02 15:04:05"), track.Album.Title, track.Album.RecordType)
			logFile.Close()
		}
	} else if track.Album.Artist != nil && track.Album.Artist.Name != "" {
		// Use album artist if available
		track.AlbumArtist = track.Album.Artist.Name
	}

	// Build output path
	outputPath := m.buildOutputPath(track)

	// Check if file already exists (resume functionality)
	if fileInfo, err := os.Stat(outputPath); err == nil {
		// File exists - check if it's complete by comparing size
		if fileInfo.Size() > 0 {
			if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
				fmt.Fprintf(logFile, "[%s] File already exists (%d bytes), skipping download and applying metadata\n", 
					time.Now().Format("2006-01-02 15:04:05"), fileInfo.Size())
				logFile.Close()
			}
			
			// File exists, just apply metadata and mark as completed
			// Apply metadata synchronously since we're not downloading
			metadataErr := m.applyMetadataTags(ctx, outputPath, track)
			if metadataErr != nil {
				if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
					fmt.Fprintf(logFile, "[%s] Failed to apply metadata tags: %v\n", time.Now().Format("2006-01-02 15:04:05"), metadataErr)
					logFile.Close()
				}
				// Don't mark as completed if metadata failed
				item.Status = "failed"
				item.ErrorMessage = fmt.Sprintf("Metadata error: %v", metadataErr)
				if err := m.queueStore.Update(item); err != nil {
					return fmt.Errorf("failed to update queue item: %w", err)
				}
				return metadataErr
			}
			
			// Download lyrics if enabled
			if m.config.Lyrics.Enabled && m.config.Lyrics.SaveSyncedFile {
				if err := m.downloadAndSaveLyrics(ctx, outputPath, track); err != nil {
					if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
						fmt.Fprintf(logFile, "[%s] Failed to download lyrics: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
						logFile.Close()
					}
					// Lyrics failure is not critical, continue
				}
			}
			
			// Mark as completed only if metadata was successfully applied
			item.Status = "completed"
			item.Progress = 100
			item.OutputPath = outputPath
			now := time.Now()
			item.CompletedAt = &now
			if err := m.queueStore.Update(item); err != nil {
				return fmt.Errorf("failed to update queue item: %w", err)
			}
			
			if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
				fmt.Fprintf(logFile, "[%s] Track marked as completed: %s\n", time.Now().Format("2006-01-02 15:04:05"), item.ID)
				logFile.Close()
			}
			
			// Update parent progress
			if item.ParentID != "" {
				m.updateParentProgress(item.ParentID)
			}
			
			// Notify completed
			if m.notifier != nil {
				m.notifier.NotifyCompleted(job.ID)
			}
			
			if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
				fmt.Fprintf(logFile, "[%s] Track resumed and completed successfully\n", time.Now().Format("2006-01-02 15:04:05"))
				logFile.Close()
			}
			
			return nil
		}
	}

	// Progress callback
	progressCallback := func(bytesProcessed, totalBytes int64) {
		if totalBytes > 0 {
			progress := int((bytesProcessed * 100) / totalBytes)
			item.Progress = progress
			m.queueStore.Update(item)

			if m.notifier != nil {
				m.notifier.NotifyProgress(job.ID, progress, bytesProcessed, totalBytes)
			}
		}
	}

	// Download and decrypt
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	}

	result, err := m.processor.DownloadAndDecrypt(
		downloadURLInfo.URL,
		job.TrackID,
		outputPath,
		progressCallback,
		headers,
		m.config.Network.Timeout,
	)

	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("download failed: %s", result.ErrorMessage)
	}

	// Download artwork if enabled
	if m.config.Download.EmbedArtwork {
		trackDir := filepath.Dir(outputPath)
		
		if track.Playlist != nil {
			// Playlist download - download playlist cover
			if err := m.downloadPlaylistArtwork(ctx, track.Playlist, trackDir); err != nil {
				// Log error but don't fail the download
				if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
					fmt.Fprintf(logFile, "[%s] Failed to download playlist artwork: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
					logFile.Close()
				}
			}
			// No artist image for playlists
		} else {
			// Album download - download album artwork
			if err := m.downloadAlbumArtwork(ctx, track.Album, trackDir); err != nil {
				// Log error but don't fail the download
				fmt.Printf("Failed to download album artwork: %v\n", err)
			}
			
			// Download artist image (to artist folder) - but NOT for compilations/soundtracks
			// Compilations use "Various Artists" and shouldn't have individual artist images
			if track.AlbumArtist != "Various Artists" {
				artistDir := filepath.Dir(trackDir) // Go up one level from album to artist
				if err := m.downloadArtistImage(ctx, track.Artist, artistDir); err != nil {
					// Log error to debug file but don't fail the download
					if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
						fmt.Fprintf(logFile, "[%s] Failed to download artist image: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
						logFile.Close()
					}
				}
			} else {
				if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
					fmt.Fprintf(logFile, "[%s] Skipping artist image for compilation/soundtrack (AlbumArtist=Various Artists)\n", 
						time.Now().Format("2006-01-02 15:04:05"))
					logFile.Close()
				}
			}
		}
	}

	// Apply metadata tags with panic recovery (in background to not slow down queue)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Panic in metadata tagging: %v\n", r)
				if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
					fmt.Fprintf(logFile, "[%s] PANIC in metadata tagging: %v\n", time.Now().Format("2006-01-02 15:04:05"), r)
					logFile.Close()
				}
			}
		}()
		
		// Small delay to ensure file is fully written and closed
		time.Sleep(100 * time.Millisecond)
		
		if err := m.applyMetadataTags(ctx, outputPath, track); err != nil {
			// Silently fail - metadata is not critical
			if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
				fmt.Fprintf(logFile, "[%s] Failed to apply metadata tags: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
				logFile.Close()
			}
		}
	}()

	// Download and save lyrics with panic recovery (in background to not slow down queue)
	if m.config.Lyrics.Enabled && m.config.Lyrics.SaveSyncedFile {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Panic in lyrics download: %v\n", r)
					if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
						fmt.Fprintf(logFile, "[%s] PANIC in lyrics download: %v\n", time.Now().Format("2006-01-02 15:04:05"), r)
						logFile.Close()
					}
				}
			}()
			
			// Small delay to ensure file is fully written
			time.Sleep(100 * time.Millisecond)
			
			if err := m.downloadAndSaveLyrics(ctx, outputPath, track); err != nil {
				// Silently fail - lyrics are not critical
				if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
					fmt.Fprintf(logFile, "[%s] Failed to download lyrics: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
					logFile.Close()
				}
			}
		}()
	}

	// Update queue item
	item.Status = "completed"
	item.Progress = 100
	item.OutputPath = outputPath
	now := time.Now()
	item.CompletedAt = &now
	if err := m.queueStore.Update(item); err != nil {
		return fmt.Errorf("failed to update queue item: %w", err)
	}

	// If this track belongs to an album, update the album's completed count
	if item.ParentID != "" {
		m.updateParentProgress(item.ParentID)
	}

	// Add to history
	if err := m.queueStore.AddToHistory(
		job.TrackID,
		track.Title,
		track.Artist.Name,
		track.Album.Title,
		outputPath,
		m.config.Download.Quality,
		result.FileSize,
	); err != nil {
		// Log error but don't fail the download
		fmt.Printf("Failed to add to history: %v\n", err)
	}

	// Notify completed
	if m.notifier != nil {
		m.notifier.NotifyCompleted(job.ID)
	}

	return nil
}

// downloadAlbumJob downloads all tracks in an album
func (m *Manager) downloadAlbumJob(ctx context.Context, job *Job) error {
	// Log to temp file
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] downloadAlbumJob started for album %s\n", time.Now().Format("2006-01-02 15:04:05"), job.AlbumID)
		logFile.Close()
	}

	// Mark album as downloading to prevent reprocessing
	if albumItem, err := m.queueStore.GetByID(job.ID); err == nil && albumItem != nil {
		albumItem.Status = "downloading"
		if err := m.queueStore.Update(albumItem); err != nil {
			if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
				fmt.Fprintf(logFile, "[%s] WARNING: Failed to update album status to downloading: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
				logFile.Close()
			}
		}
	}

	// Get album details
	album, err := m.deezerAPI.GetAlbum(ctx, job.AlbumID)
	if err != nil {
		if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
			fmt.Fprintf(logFile, "[%s] ERROR getting album details: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
			logFile.Close()
		}
		return fmt.Errorf("failed to get album details: %w", err)
	}

	totalTracks := len(album.Tracks.Data)
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] Album has %d tracks\n", time.Now().Format("2006-01-02 15:04:05"), totalTracks)
		logFile.Close()
	}

	// Detect if this is a multi-disc album
	// Method 1: Check if album.DiscCount > 1 (from nb_disk field)
	isMultiDisc := album.DiscCount > 1
	totalDiscs := album.DiscCount
	
	// Method 2: Check actual track disc numbers from album API (often not populated)
	for _, track := range album.Tracks.Data {
		if track.DiscNumber > totalDiscs {
			totalDiscs = track.DiscNumber
		}
		if track.DiscNumber > 1 {
			isMultiDisc = true
		}
	}
	
	// Method 3: If still not detected as multi-disc OR if we need to find total disc count,
	// fetch sample tracks to check. This is necessary because album API often doesn't include disc numbers
	// Check tracks from beginning, middle, and end to find disc 2+ tracks and determine total discs
	if len(album.Tracks.Data) > 0 && (totalDiscs == 0 || !isMultiDisc) {
		totalTracks := len(album.Tracks.Data)
		
		// Sample tracks to check: first, middle, last, and a few in between
		// For multi-disc albums, the last track is most likely to have the highest disc number
		indicesToCheck := []int{0} // Always check first track
		
		if totalTracks > 1 {
			indicesToCheck = append(indicesToCheck, totalTracks-1) // Last track (IMPORTANT for total disc count!)
		}
		if totalTracks > 2 {
			indicesToCheck = append(indicesToCheck, totalTracks/2) // Middle track
		}
		if totalTracks > 5 {
			indicesToCheck = append(indicesToCheck, totalTracks/3, (totalTracks*2)/3) // 1/3 and 2/3 points
		}
		if totalTracks > 10 {
			// For large albums, check more points to ensure we find all discs
			indicesToCheck = append(indicesToCheck, totalTracks/4, (totalTracks*3)/4) // 1/4 and 3/4 points
		}
		
		if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			fmt.Fprintf(logFile, "[%s] Checking %d sample tracks for multi-disc detection (total tracks: %d)\n", 
				time.Now().Format("2006-01-02 15:04:05"), len(indicesToCheck), totalTracks)
			logFile.Close()
		}
		
		for _, idx := range indicesToCheck {
			if idx >= totalTracks {
				continue
			}
			
			trackID := album.Tracks.Data[idx].ID.String()
			track, err := m.deezerAPI.GetTrack(ctx, trackID)
			if err != nil {
				if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
					fmt.Fprintf(logFile, "[%s] Failed to fetch track %d for multi-disc check: %v\n", 
						time.Now().Format("2006-01-02 15:04:05"), idx+1, err)
					logFile.Close()
				}
				continue
			}
			
			if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				fmt.Fprintf(logFile, "[%s] Checked track %d/%d: DiscNumber=%d\n", 
					time.Now().Format("2006-01-02 15:04:05"), idx+1, totalTracks, track.DiscNumber)
				logFile.Close()
			}
			
			// Update totalDiscs if this track has a higher disc number
			if track.DiscNumber > totalDiscs {
				totalDiscs = track.DiscNumber
			}
			
			if track.DiscNumber > 1 {
				isMultiDisc = true
				if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
					fmt.Fprintf(logFile, "[%s] Multi-disc detected! Track %d has DiscNumber=%d, TotalDiscs now=%d\n", 
						time.Now().Format("2006-01-02 15:04:05"), idx+1, track.DiscNumber, totalDiscs)
					logFile.Close()
				}
				// Don't break - continue checking to find the maximum disc number
			}
		}
	}
	
	// Ensure totalDiscs is at least 1 for single-disc albums, and at least 2 for multi-disc
	if totalDiscs == 0 {
		if isMultiDisc {
			totalDiscs = 2 // Multi-disc but count unknown, assume at least 2
		} else {
			totalDiscs = 1 // Single disc
		}
	}
	
	// Pre-populate the cache so all tracks will know this album is multi-disc
	// This prevents race conditions where disc 1 tracks are processed before disc 2
	albumID := job.AlbumID
	multiDiscCacheMu.Lock()
	multiDiscCache[albumID] = &DiscInfo{
		IsMultiDisc: isMultiDisc,
		TotalDiscs:  totalDiscs,
	}
	multiDiscCacheMu.Unlock()
	
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] Multi-disc detection for album %s: album.DiscCount=%d, totalDiscs=%d, isMultiDisc=%v (cached for all tracks)\n", 
			time.Now().Format("2006-01-02 15:04:05"), albumID, album.DiscCount, totalDiscs, isMultiDisc)
		logFile.Close()
	}
	
	// Mark all tracks with multi-disc flag and total disc count
	for _, track := range album.Tracks.Data {
		track.IsMultiDiscAlbum = isMultiDisc
		track.TotalDiscs = totalDiscs
		if isMultiDisc && track.DiscNumber == 0 {
			// Multi-disc album: ensure all tracks have disc number (default to 1)
			track.DiscNumber = 1
		}
		// Note: We keep the original disc number for metadata even in single-disc albums
	}

	// Update album item with total tracks
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] Trying to update album item %s with %d total tracks\n", time.Now().Format("2006-01-02 15:04:05"), job.ID, totalTracks)
		logFile.Close()
	}
	
	albumItem, err := m.queueStore.GetByID(job.ID)
	if err != nil {
		if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
			fmt.Fprintf(logFile, "[%s] ERROR: Failed to get album item %s: %v\n", time.Now().Format("2006-01-02 15:04:05"), job.ID, err)
			logFile.Close()
		}
	} else if albumItem == nil {
		if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
			fmt.Fprintf(logFile, "[%s] ERROR: Album item %s is nil\n", time.Now().Format("2006-01-02 15:04:05"), job.ID)
			logFile.Close()
		}
	} else {
		albumItem.TotalTracks = totalTracks
		albumItem.CompletedTracks = 0
		if updateErr := m.queueStore.Update(albumItem); updateErr != nil {
			if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
				fmt.Fprintf(logFile, "[%s] ERROR: Failed to update album item %s: %v\n", time.Now().Format("2006-01-02 15:04:05"), job.ID, updateErr)
				logFile.Close()
			}
		} else {
			if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
				fmt.Fprintf(logFile, "[%s] Successfully updated album item %s with %d total tracks\n", time.Now().Format("2006-01-02 15:04:05"), job.ID, totalTracks)
				logFile.Close()
			}
		}
	}

	// Create jobs for each track
	for i, track := range album.Tracks.Data {
		// Check if cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		trackID := fmt.Sprintf("track_%s_%s", job.AlbumID, track.ID)

		// Try to get existing track
		existingTrack, err := m.queueStore.GetByID(trackID)
		if err == nil && existingTrack != nil {
			// Track exists - check if it needs to be reprocessed
			if existingTrack.Status == "completed" {
				// Skip completed tracks
				if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
					fmt.Fprintf(logFile, "[%s] Track %d already completed, skipping\n", time.Now().Format("2006-01-02 15:04:05"), i)
					logFile.Close()
				}
				continue
			}
			
			// Track exists but not completed - reset to pending and submit job
			if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
				fmt.Fprintf(logFile, "[%s] Track %d exists with status %s, resetting to pending and resubmitting\n", time.Now().Format("2006-01-02 15:04:05"), i, existingTrack.Status)
				logFile.Close()
			}
			
			// Reset status to pending so it gets picked up by processQueue
			existingTrack.Status = "pending"
			existingTrack.Progress = 0
			existingTrack.ErrorMessage = ""
			updateErr := m.queueStore.Update(existingTrack)
			
			if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
				if updateErr != nil {
					fmt.Fprintf(logFile, "[%s] ERROR updating track %d status: %v\n", time.Now().Format("2006-01-02 15:04:05"), i, updateErr)
				} else {
					fmt.Fprintf(logFile, "[%s] Track %d status updated to pending in DB\n", time.Now().Format("2006-01-02 15:04:05"), i)
				}
				logFile.Close()
			}
			
			trackJob := &Job{
				ID:      trackID,
				Type:    JobTypeTrack,
				TrackID: track.ID.String(),
			}

			if err := m.workerPool.Submit(trackJob); err != nil {
				if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
					fmt.Fprintf(logFile, "[%s] ERROR submitting existing track job %d: %v\n", time.Now().Format("2006-01-02 15:04:05"), i, err)
					logFile.Close()
				}
				// Don't return error, just continue with next track
				continue
			}
			
			if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				fmt.Fprintf(logFile, "[%s] Existing track %d reset and resubmitted: %s\n", time.Now().Format("2006-01-02 15:04:05"), i, trackID)
				logFile.Close()
			}
			continue
		}

		// Track doesn't exist - create it
		trackItem := &store.QueueItem{
			ID:       trackID,
			Type:     "track",
			Title:    track.Title,
			Artist:   track.Artist.Name,
			Album:    album.Title,
			Status:   "pending",
			ParentID: job.ID, // Link track to parent album
		}

		if err := m.queueStore.Add(trackItem); err != nil {
			// Failed to add, log and continue
			if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
				fmt.Fprintf(logFile, "[%s] Track %d error adding: %v\n", time.Now().Format("2006-01-02 15:04:05"), i, err)
				logFile.Close()
			}
			continue
		}

		// Submit track job
		trackJob := &Job{
			ID:      trackItem.ID,
			Type:    JobTypeTrack,
			TrackID: track.ID.String(),
		}

		if err := m.workerPool.Submit(trackJob); err != nil {
			if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
				fmt.Fprintf(logFile, "[%s] ERROR submitting track job %d: %v\n", time.Now().Format("2006-01-02 15:04:05"), i, err)
				logFile.Close()
			}
			// Don't return error, just continue with next track
			continue
		}
		
		if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			fmt.Fprintf(logFile, "[%s] New track %d submitted: %s\n", time.Now().Format("2006-01-02 15:04:05"), i, trackItem.ID)
			logFile.Close()
		}
	}

	// Don't mark album as completed yet - it will be marked completed when all tracks finish
	// The updateParentProgress function will handle this
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] Album job completed - tracks will be processed by workers\n", time.Now().Format("2006-01-02 15:04:05"))
		logFile.Close()
	}

	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] downloadAlbumJob completed successfully\n", time.Now().Format("2006-01-02 15:04:05"))
		logFile.Close()
	}

	return nil
}

// downloadPlaylistJob downloads all tracks in a playlist
func (m *Manager) downloadPlaylistJob(ctx context.Context, job *Job) error {
	// Log to temp file
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] downloadPlaylistJob started for playlist %s\n", time.Now().Format("2006-01-02 15:04:05"), job.PlaylistID)
		logFile.Close()
	}

	// Mark playlist as downloading to prevent reprocessing
	if playlistItem, err := m.queueStore.GetByID(job.ID); err == nil && playlistItem != nil {
		playlistItem.Status = "downloading"
		if err := m.queueStore.Update(playlistItem); err != nil {
			if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
				fmt.Fprintf(logFile, "[%s] WARNING: Failed to update playlist status to downloading: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
				logFile.Close()
			}
		}
	}

	// Get playlist details
	playlist, err := m.deezerAPI.GetPlaylist(ctx, job.PlaylistID)
	if err != nil {
		if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
			fmt.Fprintf(logFile, "[%s] ERROR getting playlist details: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
			logFile.Close()
		}
		return fmt.Errorf("failed to get playlist details: %w", err)
	}

	totalTracks := len(playlist.Tracks.Data)
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] Playlist has %d tracks\n", time.Now().Format("2006-01-02 15:04:05"), totalTracks)
		logFile.Close()
	}

	// Update playlist item with total tracks
	playlistItem, err := m.queueStore.GetByID(job.ID)
	if err == nil && playlistItem != nil {
		playlistItem.TotalTracks = totalTracks
		playlistItem.CompletedTracks = 0
		if updateErr := m.queueStore.Update(playlistItem); updateErr != nil {
			if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
				fmt.Fprintf(logFile, "[%s] ERROR: Failed to update playlist item %s: %v\n", time.Now().Format("2006-01-02 15:04:05"), job.ID, updateErr)
				logFile.Close()
			}
		}
	}

	// Create jobs for each track
	for i, track := range playlist.Tracks.Data {
		// Check if cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		trackID := fmt.Sprintf("track_%s_%s", job.PlaylistID, track.ID)

		// Try to get existing track
		existingTrack, err := m.queueStore.GetByID(trackID)
		if err == nil && existingTrack != nil {
			// Track exists - check if it needs to be reprocessed
			if existingTrack.Status == "completed" {
				if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
					fmt.Fprintf(logFile, "[%s] Track %d already completed, skipping\n", time.Now().Format("2006-01-02 15:04:05"), i)
					logFile.Close()
				}
				continue
			}
			
			// Reset to pending
			existingTrack.Status = "pending"
			existingTrack.Progress = 0
			existingTrack.ErrorMessage = ""
			m.queueStore.Update(existingTrack)
			
			trackJob := &Job{
				ID:      trackID,
				Type:    JobTypeTrack,
				TrackID: track.ID.String(),
			}

			if err := m.workerPool.Submit(trackJob); err != nil {
				if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
					fmt.Fprintf(logFile, "[%s] ERROR submitting existing track job %d: %v\n", time.Now().Format("2006-01-02 15:04:05"), i, err)
					logFile.Close()
				}
				continue
			}
			continue
		}

		// Create queue item for track
		trackItem := &store.QueueItem{
			ID:       trackID,
			Type:     "track",
			Title:    track.Title,
			Artist:   track.Artist.Name,
			Album:    track.Album.Title,
			Status:   "pending",
			ParentID: job.ID, // Link track to parent playlist
		}

		if err := m.queueStore.Add(trackItem); err != nil {
			// Track might already exist, continue
			if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
				fmt.Fprintf(logFile, "[%s] Track %d error adding: %v\n", time.Now().Format("2006-01-02 15:04:05"), i, err)
				logFile.Close()
			}
			continue
		}

		// Submit track job
		trackJob := &Job{
			ID:      trackItem.ID,
			Type:    JobTypeTrack,
			TrackID: track.ID.String(),
		}

		if err := m.workerPool.Submit(trackJob); err != nil {
			if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
				fmt.Fprintf(logFile, "[%s] ERROR submitting track job %d: %v\n", time.Now().Format("2006-01-02 15:04:05"), i, err)
				logFile.Close()
			}
			continue
		}
		
		if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			fmt.Fprintf(logFile, "[%s] New track %d submitted: %s\n", time.Now().Format("2006-01-02 15:04:05"), i, trackItem.ID)
			logFile.Close()
		}
	}

	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] downloadPlaylistJob completed successfully\n", time.Now().Format("2006-01-02 15:04:05"))
		logFile.Close()
	}

	return nil
}

// processResults processes job results from the worker pool
func (m *Manager) processResults() {
	for result := range m.workerPool.Results() {
		if !result.Success && result.Error != nil {
			// Get queue item
			item, err := m.queueStore.GetByID(result.JobID)
			if err != nil {
				continue
			}

			// Update status to failed
			item.Status = "failed"
			item.ErrorMessage = result.Error.Error()
			item.RetryCount++
			m.queueStore.Update(item)

			// Notify failed
			if m.notifier != nil {
				m.notifier.NotifyFailed(result.JobID, result.Error)
			}

			// Retry if under limit
			if item.RetryCount < m.config.Network.MaxRetries {
				// Create retry job
				job := &Job{
					ID:         item.ID,
					Type:       JobType(item.Type),
					TrackID:    item.ID, // Simplified, should extract from metadata
					RetryCount: item.RetryCount,
				}

				// Submit with delay
				go func() {
					delay := time.Duration(item.RetryCount) * 2 * time.Second
					time.Sleep(delay)
					m.workerPool.Submit(job)
				}()
			}
		}
	}
}

// processQueue continuously processes pending queue items
func (m *Manager) processQueue(ctx context.Context) {
	// Use a file logger since stderr might not be captured
	logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		defer logFile.Close()
		fmt.Fprintf(logFile, "[%s] processQueue goroutine STARTED\n", time.Now().Format("2006-01-02 15:04:05"))
	}
	
	fmt.Fprintf(os.Stderr, "[INFO] processQueue goroutine started\n")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if logFile != nil {
				fmt.Fprintf(logFile, "[%s] processQueue goroutine STOPPED (context done)\n", time.Now().Format("2006-01-02 15:04:05"))
			}
			fmt.Fprintf(os.Stderr, "[INFO] processQueue goroutine stopped (context done)\n")
			return
		case <-ticker.C:
			if logFile != nil {
				fmt.Fprintf(logFile, "[%s] processQueue TICK - checking for pending items\n", time.Now().Format("2006-01-02 15:04:05"))
			}
			fmt.Fprintf(os.Stderr, "[DEBUG] processQueue tick - checking for pending items\n")
			m.processPendingItems()
		}
	}
}

// processPendingItems processes pending items in the queue
func (m *Manager) processPendingItems() {
	// Get pending items
	items, err := m.queueStore.GetPending(m.config.Download.ConcurrentDownloads * 2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to get pending items: %v\n", err)
		// Also log to temp file
		if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			fmt.Fprintf(logFile, "[%s] ERROR: Failed to get pending items: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
			logFile.Close()
		}
		return
	}

	// Always log to temp file
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] GetPending returned %d items\n", time.Now().Format("2006-01-02 15:04:05"), len(items))
		for i, item := range items {
			fmt.Fprintf(logFile, "[%s]   Item %d: ID=%s, Type=%s, Status=%s, Title=%s\n", time.Now().Format("2006-01-02 15:04:05"), i, item.ID, item.Type, item.Status, item.Title)
		}
		logFile.Close()
	}

	if len(items) > 0 {
		fmt.Fprintf(os.Stderr, "[INFO] Processing %d pending items\n", len(items))
	}

	// Open log file once for all items
	logFile, logErr := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if logErr == nil {
		defer logFile.Close()
	}

	for _, item := range items {
		// Log to temp file
		if logFile != nil {
			fmt.Fprintf(logFile, "[%s] Processing item: %s\n", time.Now().Format("2006-01-02 15:04:05"), item.ID)
		}
		
		// Check if already active
		if m.workerPool.IsJobActive(item.ID) {
			if logFile != nil {
				fmt.Fprintf(logFile, "[%s]   Skipping %s - already active\n", time.Now().Format("2006-01-02 15:04:05"), item.ID)
			}
			continue
		}

		// Check if paused
		if m.isJobPaused(item.ID) {
			if logFile != nil {
				fmt.Fprintf(logFile, "[%s]   Skipping %s - paused\n", time.Now().Format("2006-01-02 15:04:05"), item.ID)
			}
			continue
		}

			// Create job with proper ID extraction
			job := &Job{
				ID:         item.ID,
				Type:       JobType(item.Type),
				RetryCount: item.RetryCount,
			}

			// Extract the actual ID from the item.ID based on type
			// Format: "track_123", "album_456", "playlist_789"
			// For tracks from albums: "track_ALBUMID_TRACKID" - we need the last part
			parts := strings.Split(item.ID, "_")
			if len(parts) >= 2 {
				var actualID string
				if item.Type == "track" && len(parts) == 3 {
					// Track from album: track_ALBUMID_TRACKID -> use TRACKID
					actualID = parts[2]
				} else {
					// Direct download: track_TRACKID, album_ALBUMID, etc -> use second part
					actualID = parts[1]
				}
				
				switch item.Type {
				case "track":
					job.TrackID = actualID
				case "album":
					job.AlbumID = actualID
				case "playlist":
					job.PlaylistID = actualID
				}
			}

		if logFile != nil {
			fmt.Fprintf(logFile, "[%s]   Created job: ID=%s, Type=%s, TrackID=%s, AlbumID=%s, PlaylistID=%s\n", 
				time.Now().Format("2006-01-02 15:04:05"), job.ID, job.Type, job.TrackID, job.AlbumID, job.PlaylistID)
		}

		fmt.Fprintf(os.Stderr, "[INFO] Submitting job: ID=%s, Type=%s, TrackID=%s, AlbumID=%s\n", job.ID, job.Type, job.TrackID, job.AlbumID)

		// Submit job
		if err := m.workerPool.Submit(job); err != nil {
			if logFile != nil {
				fmt.Fprintf(logFile, "[%s]   ERROR submitting job %s: %v\n", time.Now().Format("2006-01-02 15:04:05"), job.ID, err)
			}
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to submit job %s: %v\n", job.ID, err)
			// Queue might be full, try again later
			continue
		}
		
		if logFile != nil {
			fmt.Fprintf(logFile, "[%s]   Job %s submitted successfully\n", time.Now().Format("2006-01-02 15:04:05"), job.ID)
		}
	}
}

// DownloadTrack adds a track to the download queue
func (m *Manager) DownloadTrack(ctx context.Context, trackID string) error {
	// Get track details
	track, err := m.deezerAPI.GetTrack(ctx, trackID)
	if err != nil {
		return fmt.Errorf("failed to get track details: %w", err)
	}

	// Create queue item
	item := &store.QueueItem{
		ID:     fmt.Sprintf("track_%s", trackID),
		Type:   "track",
		Title:  track.Title,
		Artist: track.Artist.Name,
		Album:  track.Album.Title,
		Status: "pending",
	}

	if err := m.queueStore.Add(item); err != nil {
		return fmt.Errorf("failed to add to queue: %w", err)
	}

	return nil
}

// DownloadAlbum adds an album to the download queue
func (m *Manager) DownloadAlbum(ctx context.Context, albumID string) error {
	fmt.Printf("[Manager] DownloadAlbum called with albumID: '%s'\n", albumID)
	
	// Get album details
	apiStart := time.Now()
	fmt.Printf("[Manager] Calling GetAlbum API...\n")
	album, err := m.deezerAPI.GetAlbum(ctx, albumID)
	if err != nil {
		fmt.Printf("[Manager] GetAlbum failed: %v\n", err)
		return fmt.Errorf("failed to get album details: %w", err)
	}
	fmt.Printf("[Manager] Got album: %s by %s (%d tracks) in %v\n", album.Title, album.Artist.Name, album.TrackCount, time.Since(apiStart))

	// Create queue item for album
	itemID := fmt.Sprintf("album_%s", albumID)
	
	// Check if item already exists
	existingItem, err := m.queueStore.GetByID(itemID)
	if err == nil && existingItem != nil {
		fmt.Printf("[Manager] Album already in queue with status: %s\n", existingItem.Status)
		// If it's pending or downloading, return error to notify user
		if existingItem.Status == "pending" || existingItem.Status == "downloading" {
			return fmt.Errorf("album already in queue")
		}
		// If it's failed or completed, reset it to pending
		if existingItem.Status == "failed" || existingItem.Status == "completed" {
			existingItem.Status = "pending"
			existingItem.ErrorMessage = ""
			existingItem.RetryCount = 0
			if err := m.queueStore.Update(existingItem); err != nil {
				fmt.Printf("[Manager] Failed to update existing item: %v\n", err)
				return fmt.Errorf("failed to update queue item: %w", err)
			}
			fmt.Printf("[Manager] Reset existing album to pending\n")
		}
	} else {
		// Item doesn't exist, create it
		item := &store.QueueItem{
			ID:             itemID,
			Type:           "album",
			Title:          album.Title,
			Artist:         album.Artist.Name,
			Album:          album.Title,
			Status:         "pending",
			TotalTracks:    album.TrackCount,
			CompletedTracks: 0,
		}

		fmt.Printf("[Manager] Adding album to queue with ID: %s, TotalTracks: %d\n", item.ID, item.TotalTracks)
		if err := m.queueStore.Add(item); err != nil {
			fmt.Printf("[Manager] Failed to add to queue: %v\n", err)
			return fmt.Errorf("failed to add to queue: %w", err)
		}
	}

	// Submit album job
	job := &Job{
		ID:      itemID,
		Type:    JobTypeAlbum,
		AlbumID: albumID,
	}

	fmt.Printf("[Manager] Submitting album job to worker pool...\n")
	err = m.workerPool.Submit(job)
	if err != nil {
		fmt.Printf("[Manager] Failed to submit job: %v\n", err)
		return err
	}
	
	fmt.Printf("[Manager] Album job submitted successfully\n")
	return nil
}

// DownloadPlaylist adds a playlist to the download queue
func (m *Manager) DownloadPlaylist(ctx context.Context, playlistID string) error {
	fmt.Printf("[Manager] DownloadPlaylist called with playlistID: '%s'\n", playlistID)
	
	// Get playlist details
	apiStart := time.Now()
	fmt.Printf("[Manager] Calling GetPlaylist API...\n")
	playlist, err := m.deezerAPI.GetPlaylist(ctx, playlistID)
	if err != nil {
		fmt.Printf("[Manager] GetPlaylist failed: %v\n", err)
		return fmt.Errorf("failed to get playlist details: %w", err)
	}
	fmt.Printf("[Manager] Got playlist: %s by %s (%d tracks) in %v\n", playlist.Title, playlist.Creator.Name, playlist.TrackCount, time.Since(apiStart))

	// Create queue item for playlist
	itemID := fmt.Sprintf("playlist_%s", playlistID)
	
	// Check if item already exists
	existingItem, err := m.queueStore.GetByID(itemID)
	if err == nil && existingItem != nil {
		fmt.Printf("[Manager] Playlist already in queue with status: %s\n", existingItem.Status)
		// If it's pending or downloading, return error to notify user
		if existingItem.Status == "pending" || existingItem.Status == "downloading" {
			return fmt.Errorf("playlist already in queue")
		}
		// If it's failed or completed, reset it to pending
		if existingItem.Status == "failed" || existingItem.Status == "completed" {
			existingItem.Status = "pending"
			existingItem.ErrorMessage = ""
			existingItem.RetryCount = 0
			existingItem.TotalTracks = playlist.TrackCount
			existingItem.CompletedTracks = 0
			if err := m.queueStore.Update(existingItem); err != nil {
				fmt.Printf("[Manager] Failed to update existing item: %v\n", err)
				return fmt.Errorf("failed to update queue item: %w", err)
			}
			fmt.Printf("[Manager] Reset existing playlist to pending\n")
		}
	} else {
		// Item doesn't exist, create it
		item := &store.QueueItem{
			ID:              itemID,
			Type:            "playlist",
			Title:           playlist.Title,
			Artist:          "Various Artists",
			Album:           playlist.Title,
			Status:          "pending",
			TotalTracks:     playlist.TrackCount,
			CompletedTracks: 0,
		}

		fmt.Printf("[Manager] Adding playlist to queue with ID: %s, TotalTracks: %d\n", item.ID, item.TotalTracks)
		if err := m.queueStore.Add(item); err != nil {
			fmt.Printf("[Manager] Failed to add to queue: %v\n", err)
			return fmt.Errorf("failed to add to queue: %w", err)
		}
	}

	// Submit playlist job
	job := &Job{
		ID:         itemID,
		Type:       JobTypePlaylist,
		PlaylistID: playlistID,
	}

	fmt.Printf("[Manager] Submitting playlist job to worker pool...\n")
	err = m.workerPool.Submit(job)
	if err != nil {
		fmt.Printf("[Manager] Failed to submit job: %v\n", err)
		return err
	}
	
	fmt.Printf("[Manager] Playlist job submitted successfully\n")
	return nil
}

// PauseDownload pauses a download
func (m *Manager) PauseDownload(itemID string) error {
	m.mu.Lock()
	m.pausedJobs[itemID] = true
	m.mu.Unlock()

	// Cancel the job if it's active
	if err := m.workerPool.CancelJob(itemID); err != nil {
		// Job might not be active, that's okay
	}

	// Update queue item status
	item, err := m.queueStore.GetByID(itemID)
	if err != nil {
		return fmt.Errorf("failed to get queue item: %w", err)
	}

	if item.Status == "downloading" {
		item.Status = "pending"
		if err := m.queueStore.Update(item); err != nil {
			return fmt.Errorf("failed to update queue item: %w", err)
		}
	}

	return nil
}

// ResumeDownload resumes a paused download
func (m *Manager) ResumeDownload(itemID string) error {
	m.mu.Lock()
	delete(m.pausedJobs, itemID)
	m.mu.Unlock()

	// Update queue item status
	item, err := m.queueStore.GetByID(itemID)
	if err != nil {
		return fmt.Errorf("failed to get queue item: %w", err)
	}

	if item.Status != "completed" && item.Status != "downloading" {
		item.Status = "pending"
		if err := m.queueStore.Update(item); err != nil {
			return fmt.Errorf("failed to update queue item: %w", err)
		}
	}

	return nil
}

// CancelDownload cancels a download and removes it from the queue
func (m *Manager) CancelDownload(itemID string) error {
	// Cancel the job if it's active
	if err := m.workerPool.CancelJob(itemID); err != nil {
		// Job might not be active, that's okay
	}

	// Remove from paused jobs
	m.mu.Lock()
	delete(m.pausedJobs, itemID)
	m.mu.Unlock()

	// Delete from queue
	if err := m.queueStore.Delete(itemID); err != nil {
		return fmt.Errorf("failed to delete queue item: %w", err)
	}

	return nil
}

// isJobPaused checks if a job is paused
func (m *Manager) isJobPaused(jobID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pausedJobs[jobID]
}

// buildOutputPath builds the output file path for a track
func (m *Manager) buildOutputPath(track *api.Track) string {
	// Sanitize names
	artist := sanitizeFilename(track.Artist.Name)
	albumArtist := sanitizeFilename(track.AlbumArtist)
	if albumArtist == "" {
		albumArtist = artist // Fallback to track artist if not set
	}
	album := sanitizeFilename(track.Album.Title)
	title := sanitizeFilename(track.Title)
	
	var folderPath string
	var filename string
	
	// Check if this is a playlist download
	if track.Playlist != nil && m.config.Download.CreatePlaylistFolder {
		// Playlist download - use playlist folder structure
		playlistName := sanitizeFilename(track.Playlist.Title)
		
		// Use playlist folder template if configured
		playlistFolderTemplate := m.config.Download.PlaylistFolderTemplate
		if playlistFolderTemplate == "" {
			playlistFolderTemplate = "{playlist}"
		}
		
		// Replace placeholders
		playlistFolder := strings.ReplaceAll(playlistFolderTemplate, "{playlist}", playlistName)
		folderPath = playlistFolder
		
		// Use playlist track template for filename
		playlistTrackTemplate := m.config.Download.PlaylistTrackTemplate
		if playlistTrackTemplate == "" {
			playlistTrackTemplate = "{playlist_position:02d} - {artist} - {title}"
		}
		
		// Get album artist (will be "Various Artists" for playlists in metadata)
		albumArtist := "Various Artists"
		
		// Replace placeholders in filename
		filename = playlistTrackTemplate
		filename = strings.ReplaceAll(filename, "{playlist_position:02d}", fmt.Sprintf("%02d", track.PlaylistPosition))
		filename = strings.ReplaceAll(filename, "{playlist_position}", fmt.Sprintf("%d", track.PlaylistPosition))
		filename = strings.ReplaceAll(filename, "{artist}", artist)
		filename = strings.ReplaceAll(filename, "{album_artist}", albumArtist)
		filename = strings.ReplaceAll(filename, "{title}", title)
		filename = strings.ReplaceAll(filename, "{album}", album)
		filename = strings.ReplaceAll(filename, "{playlist}", playlistName)
		filename = strings.ReplaceAll(filename, "{playlist_name}", playlistName)
		filename += ".mp3"
		
		if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			fmt.Fprintf(logFile, "[%s] Playlist track path: %s (Playlist=%s, Position=%d)\n", 
				time.Now().Format("2006-01-02 15:04:05"), filepath.Join(folderPath, filename), playlistName, track.PlaylistPosition)
			logFile.Close()
		}
	} else {
		// Album or single track download - use album artist/album folder structure
		// This ensures compilations/soundtracks go to "Various Artists" folder
		folderPath = filepath.Join(albumArtist, album)
		
		if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			fmt.Fprintf(logFile, "[%s] Building folder path: AlbumArtist='%s', Album='%s', FolderPath='%s'\n", 
				time.Now().Format("2006-01-02 15:04:05"), albumArtist, album, folderPath)
			logFile.Close()
		}
		
		// Add CD folder for multi-disc albums if enabled
		if m.config.Download.CreateCDFolder && track.IsMultiDiscAlbum && track.DiscNumber > 0 {
			cdFolderTemplate := m.config.Download.CDFolderTemplate
			if cdFolderTemplate == "" {
				cdFolderTemplate = "CD {disc_number}"
			}
			
			cdFolder := strings.ReplaceAll(cdFolderTemplate, "{disc_number}", fmt.Sprintf("%d", track.DiscNumber))
			folderPath = filepath.Join(folderPath, cdFolder)
			
			if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				fmt.Fprintf(logFile, "[%s] Creating CD folder: %s (Album=%s, DiscNumber=%d, IsMultiDisc=%v)\n", 
					time.Now().Format("2006-01-02 15:04:05"), cdFolder, track.Album.ID.String(), track.DiscNumber, track.IsMultiDiscAlbum)
				logFile.Close()
			}
		}
		
		// Build filename using track number if available
		if track.TrackNumber > 0 {
			// Album track format
			filename = fmt.Sprintf("%02d - %s - %s.mp3", track.TrackNumber, artist, title)
		} else {
			// Single track format
			filename = fmt.Sprintf("%s - %s.mp3", artist, title)
		}
	}
	
	// Combine base dir, folder structure, and filename
	fullPath := filepath.Join(m.config.Download.OutputDir, folderPath, filename)
	
	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		// Fallback to flat structure if directory creation fails
		safeFilename := fmt.Sprintf("track_%s.mp3", track.ID)
		fullPath = filepath.Join(m.config.Download.OutputDir, safeFilename)
	}
	
	return fullPath
}

// DiscInfo stores disc information for an album
type DiscInfo struct {
	IsMultiDisc bool
	TotalDiscs  int
}

// Cache for multi-disc album detection to avoid repeated API calls
var multiDiscCache = make(map[string]*DiscInfo)
var multiDiscCacheMu sync.RWMutex

// isAlbumMultiDisc checks if an album has multiple discs
// This uses a cache to avoid repeated API calls
func (m *Manager) isAlbumMultiDisc(albumID string) bool {
	if albumID == "" {
		return false
	}
	
	// Check cache first
	multiDiscCacheMu.RLock()
	if cached, ok := multiDiscCache[albumID]; ok {
		multiDiscCacheMu.RUnlock()
		if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			fmt.Fprintf(logFile, "[%s] isAlbumMultiDisc: Using cached result for album %s: %v\n", time.Now().Format("2006-01-02 15:04:05"), albumID, cached.IsMultiDisc)
			logFile.Close()
		}
		return cached.IsMultiDisc
	}
	multiDiscCacheMu.RUnlock()
	
	// Use a context with timeout to avoid blocking
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Fetch album details
	album, err := m.deezerAPI.GetAlbum(ctx, albumID)
	if err != nil {
		if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
			fmt.Fprintf(logFile, "[%s] isAlbumMultiDisc: Failed to fetch album %s: %v\n", time.Now().Format("2006-01-02 15:04:05"), albumID, err)
			logFile.Close()
		}
		return false
	}
	
	// Method 1: Check nb_disk field from Deezer API
	isMultiDisc := album.DiscCount > 1
	totalDiscs := album.DiscCount
	
	// Method 2: Also check if any track has disc_number > 1 (more reliable)
	if !isMultiDisc && album.Tracks != nil && len(album.Tracks.Data) > 0 {
		for _, track := range album.Tracks.Data {
			if track.DiscNumber > totalDiscs {
				totalDiscs = track.DiscNumber
			}
			if track.DiscNumber > 1 {
				isMultiDisc = true
			}
		}
	}
	
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] isAlbumMultiDisc: Album %s - DiscCount=%d, TotalDiscs=%d, isMultiDisc=%v\n", 
			time.Now().Format("2006-01-02 15:04:05"), albumID, album.DiscCount, totalDiscs, isMultiDisc)
		logFile.Close()
	}
	
	// Cache the result
	multiDiscCacheMu.Lock()
	multiDiscCache[albumID] = &DiscInfo{
		IsMultiDisc: isMultiDisc,
		TotalDiscs:  totalDiscs,
	}
	multiDiscCacheMu.Unlock()
	
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] isAlbumMultiDisc: Album %s result: %v\n", time.Now().Format("2006-01-02 15:04:05"), albumID, isMultiDisc)
		logFile.Close()
	}
	
	return isMultiDisc
}

// sanitizeFilename removes or replaces characters that are invalid in filenames
func sanitizeFilename(name string) string {
	// Replace path separators and other invalid characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"\x00", "",
	)
	
	sanitized := replacer.Replace(name)
	
	// Remove leading/trailing spaces and dots
	sanitized = strings.TrimSpace(sanitized)
	sanitized = strings.Trim(sanitized, ".")
	
	// Ensure filename is not empty
	if sanitized == "" {
		sanitized = "unknown"
	}
	
	return sanitized
}

// GetStats returns download statistics
func (m *Manager) GetStats() (map[string]interface{}, error) {
	queueStats, err := m.queueStore.GetStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}

	return map[string]interface{}{
		"queue_total":       queueStats.Total,
		"queue_pending":     queueStats.Pending,
		"queue_downloading": queueStats.Downloading,
		"queue_completed":   queueStats.Completed,
		"queue_failed":      queueStats.Failed,
		"active_downloads":  m.workerPool.GetActiveJobCount(),
		"max_workers":       m.workerPool.GetMaxWorkers(),
	}, nil
}

// downloadAlbumArtwork downloads the album cover art to the album directory
func (m *Manager) downloadAlbumArtwork(ctx context.Context, album *api.Album, albumDir string) error {
	// Check if artwork file already exists
	artworkPath := filepath.Join(albumDir, "cover.jpg")
	if _, err := os.Stat(artworkPath); err == nil {
		// Artwork already exists, skip download
		return nil
	}

	// Build custom size URL using MD5 hash
	// Format: https://e-cdns-images.dzcdn.net/images/cover/{md5}/{size}x{size}-000000-80-0-0.jpg
	var coverURL string
	if album.MD5Image != "" {
		size := m.config.Download.ArtworkSize
		if size == 0 {
			size = 1200 // Default to 1200 if not set
		}
		coverURL = fmt.Sprintf("https://e-cdns-images.dzcdn.net/images/cover/%s/%dx%d-000000-80-0-0.jpg", 
			album.MD5Image, size, size)
	} else {
		// Fallback to predefined URLs if MD5 not available
		coverURL = album.CoverXL
		if coverURL == "" {
			coverURL = album.CoverBig
		}
		if coverURL == "" {
			coverURL = album.CoverMedium
		}
	}

	if coverURL == "" {
		return fmt.Errorf("no cover art available")
	}

	// Download the artwork
	req, err := http.NewRequestWithContext(ctx, "GET", coverURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create artwork request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download artwork: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("artwork download failed with status: %d", resp.StatusCode)
	}

	// Create the artwork file
	artworkFile, err := os.Create(artworkPath)
	if err != nil {
		return fmt.Errorf("failed to create artwork file: %w", err)
	}
	defer artworkFile.Close()

	// Copy the artwork data
	_, err = io.Copy(artworkFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save artwork: %w", err)
	}

	return nil
}

// downloadPlaylistArtwork downloads the playlist cover art to the playlist directory
func (m *Manager) downloadPlaylistArtwork(ctx context.Context, playlist *api.Playlist, playlistDir string) error {
	// Check if artwork file already exists
	artworkPath := filepath.Join(playlistDir, "cover.jpg")
	if _, err := os.Stat(artworkPath); err == nil {
		// Artwork already exists, skip download
		return nil
	}

	// Build custom size URL using playlist picture
	var coverURL string
	size := m.config.Download.ArtworkSize
	if size == 0 {
		size = 1200 // Default to 1200 if not set
	}

	// Try to extract MD5 from PictureXL URL and build custom size URL
	urlToCheck := playlist.PictureXL
	if urlToCheck == "" {
		urlToCheck = playlist.Picture
	}
	
	if urlToCheck != "" && (strings.Contains(urlToCheck, "cdn-images.dzcdn.net") || strings.Contains(urlToCheck, "e-cdns-images.dzcdn.net")) {
		parts := strings.Split(urlToCheck, "/")
		for i, part := range parts {
			if part == "playlist" && i+1 < len(parts) {
				md5 := parts[i+1]
				// Build custom size URL
				coverURL = fmt.Sprintf("https://e-cdns-images.dzcdn.net/images/playlist/%s/%dx%d-000000-80-0-0.jpg", 
					md5, size, size)
				break
			}
		}
	}

	// Fallback to predefined URLs if custom URL couldn't be built
	if coverURL == "" {
		coverURL = playlist.PictureXL
		if coverURL == "" {
			coverURL = playlist.PictureBig
		}
		if coverURL == "" {
			coverURL = playlist.PictureMedium
		}
		if coverURL == "" {
			coverURL = playlist.Picture
		}
	}

	if coverURL == "" {
		return fmt.Errorf("no playlist cover art available")
	}

	// Download the artwork
	req, err := http.NewRequestWithContext(ctx, "GET", coverURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create playlist artwork request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download playlist artwork: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("playlist artwork download failed with status: %d", resp.StatusCode)
	}

	// Ensure playlist directory exists
	if err := os.MkdirAll(playlistDir, 0755); err != nil {
		return fmt.Errorf("failed to create playlist directory: %w", err)
	}

	// Create the artwork file
	artworkFile, err := os.Create(artworkPath)
	if err != nil {
		return fmt.Errorf("failed to create playlist artwork file: %w", err)
	}
	defer artworkFile.Close()

	// Copy the artwork data
	_, err = io.Copy(artworkFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save playlist artwork: %w", err)
	}

	return nil
}

// downloadArtistImage downloads the artist image to the artist directory
func (m *Manager) downloadArtistImage(ctx context.Context, artist *api.Artist, artistDir string) error {
	// Check if artist image file already exists
	artistImagePath := filepath.Join(artistDir, "folder.jpg")
	if _, err := os.Stat(artistImagePath); err == nil {
		// Artist image already exists, skip download
		return nil
	}

	// Get full artist details to access MD5 hash for custom size URL
	artistID := fmt.Sprintf("%v", artist.ID)
	fullArtist, err := m.deezerAPI.GetArtist(ctx, artistID)
	if err != nil {
		// Fallback to basic artist picture if full details unavailable
		if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
			fmt.Fprintf(logFile, "[%s] Failed to get full artist details for %s: %v, using fallback\n", time.Now().Format("2006-01-02 15:04:05"), artistID, err)
			logFile.Close()
		}
		fullArtist = artist
	}

	// Build custom size URL using MD5 if available
	var pictureURL string
	size := m.config.Download.ArtworkSize
	if size == 0 {
		size = 1200 // Default to 1200 if not set
	}

	// Try to extract MD5 from PictureXL URL and build custom size URL
	// PictureXL format: https://cdn-images.dzcdn.net/images/artist/{md5}/1000x1000-000000-80-0-0.jpg
	urlToCheck := fullArtist.PictureXL
	if urlToCheck == "" {
		urlToCheck = fullArtist.Picture
	}
	
	if urlToCheck != "" && (strings.Contains(urlToCheck, "cdn-images.dzcdn.net") || strings.Contains(urlToCheck, "e-cdns-images.dzcdn.net")) {
		parts := strings.Split(urlToCheck, "/")
		for i, part := range parts {
			if part == "artist" && i+1 < len(parts) {
				md5 := parts[i+1]
				// Build custom size URL - use cdn-images.dzcdn.net (not e-cdns)
				pictureURL = fmt.Sprintf("https://cdn-images.dzcdn.net/images/artist/%s/%dx%d-000000-80-0-0.jpg", 
					md5, size, size)
				break
			}
		}
	}

	// Fallback to predefined URLs if custom URL couldn't be built
	if pictureURL == "" {
		// Try PictureXL first, but it's only 1000x1000
		pictureURL = fullArtist.PictureXL
		if pictureURL == "" {
			pictureURL = fullArtist.PictureBig
		}
		if pictureURL == "" {
			pictureURL = fullArtist.PictureMedium
		}
		if pictureURL == "" {
			pictureURL = fullArtist.Picture
		}
	}

	if pictureURL == "" {
		return fmt.Errorf("no artist picture available")
	}

	// Download the artist image
	req, err := http.NewRequestWithContext(ctx, "GET", pictureURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create artist image request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download artist image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("artist image download failed with status: %d", resp.StatusCode)
	}

	// Ensure artist directory exists
	if err := os.MkdirAll(artistDir, 0755); err != nil {
		return fmt.Errorf("failed to create artist directory: %w", err)
	}

	// Create the artist image file
	artistImageFile, err := os.Create(artistImagePath)
	if err != nil {
		return fmt.Errorf("failed to create artist image file: %w", err)
	}
	defer artistImageFile.Close()

	// Copy the artist image data
	_, err = io.Copy(artistImageFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save artist image: %w", err)
	}

	return nil
}

// StopAll stops all downloads and clears the entire queue
func (m *Manager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Cancel all active jobs in the worker pool
	m.workerPool.CancelAll()

	// Clear all items from the queue
	if err := m.queueStore.ClearAll(); err != nil {
		return fmt.Errorf("failed to clear queue: %w", err)
	}

	return nil
}

// updateParentProgress updates the completed track count for a parent album/playlist
func (m *Manager) updateParentProgress(parentID string) {
	// Get parent item
	parent, err := m.queueStore.GetByID(parentID)
	if err != nil {
		return
	}

	// Count completed child tracks
	completedCount := m.queueStore.CountCompletedChildren(parentID)
	
	// Update parent
	parent.CompletedTracks = completedCount
	if parent.TotalTracks > 0 {
		parent.Progress = (completedCount * 100) / parent.TotalTracks
	}
	
	// Mark parent as completed if all tracks are done
	if completedCount >= parent.TotalTracks && parent.TotalTracks > 0 {
		if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			fmt.Fprintf(logFile, "[%s] Marking album %s as completed: %d/%d tracks\n", time.Now().Format("2006-01-02 15:04:05"), parentID, completedCount, parent.TotalTracks)
			logFile.Close()
		}
		parent.Status = "completed"
		now := time.Now()
		parent.CompletedAt = &now
	} else {
		if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			fmt.Fprintf(logFile, "[%s] Album %s NOT completed yet: %d/%d tracks, Status=%s\n", time.Now().Format("2006-01-02 15:04:05"), parentID, completedCount, parent.TotalTracks, parent.Status)
			logFile.Close()
		}
	}
	
	err = m.queueStore.Update(parent)
	if err != nil {
		if logFile, logErr := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); logErr == nil {
			fmt.Fprintf(logFile, "[%s] ERROR updating parent %s: %v\n", time.Now().Format("2006-01-02 15:04:05"), parentID, err)
			logFile.Close()
		}
	} else {
		if logFile, logErr := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); logErr == nil {
			fmt.Fprintf(logFile, "[%s] Successfully updated parent %s in database, Status=%s, Progress=%d\n", time.Now().Format("2006-01-02 15:04:05"), parentID, parent.Status, parent.Progress)
			logFile.Close()
		}
	}
	
	// Notify progress update for parent
	if m.notifier != nil {
		m.notifier.NotifyProgress(parentID, parent.Progress, int64(completedCount), int64(parent.TotalTracks))
		
		// If parent just completed, also send status notification
		if parent.Status == "completed" {
			m.notifier.NotifyCompleted(parentID)
		}
	}
}

// applyMetadataTags applies metadata tags to a downloaded audio file
func (m *Manager) applyMetadataTags(ctx context.Context, filePath string, track *api.Track) error {
	// Nil checks
	if track == nil {
		return fmt.Errorf("track is nil")
	}
	if track.Artist == nil || track.Album == nil {
		return fmt.Errorf("track artist or album is nil")
	}

	// Create metadata manager
	metadataManager := metadata.NewManager(&metadata.Config{
		EmbedArtwork: m.config.Download.EmbedArtwork,
		ArtworkSize:  1200,
	})

	// Prepare metadata with safe access
	// Use album artist if available, otherwise fall back to track artist
	albumArtist := track.Artist.Name // Default to track artist
	albumTitle := track.Album.Title
	trackNumber := track.TrackNumber
	discNumber := track.DiscNumber
	totalDiscs := track.TotalDiscs
	
	// Debug log album record type
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] Album RecordType check: Album='%s', RecordType='%s'\n", 
			time.Now().Format("2006-01-02 15:04:05"), albumTitle, track.Album.RecordType)
		logFile.Close()
	}
	
	// For playlist downloads, use "Various Artists" as album artist and playlist name as album
	if track.Playlist != nil {
		albumArtist = "Various Artists"
		albumTitle = track.Playlist.Title
		trackNumber = track.PlaylistPosition // Use playlist position as track number
		discNumber = 0                        // No disc number for playlists
		totalDiscs = 0                        // No total discs for playlists
		
		if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			fmt.Fprintf(logFile, "[%s] Playlist track metadata: Album=%s, AlbumArtist=%s, TrackNumber=%d (playlist position)\n", 
				time.Now().Format("2006-01-02 15:04:05"), albumTitle, albumArtist, trackNumber)
			logFile.Close()
		}
	} else if track.Album.RecordType != "single" && track.Album.RecordType != "ep" &&
	          (track.Album.RecordType == "compilation" || 
	           strings.Contains(strings.ToLower(albumTitle), "soundtrack") ||
	           strings.Contains(strings.ToLower(albumTitle), "original score") ||
	           strings.Contains(strings.ToLower(albumTitle), "original motion picture")) {
		// For compilations and soundtracks, use "Various Artists" as album artist
		// Exclude singles and EPs even if they have "soundtrack" in the name
		albumArtist = "Various Artists"
		
		if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			fmt.Fprintf(logFile, "[%s] Compilation/Soundtrack detected: Album='%s', RecordType='%s', using AlbumArtist=Various Artists\n", 
				time.Now().Format("2006-01-02 15:04:05"), albumTitle, track.Album.RecordType)
			logFile.Close()
		}
	} else if track.Album.Artist != nil && track.Album.Artist.Name != "" {
		albumArtist = track.Album.Artist.Name
	}

	// Build artist string with featured artists
	// Artist field should include featured artists: "Main Artist feat. Featured Artist"
	// Album Artist should remain just the main artist (or "Various Artists" for playlists)
	artistName := buildArtistString(track)

	trackMetadata := &metadata.TrackMetadata{
		Title:       track.Title,
		Artist:      artistName,
		Album:       albumTitle,
		AlbumArtist: albumArtist,
		TrackNumber: trackNumber,
		DiscNumber:  discNumber,
		TotalDiscs:  totalDiscs,
		Year:        extractYear(track.Album.ReleaseDate),
		Genre:       "", // Deezer doesn't provide genre in track API
		Duration:    track.Duration,
		ISRC:        track.ISRC,
		Label:       track.Album.Label,
		Copyright:   "", // Not available in API
	}

	// Debug log metadata values
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] Metadata: Artist=%s, AlbumArtist=%s, DiscNumber=%d/%d, TrackNumber=%d\n", 
			time.Now().Format("2006-01-02 15:04:05"), trackMetadata.Artist, trackMetadata.AlbumArtist, trackMetadata.DiscNumber, trackMetadata.TotalDiscs, trackMetadata.TrackNumber)
		logFile.Close()
	}

	// Download and embed artwork if enabled
	if m.config.Download.EmbedArtwork && track.Album != nil && track.Album.CoverXL != "" {
		// Get high-resolution artwork URL (1200x1200)
		artworkURL := getHighResArtworkURL(track.Album.CoverXL, m.config.Download.ArtworkSize)
		artworkData, mimeType, err := m.downloadArtworkData(ctx, artworkURL)
		if err == nil {
			trackMetadata.ArtworkData = artworkData
			trackMetadata.ArtworkMIME = mimeType
		}
	}

	// Apply metadata to file
	return metadataManager.ApplyMetadata(filePath, trackMetadata)
}

// downloadArtworkData downloads artwork and returns the raw data
func (m *Manager) downloadArtworkData(ctx context.Context, artworkURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", artworkURL, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("artwork download failed with status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg" // Default to JPEG
	}

	return data, mimeType, nil
}

// extractYear extracts the year from a date string (YYYY-MM-DD format)
func extractYear(dateStr string) int {
	if len(dateStr) >= 4 {
		if year, err := strconv.Atoi(dateStr[:4]); err == nil {
			return year
		}
	}
	return 0
}

// downloadAndSaveLyrics downloads and saves lyrics for a track
func (m *Manager) downloadAndSaveLyrics(ctx context.Context, audioFilePath string, track *api.Track) error {
	// Get lyrics from API
	lyrics, err := m.deezerAPI.GetLyrics(ctx, track.ID.String())
	if err != nil {
		return fmt.Errorf("failed to get lyrics: %w", err)
	}

	// Check if synced lyrics are available
	if lyrics.SyncedLyrics == "" {
		return nil // No lyrics available, not an error
	}

	// Determine lyrics file path (same directory and name as audio file, but with .lrc extension)
	lyricsPath := strings.TrimSuffix(audioFilePath, filepath.Ext(audioFilePath)) + ".lrc"

	// Write lyrics to file
	if err := os.WriteFile(lyricsPath, []byte(lyrics.SyncedLyrics), 0644); err != nil {
		return fmt.Errorf("failed to write lyrics file: %w", err)
	}

	return nil
}

// getHighResArtworkURL modifies a Deezer cover URL to request a specific size
func getHighResArtworkURL(coverURL string, size int) string {
	// Deezer cover URLs are in format: https://e-cdns-images.dzcdn.net/images/cover/{hash}/{size}x{size}.jpg
	// We can replace the size parameter to get higher resolution
	// Default CoverXL is 1000x1000, but we can request up to 1500x1500
	
	if size <= 0 {
		size = 1200 // Default to 1200x1200
	}
	
	// Replace the size in the URL
	// CoverXL format: https://e-cdns-images.dzcdn.net/images/cover/{hash}/1000x1000-000000-80-0-0.jpg
	// We want: https://e-cdns-images.dzcdn.net/images/cover/{hash}/1200x1200-000000-80-0-0.jpg
	coverURL = strings.Replace(coverURL, "1000x1000", fmt.Sprintf("%dx%d", size, size), 1)
	
	return coverURL
}

// buildArtistString builds the artist string including featured artists
// Returns "Main Artist feat. Featured Artist 1, Featured Artist 2" format
func buildArtistString(track *api.Track) string {
	if track == nil || track.Artist == nil {
		return "Unknown Artist"
	}

	mainArtist := track.Artist.Name
	
	// If no contributors, just return main artist
	if len(track.Contributors) == 0 {
		return mainArtist
	}

	// Find featured artists from contributors
	// Contributors with role "Featured" or who are not the main artist
	var featuredArtists []string
	mainArtistID := track.Artist.ID.String()
	
	for _, contributor := range track.Contributors {
		if contributor == nil {
			continue
		}
		
		contributorID := contributor.ID.String()
		
		// Skip the main artist
		if contributorID == mainArtistID {
			continue
		}
		
		// Include artists with "Featured" role or any non-main artist
		// Deezer uses roles like "Main", "Featured", etc.
		if contributor.Role == "Featured" || contributor.Role == "" {
			featuredArtists = append(featuredArtists, contributor.Name)
		}
	}

	// If no featured artists found, return main artist only
	if len(featuredArtists) == 0 {
		return mainArtist
	}

	// Build the artist string: "Main Artist feat. Featured1, Featured2"
	featuredString := strings.Join(featuredArtists, ", ")
	return fmt.Sprintf("%s feat. %s", mainArtist, featuredString)
}
