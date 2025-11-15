using System;
using System.Globalization;
using System.Windows.Data;
using System.Windows.Media.Imaging;
using DeeMusic.Desktop.Services;

namespace DeeMusic.Desktop.Converters
{
    /// <summary>
    /// Converter that loads images through the cache service
    /// </summary>
    public class CachedImageConverter : IValueConverter
    {
        public object? Convert(object value, Type targetType, object parameter, CultureInfo culture)
        {
            if (value is not string url || string.IsNullOrWhiteSpace(url))
                return null;

            // Start async load and return null initially
            // The image will be loaded asynchronously
            var task = ImageCacheService.Instance.GetImageAsync(url);
            
            // If already completed (cached), return immediately
            if (task.IsCompleted)
                return task.Result;

            // Otherwise, trigger async load and return placeholder
            _ = task.ContinueWith(t =>
            {
                // The binding will update when the image is loaded
                // This is handled by the CachedImage control
            });

            return null;
        }

        public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
        {
            throw new NotImplementedException();
        }
    }
}
