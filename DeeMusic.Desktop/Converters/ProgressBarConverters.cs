using System;
using System.Globalization;
using System.Windows.Data;
using System.Windows.Media;
using DeeMusic.Desktop.Models;

namespace DeeMusic.Desktop.Converters
{
    public class ProgressBarColorConverter : IValueConverter, IMultiValueConverter
    {
        private static readonly LinearGradientBrush DefaultBrush = new LinearGradientBrush
        {
            StartPoint = new System.Windows.Point(0, 0),
            EndPoint = new System.Windows.Point(1, 0),
            GradientStops = new GradientStopCollection
            {
                new GradientStop(Color.FromRgb(0x3b, 0x82, 0xf6), 0),
                new GradientStop(Color.FromRgb(0x25, 0x63, 0xeb), 1)
            }
        };

        private static readonly LinearGradientBrush CompletedBrush = new LinearGradientBrush
        {
            StartPoint = new System.Windows.Point(0, 0),
            EndPoint = new System.Windows.Point(1, 0),
            GradientStops = new GradientStopCollection
            {
                new GradientStop(Color.FromRgb(0x10, 0xb9, 0x81), 0),
                new GradientStop(Color.FromRgb(0x05, 0x96, 0x69), 1)
            }
        };

        private static readonly LinearGradientBrush PartialBrush = new LinearGradientBrush
        {
            StartPoint = new System.Windows.Point(0, 0),
            EndPoint = new System.Windows.Point(1, 0),
            GradientStops = new GradientStopCollection
            {
                new GradientStop(Color.FromRgb(0xf5, 0x9e, 0x0b), 0),
                new GradientStop(Color.FromRgb(0xd9, 0x77, 0x06), 1)
            }
        };

        public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
        {
            try
            {
                // Check if it's a QueueItem object
                if (value is QueueItem item)
                {
                    if (item.IsPartialSuccess)
                    {
                        System.Diagnostics.Debug.WriteLine($"[ProgressBarColorConverter] QueueItem '{item.Title}' IsPartialSuccess=true -> ORANGE");
                        return PartialBrush;
                    }
                    if (item.IsCompleted)
                    {
                        System.Diagnostics.Debug.WriteLine($"[ProgressBarColorConverter] QueueItem '{item.Title}' IsCompleted=true -> GREEN");
                        return CompletedBrush;
                    }
                    System.Diagnostics.Debug.WriteLine($"[ProgressBarColorConverter] QueueItem '{item.Title}' Status={item.Status} -> BLUE");
                    return DefaultBrush;
                }
                
                if (value is string status)
                {
                    var result = status == "completed" ? CompletedBrush : DefaultBrush;
                    System.Diagnostics.Debug.WriteLine($"[ProgressBarColorConverter] Status='{status}' -> {(status == "completed" ? "GREEN" : "BLUE")}");
                    return result;
                }
                
                if (value is bool isCompleted)
                {
                    var result = isCompleted ? CompletedBrush : DefaultBrush;
                    System.Diagnostics.Debug.WriteLine($"[ProgressBarColorConverter] IsCompleted={isCompleted} -> {(isCompleted ? "GREEN" : "BLUE")}");
                    return result;
                }
                
                System.Diagnostics.Debug.WriteLine($"[ProgressBarColorConverter] Unexpected value type: {value?.GetType().Name ?? "null"}, value={value}");
                return DefaultBrush;
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"[ProgressBarColorConverter] ERROR: {ex.Message}");
                return DefaultBrush;
            }
        }

        public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
        {
            throw new NotImplementedException();
        }
        
        // IMultiValueConverter implementation for background colors
        public object Convert(object[] values, Type targetType, object parameter, CultureInfo culture)
        {
            try
            {
                // values[0] = Status (string)
                // values[1] = CompletedTracks (int)
                // values[2] = TotalTracks (int)
                // values[3] = Type (string)
                
                if (values.Length < 4 || values[0] == null || values[1] == null || values[2] == null || values[3] == null)
                    return System.Windows.Media.Brushes.Transparent;
                
                string status = values[0].ToString() ?? "";
                int completedTracks = values[1] is int ct ? ct : 0;
                int totalTracks = values[2] is int tt ? tt : 0;
                string type = values[3].ToString() ?? "";
                
                bool isAlbumOrPlaylist = (type == "album" || type == "playlist") && totalTracks > 0;
                bool isCompleted = status == "completed";
                bool isPartialSuccess = isAlbumOrPlaylist && isCompleted && completedTracks < totalTracks;
                
                System.Diagnostics.Debug.WriteLine($"[MultiValueConverter] Status={status}, Completed={completedTracks}/{totalTracks}, Type={type}, IsPartialSuccess={isPartialSuccess}");
                
                // Return solid colors for backgrounds (not gradients)
                if (isPartialSuccess)
                {
                    System.Diagnostics.Debug.WriteLine($"[MultiValueConverter] Returning ORANGE for partial success");
                    return new SolidColorBrush((Color)ColorConverter.ConvertFromString("#FEF3C7"));
                }
                if (isCompleted)
                {
                    System.Diagnostics.Debug.WriteLine($"[MultiValueConverter] Returning GREEN for completed");
                    return new SolidColorBrush((Color)ColorConverter.ConvertFromString("#E8F5E9"));
                }
                
                return System.Windows.Media.Brushes.Transparent;
            }
            catch (Exception ex)
            {
                System.Diagnostics.Debug.WriteLine($"[MultiValueConverter] ERROR: {ex.Message}");
                return System.Windows.Media.Brushes.Transparent;
            }
        }
        
        public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
        {
            throw new NotImplementedException();
        }
    }

    public class ProgressTextColorConverter : IValueConverter
    {
        private static readonly SolidColorBrush DefaultBrush = new SolidColorBrush(Color.FromRgb(0x1e, 0x40, 0xaf));
        private static readonly SolidColorBrush CompletedBrush = new SolidColorBrush(Color.FromRgb(0x05, 0x96, 0x69));
        private static readonly SolidColorBrush PartialBrush = new SolidColorBrush(Color.FromRgb(0xd9, 0x77, 0x06));

        public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
        {
            // Check if it's a QueueItem object
            if (value is QueueItem item)
            {
                if (item.IsPartialSuccess)
                    return PartialBrush;
                if (item.IsCompleted)
                    return CompletedBrush;
                return DefaultBrush;
            }
            
            if (value is string status)
            {
                return status == "completed" ? CompletedBrush : DefaultBrush;
            }
            
            if (value is bool isCompleted)
            {
                return isCompleted ? CompletedBrush : DefaultBrush;
            }
            
            return DefaultBrush;
        }

        public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
        {
            throw new NotImplementedException();
        }
    }
}
