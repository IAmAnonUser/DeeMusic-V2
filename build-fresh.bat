@echo off
echo Clearing caches and building fresh...
echo.

REM Kill any running instances
taskkill /F /IM DeeMusic.Desktop.exe 2>nul

REM Wait a moment
timeout /t 2 /nobreak >nul

REM Clean build folders
if exist "DeeMusic.Desktop\bin" rmdir /s /q "DeeMusic.Desktop\bin"
if exist "DeeMusic.Desktop\obj" rmdir /s /q "DeeMusic.Desktop\obj"
if exist "scripts\build" rmdir /s /q "scripts\build"

REM Build using the build script
cd scripts
echo 2 | powershell.exe -ExecutionPolicy Bypass -File "build.ps1" -Version "2.0.0-fresh"

echo.
echo Build complete! Check scripts\build folder for the portable ZIP.
pause
