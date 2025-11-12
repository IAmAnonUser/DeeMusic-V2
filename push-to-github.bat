@echo off
REM DeeMusic V2 - Push to GitHub Script (Batch wrapper)
REM This script calls the PowerShell script with proper execution policy

setlocal

REM Check if commit message was provided
set "COMMIT_MSG=%~1"
if "%COMMIT_MSG%"=="" set "COMMIT_MSG=Update project files"

echo.
echo ========================================
echo   DeeMusic V2 - GitHub Push Script
echo ========================================
echo.

REM Run PowerShell script with bypass execution policy
powershell.exe -ExecutionPolicy Bypass -File "%~dp0push-to-github.ps1" -CommitMessage "%COMMIT_MSG%"

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo Press any key to exit...
    pause >nul
    exit /b %ERRORLEVEL%
)

echo.
echo Press any key to exit...
pause >nul
