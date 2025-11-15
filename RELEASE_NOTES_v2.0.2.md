# DeeMusic v2.0.2 Release Notes

## 🎉 What's New

### Queue Visual Feedback
- **Track Progress Display**: Queue now shows track progress for each album (e.g., "05/12 tracks")
- **Completion Indicators**: Albums turn green when all tracks are downloaded
- **Real-time Updates**: Progress updates live as downloads complete
- **Album-based Stats**: Stats bar now shows album counts instead of track counts for better clarity

### Theme Persistence
- **Fixed Theme Toggle**: Theme preference now properly persists across app restarts
- **Correct Initialization**: App now loads with the last selected theme (dark/light)
- **Smooth Transitions**: Theme switching includes smooth animations

### Enhanced Welcome Dialog
- **Download Directory Setup**: Now prompts for download location during initial setup
- **Browse Button**: Easy folder selection with file browser
- **More Options**: Three buttons for better workflow:
  - "More Settings" - Opens full settings dialog
  - "Skip" - Skip setup and configure later
  - "Save and Continue" - Quick setup with essentials
- **Better Layout**: Larger dialog with improved spacing and instructions

### Settings Dialog Improvements
- **Larger Window**: Increased from 700x750 to 900x900 pixels
- **Better Tab Visibility**: All 6 tabs (Deezer, Spotify, Artwork, Filenaming, Lyrics, Cache) now visible
- **No Height Restrictions**: Can maximize to full screen without cutting off content
- **Improved Minimum Size**: MinWidth 850px, MinHeight 800px for better usability

## 🐛 Bug Fixes

### Theme System
- Fixed App.xaml hardcoded light theme that prevented dark theme from loading
- Fixed theme toggle not saving to settings file
- Fixed Material Design BaseTheme not updating with theme changes

### Queue Display
- Fixed stats bar showing incorrect track counts
- Fixed queue items not updating when downloads complete
- Fixed album completion status not reflecting in UI

### Settings Dialog
- Fixed content being cut off at bottom of dialog
- Fixed tabs being cut off on the right side
- Fixed dialog not using full screen when maximized

## 🔧 Technical Improvements

### Code Quality
- Added comprehensive logging for theme operations
- Improved async/await handling in theme toggle
- Better error handling in settings save operations
- Enhanced debug logging for troubleshooting

### Architecture
- Proper Settings.System.Theme property binding
- Direct ForceSaveAsync calls instead of command execution
- Thread-safe property updates in QueueViewModel
- Better separation of concerns in MainViewModel

## 📝 Implementation Details

### Files Modified
- `DeeMusic.Desktop/ViewModels/MainViewModel.cs` - Theme toggle with persistence
- `DeeMusic.Desktop/ViewModels/SettingsViewModel.cs` - Enhanced logging and Theme property
- `DeeMusic.Desktop/ViewModels/QueueViewModel.cs` - Track progress and album completion
- `DeeMusic.Desktop/Models/QueueItem.cs` - Track progress properties
- `DeeMusic.Desktop/Views/QueueView.xaml` - Progress display UI
- `DeeMusic.Desktop/MainWindow.xaml` - Theme initialization
- `DeeMusic.Desktop/MainWindow.xaml.cs` - Welcome dialog and settings window sizing
- `DeeMusic.Desktop/App.xaml` - Default theme configuration

### New Features Implementation
1. **Track Progress**: Uses existing `CompletedTracks` and `TotalTracks` properties
2. **Green Completion**: Background color changes when `CompletedTracks == TotalTracks`
3. **Theme Persistence**: Saves to `settings.json` → `system.theme` field
4. **Welcome Dialog**: Integrated folder browser and multi-button layout

## 🎯 User Experience Improvements

- **Clearer Queue Status**: Users can now see exactly how many tracks are done per album
- **Visual Feedback**: Green backgrounds make it obvious when albums are complete
- **Persistent Preferences**: Theme choice is remembered between sessions
- **Better Onboarding**: New users get a comprehensive setup experience
- **More Screen Space**: Settings dialog uses available space efficiently

## 🔄 Migration Notes

No migration required. Existing settings files are fully compatible.

## 📊 Statistics

- **Files Changed**: 8
- **Lines Added**: ~300
- **Lines Removed**: ~50
- **Bug Fixes**: 7
- **New Features**: 4

## 🙏 Acknowledgments

Thanks to the development team for thorough testing and feedback during this release cycle.

---

**Release Date**: November 15, 2025  
**Version**: 2.0.2  
**Build**: Stable
