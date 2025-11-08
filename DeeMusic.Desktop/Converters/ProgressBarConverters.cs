using System;
using System.Globalization;
using System.Windows.Data;
using System.Windows.Media;

namespace DeeMusic.Desktop.Converters
{
    public class ProgressBarColorConverter : IValueConverter
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

        public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
        {
            try
            {
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
    }

    public class ProgressTextColorConverter : IValueConverter
    {
        private static readonly SolidColorBrush DefaultBrush = new SolidColorBrush(Color.FromRgb(0x1e, 0x40, 0xaf));
        private static readonly SolidColorBrush CompletedBrush = new SolidColorBrush(Color.FromRgb(0x05, 0x96, 0x69));

        public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
        {
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
