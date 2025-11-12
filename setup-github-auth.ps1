# DeeMusic V2 - GitHub Authentication Setup
# This script helps you set up GitHub authentication

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  GitHub Authentication Setup" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "To push to GitHub, you need to authenticate." -ForegroundColor Yellow
Write-Host ""
Write-Host "Option 1: Personal Access Token (Recommended)" -ForegroundColor Green
Write-Host "  1. Go to: https://github.com/settings/tokens" -ForegroundColor White
Write-Host "  2. Click 'Generate new token (classic)'" -ForegroundColor White
Write-Host "  3. Select scope: 'repo' (full control)" -ForegroundColor White
Write-Host "  4. Generate and copy the token" -ForegroundColor White
Write-Host ""
Write-Host "Option 2: GitHub CLI" -ForegroundColor Green
Write-Host "  1. Install GitHub CLI: https://cli.github.com/" -ForegroundColor White
Write-Host "  2. Run: gh auth login" -ForegroundColor White
Write-Host ""

$choice = Read-Host "Do you have a Personal Access Token ready? (Y/N)"

if ($choice -eq "Y" -or $choice -eq "y") {
    Write-Host ""
    Write-Host "Great! Now run this command to push:" -ForegroundColor Green
    Write-Host ""
    Write-Host "  git push -u origin main" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "When prompted:" -ForegroundColor Yellow
    Write-Host "  Username: IAmAnonUser" -ForegroundColor White
    Write-Host "  Password: [Paste your Personal Access Token]" -ForegroundColor White
    Write-Host ""
    Write-Host "The token will be saved securely in Windows Credential Manager." -ForegroundColor Gray
} else {
    Write-Host ""
    Write-Host "Please create a Personal Access Token first:" -ForegroundColor Yellow
    Write-Host "  https://github.com/settings/tokens" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Then run this script again." -ForegroundColor White
}

Write-Host ""
Write-Host "Press any key to exit..."
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
