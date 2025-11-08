using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Media;
using System.Windows.Media.Animation;

namespace DeeMusic.Desktop.Views
{
    /// <summary>
    /// Interaction logic for SearchView.xaml
    /// </summary>
    public partial class SearchView : UserControl
    {
        public SearchView()
        {
            InitializeComponent();
        }

        /// <summary>
        /// Handle mouse wheel scrolling for the entire UserControl
        /// </summary>
        private void UserControl_PreviewMouseWheel(object sender, MouseWheelEventArgs e)
        {
            if (!e.Handled)
            {
                // Direct scroll without animation for better performance
                // Calculate scroll amount (120 units per notch, scroll ~84 pixels per notch)
                var scrollAmount = e.Delta * 0.7;
                var newOffset = MainScrollViewer.VerticalOffset - scrollAmount;
                
                // Clamp to valid range
                newOffset = Math.Max(0, Math.Min(newOffset, MainScrollViewer.ScrollableHeight));
                
                // Scroll directly
                MainScrollViewer.ScrollToVerticalOffset(newOffset);
                
                e.Handled = true;
            }
        }

        /// <summary>
        /// Handle mouse wheel scrolling for horizontal ScrollViewers
        /// Pass the event to parent for vertical scrolling
        /// </summary>
        private void ScrollViewer_PreviewMouseWheel(object sender, MouseWheelEventArgs e)
        {
            // This is now handled by UserControl_PreviewMouseWheel
            // Just let it bubble up
        }

        /// <summary>
        /// Scroll left in horizontal ScrollViewer
        /// </summary>
        private void ScrollLeft_Click(object sender, RoutedEventArgs e)
        {
            if (sender is Button button && button.Tag is string scrollerName)
            {
                var scroller = FindName(scrollerName) as ScrollViewer;
                if (scroller != null)
                {
                    scroller.ScrollToHorizontalOffset(scroller.HorizontalOffset - 400);
                }
            }
        }

        /// <summary>
        /// Scroll right in horizontal ScrollViewer
        /// </summary>
        private void ScrollRight_Click(object sender, RoutedEventArgs e)
        {
            if (sender is Button button && button.Tag is string scrollerName)
            {
                var scroller = FindName(scrollerName) as ScrollViewer;
                if (scroller != null)
                {
                    scroller.ScrollToHorizontalOffset(scroller.HorizontalOffset + 400);
                }
            }
        }
    }
}
