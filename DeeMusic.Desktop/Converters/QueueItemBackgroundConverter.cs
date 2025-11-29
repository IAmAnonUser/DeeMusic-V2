using System;
using System.Globalization;
using System.Windows.Data;
using System.Windows.Media;

namespace DeeMusic.Desktop.Converters
{
    /// <summary>
    /// Converts IsPartialSuccess boolean to background brush.
    /// </summary>
    public class PartialSuccessBackgroundConverter : IValueConverter
    {
        private static readonly SolidColorBrush PartialSuccessBrush;
        private static readonly SolidColorBrush TransparentBrush;

        static PartialSuccessBackgroundConverter()
        {
            PartialSuccessBrush = new SolidColorBrush((Color)ColorConverter.ConvertFromString("#FEF3C7")!);
            TransparentBrush = Brushes.Transparent;
            PartialSuccessBrush.Freeze();
        }

        public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
        {
            if (value is bool isPartial && isPartial)
                return PartialSuccessBrush;
            return TransparentBrush;
        }

        public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
        {
            throw new NotImplementedException();
        }
    }

    /// <summary>
    /// Converts status string to background brush for completed/failed states.
    /// </summary>
    public class StatusBackgroundConverter : IValueConverter
    {
        private static readonly SolidColorBrush FailedBrush;
        private static readonly SolidColorBrush CompletedBrush;
        private static readonly SolidColorBrush TransparentBrush;

        static StatusBackgroundConverter()
        {
            FailedBrush = new SolidColorBrush((Color)ColorConverter.ConvertFromString("#FEE2E2")!);
            CompletedBrush = new SolidColorBrush((Color)ColorConverter.ConvertFromString("#E8F5E9")!);
            TransparentBrush = Brushes.Transparent;
            FailedBrush.Freeze();
            CompletedBrush.Freeze();
        }

        public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
        {
            if (value is string status)
            {
                return status switch
                {
                    "failed" => FailedBrush,
                    "completed" => CompletedBrush,
                    _ => TransparentBrush
                };
            }
            return TransparentBrush;
        }

        public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
        {
            throw new NotImplementedException();
        }
    }

    /// <summary>
    /// Multi-value converter for queue item background.
    /// Takes IsPartialSuccess, IsFailed, IsCompleted to determine the correct background.
    /// </summary>
    public class QueueItemBackgroundConverter : IMultiValueConverter
    {
        private static readonly SolidColorBrush PartialSuccessBrush;
        private static readonly SolidColorBrush FailedBrush;
        private static readonly SolidColorBrush CompletedBrush;
        private static readonly SolidColorBrush TransparentBrush;

        static QueueItemBackgroundConverter()
        {
            PartialSuccessBrush = new SolidColorBrush((Color)ColorConverter.ConvertFromString("#FEF3C7")!);
            FailedBrush = new SolidColorBrush((Color)ColorConverter.ConvertFromString("#FEE2E2")!);
            CompletedBrush = new SolidColorBrush((Color)ColorConverter.ConvertFromString("#E8F5E9")!);
            TransparentBrush = Brushes.Transparent;
            
            PartialSuccessBrush.Freeze();
            FailedBrush.Freeze();
            CompletedBrush.Freeze();
        }

        public object Convert(object[] values, Type targetType, object parameter, CultureInfo culture)
        {
            bool isPartialSuccess = values.Length > 0 && values[0] is bool ps && ps;
            bool isFailed = values.Length > 1 && values[1] is bool f && f;
            bool isCompleted = values.Length > 2 && values[2] is bool c && c;
            
            if (isPartialSuccess) return PartialSuccessBrush;
            if (isFailed) return FailedBrush;
            if (isCompleted) return CompletedBrush;
            return TransparentBrush;
        }

        public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
        {
            throw new NotImplementedException();
        }
    }

    /// <summary>
    /// Converts a color string like "#FEF3C7" to a SolidColorBrush.
    /// </summary>
    public class StringToBrushConverter : IValueConverter
    {
        public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
        {
            if (value is string colorString && !string.IsNullOrEmpty(colorString))
            {
                if (colorString == "Transparent")
                    return Brushes.Transparent;
                
                try
                {
                    var color = (Color)ColorConverter.ConvertFromString(colorString);
                    var brush = new SolidColorBrush(color);
                    brush.Freeze();
                    return brush;
                }
                catch { }
            }
            return Brushes.Transparent;
        }

        public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
        {
            throw new NotImplementedException();
        }
    }
}
