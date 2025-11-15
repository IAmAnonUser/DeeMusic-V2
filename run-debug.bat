@echo off
echo Killing any running DeeMusic processes...
taskkill /F /IM DeeMusic.Desktop.exe 2>nul
timeout /t 2 /nobreak >nul

echo Starting DeeMusic from Debug folder...
echo Path: %CD%\DeeMusic.Desktop\bin\Debug\net8.0-windows\DeeMusic.Desktop.exe
start "" "%CD%\DeeMusic.Desktop\bin\Debug\net8.0-windows\DeeMusic.Desktop.exe"

echo.
echo App started! You should now see:
echo - Orange text showing "Type: album"
echo - Orange text showing "TotalTracks: X"
echo - Red bold text showing track progress
echo.
pause
