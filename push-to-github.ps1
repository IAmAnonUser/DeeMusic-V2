# DeeMusic V2 - Push to GitHub Script
# This script stages, commits, and pushes changes to GitHub

param(
    [string]$CommitMessage = "Update project files"
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  DeeMusic V2 - GitHub Push Script" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if git is installed
try {
    $gitVersion = git --version
    Write-Host "✓ Git found: $gitVersion" -ForegroundColor Green
} catch {
    Write-Host "✗ Git is not installed or not in PATH" -ForegroundColor Red
    Write-Host "  Please install Git from https://git-scm.com/" -ForegroundColor Yellow
    exit 1
}

# Check if we're in a git repository
if (-not (Test-Path ".git")) {
    Write-Host "✗ Not a git repository" -ForegroundColor Red
    Write-Host "  Initializing git repository..." -ForegroundColor Yellow
    git init
    Write-Host "✓ Git repository initialized" -ForegroundColor Green
    
    # Add remote if not exists
    $remoteUrl = "https://github.com/IAmAnonUser/DeeMusic-V2.git"
    Write-Host "  Adding remote origin: $remoteUrl" -ForegroundColor Yellow
    git remote add origin $remoteUrl
    Write-Host "✓ Remote origin added" -ForegroundColor Green
}

Write-Host ""
Write-Host "Checking repository status..." -ForegroundColor Cyan

# Check for changes
$status = git status --porcelain
if ([string]::IsNullOrWhiteSpace($status)) {
    Write-Host "✓ No changes to commit" -ForegroundColor Green
    Write-Host ""
    Write-Host "Repository is up to date!" -ForegroundColor Green
    exit 0
}

Write-Host ""
Write-Host "Changes detected:" -ForegroundColor Yellow
git status --short
Write-Host ""

# Confirm with user
Write-Host "Commit message: " -NoNewline -ForegroundColor Cyan
Write-Host "'$CommitMessage'" -ForegroundColor White
Write-Host ""
$confirm = Read-Host "Do you want to proceed with commit and push? (Y/N)"

if ($confirm -ne "Y" -and $confirm -ne "y") {
    Write-Host "✗ Operation cancelled by user" -ForegroundColor Yellow
    exit 0
}

Write-Host ""
Write-Host "Processing..." -ForegroundColor Cyan
Write-Host ""

# Stage all changes
Write-Host "1. Staging all changes..." -ForegroundColor Yellow
git add .
if ($LASTEXITCODE -eq 0) {
    Write-Host "   ✓ Files staged successfully" -ForegroundColor Green
} else {
    Write-Host "   ✗ Failed to stage files" -ForegroundColor Red
    exit 1
}

# Commit changes
Write-Host "2. Committing changes..." -ForegroundColor Yellow
git commit -m "$CommitMessage"
if ($LASTEXITCODE -eq 0) {
    Write-Host "   ✓ Changes committed successfully" -ForegroundColor Green
} else {
    Write-Host "   ✗ Failed to commit changes" -ForegroundColor Red
    exit 1
}

# Get current branch
$currentBranch = git branch --show-current
if ([string]::IsNullOrWhiteSpace($currentBranch)) {
    Write-Host "   Setting default branch to 'main'..." -ForegroundColor Yellow
    git branch -M main
    $currentBranch = "main"
}

# Push to GitHub
Write-Host "3. Pushing to GitHub (branch: $currentBranch)..." -ForegroundColor Yellow
git push -u origin $currentBranch
if ($LASTEXITCODE -eq 0) {
    Write-Host "   ✓ Successfully pushed to GitHub!" -ForegroundColor Green
} else {
    Write-Host "   ✗ Failed to push to GitHub" -ForegroundColor Red
    Write-Host ""
    Write-Host "Common issues:" -ForegroundColor Yellow
    Write-Host "  • Authentication failed: Use a Personal Access Token (PAT)" -ForegroundColor White
    Write-Host "    Go to: GitHub Settings → Developer settings → Personal access tokens" -ForegroundColor White
    Write-Host "  • Remote rejected: Check if you have write access to the repository" -ForegroundColor White
    Write-Host "  • Branch protection: The branch may have protection rules" -ForegroundColor White
    exit 1
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  ✓ Successfully pushed to GitHub!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Repository: https://github.com/IAmAnonUser/DeeMusic-V2" -ForegroundColor Cyan
Write-Host ""
