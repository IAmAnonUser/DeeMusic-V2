# Error Handling Quick Reference

## Quick Start

### 1. Basic Logging

```csharp
// Info logging
LoggingService.Instance.LogInfo("User clicked download button");

// Error logging
LoggingService.Instance.LogError("Download failed", exception);

// Critical logging
LoggingService.Instance.LogCritical("Application crash prevented", exception);
```

### 2. Error Handling in ViewModels

```csharp
private async Task DownloadTrackAsync(string trackId)
{
    try
    {
        LoggingService.Instance.LogInfo($"Starting download: {trackId}");
        await _service.DownloadTrackAsync(trackId);
        LoggingService.Instance.LogInfo($"Download completed: {trackId}");
    }
    catch (BackendException ex)
    {
        ErrorHandler.HandleBackendException(ex, "Download Track");
    }
    catch (Exception ex)
    {
        ErrorHandler.HandleException(ex, "Download Track");
    }
}
```

### 3. Error Handling in Services

```csharp
public async Task<SearchResults> SearchAsync(string query)
{
    try
    {
        // Service automatically retries transient errors
        return await _deeMusicService.SearchAsync<SearchResults>(query, "track");
    }
    catch (BackendException ex)
    {
        LoggingService.Instance.LogError($"Search failed: {query}", ex);
        throw; // Let caller handle
    }
}
```

### 4. User Confirmations

```csharp
if (ErrorHandler.ShowConfirmation("Clear Queue", "Are you sure you want to clear all completed downloads?"))
{
    await _service.ClearCompletedAsync();
}
```

### 5. User Information

```csharp
ErrorHandler.ShowInfo("Download Complete", "All tracks have been downloaded successfully!");
```

## Error Codes Reference

| Code | Meaning | Transient | Action |
|------|---------|-----------|--------|
| -1 | Not initialized | No | Restart app |
| -2 | Operation failed | No | Check logs |
| -3 | Invalid config | No | Fix settings |
| -4 | Database error | No | Check database |
| -5 | Migration failed | No | Check logs |
| -6 | Download manager failed | No | Restart app |
| -7 | Authentication failed | No | Update ARL |
| -8 | Network error | Yes | Auto-retry |
| -9 | File system error | No | Check permissions |
| -10 | Invalid parameter | No | Fix input |
| -11 | Resource not found | No | Check ID |
| -12 | Permission denied | No | Check permissions |
| -13 | Timeout | Yes | Auto-retry |
| -14 | Rate limit | Yes | Auto-retry |

## Common Patterns

### Pattern 1: Simple Operation
```csharp
try
{
    await _service.SomeOperationAsync();
}
catch (Exception ex)
{
    ErrorHandler.HandleException(ex, "Operation Name");
}
```

### Pattern 2: With Logging
```csharp
try
{
    LoggingService.Instance.LogInfo("Starting operation");
    await _service.SomeOperationAsync();
    LoggingService.Instance.LogInfo("Operation completed");
}
catch (Exception ex)
{
    ErrorHandler.HandleException(ex, "Operation Name");
}
```

### Pattern 3: Backend-Specific
```csharp
try
{
    await _service.SomeOperationAsync();
}
catch (BackendException ex)
{
    ErrorHandler.HandleBackendException(ex, "Operation Name");
}
catch (Exception ex)
{
    ErrorHandler.HandleException(ex, "Operation Name");
}
```

### Pattern 4: Silent Logging (No Dialog)
```csharp
try
{
    await _service.SomeOperationAsync();
}
catch (Exception ex)
{
    LoggingService.Instance.LogError("Operation failed", ex);
    // No dialog shown
}
```

### Pattern 5: Custom Error Message
```csharp
try
{
    await _service.SomeOperationAsync();
}
catch (Exception ex)
{
    LoggingService.Instance.LogError("Operation failed", ex);
    ErrorHandler.ShowErrorDialog(
        "Custom Title",
        "Custom message for the user",
        isCritical: false
    );
}
```

## Log Levels Guide

| Level | When to Use | Example |
|-------|-------------|---------|
| Debug | Development debugging | "Variable value: {value}" |
| Info | Normal operations | "User logged in", "Download started" |
| Warning | Recoverable issues | "Retry attempt 2/3", "Setting not found, using default" |
| Error | Operation failures | "Download failed", "Database query error" |
| Critical | Application-level issues | "Unhandled exception", "Initialization failed" |

## Best Practices

### DO ✅
- Always log errors with context
- Use specific exception types when available
- Provide context in error messages
- Let transient errors retry automatically
- Log important user actions
- Use appropriate log levels

### DON'T ❌
- Don't log sensitive data (passwords, tokens)
- Don't show technical details to users
- Don't retry non-transient errors
- Don't swallow exceptions silently
- Don't log in tight loops
- Don't use Debug level in production

## Debugging Tips

### Finding Logs
```
%APPDATA%\DeeMusicV2\logs\deemusic_YYYYMMDD.log
```

### Opening Logs Folder
```csharp
LoggingService.Instance.OpenLogsFolder();
```

### Log File Format
```
[2025-10-26 14:30:45.123] [Info] Operation started
[2025-10-26 14:30:46.456] [Error] Operation failed
Exception: BackendException
Message: Download failed: Network error
StackTrace: ...
```

### Common Issues

**Issue:** Logs not appearing
- Check `%APPDATA%\DeeMusicV2\logs\` exists
- Check file permissions
- Check disk space

**Issue:** Too many log files
- Automatic rotation keeps only 5 files
- Each file max 10 MB
- Old files automatically deleted

**Issue:** Error dialog not showing
- Check if running on UI thread
- Check Application.Current is not null
- Check for exceptions in error handler

## Testing Error Handling

### Test Network Error
```csharp
// Disconnect internet, then:
await _service.SearchAsync<SearchResults>("test", "track");
// Should retry 3 times, then show error dialog
```

### Test Authentication Error
```csharp
// Set invalid ARL in settings, then:
await _service.DownloadTrackAsync("123456");
// Should show authentication error dialog
```

### Test Unhandled Exception
```csharp
throw new Exception("Test unhandled exception");
// Should be caught by global handler, logged, and dialog shown
```

## Integration with Existing Code

### In ViewModels
```csharp
public class MyViewModel : ViewModelBase
{
    private readonly DeeMusicService _service;
    
    public MyViewModel(DeeMusicService service)
    {
        _service = service;
    }
    
    private async Task MyOperationAsync()
    {
        try
        {
            await _service.SomeOperationAsync();
        }
        catch (Exception ex)
        {
            ErrorHandler.HandleException(ex, "My Operation");
        }
    }
}
```

### In Services
```csharp
public class MyService
{
    public async Task DoSomethingAsync()
    {
        try
        {
            LoggingService.Instance.LogInfo("Starting operation");
            // ... operation code ...
            LoggingService.Instance.LogInfo("Operation completed");
        }
        catch (Exception ex)
        {
            LoggingService.Instance.LogError("Operation failed", ex);
            throw; // Let caller handle
        }
    }
}
```

## Performance Considerations

- Logging is async (doesn't block UI)
- Log rotation is automatic
- Only important operations logged
- Retry uses exponential backoff
- No performance impact on normal operations

## Support

For detailed documentation, see:
- `ERROR_HANDLING.md` - Complete documentation
- `TASK_16_ERROR_HANDLING_SUMMARY.md` - Implementation summary
