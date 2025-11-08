# DeeMusic Desktop - C# WPF Frontend

This is the native Windows desktop frontend for DeeMusic, built with C# WPF and Material Design.

## Project Structure

```
DeeMusic.Desktop/
├── App.xaml                    # Application entry point
├── App.xaml.cs
├── MainWindow.xaml             # Main application window
├── MainWindow.xaml.cs
├── ViewModels/                 # MVVM ViewModels
├── Views/                      # XAML views
├── Models/                     # Data models
├── Services/                   # Service classes (P/Invoke wrapper, etc.)
├── Controls/                   # Custom WPF controls
└── Resources/
    ├── Styles/                 # XAML style resources
    │   ├── Colors.xaml
    │   ├── Buttons.xaml
    │   └── Cards.xaml
    └── Icons/                  # Application icons
```

## Technology Stack

- **.NET 8.0** - Target framework
- **WPF** - Windows Presentation Foundation for native UI
- **MaterialDesignThemes** - Material Design styling
- **CommunityToolkit.Mvvm** - MVVM helpers and commands

## Building

### Prerequisites

- Visual Studio 2022 or later
- .NET 6.0 SDK or later
- Go DLL (`deemusic-core.dll`) from the backend build

### Build Steps

1. Restore NuGet packages:
   ```bash
   dotnet restore
   ```

2. Build the project:
   ```bash
   dotnet build
   ```

3. Run the application:
   ```bash
   dotnet run
   ```

## Integration with Go Backend

The C# frontend communicates with the Go backend DLL via P/Invoke. The Go DLL (`deemusic-core.dll`) is automatically copied to the output directory during build.

The `Services/GoBackendService.cs` class (to be implemented) will handle all P/Invoke declarations and communication with the Go backend.

## Next Steps

The following components need to be implemented:

1. **P/Invoke Wrapper** (`Services/GoBackendService.cs`)
2. **Data Models** (Track, Album, QueueItem, etc.)
3. **ViewModels** (MainViewModel, SearchViewModel, QueueViewModel, SettingsViewModel)
4. **Views** (SearchView, QueueView, SettingsView, etc.)
5. **Custom Controls** (ModernButton, ProgressCard, SearchResultCard)

See the implementation plan in `.kiro/specs/standalone-windows-app/tasks.md` for details.
