# Quick Test Guide - Quality Settings Fix

## What Was Fixed

The app now automatically restarts when you change the download quality setting (MP3 320 ↔ FLAC). This ensures the new quality is properly applied to all downloads.

## How to Test

### 1. Extract and Run
```
Extract: scripts\build\DeeMusic-Portable-v2.2.6.zip
Run: DeeMusic.Desktop.exe
```

### 2. Test Quality Change

**Test A: Change to FLAC**
1. Open Settings (gear icon)
2. Go to Download tab
3. Change "Download Quality" to FLAC
4. Click Save
5. **Expected:** Notification appears "Quality changed to FLAC. Restarting app..."
6. **Expected:** App closes and reopens after 1.5 seconds
7. Add a song to queue and download it
8. **Expected:** File has `.flac` extension and is FLAC format

**Test B: Change to MP3 320**
1. Open Settings
2. Change "Download Quality" to MP3 320
3. Click Save
4. **Expected:** Notification appears "Quality changed to MP3_320. Restarting app..."
5. **Expected:** App restarts
6. Download a song
7. **Expected:** File has `.mp3` extension and is MP3 320 format

**Test C: No Change**
1. Open Settings
2. Don't change quality
3. Click Save
4. **Expected:** Settings save normally, NO restart

### 3. Verify Files

**Check file extension:**
- FLAC downloads should be `.flac`
- MP3 downloads should be `.mp3`

**Check file format:**
- Right-click file → Properties → Details
- FLAC: Should show "FLAC" as audio format
- MP3: Should show "MP3" with 320 kbps bitrate

## What to Look For

✅ **Success Indicators:**
- Notification shows when quality changes
- App restarts automatically
- Downloaded files have correct extension
- Downloaded files are in correct format

❌ **Failure Indicators:**
- No notification when changing quality
- App doesn't restart
- Files have wrong extension (.mp3 for FLAC or vice versa)
- Files are in wrong format (MP3 when FLAC selected)

## Logs to Check (If Issues)

**Frontend Log:**
```
%APPDATA%\DeeMusicV2\logs\deemusic-2026-04-25.log
```
Look for: "Quality changed from X to Y, restarting app..."

**Backend Log:**
```
%APPDATA%\DeeMusicV2\logs\go-backend.log
```
Look for: "UpdateSettings called with quality: FLAC"

**Debug Log:**
```
%TEMP%\deemusic-download-debug.log
```
Look for: "Building output path with format: flac"

## Quick Commands

**View recent frontend log:**
```powershell
Get-Content "$env:APPDATA\DeeMusicV2\logs\deemusic-$(Get-Date -Format 'yyyy-MM-dd').log" -Tail 50
```

**View backend log:**
```powershell
Get-Content "$env:APPDATA\DeeMusicV2\logs\go-backend.log" -Tail 50
```

**View debug log:**
```powershell
Get-Content "$env:TEMP\deemusic-download-debug.log" -Tail 100
```

## Expected Timeline

1. Change quality → Click Save: **Instant**
2. Notification appears: **Instant**
3. App restarts: **1.5 seconds**
4. App fully loaded: **2-3 seconds**
5. Ready to download: **Total ~5 seconds**

## Notes

- The restart is necessary because of WPF binding timing issues
- The 1.5 second delay ensures you see the notification
- Settings are saved before restart, so no data is lost
- Downloads in progress will be interrupted (by design)

---

**Build:** v2.2.6  
**Date:** 2026-04-25  
**Status:** Ready for testing
