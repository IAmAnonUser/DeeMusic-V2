# System Tray Integration

## Overview

The system tray integration provides a native Windows experience by allowing DeeMusic to run in the background with a system tray icon. Users can interact with the application through the tray icon even when the main window is hidden.

## Features

### 1. System Tray Icon
- **Icon**: Displays a custom "DM" icon in the system tray
- **Tooltip**: Shows "DeeMusic" when hovering over the icon
- **Double-click**: Opens/shows the main window

### 2. Context Menu
Right-clicking the tray icon shows a menu with the following options:

- **Open** (Bold): Shows and activates the main window
- **Show Queue**: Opens the main window and navigates to the Queue page
- **Pause All**: Pauses all active downloads
- **Resume All**: Resumes all paused downloads
- **Settings**: Opens the main window and navigates to Settings
- **Exit**: Closes the application completely

### 3. Minimize to Tray
- When enabled in settings, clicking the close button (X) minimizes the app to tray instead of closing it
- The setting `system.minimize_to_tray` controls this behavior (default: true)
- To actually exit the application, use the "Exit" option from the tray menu

### 4. Notifications
The tray icon displays balloon notifications for:

- **Download Completed**: Shows when a track finishes downloading successfully
- **Download Failed**: Shows when a download fails with error details

## Implementation Details

### TrayService Class
Located in `Services/TrayService.cs`, this service manages all tray-related functionality:

```csharp
public class TrayService : IDisposable
{
    // Initialize the tray icon and context menu
    public void Initialize()
    
    // Show a notification balloon
    public void ShowNotification(string title, string message, ToolTipIcon icon)
    
    // Show download completion notification
    public void ShowDownloadCompleted(string trackName)
    
    // Show download error notification
    public void ShowDownloadError(string trackName, string error)
    
    // Show the main window
    public void ShowMainWindow()
    
    // Hide the window to tray
    public void HideToTray()
}
```

### Integration Points

1. **App.xaml.cs**: Initializes the TrayService on application startup
2. **MainWindow.xaml.cs**: Handles minimize to tray behavior and window closing
3. **QueueViewModel.cs**: Triggers notifications when download status changes
4. **MainViewModel.cs**: Provides navigation methods for tray menu actions

### Settings Integration

The minimize to tray behavior is controlled by the `system.minimize_to_tray` setting in `settings.json`:

```json
{
  "system": {
    "minimize_to_tray": true,
    "start_minimized": false
  }
}
```

## User Experience

### Normal Operation
1. User launches DeeMusic - main window appears
2. User starts downloads
3. User clicks close button (X) - window hides to tray
4. Downloads continue in background
5. Notification appears when downloads complete
6. User double-clicks tray icon - window reappears

### Tray Menu Usage
1. User right-clicks tray icon
2. Selects "Show Queue" - window opens to Queue page
3. Selects "Pause All" - all downloads pause without opening window
4. Selects "Exit" - application closes completely

## Requirements Satisfied

This implementation satisfies the following requirements from the spec:

- **6.1**: System tray integration functionality retained
- **6.2**: Windows startup management functionality retained (setting available)
- **6.3**: Support for minimizing to system tray when window is closed
- **6.4**: System tray menu options for show/hide, settings, and quit
- **6.5**: Native Windows notifications for download completion and errors

## Future Enhancements

Potential improvements for future versions:

1. **Custom Icon**: Replace the generated "DM" icon with a proper .ico file
2. **Progress in Tray**: Show download progress in the tray icon tooltip
3. **Notification Settings**: Allow users to configure which notifications to show
4. **Tray-only Mode**: Option to start minimized to tray without showing window
5. **Quick Actions**: Add more quick actions to the tray menu (e.g., "Download from Clipboard")
