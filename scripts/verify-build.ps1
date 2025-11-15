# Build Verification Script
# Checks if the build is truly self-contained

param(
    [string]$BuildPath = "DeeMusic.Desktop\bin\Release\net8.0-windows\win-x64\publish"
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Build Verification" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

if (-not (Test-Path $BuildPath)) {
    Write-Host "ERROR: Build path not found: $BuildPath" -ForegroundColor Red
    Write-Host "Run the build script first!" -ForegroundColor Yellow
    exit 1
}

Write-Host "Checking build at: $BuildPath" -ForegroundColor White
Write-Host ""

# Check for main executable
$exeFile = Join-Path $BuildPath "DeeMusic.Desktop.exe"
if (Test-Path $exeFile) {
    Write-Host "[OK] Main executable found" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Main executable NOT found" -ForegroundColor Red
    exit 1
}

# Check for Go DLL
$goDll = Join-Path $BuildPath "deemusic-core.dll"
if (Test-Path $goDll) {
    Write-Host "[OK] Go backend DLL found" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Go backend DLL NOT found" -ForegroundColor Red
    exit 1
}

# Check for .NET runtime files (self-contained check)
$runtimeFiles = @(
    "System.Runtime.dll",
    "System.Private.CoreLib.dll",
    "hostfxr.dll",
    "hostpolicy.dll"
)

$missingRuntime = @()
foreach ($file in $runtimeFiles) {
    $filePath = Join-Path $BuildPath $file
    if (Test-Path $filePath) {
        Write-Host "[OK] Runtime file: $file" -ForegroundColor Green
    } else {
        Write-Host "[WARN] Runtime file missing: $file" -ForegroundColor Yellow
        $missingRuntime += $file
    }
}

# Check for WPF dependencies
$wpfFiles = @(
    "PresentationCore.dll",
    "PresentationFramework.dll",
    "WindowsBase.dll"
)

$missingWpf = @()
foreach ($file in $wpfFiles) {
    $filePath = Join-Path $BuildPath $file
    if (Test-Path $filePath) {
        Write-Host "[OK] WPF file: $file" -ForegroundColor Green
    } else {
        Write-Host "[WARN] WPF file missing: $file" -ForegroundColor Yellow
        $missingWpf += $file
    }
}

# Check for NuGet packages
$packageFiles = @(
    "MaterialDesignThemes.Wpf.dll",
    "CommunityToolkit.Mvvm.dll"
)

foreach ($file in $packageFiles) {
    $filePath = Join-Path $BuildPath $file
    if (Test-Path $filePath) {
        Write-Host "[OK] Package: $file" -ForegroundColor Green
    } else {
        Write-Host "[WARN] Package missing: $file" -ForegroundColor Yellow
    }
}

# Count total files
$totalFiles = (Get-ChildItem $BuildPath -Recurse -File).Count
Write-Host ""
Write-Host "Total files in build: $totalFiles" -ForegroundColor Cyan

# Check build size
$totalSize = (Get-ChildItem $BuildPath -Recurse -File | Measure-Object -Property Length -Sum).Sum / 1MB
Write-Host "Total build size: $([math]::Round($totalSize, 2)) MB" -ForegroundColor Cyan

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan

# Summary
if ($missingRuntime.Count -eq 0 -and $missingWpf.Count -eq 0) {
    Write-Host "Build Verification: PASSED" -ForegroundColor Green
    Write-Host "This is a self-contained build!" -ForegroundColor Green
} else {
    Write-Host "Build Verification: WARNING" -ForegroundColor Yellow
    if ($missingRuntime.Count -gt 0) {
        Write-Host "Missing runtime files: $($missingRuntime -join ', ')" -ForegroundColor Yellow
    }
    if ($missingWpf.Count -gt 0) {
        Write-Host "Missing WPF files: $($missingWpf -join ', ')" -ForegroundColor Yellow
    }
    Write-Host ""
    Write-Host "The build may still work if these are embedded or not required." -ForegroundColor Gray
}

Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
