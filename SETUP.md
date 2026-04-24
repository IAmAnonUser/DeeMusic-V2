# DeeMusic Development Environment Setup

This guide will help you set up your development environment for DeeMusic after a fresh Windows installation.

## Quick Setup (Recommended)

1. **Download the repository** (or clone if Git is already installed)
2. **Right-click** `scripts/setup-dev-environment.bat`
3. **Select** "Run as administrator"
4. **Follow the prompts** - the script will install everything automatically

The setup script will install:
- Windows Package Manager (winget)
- Git
- GitHub CLI
- Go (Golang)
- .NET 8 SDK
- MinGW-w64 (GCC for Windows)
- Visual Studio Code (optional)

## Manual Setup

If you prefer to install tools manually, here's what you need:

### 1. Git
```powershell
winget install Git.Git
```
Or download from: https://git-scm.com/download/win

### 2. GitHub CLI
```powershell
winget install GitHub.cli
```
Or download from: https://cli.github.com/

### 3. Go (Golang)
```powershell
winget install GoLang.Go
```
Or download from: https://go.dev/dl/

### 4. .NET 8 SDK
```powershell
winget install Microsoft.DotNet.SDK.8
```
Or download from: https://dotnet.microsoft.com/download/dotnet/8.0

### 5. MinGW-w64 (GCC for Windows)
```powershell
winget install mingw.mingw
```
Or download from: https://www.mingw-w64.org/

**Important**: After installing MinGW, add it to your PATH:
- Default location: `C:\mingw64\bin`
- Add to System Environment Variables → Path

### 6. Visual Studio Code (Optional)
```powershell
winget install Microsoft.VisualStudioCode
```
Or download from: https://code.visualstudio.com/

## After Installation

### 1. Configure Git
```bash
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"
```

### 2. Authenticate with GitHub
```bash
gh auth login
```
Follow the prompts to authenticate with your GitHub account.

### 3. Clone the Repository
```bash
git clone https://github.com/IAmAnonUser/DeeMusic-V2.git
cd DeeMusic-V2
```

### 4. Build and Run
```bash
.\run.bat
```

This will:
- Build the Go backend (deemusic-core.dll)
- Build the C# frontend (DeeMusic.Desktop.exe)
- Launch the application

## Verify Installation

Run these commands to verify everything is installed:

```powershell
git --version
gh --version
go version
dotnet --version
gcc --version
```

All commands should return version information without errors.

## Troubleshooting

### "Command not found" errors
- Close and reopen your terminal/PowerShell
- The PATH environment variable needs to be refreshed

### MinGW/GCC not found
- Verify MinGW is installed at `C:\mingw64`
- Check that `C:\mingw64\bin` is in your PATH
- Restart your terminal after adding to PATH

### .NET SDK not found
- Install .NET 8 SDK (not just Runtime)
- Restart your terminal after installation

### Go build errors
- Ensure Go is version 1.21 or higher: `go version`
- Verify CGO is enabled: `go env CGO_ENABLED` (should be "1")

### Permission errors during setup
- Run the setup script as Administrator
- Right-click → "Run as administrator"

## Development Tools (Optional)

### Kiro IDE
If you're using Kiro for AI-assisted development, no additional setup is needed. Kiro will use the installed tools automatically.

### Visual Studio Code Extensions
If using VS Code, install these extensions:
- Go (golang.go)
- C# (ms-dotnettools.csharp)
- PowerShell (ms-vscode.powershell)

## Project Structure

```
DeeMusic-V2/
├── cmd/deemusic-core/     # Go backend entry point
├── internal/              # Go backend packages
│   ├── api/              # Deezer API client
│   ├── download/         # Download manager
│   ├── decryption/       # Audio decryption
│   └── ...
├── DeeMusic.Desktop/      # C# WPF frontend
│   ├── Views/            # XAML views
│   ├── ViewModels/       # View models
│   ├── Services/         # Services
│   └── Models/           # Data models
├── scripts/               # Build and utility scripts
│   ├── run.bat           # Quick build and run
│   ├── github-manager.bat # GitHub operations
│   └── setup-dev-environment.bat # This setup script
└── docs/                  # Documentation

```

## Build Scripts

- `run.bat` - Build and run the application
- `scripts/github-manager.bat` - Manage Git operations, releases, etc.
- `scripts/create-release.ps1` - Create a new release

## Next Steps

1. Read the main [README.md](README.md) for project overview
2. Check [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines (if exists)
3. Review the code structure in `internal/` and `DeeMusic.Desktop/`
4. Run `.\run.bat` to build and test the application

## Getting Help

- Check the logs in `%APPDATA%\DeeMusicV2\logs\`
- Review debug logs in `%TEMP%\deemusic-download-debug.log`
- Open an issue on GitHub if you encounter problems

---

**Last Updated**: 2025-03-29
**Version**: 2.2.5
