# DeeMusic Desktop - Project Structure

This document describes the C# WPF project structure created for the DeeMusic Desktop application.

## Created Files and Folders

### Solution and Project Files

- **DeeMusic.Desktop.sln** - Visual Studio solution file
- **DeeMusic.Desktop/DeeMusic.Desktop.csproj** - WPF project file targeting .NET 8.0

### Application Entry Points

- **DeeMusic.Desktop/App.xaml** - Application definition with Material Design theme
- **DeeMusic.Desktop/App.xaml.cs** - Application code-behind
- **DeeMusic.Desktop/MainWindow.xaml** - Main window XAML
- **DeeMusic.Desktop/MainWindow.xaml.cs** - Main window code-behind

### Folder Structure

```
DeeMusic.Desktop/
├── ViewModels/          # MVVM ViewModels (ready for implementation)
├── Views/               # XAML views (ready for implementation)
├── Models/              # Data models (ready for implementation)
├── Services/            # Service classes including P/Invoke wrapper
├── Controls/            # Custom WPF controls
└── Resources/
    ├── Styles/          # XAML style resources
    │   ├── Colors.xaml      # Color palette (dark/light themes)
    │   ├── Buttons.xaml     # Button styles
    │   └── Cards.xaml       # Card styles
    └── Icons/           # Application icons
```

### Resource Files

**Colors.xaml** - Defines the color palette:
- Dark theme colors (background, surface, primary, secondary, etc.)
- Light theme colors
- Brush resources for easy binding

**Buttons.xaml** - Modern button styles:
- ModernButtonStyle with hover and press effects
- Rounded corners and smooth transitions

**Cards.xaml** - Card component styles:
- Card style with hover shadow effects
- Consistent padding and margins

### Build Scripts

- **scripts/build-wpf.ps1** - Builds the C# WPF application
- **scripts/build-all.ps1** - Builds both Go DLL and C# WPF app

### Documentation

- **DeeMusic.Desktop/README.md** - Project overview and structure
- **DeeMusic.Desktop/SETUP.md** - Development setup guide
- **DeeMusic.Desktop/PROJECT_STRUCTURE.md** - This file

### Configuration

- **DeeMusic.Desktop/.gitignore** - Git ignore rules for C# build artifacts
- **.gitignore** - Updated to include DeeMusic.Desktop build outputs

## NuGet Packages

The project includes the following NuGet packages:

1. **MaterialDesignThemes (v4.9.0)**
   - Material Design styling for WPF
   - Modern UI components and themes

2. **CommunityToolkit.Mvvm (v8.2.2)**
   - MVVM helpers and base classes
   - RelayCommand and ObservableObject implementations

## Build Configuration

The project is configured to:
- Target .NET 8.0 Windows
- Use WPF framework
- Enable nullable reference types
- Copy `deemusic-core.dll` to output directory automatically

## Next Steps

With the project structure in place, the following can now be implemented:

1. **Task 5: Implement C# P/Invoke wrapper**
   - Create `Services/GoBackendService.cs`
   - Define P/Invoke declarations
   - Implement callback handlers

2. **Task 6: Create C# data models**
   - Define Track, Album, Artist, Playlist models
   - Define QueueItem and Settings models

3. **Task 7: Implement MVVM ViewModels**
   - MainViewModel, SearchViewModel, QueueViewModel, SettingsViewModel

4. **Task 8: Design and implement XAML views**
   - SearchView, QueueView, SettingsView, etc.

5. **Task 9: Implement custom WPF controls**
   - ModernButton, ProgressCard, SearchResultCard

## Building the Project

### Prerequisites
- .NET 8.0 SDK or later
- Visual Studio 2022 (recommended) or VS Code with C# extension

### Build Commands

```powershell
# Restore NuGet packages
dotnet restore DeeMusic.Desktop/DeeMusic.Desktop.csproj

# Build the project
dotnet build DeeMusic.Desktop/DeeMusic.Desktop.csproj

# Run the application
dotnet run --project DeeMusic.Desktop/DeeMusic.Desktop.csproj

# Or use the build script
.\scripts\build-wpf.ps1

# Build everything (Go + C#)
.\scripts\build-all.ps1
```

## Integration with Go Backend

The C# application will communicate with the Go backend DLL (`deemusic-core.dll`) via P/Invoke. The DLL is automatically copied to the output directory during build.

The integration will be implemented in the `Services/GoBackendService.cs` class, which will:
- Define P/Invoke declarations for all Go exports
- Handle string marshaling between C# and Go
- Manage callback delegates for progress updates
- Provide async wrapper methods for UI consumption

## Design System

The application uses a modern design system with:

**Dark Theme (Default):**
- Background: #0a0a0a
- Surface: #1a1a1a
- Primary: #6366f1 (Indigo)
- Secondary: #8b5cf6 (Purple)

**Light Theme:**
- Background: #ffffff
- Surface: #f9fafb
- Primary: #4f46e5 (Indigo)
- Secondary: #7c3aed (Purple)

**Typography:**
- Material Design font family
- Consistent sizing and weights

**Components:**
- Rounded corners (4-8px)
- Smooth transitions (150-300ms)
- Hover effects and shadows
- Material Design elevation

## Status

✅ **Task 4 Complete**: C# WPF project structure is fully set up and ready for implementation.

The project structure provides a solid foundation for building the native Windows desktop application with proper separation of concerns, modern styling, and integration with the Go backend.
