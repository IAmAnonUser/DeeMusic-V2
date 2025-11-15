# DeeMusic Desktop - Development Setup

This guide will help you set up the development environment for the DeeMusic Desktop application.

## Prerequisites

### Required

1. **Go 1.21 or later**
   - Download from: https://go.dev/download/
   - Required for building the backend DLL

2. **.NET 8.0 SDK or later**
   - Download from: https://dotnet.microsoft.com/download
   - Required for building the WPF frontend

3. **Visual Studio 2022** (recommended) or **Visual Studio Code**
   - Visual Studio 2022: https://visualstudio.microsoft.com/downloads/
   - Install the ".NET desktop development" workload
   - VS Code: Install C# extension

### Optional

- **Git** for version control
- **Windows 10/11** (WPF is Windows-only)

## Installation Steps

### 1. Verify Prerequisites

Open PowerShell and verify installations:

```powershell
# Check Go
go version

# Check .NET SDK
dotnet --version

# Check GCC (for Go CGO)
gcc --version
```

### 2. Clone Repository

```powershell
git clone <repository-url>
cd DeeMusic
```

### 3. Build Go Backend DLL

```powershell
.\scripts\build-dll.ps1
```

This will create `deemusic-core.dll` in the root directory.

### 4. Restore NuGet Packages

```powershell
cd DeeMusic.Desktop
dotnet restore
```

### 5. Build WPF Application

```powershell
dotnet build
```

Or use the build script:

```powershell
cd ..
.\scripts\build-wpf.ps1
```

### 6. Run the Application

```powershell
cd DeeMusic.Desktop
dotnet run
```

Or run the executable directly:

```powershell
.\bin\Debug\net6.0-windows\DeeMusic.Desktop.exe
```

## Complete Build

To build everything in one command:

```powershell
.\scripts\build-all.ps1
```

For release build:

```powershell
.\scripts\build-all.ps1 -Configuration Release
```

## Development Workflow

### Using Visual Studio 2022

1. Open `DeeMusic.Desktop.sln` in Visual Studio
2. Set `DeeMusic.Desktop` as the startup project
3. Press F5 to build and run

### Using Visual Studio Code

1. Open the `DeeMusic.Desktop` folder in VS Code
2. Install recommended extensions (C#, C# Dev Kit)
3. Press F5 to build and run

### Hot Reload

Visual Studio 2022 supports XAML Hot Reload:
- Make changes to XAML files
- Changes appear immediately without rebuilding

## Project Structure

```
DeeMusic/
├── cmd/deemusic-core/          # Go backend entry point
├── internal/                    # Go backend packages
├── DeeMusic.Desktop/           # C# WPF frontend
│   ├── ViewModels/
│   ├── Views/
│   ├── Models/
│   ├── Services/
│   ├── Controls/
│   └── Resources/
├── scripts/
│   ├── build-dll.ps1           # Build Go DLL
│   ├── build-wpf.ps1           # Build WPF app
│   └── build-all.ps1           # Build everything
└── deemusic-core.dll           # Go backend (generated)
```

## Troubleshooting

### .NET SDK Not Found

**Error:** `No .NET SDKs were found`

**Solution:** Install .NET 8.0 SDK from https://dotnet.microsoft.com/download

### Go DLL Build Fails

**Error:** `gcc: command not found`

**Solution:** Install TDM-GCC or MinGW-w64 for CGO support

### NuGet Package Restore Fails

**Error:** Package restore failed

**Solution:**
```powershell
dotnet nuget locals all --clear
dotnet restore --force
```

### MaterialDesignThemes Not Found

**Error:** Could not load MaterialDesignThemes

**Solution:** Ensure NuGet packages are restored:
```powershell
dotnet restore DeeMusic.Desktop/DeeMusic.Desktop.csproj
```

## Next Steps

Once the project builds successfully:

1. Review the implementation plan: `.kiro/specs/standalone-windows-app/tasks.md`
2. Start implementing features according to the task list
3. Test the P/Invoke integration between C# and Go
4. Implement ViewModels and Views

## Resources

- [WPF Documentation](https://docs.microsoft.com/en-us/dotnet/desktop/wpf/)
- [Material Design in XAML](http://materialdesigninxaml.net/)
- [MVVM Toolkit](https://learn.microsoft.com/en-us/dotnet/communitytoolkit/mvvm/)
- [Go CGO Documentation](https://pkg.go.dev/cmd/cgo)
