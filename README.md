# DeeMusic V2

A modern, high-performance music downloader for D**zer with sp*tify playlist import support. Built with C# (WPF) frontend and Go backend for optimal performance.

![Platform](https://img.shields.io/badge/platform-Windows-lightgrey.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)

## ✨ Features

### Core Functionality
- 🎵 **Download Music from D**zer**
  - Individual tracks
  - Full albums with proper metadata
  - Playlists
  - High-quality audio (MP3 320kbps or FLAC)

- 🎧 **sp*tify Integration**
  - Import sp*tify playlists directly
  - Automatic track matching on D**zer
  - Preserves playlist structure and metadata
  - Download sp*tify playlists as D**zer tracks

### Advanced Features
- ⚡ **High-Performance Downloads**
  - Concurrent downloads (1-12 simultaneous)
  - Go-powered backend for speed
  - Efficient memory management
  - Resume failed downloads

- 📁 **Smart Organization**
  - Customizable folder structure
  - Artist/Album/Track organization
  - Multi-disc album support with CD folders
  - Automatic artwork download (up to 1500x1500)

- 🎨 **Modern UI**
  - Clean, intuitive interface
  - Real-time download progress
  - Queue management
  - Search with filters (tracks, albums, artists, playlists)
  - Featured content on home page

- 🔧 **Flexible Configuration**
  - Custom download paths
  - Filename templates
  - Quality settings
  - Concurrent download limits
  - sp*tify API integration

### Metadata & Tagging
- 📝 **Complete ID3 Tags**
  - Title, Artist, Album
  - Track number and disc number
  - Album artist
  - Release date
  - Genre
  - ISRC codes
  - Embedded album artwork

## 🚀 Getting Started

### Prerequisites
- Windows 10/11
- .NET 8.0 Runtime
- D**zer Premium account (ARL token required)
- (Optional) sp*tify API credentials for playlist import

### Installation

1. **Download the latest release** from the [Releases](https://github.com/IAmAnonUser/DeeMusic-V2/releases) page

2. **Extract the archive** to your preferred location

3. **Run `DeeMusic.Desktop.exe`**

4. **Configure on first launch:**
   - Enter your D**zer ARL token
   - Set your download directory
   - (Optional) Add sp*tify API credentials

### Getting Your D**zer ARL Token

1. Log into [d**zer.com](https://www.d**zer.com) in your browser
2. Open Developer Tools (F12)
3. Go to **Application** → **Cookies** → `https://www.d**zer.com`
4. Copy the value of the `arl` cookie (192 characters)
5. Paste it into DeeMusic settings

### Getting sp*tify API Credentials (Optional)

1. Go to [sp*tify Developer Dashboard](https://developer.sp*tify.com/dashboard)
2. Create a new app
3. Copy the **Client ID** and **Client Secret**
4. Add them in DeeMusic Settings → sp*tify Integration

## 📖 Usage

### Downloading Music

**Search and Download:**
1. Enter a search query in the search box
2. Filter by type (All, Tracks, Albums, Artists, Playlists)
3. Click the download button on any result

**Import sp*tify Playlist:**
1. Copy a sp*tify playlist URL
2. Paste it into the search box
3. Press Enter
4. Review matched tracks
5. Click "Download Playlist"

### Queue Management

- View all downloads in the Queue tab
- Monitor progress in real-time
- Pause/Resume individual downloads
- Retry failed downloads
- Clear completed downloads

### Settings

**Download Settings:**
- Output directory
- Audio quality (MP3 320 / FLAC)
- Concurrent downloads (1-12)
- Create CD folders for multi-disc albums
- Artwork size (up to 1500x1500)

**Filename Templates:**
- Customize folder structure
- Available placeholders:
  - `{artist}`, `{album_artist}`
  - `{album}`, `{title}`
  - `{track_number}`, `{disc_number}`
  - `{year}`, `{label}`

**Example Templates:**
- Album track: `{track_number:02d} - {artist} - {title}`
- Folder: `{artist}/{album}`
- CD folder: `CD {disc_number}`

## 🏗️ Architecture

### Technology Stack
- **Frontend:** C# / WPF (.NET 8.0)
- **Backend:** Go (compiled as C DLL)
- **Database:** SQLite
- **APIs:** D**zer API, sp*tify Web API

### Project Structure
```
DeeMusic-V2/
├── DeeMusic.Desktop/          # C# WPF Application
│   ├── ViewModels/            # MVVM ViewModels
│   ├── Views/                 # WPF Views
│   ├── Services/              # Service layer
│   └── Models/                # Data models
├── internal/                  # Go Backend
│   ├── api/                   # API clients (D**zer, sp*tify)
│   ├── download/              # Download manager
│   ├── store/                 # Database layer
│   └── config/                # Configuration
├── cmd/deemusic-core/         # Go DLL entry point
└── docs/                      # Documentation
```

## 🔧 Building from Source

### Prerequisites
- Visual Studio 2022 or later
- .NET 8.0 SDK
- Go 1.21 or later
- MinGW-w64 (for Go DLL compilation)

### Build Steps

1. **Clone the repository:**
```bash
git clone https://github.com/IAmAnonUser/DeeMusic-V2.git
cd DeeMusic-V2
```

2. **Build the Go backend:**
```bash
go build -buildmode=c-shared -o deemusic-core.dll ./cmd/deemusic-core
```

3. **Build the C# frontend:**
```bash
dotnet build DeeMusic.Desktop/DeeMusic.Desktop.csproj
```

4. **Copy the DLL:**
```bash
copy deemusic-core.dll DeeMusic.Desktop\bin\Debug\net8.0-windows\
```

5. **Run:**
```bash
dotnet run --project DeeMusic.Desktop/DeeMusic.Desktop.csproj
```

Or use the provided build script:
```bash
.\run.bat
```

## 📝 Recent Updates

### Version 2.0.0 (Latest)

**New Features:**
- ✅ sp*tify playlist import with automatic track matching
- ✅ Multi-disc album support with CD folder creation
- ✅ High-resolution artwork download (up to 1500x1500)
- ✅ Featured content on home page (New Releases, Top Albums, etc.)
- ✅ Improved search with category filters
- ✅ Real-time download progress tracking

**Improvements:**
- ✅ Fixed progress bar accuracy for playlists
- ✅ Prevented duplicate track downloads
- ✅ Better error handling and retry logic
- ✅ Improved navigation flow
- ✅ Enhanced UI responsiveness
- ✅ Optimized concurrent download performance

**Bug Fixes:**
- ✅ Fixed playlist progress not reaching 100%
- ✅ Fixed navigation issues after sp*tify import
- ✅ Fixed multi-disc album detection
- ✅ Fixed track retry logic causing progress drops
- ✅ Fixed completed tracks being re-downloaded

## 🐛 Known Issues

- Some sp*tify tracks may not be available on D**zer
- Very large playlists (500+ tracks) may take time to process
- D**zer ARL tokens expire periodically and need to be refreshed

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## ⚠️ Disclaimer

This tool is for educational purposes only. Users are responsible for complying with D**zer's Terms of Service and applicable copyright laws. The developers are not responsible for any misuse of this software.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- D**zer for their music streaming service
- sp*tify for their Web API
- The Go and .NET communities for excellent tools and libraries

## 📧 Support

For issues, questions, or suggestions:
- Open an [Issue](https://github.com/IAmAnonUser/DeeMusic-V2/issues)
- Check existing issues for solutions

---

**Note:** This is a complete rewrite of DeeMusic V1 (Python) with improved performance, stability, and features.
