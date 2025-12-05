using System;
using System.Diagnostics;
using System.IO;
using System.IO.Compression;
using System.Net.Http;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Threading.Tasks;
using System.Windows;

namespace DeeMusic.Desktop.Services
{
    /// <summary>
    /// Service for checking and applying application updates from GitHub releases
    /// </summary>
    public class UpdateService
    {
        private static UpdateService? _instance;
        private readonly HttpClient _httpClient;
        private const string GitHubApiUrl = "https://api.github.com/repos/IAmAnonUser/DeeMusic-V2/releases/latest";
        
        // Get version from assembly instead of hardcoding
        private static string CurrentVersion => System.Reflection.Assembly.GetExecutingAssembly()
            .GetName().Version?.ToString(3) ?? "0.0.0";
        
        public static UpdateService Instance => _instance ??= new UpdateService();

        private UpdateService()
        {
            _httpClient = new HttpClient();
            _httpClient.DefaultRequestHeaders.Add("User-Agent", "DeeMusic-Updater");
        }

        /// <summary>
        /// Check if a new version is available on GitHub
        /// </summary>
        public async Task<UpdateInfo?> CheckForUpdatesAsync()
        {
            try
            {
                LoggingService.Instance.LogInfo("Checking for updates...");
                
                var response = await _httpClient.GetStringAsync(GitHubApiUrl);
                var release = JsonSerializer.Deserialize<GitHubRelease>(response);
                
                if (release == null)
                {
                    LoggingService.Instance.LogWarning("Failed to parse GitHub release info");
                    return null;
                }

                var latestVersion = release.TagName?.TrimStart('v') ?? "";
                var currentVersion = CurrentVersion;

                LoggingService.Instance.LogInfo($"Current version: {currentVersion}, Latest version: {latestVersion}");

                if (IsNewerVersion(latestVersion, currentVersion))
                {
                    // Find the portable ZIP asset
                    var asset = FindPortableAsset(release);
                    if (asset != null)
                    {
                        return new UpdateInfo
                        {
                            Version = latestVersion,
                            ReleaseNotes = release.Body ?? "No release notes available",
                            DownloadUrl = asset.BrowserDownloadUrl ?? "",
                            FileName = asset.Name ?? "update.zip",
                            FileSize = asset.Size
                        };
                    }
                }

                LoggingService.Instance.LogInfo("No updates available");
                return null;
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to check for updates", ex);
                return null;
            }
        }

        /// <summary>
        /// Download the update package
        /// </summary>
        public async Task<string?> DownloadUpdateAsync(UpdateInfo updateInfo, IProgress<int>? progress = null)
        {
            try
            {
                LoggingService.Instance.LogInfo($"Downloading update: {updateInfo.FileName}");
                
                var tempPath = Path.Combine(Path.GetTempPath(), "DeeMusic_Update");
                Directory.CreateDirectory(tempPath);
                
                var downloadPath = Path.Combine(tempPath, updateInfo.FileName);

                using var response = await _httpClient.GetAsync(updateInfo.DownloadUrl, HttpCompletionOption.ResponseHeadersRead);
                response.EnsureSuccessStatusCode();

                var totalBytes = response.Content.Headers.ContentLength ?? 0;
                var downloadedBytes = 0L;

                using var contentStream = await response.Content.ReadAsStreamAsync();
                using var fileStream = new FileStream(downloadPath, FileMode.Create, FileAccess.Write, FileShare.None, 8192, true);

                var buffer = new byte[8192];
                int bytesRead;

                while ((bytesRead = await contentStream.ReadAsync(buffer, 0, buffer.Length)) > 0)
                {
                    await fileStream.WriteAsync(buffer, 0, bytesRead);
                    downloadedBytes += bytesRead;

                    if (totalBytes > 0)
                    {
                        var progressPercentage = (int)((downloadedBytes * 100) / totalBytes);
                        progress?.Report(progressPercentage);
                    }
                }

                LoggingService.Instance.LogInfo($"Update downloaded to: {downloadPath}");
                return downloadPath;
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to download update", ex);
                return null;
            }
        }

        /// <summary>
        /// Apply the downloaded update
        /// </summary>
        public bool ApplyUpdate(string updatePackagePath)
        {
            try
            {
                LoggingService.Instance.LogInfo($"Applying update from: {updatePackagePath}");

                // Check if we need admin privileges
                var appPath = AppDomain.CurrentDomain.BaseDirectory;
                var needsAdmin = !HasWriteAccess(appPath);
                
                if (needsAdmin)
                {
                    LoggingService.Instance.LogInfo("Update requires administrator privileges");
                    
                    var result = MessageBox.Show(
                        "This update requires administrator privileges to modify files in Program Files.\n\n" +
                        "Click OK to restart with administrator privileges, or Cancel to skip the update.",
                        "Administrator Required",
                        MessageBoxButton.OKCancel,
                        MessageBoxImage.Warning);
                    
                    if (result != MessageBoxResult.OK)
                    {
                        LoggingService.Instance.LogInfo("User declined admin elevation");
                        return false;
                    }
                }

                // Create update script
                var scriptPath = CreateUpdateScript(updatePackagePath);
                
                if (scriptPath == null)
                {
                    LoggingService.Instance.LogError("Failed to create update script");
                    return false;
                }

                // Launch update script
                var psi = new ProcessStartInfo
                {
                    FileName = "powershell.exe",
                    Arguments = $"-ExecutionPolicy Bypass -WindowStyle Normal -File \"{scriptPath}\"",
                    UseShellExecute = true
                };

                // Request admin elevation if needed
                if (needsAdmin)
                {
                    psi.Verb = "runas";
                }

                try
                {
                    Process.Start(psi);
                }
                catch (System.ComponentModel.Win32Exception ex)
                {
                    // User cancelled UAC prompt
                    if (ex.NativeErrorCode == 1223)
                    {
                        LoggingService.Instance.LogWarning("User cancelled update (UAC prompt)");
                        MessageBox.Show(
                            "Update cancelled. Administrator privileges are required to update the application.",
                            "Update Cancelled",
                            MessageBoxButton.OK,
                            MessageBoxImage.Information);
                        return false;
                    }
                    throw;
                }
                
                LoggingService.Instance.LogInfo("Update script launched, application will restart");
                
                // Exit the application
                Application.Current.Dispatcher.Invoke(() =>
                {
                    Application.Current.Shutdown();
                });

                return true;
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to apply update", ex);
                MessageBox.Show(
                    $"Failed to apply update: {ex.Message}\n\nPlease try downloading the update manually from GitHub.",
                    "Update Error",
                    MessageBoxButton.OK,
                    MessageBoxImage.Error);
                return false;
            }
        }

        /// <summary>
        /// Check if we have write access to a directory
        /// </summary>
        private bool HasWriteAccess(string directoryPath)
        {
            try
            {
                var testFile = Path.Combine(directoryPath, $"_write_test_{Guid.NewGuid()}.tmp");
                File.WriteAllText(testFile, "test");
                File.Delete(testFile);
                return true;
            }
            catch
            {
                return false;
            }
        }

        /// <summary>
        /// Create PowerShell script to apply the update
        /// </summary>
        private string? CreateUpdateScript(string updatePackagePath)
        {
            try
            {
                var appPath = AppDomain.CurrentDomain.BaseDirectory;
                var scriptPath = Path.Combine(Path.GetTempPath(), "DeeMusic_Update", "update.ps1");
                var logPath = Path.Combine(Path.GetTempPath(), "DeeMusic_Update", "update.log");

                var script = $@"
# DeeMusic Update Script
$ErrorActionPreference = 'Stop'
$logFile = '{logPath}'

function Write-Log {{
    param($Message)
    $timestamp = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'
    ""[$timestamp] $Message"" | Out-File -FilePath $logFile -Append
    Write-Host $Message
}}

try {{
    Write-Log 'DeeMusic Update Script Started'
    Write-Log 'Waiting for application to close...'
    Start-Sleep -Seconds 2

    $appPath = '{appPath}'
    $updateZip = '{updatePackagePath}'
    $backupPath = '{Path.Combine(Path.GetTempPath(), "DeeMusic_Backup")}'
    $tempExtract = '{Path.Combine(Path.GetTempPath(), "DeeMusic_Extract")}'

    Write-Log ""App Path: $appPath""
    Write-Log ""Update ZIP: $updateZip""

    # Test write access to app directory
    $testFile = Join-Path $appPath ""_update_test.tmp""
    try {{
        [System.IO.File]::WriteAllText($testFile, ""test"")
        Remove-Item $testFile -Force
        Write-Log 'Write access confirmed'
    }} catch {{
        Write-Log 'ERROR: No write access to application directory!'
        Write-Log 'This update requires administrator privileges.'
        Write-Log 'Please run the application as administrator and try again.'
        [System.Windows.MessageBox]::Show(""Update failed: No write access to application directory.`n`nPlease run DeeMusic as administrator and try the update again."", ""Update Error"", ""OK"", ""Error"")
        exit 1
    }}

    Write-Log 'Creating backup...'
    if (Test-Path $backupPath) {{
        Remove-Item -Path $backupPath -Recurse -Force
    }}
    New-Item -ItemType Directory -Path $backupPath -Force | Out-Null
    Copy-Item -Path ""$appPath\*"" -Destination $backupPath -Recurse -Force
    Write-Log 'Backup created'

    Write-Log 'Extracting update...'
    if (Test-Path $tempExtract) {{
        Remove-Item -Path $tempExtract -Recurse -Force
    }}
    Expand-Archive -Path $updateZip -DestinationPath $tempExtract -Force
    Write-Log 'Update extracted'

    # Find the actual content folder (portable ZIP may have a subfolder)
    $contentPath = $tempExtract
    $possibleSubfolder = Get-ChildItem -Path $tempExtract -Directory | Where-Object {{ $_.Name -like ""DeeMusic*"" }} | Select-Object -First 1
    if ($possibleSubfolder) {{
        $contentPath = $possibleSubfolder.FullName
        Write-Log ""Found content in subfolder: $($possibleSubfolder.Name)""
    }}

    # Verify the update contains the executable
    $exePath = Join-Path $contentPath ""DeeMusic.Desktop.exe""
    if (-not (Test-Path $exePath)) {{
        Write-Log 'ERROR: Update package does not contain DeeMusic.Desktop.exe'
        throw ""Invalid update package""
    }}
    Write-Log 'Update package validated'

    Write-Log 'Copying files to application directory...'
    # Copy all files from content path to app path
    Get-ChildItem -Path $contentPath -Recurse | ForEach-Object {{
        $targetPath = $_.FullName.Replace($contentPath, $appPath)
        if ($_.PSIsContainer) {{
            if (-not (Test-Path $targetPath)) {{
                New-Item -ItemType Directory -Path $targetPath -Force | Out-Null
            }}
        }} else {{
            Copy-Item -Path $_.FullName -Destination $targetPath -Force
            Write-Log ""Copied: $($_.Name)""
        }}
    }}
    Write-Log 'Files copied successfully'

    # Clean up
    Write-Log 'Cleaning up temporary files...'
    Remove-Item -Path $tempExtract -Recurse -Force
    Remove-Item -Path $updateZip -Force
    Write-Log 'Cleanup complete'

    # Restart application
    Write-Log 'Restarting application...'
    Start-Sleep -Seconds 1
    Start-Process -FilePath ""$appPath\DeeMusic.Desktop.exe""
    Write-Log 'Application restarted'

    Write-Log 'Update completed successfully!'

    # Clean up backup after 5 seconds
    Start-Sleep -Seconds 5
    if (Test-Path $backupPath) {{
        Remove-Item -Path $backupPath -Recurse -Force
        Write-Log 'Backup cleaned up'
    }}

}} catch {{
    Write-Log ""ERROR: Update failed - $($_.Exception.Message)""
    Write-Log 'Restoring backup...'
    
    try {{
        if (Test-Path $backupPath) {{
            Copy-Item -Path ""$backupPath\*"" -Destination $appPath -Recurse -Force
            Write-Log 'Backup restored successfully'
        }}
    }} catch {{
        Write-Log ""ERROR: Failed to restore backup - $($_.Exception.Message)""
    }}
    
    Write-Log 'Starting application...'
    Start-Process -FilePath ""$appPath\DeeMusic.Desktop.exe""
    
    [System.Windows.MessageBox]::Show(""Update failed: $($_.Exception.Message)`n`nYour previous version has been restored.`n`nCheck the log file at: $logFile"", ""Update Error"", ""OK"", ""Error"")
    exit 1
}}
";

                Directory.CreateDirectory(Path.GetDirectoryName(scriptPath)!);
                File.WriteAllText(scriptPath, script);
                
                return scriptPath;
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to create update script", ex);
                return null;
            }
        }

        /// <summary>
        /// Compare version strings
        /// </summary>
        private bool IsNewerVersion(string latestVersion, string currentVersion)
        {
            try
            {
                var latest = Version.Parse(latestVersion);
                var current = Version.Parse(currentVersion);
                return latest > current;
            }
            catch
            {
                return false;
            }
        }

        /// <summary>
        /// Find the portable ZIP asset in the release
        /// </summary>
        private GitHubAsset? FindPortableAsset(GitHubRelease release)
        {
            if (release.Assets == null) return null;

            foreach (var asset in release.Assets)
            {
                var name = asset.Name?.ToLower() ?? "";
                if (name.Contains("portable") && name.EndsWith(".zip"))
                {
                    return asset;
                }
            }

            return null;
        }
    }

    #region GitHub API Models

    public class UpdateInfo
    {
        public string Version { get; set; } = "";
        public string ReleaseNotes { get; set; } = "";
        public string DownloadUrl { get; set; } = "";
        public string FileName { get; set; } = "";
        public long FileSize { get; set; }
    }

    public class GitHubRelease
    {
        [JsonPropertyName("tag_name")]
        public string? TagName { get; set; }

        [JsonPropertyName("name")]
        public string? Name { get; set; }

        [JsonPropertyName("body")]
        public string? Body { get; set; }

        [JsonPropertyName("published_at")]
        public DateTime PublishedAt { get; set; }

        [JsonPropertyName("assets")]
        public GitHubAsset[]? Assets { get; set; }
    }

    public class GitHubAsset
    {
        [JsonPropertyName("name")]
        public string? Name { get; set; }

        [JsonPropertyName("browser_download_url")]
        public string? BrowserDownloadUrl { get; set; }

        [JsonPropertyName("size")]
        public long Size { get; set; }
    }

    #endregion
}
