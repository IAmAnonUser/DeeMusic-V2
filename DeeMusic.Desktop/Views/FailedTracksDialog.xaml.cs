using System.Collections.Generic;
using System.Windows;
using DeeMusic.Desktop.Models;

namespace DeeMusic.Desktop.Views
{
    public partial class FailedTracksDialog : Window
    {
        public FailedTracksDialog(string albumTitle, List<FailedTrack> failedTracks)
        {
            InitializeComponent();

            int failedCount = failedTracks?.Count ?? 0;
            SummaryText.Text = $"{failedCount} track{(failedCount != 1 ? "s" : "")} failed to download from \"{albumTitle}\"";

            FailedTracksList.ItemsSource = failedTracks;
        }

        private void CloseButton_Click(object sender, RoutedEventArgs e)
        {
            Close();
        }
    }
}
