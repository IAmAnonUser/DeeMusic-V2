using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media;
using System.Windows.Media.Imaging;
using DeeMusic.Desktop.Services;

namespace DeeMusic.Desktop.Controls
{
    /// <summary>
    /// Image control that automatically caches images
    /// </summary>
    public class CachedImage : Image
    {
        public static readonly DependencyProperty ImageUrlProperty =
            DependencyProperty.Register(
                nameof(ImageUrl),
                typeof(string),
                typeof(CachedImage),
                new PropertyMetadata(null, OnImageUrlChanged));

        public string? ImageUrl
        {
            get => (string?)GetValue(ImageUrlProperty);
            set => SetValue(ImageUrlProperty, value);
        }

        private static async void OnImageUrlChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
        {
            if (d is not CachedImage control)
                return;

            var url = e.NewValue as string;
            if (string.IsNullOrWhiteSpace(url))
            {
                control.Source = null;
                return;
            }

            try
            {
                var image = await ImageCacheService.Instance.GetImageAsync(url);
                
                // Check if URL hasn't changed while we were loading
                if (control.ImageUrl == url)
                {
                    control.Source = image;
                }
            }
            catch (Exception ex)
            {
                LoggingService.Instance.LogError($"Failed to load cached image: {url}", ex);
                control.Source = null;
            }
        }
    }
}
