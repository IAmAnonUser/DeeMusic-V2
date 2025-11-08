# DeeMusic

A modern Windows desktop application for downloading music from Deezer with high-quality audio, automatic metadata, and multi-disc album support.

![Windows](https://img.shields.io/badge/Windows-10%2F11-blue?logo=windows)
![.NET](https://img.shields.io/badge/.NET-8.0-512BD4?logo=dotnet)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![License](https://img.shields.io/badge/License-MIT-green)

## Features

### 🎵 Download & Quality
- Download tracks, albums, playlists, and artist discographies
- **MP3 320kbps** or **FLAC lossless** quality
- Concurrent downloads (1-12 simultaneous)
- Automatic retry on failure
- Resume interrupted downloads

### 🎨 Metadata & Organization
- Automatic ID3 tags (artist, album, title, year, genre)
- High-resolution album artwork (up to 1500x1500)
- Lyrics download and embedding
- **Multi-disc album support** with CD folders
- Customizable filename templates
- Flexible folder structure

### 🖥️ Modern Interface
- Native Windows WPF application
- Dark and light themes
- Real-time download progress
- Queue management (pause, resume, cancel)
- System tray integration
- Search with instant results

### ⚡ Performance
- Go backend for high-speed downloads
- Efficient memory usage
- Handles 10,000+ queue items
- SQLite database for persistence
- Local-only operation (no HTTP server)

### 📦 Distribution
- **Installer**: Traditional Windows setup with Start Menu integration
- **Portable**: Zero-installation, run from USB drive
- **Self-contained**: No prerequisites required (.NET runtime included)
- Works on any Windows 10/11 (64-bit) PC

## Quick Start

### Download

Get the latest release from [Releases](https://github.com/yourusername/deemusic-go/releases):

**Installer (Recommended)**
- `DeeMusic-Setup-{version}.exe` (~150 MB)
- Installs to Program Files
- Start Menu shortcuts
- Automatic updates

**Portable**
- `DeeMusic-Portable-{version}.zip` (~150 MB)
- Extract and run anywhere
- No installation needed
- Perfect for USB drives
- **Note**: Extract the ZIP first, then run `DeeMusic.Desktop.exe` from the extracted folder

### Setup

1. **Get Deezer ARL Token**
   - Log in to [deezer.com](https://www.deezer.com)
   - Press F12 → Application → Cookies → deezer.com
   - Copy the `arl` cookie value (192 characters)

2. **Configure DeeMusic**
   - Open Settings (⚙️)
   - Paste your ARL token
   - Set download folder and quality
   - Save

3. **Start Downloading**
   - Search for music
   - Click download buttons
   - Monitor progress in queue

## System Requirements

### For Users
- Windows 10 or Windows 11 (64-bit)
- 4 GB RAM (8 GB recommended)
- 200 MB disk space + downloads
- Internet connection

**No other software required!** Both installer and portable versions include everything needed.

### For Developers
- Go 1.21+ (backend)
- .NET 8.0 SDK (frontend)
- NSIS 3.0+ (installer, optional)
- MinGW-w64 (Go CGO)

## Building from Source

### Quick Build

```powershell
# Build both installer and portable versions
.\scripts\build-release.ps1

# Skip tests for faster build
.\scripts\build-release.ps1 -SkipTests

# Clean build with specific version
.\scripts\build-release.ps1 -Version "2.1.0" -Clean
```

Output in `dist/` folder:
- `DeeMusic-Setup-{version}.exe` - Windows installer
- `DeeMusic-Portable-{version}.zip` - Portable package
- `checksums-{version}.txt` - SHA256 checksums

### Manual Build

```powershell
# 1. Build Go backend
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "1"
go build -buildmode=c-shared -o cmd/deemusic-core/deemusic-core.dll cmd/deemusic-core/main.go

# 2. Build C# frontend
dotnet publish DeeMusic.Desktop/DeeMusic.Desktop.csproj -c Release -r win-x64 --self-contained true

# 3. Copy DLL to output
Copy-Item cmd/deemusic-core/deemusic-core.dll DeeMusic.Desktop/bin/Release/net8.0-windows/publish/
```

See [scripts/README.md](scripts/README.md) for detailed build instructions.

## Architecture

**Frontend**: C# WPF with MVVM pattern
- Native Windows UI with Material Design
- P/Invoke for Go DLL integration
- Async/await for responsive UI

**Backend**: Go compiled as C-shared DLL
- High-performance download engine
- Deezer API integration
- Audio decryption and metadata
- SQLite queue persistence

**Communication**: Direct function calls via P/Invoke
- C# → Go: Function calls
- Go → C#: Callback delegates for progress
- No HTTP server, fully local

## Configuration

Settings stored in `%APPDATA%\DeeMusicV2\settings.json`:

```json
{
  "deezer": {
    "arl": "your_arl_token_here"
  },
  "download": {
    "output_dir": "C:\\Users\\YourName\\Music\\DeeMusic",
    "quality": "MP3_320",
    "concurrent_downloads": 8,
    "create_cd_folder": true,
    "cd_folder_template": "CD {disc_number}",
    "embed_artwork": true,
    "artwork_size": 1200
  }
}
```

## Multi-Disc Album Support

DeeMusic automatically detects and organizes multi-disc albums:

```
Artist/
└── Album Name/
    ├── CD 1/
    │   ├── 01 - Track.mp3
    │   └── 02 - Track.mp3
    └── CD 2/
        ├── 01 - Track.mp3
        └── 02 - Track.mp3
```

Configurable via `create_cd_folder` and `cd_folder_template` settings.

## Project Structure

```
deemusic/
├── cmd/deemusic-core/      # Go DLL entry point
├── internal/               # Go backend packages
│   ├── api/               # Deezer API client
│   ├── download/          # Download manager
│   ├── decryption/        # Audio decryption
│   ├── metadata/          # Metadata & lyrics
│   └── store/             # SQLite persistence
├── DeeMusic.Desktop/       # C# WPF frontend
│   ├── ViewModels/        # MVVM ViewModels
│   ├── Views/             # XAML views
│   ├── Services/          # P/Invoke wrapper
│   └── Resources/         # Themes & styles
└── scripts/               # Build scripts
```

## Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Disclaimer

**For personal use only.** Respect copyright laws and artist rights. Only download music you have the right to access. The developers are not responsible for misuse of this software.

## Acknowledgments

- Original Python version of DeeMusic
- Deezer for their music streaming service
- Open source community

---

**Made with ❤️ for music lovers**
