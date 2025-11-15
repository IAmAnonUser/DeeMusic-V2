# Windows Startup Integration

This document describes the Windows startup integration implementation for DeeMusic Desktop.

## Overview

The Windows startup integration allows users to configure DeeMusic to automatically start when Windows boots. The implementation uses the Windows Registry to manage startup entries.

## Components

### StartupManager Service

**Location:** `Services/StartupManager.cs`

The `StartupManager` is a singleton service that handles all Windows startup registry operations.

#### Key Methods

- `EnableStartup(bool startMinimized)` - Adds DeeMusic to Windows startup
- `DisableStartup()` - Removes DeeMusic from Windows startup
- `IsStartupEnabled()` - Checks if startup is currently enabled
- `UpdateStartup(bool enabled, bool startMinimized)` - Updates startup configuration
- `GetStartupCommandLine()` - Gets the current startup command from registry

#### Registry Details

- **Registry Path:** `HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\Run`
- **Value Name:** `DeeMusic`
- **Value Type:** `REG_SZ` (String)
- **Value Format:** `"C:\Path\To\DeeMusic.Desktop.exe" [--minimized]`

### Settings Integration

The startup configuration is stored in the application settings:

```json
{
  "system": {
    "run_on_startup": false,
    "start_minimized": false
  }
}
```

### SettingsViewModel

**Location:** `ViewModels/SettingsViewModel.cs`

The SettingsViewModel manages the startup settings and automatically updates the Windows registry when settings change.

#### Properties

- `StartWithWindows` - Enables/disables Windows startup
- `StartMinimized` - Controls whether the app starts minimized

#### Behavior

When `StartWithWindows` or `StartMinimized` changes:
1. The property value is updated in the Settings model
2. `UpdateWindowsStartup()` is called automatically
3. The registry is updated via `StartupManager`
4. Changes are marked as unsaved

When settings are loaded:
1. Settings are loaded from the backend
2. `SyncStartupWithRegistry()` is called
3. If there's a mismatch between settings and registry, the registry is updated

### UI Integration

**Location:** `Views/SettingsView.xaml`

The startup settings are displayed in the UI Settings section:

```xml
<CheckBox Content="Start with Windows"
         IsChecked="{Binding StartWithWindows}"
         Margin="0,0,0,8"/>

<CheckBox Content="Start Minimized"
         IsChecked="{Binding StartMinimized}"
         IsEnabled="{Binding StartWithWindows}"
         Margin="0,0,0,8"/>
```

The "Start Minimized" checkbox is only enabled when "Start with Windows" is checked.

### Command Line Arguments

**Location:** `App.xaml.cs`

The application supports the `--minimized` command line argument:

```
DeeMusic.Desktop.exe --minimized
```

When this argument is present:
1. The app starts normally
2. After initialization, the main window is minimized
3. The window is hidden (if minimize to tray is enabled)

Supported argument formats:
- `--minimized`
- `-minimized`
- `/minimized`

## Usage Flow

### Enabling Startup

1. User checks "Start with Windows" in Settings
2. `StartWithWindows` property setter is triggered
3. `UpdateWindowsStartup()` is called
4. `StartupManager.UpdateStartup(true, startMinimized)` is invoked
5. Registry entry is created: `"C:\...\DeeMusic.Desktop.exe" [--minimized]`
6. Settings are marked as changed
7. User clicks "Save Settings" to persist

### Disabling Startup

1. User unchecks "Start with Windows" in Settings
2. `StartWithWindows` property setter is triggered
3. `UpdateWindowsStartup()` is called
4. `StartupManager.UpdateStartup(false, ...)` is invoked
5. Registry entry is removed
6. Settings are marked as changed
7. User clicks "Save Settings" to persist

### Changing Start Minimized

1. User checks/unchecks "Start Minimized" (only if startup is enabled)
2. `StartMinimized` property setter is triggered
3. `UpdateWindowsStartup()` is called (only if `StartWithWindows` is true)
4. Registry entry is updated with/without `--minimized` flag
5. Settings are marked as changed
6. User clicks "Save Settings" to persist

### Application Startup

When Windows starts and DeeMusic is configured to run:

1. Windows executes the command from registry
2. DeeMusic.Desktop.exe launches
3. `App.OnStartup()` checks command line arguments
4. If `--minimized` is present, `_startMinimized` flag is set
5. Theme is initialized
6. Main window is created
7. In `App.OnActivated()`, if `_startMinimized` is true:
   - Window state is set to Minimized
   - Window is hidden (if minimize to tray is enabled)

## Error Handling

All registry operations are wrapped in try-catch blocks:

- If registry access fails, errors are logged to Debug output
- Methods return `false` on failure
- The application continues to function even if registry operations fail
- Users are not shown error messages for registry failures (silent failure)

## Security Considerations

- Uses `HKEY_CURRENT_USER` (not `HKEY_LOCAL_MACHINE`) - no admin rights required
- Only modifies the user's own startup entries
- Executable path is properly quoted to handle spaces
- No sensitive information is stored in the registry

## Testing

To test the startup integration:

1. **Enable Startup:**
   - Open Settings
   - Check "Start with Windows"
   - Optionally check "Start Minimized"
   - Click "Save Settings"
   - Verify registry entry: `regedit` → `HKEY_CURRENT_USER\SOFTWARE\Microsoft\Windows\CurrentVersion\Run`

2. **Test Startup:**
   - Log out and log back in, or restart Windows
   - Verify DeeMusic starts automatically
   - If "Start Minimized" was enabled, verify it starts minimized

3. **Disable Startup:**
   - Open Settings
   - Uncheck "Start with Windows"
   - Click "Save Settings"
   - Verify registry entry is removed

4. **Test Sync:**
   - Manually delete the registry entry
   - Restart DeeMusic
   - Settings should sync and recreate the entry if enabled

## Troubleshooting

### Startup Not Working

1. Check if registry entry exists
2. Verify executable path is correct
3. Check Windows Event Viewer for startup errors
4. Ensure no antivirus is blocking the startup

### Registry Access Denied

- Ensure the application is not running as a different user
- Check Windows permissions for the registry key
- Try running as administrator (though it shouldn't be required)

### Executable Path Issues

- The implementation handles both regular and single-file deployments
- If the assembly location is a .dll, it attempts to find the .exe
- Uses `Process.GetCurrentProcess().MainModule.FileName` as fallback

## Future Enhancements

Potential improvements:

1. Add Task Scheduler integration as an alternative to registry
2. Support for delayed startup (start X seconds after login)
3. Startup success/failure notifications
4. Startup diagnostics in the UI
5. Option to start only on specific conditions (e.g., when connected to power)
