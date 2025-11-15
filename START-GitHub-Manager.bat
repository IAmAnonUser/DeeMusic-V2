@echo off
REM DeeMusic GitHub Manager - START HERE!

title DeeMusic GitHub Manager

echo.
echo ========================================
echo   DeeMusic GitHub Manager
echo ========================================
echo.
echo   Starting interactive manager...
echo.

REM Get the directory where this batch file is located
set BATCH_DIR=%~dp0

REM Change to that directory
cd /d "%BATCH_DIR%"

REM Run the GitHub manager script with full path
powershell.exe -ExecutionPolicy Bypass -NoProfile -File "%BATCH_DIR%scripts\github-manager.ps1"

REM Keep window open if there was an error
if errorlevel 1 (
    echo.
    echo ========================================
    echo   An error occurred!
    echo ========================================
    echo.
    echo   Check the error message above.
    echo   Press any key to exit...
    echo.
    pause >nul
)
