# DeeMusic Build Script
# Simple menu-driven build for installer or portable version

param(
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot\..

# Show menu
Clear-Host
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "DeeMusic Build Menu" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Prompt for version if not provided
if ([string]::IsNullOrWhiteSpace($Version)) {
    Write-Host "Enter version number (e.g., 2.0.3):" -ForegroundColor Yellow -NoNewline
    Write-Host " " -NoNewline
    $Version = Read-Host
    
    if ([string]::IsNullOrWhiteSpace($Version)) {
        Write-Host "ERROR: Version number is required" -ForegroundColor Red
        exit 1
    }
    
    Write-Host ""
}

Write-Host "Building Version: $Version" -ForegroundColor Green
Write-Host ""
Write-Host "1. Build Installer" -ForegroundColor White
Write-Host "2. Build Portable ZIP" -ForegroundColor White
Write-Host "3. Build Both" -ForegroundColor White
Write-Host "4. Exit" -ForegroundColor White
Write-Host ""
$choice = Read-Host "Select option (1-4)"

if ($choice -eq "4") {
    Write-Host "Cancelled." -ForegroundColor Yellow
    exit 0
}

# Verify prerequisites
Write-Host ""
Write-Host "Checking prerequisites..." -ForegroundColor Cyan

$hasGo = $null -ne (Get-Command go -ErrorAction SilentlyContinue)
$hasDotnet = $null -ne (Get-Command dotnet -ErrorAction SilentlyContinue)
$nsisPath = @(
    "${env:ProgramFiles}\NSIS\makensis.exe",
    "${env:ProgramFiles(x86)}\NSIS\makensis.exe"
) | Where-Object { Test-Path $_ } | Select-Object -First 1

if (-not $hasGo) { Write-Host "ERROR: Go not found" -ForegroundColor Red; exit 1 }
if (-not $hasDotnet) { Write-Host "ERROR: .NET SDK not found" -ForegroundColor Red; exit 1 }
if (($choice -eq "1" -or $choice -eq "3") -and -not $nsisPath) { 
    Write-Host "ERROR: NSIS not found (required for installer)" -ForegroundColor Red
    exit 1
}

Write-Host "Prerequisites OK" -ForegroundColor Green
Write-Host ""

# Build Go DLL
Write-Host "Building Go backend..." -ForegroundColor Cyan
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "1"
$env:CGO_LDFLAGS = "-static-libgcc -static-libstdc++"
go build -buildmode=c-shared -ldflags="-s -w" -o cmd\deemusic-core\deemusic-core.dll cmd\deemusic-core\main.go
if ($LASTEXITCODE -ne 0) { Write-Host "Go build failed" -ForegroundColor Red; exit 1 }
Write-Host "Go DLL built" -ForegroundColor Green
Write-Host ""

# Build C# application
Write-Host "Building C# application..." -ForegroundColor Cyan
$publishDir = "DeeMusic.Desktop\bin\Release\net8.0-windows\win-x64\publish"
dotnet publish DeeMusic.Desktop\DeeMusic.Desktop.csproj -c Release -r win-x64 --self-contained true -o $publishDir 2>&1 | Out-Null
$buildResult = $LASTEXITCODE
if ($buildResult -ne 0) { 
    Write-Host "C# build failed with exit code $buildResult" -ForegroundColor Red
    Write-Host "Running build again with output..." -ForegroundColor Yellow
    dotnet publish DeeMusic.Desktop\DeeMusic.Desktop.csproj -c Release -r win-x64 --self-contained true -o $publishDir
    exit 1
}

# Copy DLL
Copy-Item cmd\deemusic-core\deemusic-core.dll $publishDir\ -Force

Write-Host "C# application built" -ForegroundColor Green
Write-Host ""

# Create build output directory
$buildDir = "$PSScriptRoot\build"
New-Item -ItemType Directory -Path $buildDir -Force | Out-Null

# Build Installer
if ($choice -eq "1" -or $choice -eq "3") {
    if (Test-Path "installer\DeeMusic.nsi") {
        Write-Host "Building installer with version $Version..." -ForegroundColor Cyan
        
        # Update version in NSIS script
        $nsiContent = Get-Content "installer\DeeMusic.nsi" -Raw
        $nsiContent = $nsiContent -replace '!define PRODUCT_VERSION ".*"', "!define PRODUCT_VERSION `"$Version`""
        $nsiContent | Set-Content "installer\DeeMusic.nsi" -NoNewline
        
        Write-Host "Updated installer version to $Version" -ForegroundColor Gray
        
        & $nsisPath /V2 "installer\DeeMusic.nsi" | Out-Null
        
        if ($LASTEXITCODE -eq 0) {
            # Move installer to build folder
            $installerFile = Get-ChildItem "." -Filter "DeeMusic-Setup-*.exe" | Select-Object -First 1
            if ($installerFile) {
                Move-Item $installerFile.FullName "$buildDir\" -Force
            }
            Write-Host "Installer built successfully" -ForegroundColor Green
        } else {
            Write-Host "Installer build failed" -ForegroundColor Red
            exit 1
        }
        Write-Host ""
    } else {
        Write-Host "Installer script not available - skipping" -ForegroundColor Yellow
        if ($choice -eq "1") {
            Write-Host "Use option 2 to build portable version instead" -ForegroundColor Yellow
            exit 0
        }
        Write-Host ""
    }
}

# Build Portable
if ($choice -eq "2" -or $choice -eq "3") {
    Write-Host "Building portable version $Version..." -ForegroundColor Cyan
    
    $portableDir = "$buildDir\DeeMusic-Portable"
    $publishDir = "DeeMusic.Desktop\bin\Release\net8.0-windows\win-x64\publish"
    
    # Clean old build (retry if locked)
    if (Test-Path $portableDir) {
        try {
            Remove-Item -Recurse -Force $portableDir -ErrorAction Stop
        } catch {
            Write-Host "Warning: Could not delete old portable folder (files in use)" -ForegroundColor Yellow
            $portableDir = "$buildDir\DeeMusic-Portable-$(Get-Date -Format 'HHmmss')"
        }
    }
    
    # Create new directory
    New-Item -ItemType Directory -Path $portableDir -Force | Out-Null
    
    # Copy all files
    Copy-Item "$publishDir\*" $portableDir -Recurse -Force
    
    # Note: .portable marker NOT created - portable version will use AppData like installed version
    # This ensures queue and settings persist even if portable folder is deleted/moved
    
    # Create settings template
    @"
{
  "deezer": { "arl": "" },
  "download": {
    "output_dir": "downloads",
    "quality": "MP3_320",
    "concurrent_downloads": 8
  }
}
"@ | Out-File -FilePath "$portableDir\settings.template.json" -Encoding UTF8
    
    # Create README
    $buildDate = Get-Date -Format "yyyy-MM-dd"
    @"
DeeMusic Portable v$Version

IMPORTANT: This portable version stores data in AppData, not in this folder.
Your queue, settings, and logs are saved to:
  %APPDATA%\DeeMusicV2\

This means:
- Queue persists even if you move/delete this folder
- Settings are shared with the installed version (if you have both)
- You can run from anywhere (USB drive, Downloads, etc.)

Quick Start:
1. Run DeeMusic.Desktop.exe
2. Go to Settings (gear icon)
3. Add your Deezer ARL token
4. Set your download folder
5. Start downloading!

Get ARL Token:
1. Login to deezer.com
2. Press F12 for DevTools
3. Go to Application > Cookies > deezer.com
4. Copy the 'arl' cookie value (192 characters)

Version: $Version
Build: $buildDate
"@ | Out-File -FilePath "$portableDir\README.txt" -Encoding UTF8
    
    # Create ZIP
    $zipName = "DeeMusic-Portable-v$Version.zip"
    $zipPath = "$buildDir\$zipName"
    Remove-Item $zipPath -Force -ErrorAction SilentlyContinue
    
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    [System.IO.Compression.ZipFile]::CreateFromDirectory($portableDir, $zipPath, [System.IO.Compression.CompressionLevel]::Optimal, $false)
    
    Write-Host "Portable ZIP created" -ForegroundColor Green
    Write-Host ""
}

# Summary
Write-Host "========================================" -ForegroundColor Green
Write-Host "Build Complete!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Output in scripts\build\ folder" -ForegroundColor Cyan
Write-Host ""

if ($choice -eq "1" -or $choice -eq "3") {
    $installer = Get-ChildItem $buildDir -Filter "*.exe" -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($installer) {
        Write-Host "Installer: $($installer.Name)" -ForegroundColor White
    }
}

if ($choice -eq "2" -or $choice -eq "3") {
    $portable = Get-ChildItem $buildDir -Filter "*.zip" -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($portable) {
        Write-Host "Portable: $($portable.Name)" -ForegroundColor White
    }
}

Write-Host ""

# If running interactively (not from another script), pause
if ([Environment]::UserInteractive -and -not ([Environment]::GetCommandLineArgs() -like '*-NonInteractive*')) {
    Write-Host "Press any key to exit..." -ForegroundColor Gray
    $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
}
