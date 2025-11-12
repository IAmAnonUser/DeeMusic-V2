using System.Windows.Controls;
using System.Windows.Input;
using System.Windows;
using System.Windows.Media;

namespace DeeMusic.Desktop.Views
{
    /// <summary>
    /// Interaction logic for PlaylistDetailView.xaml
    /// </summary>
    public partial class PlaylistDetailView : UserControl
    {
        public PlaylistDetailView()
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
}
