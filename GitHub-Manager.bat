@echo off
REM DeeMusic GitHub Manager - Quick Launch

title DeeMusic GitHub Manager

REM Get the directory where this batch file is located
set BATCH_DIR=%~dp0

REM Change to that directory
cd /d "%BATCH_DIR%"

REM Run the GitHub manager script
powershell.exe -ExecutionPolicy Bypass -NoProfile -File "%BATCH_DIR%scripts\github-manager.ps1"

REM Keep window open if there was an error
if errorlevel 1 (
    echo.
    echo An error occurred. Press any key to exit...
    pause >nul
)
