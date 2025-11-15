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
        private const string CurrentVersion = "2.0.4"; // Update this with each release
        
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

                // Create update script
                var scriptPath = CreateUpdateScript(updatePackagePath);
                
                if (scriptPath == null)
                {
                    LoggingService.Instance.LogError("Failed to create update script");
                    return false;
                }

                // Launch update script and exit application
                var psi = new ProcessStartInfo
                {
                    FileName = "powershell.exe",
                    Arguments = $"-ExecutionPolicy Bypass -File \"{scriptPath}\"",
                    UseShellExecute = false,
                    CreateNoWindow = true
                };

                Process.Start(psi);
                
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

                var script = $@"
# DeeMusic Update Script
Write-Host 'Waiting for application to close...'
Start-Sleep -Seconds 2

$appPath = '{appPath}'
$updateZip = '{updatePackagePath}'
$backupPath = '{Path.Combine(Path.GetTempPath(), "DeeMusic_Backup")}'

Write-Host 'Creating backup...'
if (Test-Path $backupPath) {{
    Remove-Item -Path $backupPath -Recurse -Force
}}
New-Item -ItemType Directory -Path $backupPath -Force | Out-Null
Copy-Item -Path ""$appPath\*"" -Destination $backupPath -Recurse -Force

Write-Host 'Extracting update...'
try {{
    # Extract to temp location first
    $tempExtract = '{Path.Combine(Path.GetTempPath(), "DeeMusic_Extract")}'
    if (Test-Path $tempExtract) {{
        Remove-Item -Path $tempExtract -Recurse -Force
    }}
    Expand-Archive -Path $updateZip -DestinationPath $tempExtract -Force
    
    # Copy files to application directory
    Copy-Item -Path ""$tempExtract\*"" -Destination $appPath -Recurse -Force
    
    Write-Host 'Update applied successfully!'
    
    # Clean up
    Remove-Item -Path $tempExtract -Recurse -Force
    Remove-Item -Path $updateZip -Force
    
    # Restart application
    Write-Host 'Restarting application...'
    Start-Sleep -Seconds 1
    Start-Process -FilePath ""$appPath\DeeMusic.Desktop.exe""
    
    Write-Host 'Update complete!'
}} catch {{
    Write-Host 'Update failed! Restoring backup...'
    Copy-Item -Path ""$backupPath\*"" -Destination $appPath -Recurse -Force
    Write-Host 'Backup restored. Starting application...'
    Start-Process -FilePath ""$appPath\DeeMusic.Desktop.exe""
}}

# Clean up backup after 5 seconds
Start-Sleep -Seconds 5
if (Test-Path $backupPath) {{
    Remove-Item -Path $backupPath -Recurse -Force
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
