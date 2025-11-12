package main

/*
#include <stdlib.h>

// Callback function types for C# interop
typedef void (*ProgressCallback)(char* itemID, int progress, long long bytesProcessed, long long totalBytes);
typedef void (*StatusCallback)(char* itemID, char* status, char* errorMsg);
typedef void (*QueueUpdateCallback)(char* statsJson);

// Helper functions to call function pointers
static inline void call_progress_callback(ProgressCallback cb, char* itemID, int progress, long long bytesProcessed, long long totalBytes) {
	if (cb != NULL) {
		cb(itemID, progress, bytesProcessed, totalBytes);
	}
}

static inline void call_status_callback(StatusCallback cb, char* itemID, char* status, char* errorMsg) {
	if (cb != NULL) {
		cb(itemID, status, errorMsg);
	}
}

static inline void call_queue_update_callback(QueueUpdateCallback cb, char* statsJson) {
	if (cb != NULL) {
		cb(statsJson);
	}
}
*/
import "C"
import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/deemusic/deemusic-go/internal/api"
	"github.com/deemusic/deemusic-go/internal/config"
	"github.com/deemusic/deemusic-go/internal/download"
	"github.com/deemusic/deemusic-go/internal/migration"
	"github.com/deemusic/deemusic-go/internal/store"
	_ "github.com/mattn/go-sqlite3"
)

// Global state for the DLL
var (
	ctx          context.Context
	cancel       context.CancelFunc
	downloadMgr  *download.Manager
	deezerAPI    *api.DeezerClient
	queueStore   *store.QueueStore
	cfg          *config.Config
	db           *sql.DB
	initialized  bool
	mu           sync.RWMutex
	debugLog     *os.File
	shutdownFlag bool // Flag to track if shutdown was intentional
	
	// Callbacks
	progressCb     C.ProgressCallback
	statusCb       C.StatusCallback
	queueUpdateCb  C.QueueUpdateCallback
	callbackMu     sync.RWMutex
)

func logDebug(format string, args ...interface{}) {
	if debugLog != nil {
		fmt.Fprintf(debugLog, "[%s] ", time.Now().Format("2006-01-02 15:04:05.000"))
		fmt.Fprintf(debugLog, format, args...)
		fmt.Fprintln(debugLog)
		debugLog.Sync()
	}
	// Also to stderr
	fmt.Fprintf(os.Stderr, format, args...)
	fmt.Fprintln(os.Stderr)
}

// CallbackNotifier implements the Notifier interface using C callbacks
type CallbackNotifier struct{}

func (n *CallbackNotifier) NotifyProgress(itemID string, progress int, bytesProcessed, totalBytes int64) {
	callbackMu.RLock()
	cb := progressCb
	callbackMu.RUnlock()
	
	if cb != nil {
		cItemID := C.CString(itemID)
		defer C.free(unsafe.Pointer(cItemID))
		
		// Call the callback function pointer
		C.call_progress_callback(cb, cItemID, C.int(progress), C.longlong(bytesProcessed), C.longlong(totalBytes))
	}
}

func (n *CallbackNotifier) NotifyStarted(itemID string) {
	callbackMu.RLock()
	cb := statusCb
	callbackMu.RUnlock()
	
	if cb != nil {
		cItemID := C.CString(itemID)
		cStatus := C.CString("started")
		defer C.free(unsafe.Pointer(cItemID))
		defer C.free(unsafe.Pointer(cStatus))
		
		// Call the callback function pointer
		C.call_status_callback(cb, cItemID, cStatus, nil)
	}
}

func (n *CallbackNotifier) NotifyCompleted(itemID string) {
	callbackMu.RLock()
	cb := statusCb
	callbackMu.RUnlock()
	
	if cb != nil {
		cItemID := C.CString(itemID)
		cStatus := C.CString("completed")
		defer C.free(unsafe.Pointer(cItemID))
		defer C.free(unsafe.Pointer(cStatus))
		
		// Call the callback function pointer
		C.call_status_callback(cb, cItemID, cStatus, nil)
	}
	
	// Also trigger queue update
	n.notifyQueueUpdate()
}

func (n *CallbackNotifier) NotifyFailed(itemID string, err error) {
	callbackMu.RLock()
	cb := statusCb
	callbackMu.RUnlock()
	
	if cb != nil {
		cItemID := C.CString(itemID)
		cStatus := C.CString("failed")
		cError := C.CString(err.Error())
		defer C.free(unsafe.Pointer(cItemID))
		defer C.free(unsafe.Pointer(cStatus))
		defer C.free(unsafe.Pointer(cError))
		
		// Call the callback function pointer
		C.call_status_callback(cb, cItemID, cStatus, cError)
	}
	
	// Also trigger queue update
	n.notifyQueueUpdate()
}

func (n *CallbackNotifier) notifyQueueUpdate() {
	callbackMu.RLock()
	cb := queueUpdateCb
	callbackMu.RUnlock()
	
	if cb != nil && queueStore != nil {
		stats, err := queueStore.GetStats()
		if err == nil {
			statsJSON, _ := json.Marshal(stats)
			cStats := C.CString(string(statsJSON))
			defer C.free(unsafe.Pointer(cStats))
			
			// Call the callback function pointer
			C.call_queue_update_callback(cb, cStats)
		}
	}
}

//export InitializeApp
func InitializeApp(configPath *C.char) C.int {
	// Add panic recovery to prevent crashes
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "[PANIC] InitializeApp panicked: %v\n", r)
			if debugLog != nil {
				fmt.Fprintf(debugLog, "[%s] [PANIC] InitializeApp panicked: %v\n", time.Now().Format("2006-01-02 15:04:05"), r)
				// Log stack trace
				for i := 0; i < 20; i++ {
					pc, file, line, ok := runtime.Caller(i)
					if !ok {
						break
					}
					fn := runtime.FuncForPC(pc)
					fmt.Fprintf(debugLog, "  %s:%d %s\n", file, line, fn.Name())
				}
				debugLog.Sync()
			}
		}
	}()
	
	mu.Lock()
	defer mu.Unlock()
	
	// Open debug log file
	if debugLog == nil {
		dataDir := config.GetDataDir()
		logPath := filepath.Join(dataDir, "logs", "go-backend.log")
		os.MkdirAll(filepath.Dir(logPath), 0755)
		var err error
		debugLog, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to open debug log: %v\n", err)
		} else {
			logDebug("=== DeeMusic Go Backend Log Started ===")
		}
	}
	
	if initialized {
		logDebug("[WARN] Backend already initialized - returning success without reinitializing")
		logDebug("[WARN] If you need to reinitialize, call ShutdownApp first")
		return 0 // Already initialized - don't create a new context!
	}
	
	logDebug("[INFO] Initializing DeeMusic backend...")
	
	// Create context with no timeout - this should live for the entire application lifetime
	ctx, cancel = context.WithCancel(context.Background())
	logDebug("Created application context (should never be cancelled until shutdown)")
	
	// Load configuration
	goConfigPath := C.GoString(configPath)
	fmt.Fprintf(os.Stderr, "[INFO] Loading configuration from: %s\n", goConfigPath)
	var err error
	cfg, err = config.Load(goConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to load config: %v\n", err)
		return -3 // Invalid configuration
	}
	
	// Initialize database
	dataDir := config.GetDataDir()
	dbPath := filepath.Join(dataDir, "data", "queue.db")
	fmt.Fprintf(os.Stderr, "[INFO] Database path: %s\n", dbPath)
	
	// Log to debug file
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] Database path: %s\n", time.Now().Format("2006-01-02 15:04:05"), dbPath)
		fmt.Fprintf(logFile, "[%s] Data directory: %s\n", time.Now().Format("2006-01-02 15:04:05"), dataDir)
		fmt.Fprintf(logFile, "[%s] Portable mode: %v\n", time.Now().Format("2006-01-02 15:04:05"), config.IsPortableMode())
		logFile.Close()
	}
	
	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to create data directory: %v\n", err)
		return -9 // File system error
	}
	
	// Open database with optimizations
	// Increased busy_timeout to 30 seconds to handle high concurrency
	db, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=30000&_synchronous=NORMAL&cache=shared")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to open database: %v\n", err)
		return -4 // Database error
	}
	
	// Set connection pool settings for better concurrency
	// Increased to handle more concurrent workers
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)
	
	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to ping database: %v\n", err)
		return -4 // Database error
	}
	
	// Run migrations
	fmt.Fprintf(os.Stderr, "[INFO] Running database migrations...\n")
	if err := store.RunMigrations(db); err != nil {
		db.Close()
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to run migrations: %v\n", err)
		return -5 // Migration failed
	}
	
	// Initialize components
	queueStore = store.NewQueueStore(db)
	deezerAPI = api.NewDeezerClient(30 * time.Second)
	
	// Authenticate with Deezer
	logDebug("Checking Deezer ARL configuration...")
	if cfg.Deezer.ARL != "" {
		logDebug("ARL found (length: %d), authenticating with Deezer...", len(cfg.Deezer.ARL))
		fmt.Fprintf(os.Stderr, "[INFO] Authenticating with Deezer...\n")
		if err := deezerAPI.Authenticate(context.Background(), cfg.Deezer.ARL); err != nil {
			logDebug("Deezer authentication FAILED: %v", err)
			fmt.Fprintf(os.Stderr, "[WARN] Failed to authenticate with Deezer: %v\n", err)
			// Continue anyway, user can set ARL later
		} else {
			logDebug("Deezer authentication SUCCESSFUL")
			fmt.Fprintf(os.Stderr, "[INFO] Deezer authentication successful\n")
		}
	} else {
		logDebug("No Deezer ARL configured!")
		fmt.Fprintf(os.Stderr, "[WARN] No Deezer ARL configured\n")
	}
	
	// Create download manager with callback notifier
	notifier := &CallbackNotifier{}
	downloadMgr = download.NewManager(cfg, queueStore, deezerAPI, notifier)
	
	// Start download manager with application-lifetime context
	fmt.Fprintf(os.Stderr, "[INFO] Starting download manager...\n")
	logDebug("Starting download manager with application-lifetime context...")
	if err := downloadMgr.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to start download manager: %v\n", err)
		logDebug("Download manager start FAILED: %v", err)
		return -6 // Failed to start download manager
	}
	logDebug("Download manager started successfully with context that will live until shutdown")
	fmt.Fprintf(os.Stderr, "[INFO] Download manager started successfully\n")
	
	// Add a goroutine to monitor context cancellation (for debugging)
	// This goroutine will also attempt to recover from panics
	fmt.Fprintf(os.Stderr, "[DEBUG] About to start context monitor goroutine\n")
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logDebug("[PANIC RECOVERY] Context monitor goroutine panicked: %v", r)
				fmt.Fprintf(os.Stderr, "[PANIC RECOVERY] Context monitor goroutine panicked: %v\n", r)
			}
		}()
		
		fmt.Fprintf(os.Stderr, "[DEBUG] Context monitor goroutine RUNNING, waiting for ctx.Done()...\n")
		<-ctx.Done()
		
		// Check if this was an intentional shutdown
		mu.RLock()
		wasIntentional := shutdownFlag
		mu.RUnlock()
		
		if wasIntentional {
			fmt.Fprintf(os.Stderr, "[INFO] Application context cancelled during intentional shutdown\n")
			logDebug("[INFO] Application context cancelled during intentional shutdown")
		} else {
			fmt.Fprintf(os.Stderr, "[CRITICAL] ===== APPLICATION CONTEXT WAS CANCELLED UNEXPECTEDLY ===== Reason: %v\n", ctx.Err())
			logDebug("[CRITICAL] UNEXPECTED CONTEXT CANCELLATION! Reason: %v", ctx.Err())
			
			// Log stack trace to see what cancelled the context
			logDebug("[CRITICAL] Stack trace at unexpected context cancellation:")
			for i := 0; i < 20; i++ {
				pc, file, line, ok := runtime.Caller(i)
				if !ok {
					break
				}
				fn := runtime.FuncForPC(pc)
				logDebug("  %s:%d %s", file, line, fn.Name())
			}
			
			// Log to debug file
			if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				fmt.Fprintf(logFile, "[%s] [CRITICAL] UNEXPECTED CONTEXT CANCELLATION! Reason: %v\n", time.Now().Format("2006-01-02 15:04:05"), ctx.Err())
				fmt.Fprintf(logFile, "[%s] [CRITICAL] This indicates a bug - context should only be cancelled during explicit shutdown!\n", time.Now().Format("2006-01-02 15:04:05"))
				logFile.Close()
			}
		}
	}()
	fmt.Fprintf(os.Stderr, "[DEBUG] Context monitor goroutine started\n")
	
	initialized = true
	fmt.Fprintf(os.Stderr, "[INFO] Backend initialized successfully\n")
	return 0
}

//export ShutdownApp
func ShutdownApp() {
	mu.Lock()
	defer mu.Unlock()
	
	if !initialized {
		logDebug("[WARN] Shutdown called but backend not initialized")
		fmt.Fprintf(os.Stderr, "[WARN] Shutdown called but backend not initialized\n")
		return
	}
	
	// Set shutdown flag to indicate this is intentional
	shutdownFlag = true
	
	logDebug("[INFO] ===== SHUTTING DOWN BACKEND =====")
	logDebug("[INFO] Shutdown requested at %s", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(os.Stderr, "[INFO] Shutting down backend...\n")
	
	// Log stack trace to see who's calling shutdown
	logDebug("[INFO] Shutdown call stack:")
	for i := 0; i < 10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		logDebug("  %s:%d %s", file, line, fn.Name())
	}
	
	// Stop download manager
	if downloadMgr != nil {
		logDebug("[INFO] Stopping download manager...")
		fmt.Fprintf(os.Stderr, "[INFO] Stopping download manager...\n")
		downloadMgr.Stop()
	}
	
	// Close database
	if db != nil {
		logDebug("[INFO] Closing database...")
		fmt.Fprintf(os.Stderr, "[INFO] Closing database...\n")
		db.Close()
	}
	
	// Cancel context ONLY during intentional shutdown
	if cancel != nil {
		logDebug("[INFO] Cancelling application context (intentional shutdown)...")
		cancel()
	}
	
	initialized = false
	logDebug("[INFO] Backend shutdown complete at %s", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(os.Stderr, "[INFO] Backend shutdown complete\n")
}

//export SetProgressCallback
func SetProgressCallback(callback C.ProgressCallback) {
	callbackMu.Lock()
	progressCb = callback
	callbackMu.Unlock()
}

//export SetStatusCallback
func SetStatusCallback(callback C.StatusCallback) {
	callbackMu.Lock()
	statusCb = callback
	callbackMu.Unlock()
}

//export SetQueueUpdateCallback
func SetQueueUpdateCallback(callback C.QueueUpdateCallback) {
	callbackMu.Lock()
	queueUpdateCb = callback
	callbackMu.Unlock()
}

//export FreeString
func FreeString(str *C.char) {
	C.free(unsafe.Pointer(str))
}

// Helper function to check if initialized
func checkInitialized() bool {
	mu.RLock()
	defer mu.RUnlock()
	return initialized
}

// Required for DLL compilation
func main() {}

//export Search
func Search(query *C.char, searchType *C.char, limit C.int) *C.char {
	if !checkInitialized() {
		logDebug("Search: Backend not initialized")
		fmt.Fprintf(os.Stderr, "[ERROR] Search called but backend not initialized\n")
		return C.CString(`{"error": "Backend not initialized"}`)
	}
	
	goQuery := C.GoString(query)
	goSearchType := C.GoString(searchType)
	goLimit := int(limit)
	
	if goLimit <= 0 {
		goLimit = 50
	}
	
	logDebug("Search called: query='%s', type='%s', limit=%d", goQuery, goSearchType, goLimit)
	fmt.Fprintf(os.Stderr, "[INFO] Search: query='%s', type='%s', limit=%d\n", goQuery, goSearchType, goLimit)
	
	var results interface{}
	var err error
	
	switch goSearchType {
	case "track":
		results, err = deezerAPI.SearchTracks(ctx, goQuery, goLimit)
	case "album":
		results, err = deezerAPI.SearchAlbums(ctx, goQuery, goLimit)
	case "artist":
		results, err = deezerAPI.SearchArtists(ctx, goQuery, goLimit)
	case "playlist":
		results, err = deezerAPI.SearchPlaylists(ctx, goQuery, goLimit)
	default:
		results, err = deezerAPI.SearchTracks(ctx, goQuery, goLimit)
	}
	
	if err != nil {
		logDebug("Search failed: %v", err)
		fmt.Fprintf(os.Stderr, "[ERROR] Search failed: %v\n", err)
		errJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		return C.CString(string(errJSON))
	}
	
	// Wrap results in SearchResponse format expected by C#
	// C# expects: {"data": [...], "total": N}
	var total int
	switch v := results.(type) {
	case []*api.Track:
		total = len(v)
		logDebug("Search returned %d tracks", total)
	case []*api.Album:
		total = len(v)
		logDebug("Search returned %d albums", total)
	case []*api.Artist:
		total = len(v)
		logDebug("Search returned %d artists", total)
	case []*api.Playlist:
		total = len(v)
		logDebug("Search returned %d playlists", total)
	}
	
	response := map[string]interface{}{
		"data":  results,
		"total": total,
	}
	
	jsonData, err := json.Marshal(response)
	if err != nil {
		logDebug("Failed to marshal search results: %v", err)
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to marshal search results: %v\n", err)
		errJSON, _ := json.Marshal(map[string]string{"error": "Failed to marshal results"})
		return C.CString(string(errJSON))
	}
	
	logDebug("Search completed successfully, returning %d results (JSON length: %d)", total, len(jsonData))
	fmt.Fprintf(os.Stderr, "[INFO] Search completed successfully, returning %d results\n", total)
	return C.CString(string(jsonData))
}

//export GetAlbum
func GetAlbum(albumID *C.char) *C.char {
	if !checkInitialized() {
		return C.CString(`{"error": "not initialized"}`)
	}
	
	goAlbumID := C.GoString(albumID)
	
	album, err := deezerAPI.GetAlbum(ctx, goAlbumID)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		return C.CString(string(errJSON))
	}
	
	jsonData, err := json.Marshal(album)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": "failed to marshal album"})
		return C.CString(string(errJSON))
	}
	
	return C.CString(string(jsonData))
}

//export GetArtist
func GetArtist(artistID *C.char) *C.char {
	if !checkInitialized() {
		return C.CString(`{"error": "not initialized"}`)
	}
	
	goArtistID := C.GoString(artistID)
	
	artist, err := deezerAPI.GetArtist(ctx, goArtistID)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		return C.CString(string(errJSON))
	}
	
	jsonData, err := json.Marshal(artist)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": "failed to marshal artist"})
		return C.CString(string(errJSON))
	}
	
	return C.CString(string(jsonData))
}

//export GetArtistAlbums
func GetArtistAlbums(artistID *C.char, limit C.int) *C.char {
	if !checkInitialized() {
		return C.CString(`{"error": "not initialized"}`)
	}
	
	goArtistID := C.GoString(artistID)
	goLimit := int(limit)
	if goLimit <= 0 {
		goLimit = 100 // Default to 100 to get all albums
	}
	
	logDebug("GetArtistAlbums called: artistID=%s, limit=%d", goArtistID, goLimit)
	
	albums, err := deezerAPI.GetArtistAlbums(ctx, goArtistID, goLimit)
	if err != nil {
		logDebug("GetArtistAlbums error: %v", err)
		errJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		return C.CString(string(errJSON))
	}
	
	logDebug("GetArtistAlbums returned %d albums", len(albums))
	for i, album := range albums {
		if i < 5 { // Log first 5 albums
			logDebug("  Album %d: %s (RecordType: %s)", i+1, album.Title, album.RecordType)
		}
	}
	
	jsonData, err := json.Marshal(albums)
	if err != nil {
		logDebug("GetArtistAlbums marshal error: %v", err)
		errJSON, _ := json.Marshal(map[string]string{"error": "failed to marshal albums"})
		return C.CString(string(errJSON))
	}
	
	return C.CString(string(jsonData))
}

//export GetPlaylist
func GetPlaylist(playlistID *C.char) *C.char {
	if !checkInitialized() {
		return C.CString(`{"error": "not initialized"}`)
	}
	
	goPlaylistID := C.GoString(playlistID)
	
	playlist, err := deezerAPI.GetPlaylist(ctx, goPlaylistID)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		return C.CString(string(errJSON))
	}
	
	jsonData, err := json.Marshal(playlist)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": "failed to marshal playlist"})
		return C.CString(string(errJSON))
	}
	
	return C.CString(string(jsonData))
}

//export GetCharts
func GetCharts(limit C.int) *C.char {
	if !checkInitialized() {
		return C.CString(`{"error": "not initialized"}`)
	}
	
	goLimit := int(limit)
	if goLimit <= 0 {
		goLimit = 25
	}
	
	charts, err := deezerAPI.GetChart(ctx, goLimit)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		return C.CString(string(errJSON))
	}
	
	jsonData, err := json.Marshal(charts)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": "failed to marshal charts"})
		return C.CString(string(errJSON))
	}
	
	return C.CString(string(jsonData))
}

//export GetEditorialReleases
func GetEditorialReleases(limit C.int) *C.char {
	if !checkInitialized() {
		return C.CString(`{"error": "not initialized"}`)
	}
	
	goLimit := int(limit)
	if goLimit <= 0 {
		goLimit = 25
	}
	
	releases, err := deezerAPI.GetEditorialReleases(ctx, goLimit)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		return C.CString(string(errJSON))
	}
	
	// Wrap in a data structure
	response := map[string]interface{}{
		"data": releases,
	}
	
	jsonData, err := json.Marshal(response)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": "failed to marshal releases"})
		return C.CString(string(errJSON))
	}
	
	return C.CString(string(jsonData))
}

//export DownloadTrack
func DownloadTrack(trackID *C.char, quality *C.char) C.int {
	if !checkInitialized() {
		fmt.Fprintf(os.Stderr, "[ERROR] DownloadTrack called but backend not initialized\n")
		return -1
	}
	
	goTrackID := C.GoString(trackID)
	
	// Update quality in config if provided
	if quality != nil {
		goQuality := C.GoString(quality)
		if goQuality != "" {
			cfg.Download.Quality = goQuality
			fmt.Fprintf(os.Stderr, "[INFO] Quality set to: %s\n", goQuality)
		}
	}
	
	fmt.Fprintf(os.Stderr, "[INFO] Downloading track: %s\n", goTrackID)
	err := downloadMgr.DownloadTrack(ctx, goTrackID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to download track %s: %v\n", goTrackID, err)
		return -2
	}
	
	fmt.Fprintf(os.Stderr, "[INFO] Track %s added to download queue\n", goTrackID)
	return 0
}

//export DownloadAlbum
func DownloadAlbum(albumID *C.char, quality *C.char) C.int {
	if !checkInitialized() {
		logDebug("DownloadAlbum: Backend not initialized")
		return -1
	}
	
	goAlbumID := C.GoString(albumID)
	
	// Log the album ID being downloaded
	logDebug("DownloadAlbum called with ID: '%s'", goAlbumID)
	
	if goAlbumID == "" {
		logDebug("DownloadAlbum: Album ID is empty!")
		return -3
	}
	
	// Update quality in config if provided
	if quality != nil {
		goQuality := C.GoString(quality)
		if goQuality != "" {
			cfg.Download.Quality = goQuality
			logDebug("DownloadAlbum: Quality set to %s", goQuality)
		}
	}
	
	logDebug("DownloadAlbum: Calling downloadMgr.DownloadAlbum...")
	err := downloadMgr.DownloadAlbum(ctx, goAlbumID)
	if err != nil {
		logDebug("DownloadAlbum: Failed to download album %s: %v", goAlbumID, err)
		// Check if it's a duplicate album error
		if strings.Contains(err.Error(), "already in queue") {
			return -15 // Specific error code for duplicate
		}
		return -2
	}
	
	logDebug("DownloadAlbum: Album %s download initiated successfully", goAlbumID)
	return 0
}

//export DownloadPlaylist
func DownloadPlaylist(playlistID *C.char, quality *C.char) C.int {
	if !checkInitialized() {
		return -1
	}
	
	goPlaylistID := C.GoString(playlistID)
	
	// Update quality in config if provided
	if quality != nil {
		goQuality := C.GoString(quality)
		if goQuality != "" {
			cfg.Download.Quality = goQuality
		}
	}
	
	err := downloadMgr.DownloadPlaylist(ctx, goPlaylistID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to download playlist: %v\n", err)
		return -2
	}
	
	return 0
}

//export DownloadCustomPlaylist
func DownloadCustomPlaylist(playlistJSON *C.char, quality *C.char) C.int {
	if !checkInitialized() {
		return -1
	}
	
	goPlaylistJSON := C.GoString(playlistJSON)
	
	// Update quality in config if provided
	if quality != nil {
		goQuality := C.GoString(quality)
		if goQuality != "" {
			cfg.Download.Quality = goQuality
		}
	}
	
	err := downloadMgr.DownloadCustomPlaylist(ctx, goPlaylistJSON)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to download custom playlist: %v\n", err)
		return -2
	}
	
	return 0
}

//export ConvertSpotifyURL
func ConvertSpotifyURL(url *C.char) *C.char {
	if !checkInitialized() {
		return C.CString(`{"error": "not initialized"}`)
	}
	
	goURL := C.GoString(url)
	
	// Check if Spotify credentials are configured
	if cfg.Spotify.ClientID == "" || cfg.Spotify.ClientSecret == "" {
		errJSON, _ := json.Marshal(map[string]string{
			"error": "Spotify API credentials not configured",
		})
		return C.CString(string(errJSON))
	}
	
	// Create Spotify client
	spotifyClient := api.NewSpotifyClient(cfg.Spotify.ClientID, cfg.Spotify.ClientSecret, 30*time.Second)
	
	// Authenticate
	if err := spotifyClient.Authenticate(ctx); err != nil {
		errJSON, _ := json.Marshal(map[string]string{
			"error": fmt.Sprintf("Spotify authentication failed: %v", err),
		})
		return C.CString(string(errJSON))
	}
	
	// Create converter
	converter := api.NewSpotifyConverter(spotifyClient, deezerAPI)
	
	// Convert playlist
	result, err := converter.ConvertPlaylist(ctx, goURL)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{
			"error": fmt.Sprintf("Conversion failed: %v", err),
		})
		return C.CString(string(errJSON))
	}
	
	// Marshal result to JSON
	jsonData, err := json.Marshal(result)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{
			"error": "failed to marshal conversion result",
		})
		return C.CString(string(errJSON))
	}
	
	return C.CString(string(jsonData))
}

//export GetQueue
func GetQueue(offset C.int, limit C.int, filter *C.char) *C.char {
	if !checkInitialized() {
		logDebug("GetQueue: Backend not initialized")
		return C.CString(`{"error": "not initialized"}`)
	}
	
	goOffset := int(offset)
	goLimit := int(limit)
	goFilter := ""
	if filter != nil {
		goFilter = C.GoString(filter)
	}
	
	logDebug("GetQueue called: offset=%d, limit=%d, filter='%s'", goOffset, goLimit, goFilter)
	
	// Default limit
	if goLimit <= 0 {
		goLimit = 100
	}
	
	// Enforce maximum limit of 1000 items to prevent memory issues
	if goLimit > 1000 {
		goLimit = 1000
	}
	
	var items []*store.QueueItem
	var totalCount int
	var err error
	
	// Get items based on filter
	switch goFilter {
	case "pending":
		items, err = queueStore.GetByStatus("pending", goOffset, goLimit)
		if err == nil {
			totalCount, _ = queueStore.GetCountByStatus("pending")
		}
	case "downloading":
		items, err = queueStore.GetByStatus("downloading", goOffset, goLimit)
		if err == nil {
			totalCount, _ = queueStore.GetCountByStatus("downloading")
		}
	case "completed":
		items, err = queueStore.GetByStatus("completed", goOffset, goLimit)
		if err == nil {
			totalCount, _ = queueStore.GetCountByStatus("completed")
		}
	case "failed":
		items, err = queueStore.GetByStatus("failed", goOffset, goLimit)
		if err == nil {
			totalCount, _ = queueStore.GetCountByStatus("failed")
		}
	default:
		items, err = queueStore.GetAll(goOffset, goLimit)
		if err == nil {
			totalCount, _ = queueStore.GetCount()
		}
	}
	
	if err != nil {
		logDebug("GetQueue: Failed to get queue items: %v", err)
		errJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		return C.CString(string(errJSON))
	}
	
	logDebug("GetQueue: Retrieved %d items (total: %d)", len(items), totalCount)
	
	// Debug: Log album items with track counts
	for _, item := range items {
		if item.Type == "album" && item.TotalTracks > 0 {
			logDebug("Album item: ID=%s, Type=%s, TotalTracks=%d, CompletedTracks=%d", item.ID, item.Type, item.TotalTracks, item.CompletedTracks)
		}
	}
	
	// Return paginated response with metadata
	response := map[string]interface{}{
		"items":  items,
		"total":  totalCount,
		"offset": goOffset,
		"limit":  goLimit,
	}
	
	jsonData, err := json.Marshal(response)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": "failed to marshal queue"})
		return C.CString(string(errJSON))
	}
	
	// Debug: Log a sample of the JSON for album items
	if len(items) > 0 && items[0].Type == "album" {
		logDebug("Sample JSON for first item: %s", string(jsonData)[:min(500, len(jsonData))])
	}
	
	return C.CString(string(jsonData))
}

//export GetQueueStats
func GetQueueStats() *C.char {
	if !checkInitialized() {
		return C.CString(`{"error": "not initialized"}`)
	}
	
	stats, err := queueStore.GetStats()
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": err.Error()})
		return C.CString(string(errJSON))
	}
	
	jsonData, err := json.Marshal(stats)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": "failed to marshal stats"})
		return C.CString(string(errJSON))
	}
	
	return C.CString(string(jsonData))
}

//export PauseDownload
func PauseDownload(itemID *C.char) C.int {
	if !checkInitialized() {
		return -1
	}
	
	goItemID := C.GoString(itemID)
	
	err := downloadMgr.PauseDownload(goItemID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to pause download: %v\n", err)
		return -2
	}
	
	return 0
}

//export ResumeDownload
func ResumeDownload(itemID *C.char) C.int {
	if !checkInitialized() {
		return -1
	}
	
	goItemID := C.GoString(itemID)
	
	err := downloadMgr.ResumeDownload(goItemID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to resume download: %v\n", err)
		return -2
	}
	
	return 0
}

//export CancelDownload
func CancelDownload(itemID *C.char) C.int {
	if !checkInitialized() {
		return -1
	}
	
	goItemID := C.GoString(itemID)
	
	err := downloadMgr.CancelDownload(goItemID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to cancel download: %v\n", err)
		return -2
	}
	
	return 0
}

//export RetryDownload
func RetryDownload(itemID *C.char) C.int {
	if !checkInitialized() {
		return -1
	}
	
	goItemID := C.GoString(itemID)
	
	// Get the item and reset its status
	item, err := queueStore.GetByID(goItemID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get queue item: %v\n", err)
		return -2
	}
	
	item.Status = "pending"
	item.ErrorMessage = ""
	item.Progress = 0
	
	err = queueStore.Update(item)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to update queue item: %v\n", err)
		return -3
	}
	
	return 0
}

//export ClearCompleted
func ClearCompleted() C.int {
	if !checkInitialized() {
		return -1
	}
	
	// Log to debug file
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] ClearCompleted called\n", time.Now().Format("2006-01-02 15:04:05"))
		logFile.Close()
	}
	
	err := queueStore.ClearCompleted()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to clear completed: %v\n", err)
		if logFile, err2 := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err2 == nil {
			fmt.Fprintf(logFile, "[%s] ClearCompleted error: %v\n", time.Now().Format("2006-01-02 15:04:05"), err)
			logFile.Close()
		}
		return -2
	}
	
	if logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "deemusic-download-debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		fmt.Fprintf(logFile, "[%s] ClearCompleted success\n", time.Now().Format("2006-01-02 15:04:05"))
		logFile.Close()
	}
	
	return 0
}

//export GetSettings
func GetSettings() *C.char {
	if !checkInitialized() {
		return C.CString(`{"error": "not initialized"}`)
	}
	
	jsonData, err := json.Marshal(cfg)
	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"error": "failed to marshal settings"})
		return C.CString(string(errJSON))
	}
	
	return C.CString(string(jsonData))
}

//export UpdateSettings
func UpdateSettings(settingsJSON *C.char) C.int {
	if !checkInitialized() {
		return -1
	}
	
	goSettingsJSON := C.GoString(settingsJSON)
	
	var newCfg config.Config
	err := json.Unmarshal([]byte(goSettingsJSON), &newCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to unmarshal settings: %v\n", err)
		return -2
	}
	
	// Validate new config
	if err := newCfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid settings: %v\n", err)
		return -3
	}
	
	// Save to file
	configPath := config.GetConfigPath()
	if err := newCfg.Save(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save settings: %v\n", err)
		return -4
	}
	
	// Update in-memory config
	cfg = &newCfg
	
	return 0
}

//export GetDownloadPath
func GetDownloadPath() *C.char {
	if !checkInitialized() {
		return C.CString("")
	}
	
	return C.CString(cfg.Download.OutputDir)
}

//export SetDownloadPath
func SetDownloadPath(path *C.char) C.int {
	if !checkInitialized() {
		return -1
	}
	
	goPath := C.GoString(path)
	
	// Validate path exists or can be created
	if err := os.MkdirAll(goPath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create download path: %v\n", err)
		return -2
	}
	
	cfg.Download.OutputDir = goPath
	
	// Save config
	configPath := config.GetConfigPath()
	if err := cfg.Save(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save settings: %v\n", err)
		return -3
	}
	
	return 0
}

//export GetVersion
func GetVersion() *C.char {
	return C.CString("2.0.0-standalone")
}

// ============================================================================
// Migration Functions
// ============================================================================

//export CheckMigrationNeeded
func CheckMigrationNeeded() C.int {
	needed, err := migration.CheckMigrationNeeded()
	if err != nil {
		return -1
	}
	if needed {
		return 1
	}
	return 0
}

//export GetMigrationInfo
func GetMigrationInfo() *C.char {
	info, err := migration.GetMigrationInfo()
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error": "%s"}`, err.Error()))
	}

	jsonData, err := json.Marshal(info)
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error": "%s"}`, err.Error()))
	}

	return C.CString(string(jsonData))
}

//export DetectPythonInstallation
func DetectPythonInstallation() *C.char {
	detector := migration.NewDetector()
	installation, err := detector.DetectPythonInstallation()
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error": "%s"}`, err.Error()))
	}

	result := map[string]interface{}{
		"data_dir":      installation.DataDir,
		"has_settings":  installation.HasSettings,
		"has_queue":     installation.HasQueue,
		"settings_path": installation.SettingsPath,
		"queue_path":    installation.QueueDBPath,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error": "%s"}`, err.Error()))
	}

	return C.CString(string(jsonData))
}

//export PerformMigration
func PerformMigration(progressCallback C.ProgressCallback) *C.char {
	migrator := migration.NewMigrator()

	// Store progress callback temporarily
	callbackMu.Lock()
	oldProgressCb := progressCb
	progressCb = progressCallback
	callbackMu.Unlock()

	// Restore old callback when done
	defer func() {
		callbackMu.Lock()
		progressCb = oldProgressCb
		callbackMu.Unlock()
	}()

	// Report progress: Detection
	if progressCallback != nil {
		msg := C.CString("Detecting Python installation...")
		C.call_progress_callback(progressCallback, msg, C.int(10), C.longlong(0), C.longlong(100))
		C.free(unsafe.Pointer(msg))
	}

	// Detect Python installation
	installation, err := migrator.DetectPythonInstallation()
	if err != nil {
		return C.CString(fmt.Sprintf(`{"success": false, "error": "%s"}`, err.Error()))
	}

	// Report progress: Backup
	if progressCallback != nil {
		msg := C.CString("Creating backup...")
		C.call_progress_callback(progressCallback, msg, C.int(20), C.longlong(0), C.longlong(100))
		C.free(unsafe.Pointer(msg))
	}

	// Create backup
	if err := migrator.CreateBackup(); err != nil {
		return C.CString(fmt.Sprintf(`{"success": false, "error": "Backup failed: %s"}`, err.Error()))
	}

	// Report progress: Settings migration
	if progressCallback != nil {
		msg := C.CString("Migrating settings...")
		C.call_progress_callback(progressCallback, msg, C.int(40), C.longlong(0), C.longlong(100))
		C.free(unsafe.Pointer(msg))
	}

	// Migrate settings
	settingsMigrated := false
	if installation.HasSettings {
		if err := migrator.MigrateSettings(); err != nil {
			return C.CString(fmt.Sprintf(`{"success": false, "error": "Settings migration failed: %s"}`, err.Error()))
		}
		settingsMigrated = true
	}

	// Report progress: Queue migration
	if progressCallback != nil {
		msg := C.CString("Migrating queue and history...")
		C.call_progress_callback(progressCallback, msg, C.int(60), C.longlong(0), C.longlong(100))
		C.free(unsafe.Pointer(msg))
	}

	// Migrate queue
	queueMigrated := false
	if installation.HasQueue {
		if err := migrator.MigrateQueue(); err != nil {
			return C.CString(fmt.Sprintf(`{"success": false, "error": "Queue migration failed: %s"}`, err.Error()))
		}
		queueMigrated = true
	}

	// Report progress: Complete
	if progressCallback != nil {
		msg := C.CString("Migration complete!")
		C.call_progress_callback(progressCallback, msg, C.int(100), C.longlong(0), C.longlong(100))
		C.free(unsafe.Pointer(msg))
	}

	result := map[string]interface{}{
		"success":           true,
		"settings_migrated": settingsMigrated,
		"queue_migrated":    queueMigrated,
		"backup_path":       installation.BackupPath,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return C.CString(fmt.Sprintf(`{"success": false, "error": "%s"}`, err.Error()))
	}

	return C.CString(string(jsonData))
}

//export GetMigrationStats
func GetMigrationStats() *C.char {
	detector := migration.NewDetector()
	installation, err := detector.DetectPythonInstallation()
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error": "%s"}`, err.Error()))
	}

	if !installation.HasQueue {
		return C.CString(`{"queue_items": 0, "history_items": 0}`)
	}

	// Get database path
	goDBPath := filepath.Join(config.GetDataDir(), "deemusic.db")
	queueMigrator := migration.NewQueueMigrator(installation.QueueDBPath, goDBPath)

	stats, err := queueMigrator.GetMigrationStats()
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error": "%s"}`, err.Error()))
	}

	jsonData, err := json.Marshal(stats)
	if err != nil {
		return C.CString(fmt.Sprintf(`{"error": "%s"}`, err.Error()))
	}

	return C.CString(string(jsonData))
}

//export StopAllDownloads
func StopAllDownloads() C.int {
	if !checkInitialized() {
		return -1
	}
	
	if downloadMgr == nil {
		return -2
	}
	
	err := downloadMgr.StopAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to stop all downloads: %v\n", err)
		return -3
	}
	
	return 0
}
