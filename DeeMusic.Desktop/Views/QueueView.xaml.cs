using System.Windows;
using System.Windows.Controls;
using DeeMusic.Desktop.Models;
using DeeMusic.Desktop.Services;

namespace DeeMusic.Desktop.Views
{
    /// <summary>
    /// Interaction logic for QueueView.xaml
    /// </summary>
    public partial class QueueView : UserControl
    {
        public QueueView()
        {
            InitializeComponent();
        }

        private async void ViewFailedTracksButton_Click(object sender, RoutedEventArgs e)
        {
            if (sender is Button button && button.Tag is QueueItem queueItem)
            {
                try
                {
                    // Get the ViewModel
                    if (DataContext is not ViewModels.QueueViewModel viewModel)
                    {
                        LoggingService.Instance.LogError("QueueViewModel not found in DataContext");
                        return;
                    }

                    // Get failed tracks through the ViewModel
                    var failedTracks = await viewModel.GetFailedTracksAsync(queueItem.Id);

                    if (failedTracks != null && failedTracks.Count > 0)
                    {
                        var dialog = new FailedTracksDialog(queueItem.Title, failedTracks);
                        dialog.Owner = Window.GetWindow(this);
                        dialog.ShowDialog();
                    }
                    else
                    {
                        MessageBox.Show("No failed track details available.", "Information", 
                            MessageBoxButton.OK, MessageBoxImage.Information);
                    }
                }
                catch (System.Exception ex)
                {
                    LoggingService.Instance.LogError($"Failed to load failed tracks: {ex.Message}");
                    MessageBox.Show($"Failed to load error details: {ex.Message}", "Error", 
                        MessageBoxButton.OK, MessageBoxImage.Error);
                }
            }
        }
    }
}
