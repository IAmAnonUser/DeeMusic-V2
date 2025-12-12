using System;
using System.Collections.Concurrent;
using System.IO;
using System.Net.Http;
using System.Threading.Tasks;
using System.Windows.Media.Imaging;

namespace DeeMusic.Desktop.Services
{
    /// <summary>
    /// Service for caching images to improve performance
    /// </summary>
    public class ImageCacheService
    {
        private static readonly Lazy<ImageCacheService> _instance = new(() => new ImageCacheService());
        public static ImageCacheService Instance => _instance.Value;

        private readonly ConcurrentDictionary<string, BitmapImage> _memoryCache;
        private readonly string _diskCacheDirectory;
        private readonly HttpClient _httpClient;

        private ImageCacheService()
        {
            _memoryCache = new ConcurrentDictionary<string, BitmapImage>();
            _diskCacheDirectory = Path.Combine(
                Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData),
                "DeeMusicV2", "ImageCache");
            
            Directory.CreateDirectory(_diskCacheDirectory);
            _httpClient = new HttpClient();
        }

        /// <summary>
        /// Try to get image from memory cache synchronously (no async overhead)
        /// </summary>
        public BitmapImage? TryGetFromMemoryCache(string? url)
        {
            if (string.IsNullOrWhiteSpace(url))
                return null;

            if (_memoryCache.TryGetValue(url, out var cachedImage))
                return cachedImage;

            return null;
        }

        /// <summary>
        /// Get an image from cache or download it
        /// </summary>
        public async Task<BitmapImage?> GetImageAsync(string? url)
        {
            if (string.IsNullOrWhiteSpace(url))
                return null;

            // Check memory cache first
            if (_memoryCache.TryGetValue(url, out var cachedImage))
                return cachedImage;

            // Check disk cache - load on background thread to avoid blocking UI
            var fileName = GetCacheFileName(url);
            var filePath = Path.Combine(_diskCacheDirectory, fileName);

            if (File.Exists(filePath))
            {
                try
                {
                    // Read file bytes on background thread
                    var imageData = await Task.Run(() => File.ReadAllBytes(filePath));
                    var image = LoadImageFromBytes(imageData);
                    _memoryCache.TryAdd(url, image);
                    return image;
                }
                catch
                {
                    // If file is corrupted, delete it and download again
                    try { File.Delete(filePath); } catch { }
                }
            }

            // Download and cache
            try
            {
                var imageData = await _httpClient.GetByteArrayAsync(url);
                
                // Save to disk cache
                await File.WriteAllBytesAsync(filePath, imageData);
                
                // Load into memory
                var image = LoadImageFromBytes(imageData);
                _memoryCache.TryAdd(url, image);
                
                return image;
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Failed to download image: {url}", ex);
                return null;
            }
        }

        /// <summary>
        /// Load image from file
        /// </summary>
        private BitmapImage LoadImageFromFile(string filePath)
        {
            var image = new BitmapImage();
            image.BeginInit();
            image.CacheOption = BitmapCacheOption.OnLoad;
            image.UriSource = new Uri(filePath, UriKind.Absolute);
            image.EndInit();
            image.Freeze(); // Make it thread-safe
            return image;
        }

        /// <summary>
        /// Load image from bytes
        /// </summary>
        private BitmapImage LoadImageFromBytes(byte[] imageData)
        {
            var image = new BitmapImage();
            image.BeginInit();
            image.CacheOption = BitmapCacheOption.OnLoad;
            image.StreamSource = new MemoryStream(imageData);
            image.EndInit();
            image.Freeze(); // Make it thread-safe
            return image;
        }

        /// <summary>
        /// Get cache file name from URL
        /// </summary>
        private string GetCacheFileName(string url)
        {
            // Use hash of URL as filename to avoid invalid characters
            var hash = url.GetHashCode().ToString("X8");
            var extension = Path.GetExtension(url);
            if (string.IsNullOrEmpty(extension) || extension.Length > 5)
                extension = ".jpg";
            return $"{hash}{extension}";
        }

        /// <summary>
        /// Clear memory cache
        /// </summary>
        public void ClearMemoryCache()
        {
            _memoryCache.Clear();
        }

        /// <summary>
        /// Clear disk cache
        /// </summary>
        public void ClearDiskCache()
        {
            try
            {
                if (Directory.Exists(_diskCacheDirectory))
                {
                    Directory.Delete(_diskCacheDirectory, true);
                    Directory.CreateDirectory(_diskCacheDirectory);
                }
                ClearMemoryCache();
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError("Failed to clear disk cache", ex);
            }
        }

        /// <summary>
        /// Get cache size in bytes
        /// </summary>
        public long GetCacheSize()
        {
            try
            {
                if (!Directory.Exists(_diskCacheDirectory))
                    return 0;

                var files = Directory.GetFiles(_diskCacheDirectory);
                long totalSize = 0;
                foreach (var file in files)
                {
                    totalSize += new FileInfo(file).Length;
                }
                return totalSize;
            }
            catch
            {
                return 0;
            }
        }
    }
}
