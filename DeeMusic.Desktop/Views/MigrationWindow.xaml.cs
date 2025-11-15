using System;
using System.Windows;
using DeeMusic.Desktop.Services;

namespace DeeMusic.Desktop.Views
{
    /// <summary>
    /// Interaction logic for MigrationWindow.xaml
    /// </summary>
    public partial class MigrationWindow : Window
    {
        private readonly MigrationService _migrationService;
        private bool _migrationCompleted = false;

        public MigrationWindow()
        {
            InitializeComponent();
            _migrationService = new MigrationService();
            _migrationService.ProgressUpdated += OnMigrationProgress;

            // Load migration info
            LoadMigrationInfo();
        }

        private async void LoadMigrationInfo()
        {
            try
            {
                // Detect Python installation
                var installation = await _migrationService.DetectPythonInstallationAsync();
                if (installation == null || installation.HasError)
                {
                    ShowError("Could not detect Python installation.");
                    return;
                }

                // Update UI with installation info
                LocationText.Text = installation.data_dir ?? "Unknown";
                SettingsText.Text = installation.has_settings ? "Found" : "Not found";

                // Get migration stats
                var stats = await _migrationService.GetMigrationStatsAsync();
                if (stats != null && !stats.HasError)
                {
                    QueueItemsText.Text = stats.queue_items.ToString();
                    HistoryItemsText.Text = stats.history_items.ToString();
                }
                else
                {
                    QueueItemsText.Text = "0";
                    HistoryItemsText.Text = "0";
                }
            }
            catch (Exception ex)
            {
                ShowError($"Error loading migration info: {ex.Message}");
            }
        }

        private async void MigrateButton_Click(object sender, RoutedEventArgs e)
        {
            try
            {
                // Show progress view
                DetectionView.Visibility = Visibility.Collapsed;
                ProgressView.Visibility = Visibility.Visible;
                MigrateButton.Visibility = Visibility.Collapsed;
                SkipButton.Visibility = Visibility.Collapsed;

                // Perform migration
                var result = await _migrationService.PerformMigrationAsync();

                if (result.Success)
                {
                    _migrationCompleted = true;
                    ShowSuccess(result);
                }
                else
                {
                    ShowError(result.Error ?? "Migration failed for unknown reason.");
                }
            }
            catch (Exception ex)
            {
                ShowError($"Migration error: {ex.Message}");
            }
        }

        private void SkipButton_Click(object sender, RoutedEventArgs e)
        {
            var result = MessageBox.Show(
                "Are you sure you want to skip migration? You can always migrate later from the settings.",
                "Skip Migration",
                MessageBoxButton.YesNo,
                MessageBoxImage.Question);

            if (result == MessageBoxResult.Yes)
            {
                DialogResult = false;
                Close();
            }
        }

        private void CloseButton_Click(object sender, RoutedEventArgs e)
        {
            DialogResult = _migrationCompleted;
            Close();
        }

        private void OnMigrationProgress(object? sender, MigrationProgressEventArgs e)
        {
            ProgressMessage.Text = e.Message;
            ProgressBar.Value = e.Progress;
            ProgressPercent.Text = $"{e.Progress}%";
        }

        private void ShowSuccess(MigrationResult result)
        {
            ProgressView.Visibility = Visibility.Collapsed;
            SuccessView.Visibility = Visibility.Visible;
            CloseButton.Visibility = Visibility.Visible;

            // Build success message
            var parts = new System.Collections.Generic.List<string>();
            if (result.settings_migrated)
                parts.Add("settings");
            if (result.queue_migrated)
                parts.Add("queue and history");

            if (parts.Count > 0)
            {
                SuccessMessage.Text = $"Your {string.Join(" and ", parts)} have been successfully migrated.";
            }

            if (!string.IsNullOrEmpty(result.backup_path))
            {
                BackupPathText.Text = $"Backup created at:\n{result.backup_path}";
            }
        }

        private void ShowError(string error)
        {
            DetectionView.Visibility = Visibility.Collapsed;
            ProgressView.Visibility = Visibility.Collapsed;
            SuccessView.Visibility = Visibility.Collapsed;
            ErrorView.Visibility = Visibility.Visible;

            MigrateButton.Visibility = Visibility.Collapsed;
            SkipButton.Visibility = Visibility.Collapsed;
            CloseButton.Visibility = Visibility.Visible;

            ErrorMessage.Text = error;
        }
    }
}
