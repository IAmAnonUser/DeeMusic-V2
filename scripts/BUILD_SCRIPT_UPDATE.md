# Build Script Update - Version Prompt

## Changes Made

The build script (`scripts/build.ps1`) has been updated to prompt for the version number at the start of the build process.

## What's New

### Version Prompt
- When you run the build script, it will now ask: **"Enter version number (e.g., 2.0.3):"**
- The version you enter will be used for both:
  - **Installer**: Updates the NSIS script with the version
  - **Portable ZIP**: Names the file with the version (e.g., `DeeMusic-Portable-v2.0.3.zip`)

### Usage

**Interactive Mode:**
```powershell
.\scripts\build.ps1
# You'll be prompted to enter the version
```

**Command Line Mode:**
```powershell
.\scripts\build.ps1 -Version "2.0.3"
# Version is provided, no prompt
```

**From build.bat:**
```cmd
.\build.bat
# Will prompt for version when you select build option
```

## Example Session

```
========================================
DeeMusic Build Menu
========================================

Enter version number (e.g., 2.0.3): 2.0.3

Building Version: 2.0.3

1. Build Installer
2. Build Portable ZIP
3. Build Both
4. Exit

Select option (1-4): 3
```

## Files Updated

- `scripts/build.ps1` - Main build script
  - Added version prompt at startup
  - Shows version being built in console output
  - Updates NSIS installer version
  - Names portable ZIP with version

## Benefits

- ✅ No need to manually edit version numbers in multiple files
- ✅ Consistent versioning across installer and portable builds
- ✅ Clear feedback about which version is being built
- ✅ Can still pass version via command line parameter
- ✅ Prevents building with wrong/outdated version numbers

## Next Steps

When building a release:
1. Run `.\build.bat` or `.\scripts\build.ps1`
2. Enter the version number (e.g., `2.0.3`)
3. Select build option (1, 2, or 3)
4. Script will automatically version both outputs

The generated files will be:
- `scripts\build\DeeMusic-Setup-2.0.3.exe` (Installer)
- `scripts\build\DeeMusic-Portable-v2.0.3.zip` (Portable)
