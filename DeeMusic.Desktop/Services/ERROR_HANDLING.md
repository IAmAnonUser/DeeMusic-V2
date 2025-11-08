# Error Handling and Logging System

This document describes the comprehensive error handling and logging system implemented in DeeMusic Desktop.

## Components

### 1. LoggingService

**Location:** `Services/LoggingService.cs`

A singleton service that provides file-based logging for debugging and error tracking.

**Features:**
- Automatic log file rotation (max 10 MB per file)
- Keeps only the most recent 5 log files
- Thread-safe logging
- Multiple log levels: Debug, Info, Warning, Error, Critical
- Logs stored in `%APPDATA%\DeeMusicV2\logs\`

**Usage:**
```csharp
LoggingService.Instance.LogInfo("Operation completed");
LoggingService.Instance.LogError("Operation failed", exception);
LoggingService.Instance.LogCritical("Critical error", exception);
```

**Log File Format:**
```
[2025-10-26 14:30:45.123] [Info] Backend initialized successfully
[2025-10-26 14:30:46.456] [Error] Failed to download track
Exception: BackendException
Message: Download failed: Network error
StackTrace: ...
```

### 2. ErrorHandler

**Location:** `Services/ErrorHandler.cs`

A static class that provides centralized error handling with user-friendly error messages.

**Features:**
- Automatic error logging
- User-friendly error dialogs
- Context-aware error messages
- Backend error code translation
- Critical vs non-critical error classification

**Usage:**
```csharp
try
{
    await service.DownloadTrackAsync(trackId);
}
catch (DeeMusicService.BackendException ex)
{
    ErrorHandler.HandleBackendException(ex, "Download Track");
}
catch (Exception ex)
{
    ErrorHandler.HandleException(ex, "Download Track");
}
```

**Error Message Examples:**
- Authentication errors → "Please update your ARL token in Settings"
- Database errors → "Your queue data may be corrupted"
- Network errors → "Please check your internet connection"
- Not found errors → "Content not available in your region"

### 3. Enhanced P/Invoke Layer

**Location:** `Services/GoBackendService.cs`

Enhanced error handling in the P/Invoke wrapper.

**Features:**
- Detailed error code mapping
- Transient error detection
- Context-aware error messages
- Error code documentation

**Error Codes:**
- `-1`: Backend not initialized
- `-2`: Operation failed
- `-3`: Invalid configuration
- `-4`: Database error
- `-5`: Migration failed
- `-6`: Failed to start download manager
- `-7`: Authentication failed
- `-8`: Network error (transient)
- `-9`: File system error
- `-10`: Invalid parameter
- `-11`: Resource not found
- `-12`: Permission denied
- `-13`: Timeout (transient)
- `-14`: Rate limit exceeded (transient)

### 4. Retry Logic

**Location:** `Services/DeeMusicService.cs`

Intelligent retry logic for transient errors.

**Features:**
- Automatic retry for transient errors (network, timeout, rate limit)
- Exponential backoff (1s, 2s, 3s)
- Maximum 3 retry attempts
- No retry for non-transient errors (authentication, database, etc.)

**Example:**
```csharp
// Automatically retries network errors up to 3 times
await service.SearchAsync<SearchResults>(query, "track");
```

### 5. Global Exception Handlers

**Location:** `App.xaml.cs`

Application-wide exception handlers to catch unhandled exceptions.

**Handlers:**
- UI thread exceptions (DispatcherUnhandledException)
- Background thread exceptions (AppDomain.UnhandledException)
- Task exceptions (TaskScheduler.UnobservedTaskException)

**Behavior:**
- All exceptions are logged to file
- User-friendly error dialogs are shown
- Application continues running (doesn't crash)

### 6. Go Backend Logging

**Location:** `cmd/deemusic-core/main.go`

Enhanced logging in the Go backend.

**Features:**
- Structured log messages with levels: [INFO], [WARN], [ERROR]
- Detailed operation logging
- Error context and stack traces
- Logs written to stderr (captured by C# layer)

**Example Output:**
```
[INFO] Initializing DeeMusic backend...
[INFO] Loading configuration from: C:\Users\...\settings.json
[INFO] Database path: C:\Users\...\queue.db
[INFO] Running database migrations...
[INFO] Authenticating with Deezer...
[INFO] Deezer authentication successful
[INFO] Starting download manager...
[INFO] Backend initialized successfully
```

## Error Flow

### 1. Backend Error Flow

```
Go Backend Error
    ↓
Error Code Returned (-1 to -14)
    ↓
C# P/Invoke Layer (GoBackend.GetErrorMessage)
    ↓
DeeMusicService (ExecuteWithRetryAsync)
    ↓
Retry if Transient / Throw if Non-Transient
    ↓
ErrorHandler.HandleBackendException
    ↓
User-Friendly Dialog + Log to File
```

### 2. General Exception Flow

```
Exception Thrown
    ↓
Caught in ViewModel/Service
    ↓
ErrorHandler.HandleException
    ↓
Determine Error Type
    ↓
Show User-Friendly Dialog + Log to File
```

### 3. Unhandled Exception Flow

```
Unhandled Exception
    ↓
Global Exception Handler (App.xaml.cs)
    ↓
Log to File (Critical Level)
    ↓
ErrorHandler.HandleException
    ↓
User-Friendly Dialog
    ↓
Application Continues (e.Handled = true)
```

## Best Practices

### For Developers

1. **Always log errors:**
   ```csharp
   catch (Exception ex)
   {
       LoggingService.Instance.LogError("Operation failed", ex);
       ErrorHandler.HandleException(ex, "Operation Name");
   }
   ```

2. **Use context in error messages:**
   ```csharp
   ErrorHandler.HandleException(ex, "Downloading Track");
   // Results in: "Error in Downloading Track: ..."
   ```

3. **Handle backend exceptions specifically:**
   ```csharp
   catch (DeeMusicService.BackendException ex)
   {
       ErrorHandler.HandleBackendException(ex, "Search");
   }
   ```

4. **Let transient errors retry automatically:**
   ```csharp
   // No need for manual retry logic
   await service.SearchAsync<SearchResults>(query, "track");
   ```

5. **Log important operations:**
   ```csharp
   LoggingService.Instance.LogInfo($"Starting download: {trackId}");
   await service.DownloadTrackAsync(trackId);
   LoggingService.Instance.LogInfo($"Download completed: {trackId}");
   ```

### For Users

1. **Finding log files:**
   - Location: `%APPDATA%\DeeMusicV2\logs\`
   - Or: Settings → Open Logs Folder (if implemented)

2. **Understanding error messages:**
   - Error dialogs provide user-friendly explanations
   - Check log files for technical details
   - Error codes help identify specific issues

3. **Reporting issues:**
   - Include the log file from `%APPDATA%\DeeMusicV2\logs\`
   - Note the exact error message shown
   - Describe steps to reproduce

## Testing Error Handling

### Test Scenarios

1. **Backend not initialized:**
   - Call any service method before initialization
   - Expected: Error dialog + log entry

2. **Network error:**
   - Disconnect internet
   - Try to search or download
   - Expected: Automatic retry → Error dialog after 3 attempts

3. **Invalid ARL:**
   - Set invalid ARL in settings
   - Try to download
   - Expected: Authentication error dialog

4. **Database corruption:**
   - Corrupt the queue.db file
   - Start application
   - Expected: Database error dialog

5. **Unhandled exception:**
   - Trigger any unexpected error
   - Expected: Error logged + dialog shown + app continues

## Performance Considerations

- **Async logging:** Log writes are async to avoid blocking UI
- **Log rotation:** Automatic rotation prevents disk space issues
- **Selective logging:** Only important operations are logged
- **Efficient retry:** Exponential backoff prevents server hammering

## Future Enhancements

- [ ] Add log viewer in Settings
- [ ] Export logs for bug reports
- [ ] Configurable log levels
- [ ] Remote error reporting (opt-in)
- [ ] Error analytics dashboard
