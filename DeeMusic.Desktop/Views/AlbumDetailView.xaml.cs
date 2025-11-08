using System;
using System.Globalization;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Input;
using System.Windows.Media;

namespace DeeMusic.Desktop.Views
{
    /// <summary>
    /// Interaction logic for AlbumDetailView.xaml
    /// </summary>
    public partial class AlbumDetailView : UserControl
    {
        public AlbumDetailView()
        {
            InitializeComponent();
        }
        
        private void ListView_PreviewMouseWheel(object sender, MouseWheelEventArgs e)
        {
            // Forward the mouse wheel event to the parent ScrollViewer
            if (sender is ListView listView)
            {
                var scrollViewer = FindParent<ScrollViewer>(listView);
                if (scrollViewer != null)
                {
                    scrollViewer.ScrollToVerticalOffset(scrollViewer.VerticalOffset - e.Delta / 3.0);
                    e.Handled = true;
                }
            }
        }
        
        private static T? FindParent<T>(DependencyObject child) where T : DependencyObject
        {
            var parent = VisualTreeHelper.GetParent(child);
            
            if (parent == null)
                return null;
                
            if (parent is T typedParent)
                return typedParent;
                
            return FindParent<T>(parent);
        }
    }
    
    /// <summary>
    /// Converter to increment AlternationIndex by 1 for track numbers
    /// </summary>
    public class AlternationIndexConverter : IValueConverter
    {
        public object Convert(object value, Type targetType, object parameter, CultureInfo culture)
        {
            if (value is int index)
                return index + 1;
            return 1;
        }

        public object ConvertBack(object value, Type targetType, object parameter, CultureInfo culture)
        {
            throw new NotImplementedException();
        }
    }
}
