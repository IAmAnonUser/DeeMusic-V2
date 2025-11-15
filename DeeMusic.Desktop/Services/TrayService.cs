using System;
using System.Windows;
using System.Windows.Forms;
using DeeMusic.Desktop.ViewModels;
using Application = System.Windows.Application;
using DrawingFont = System.Drawing.Font;
using DrawingFontStyle = System.Drawing.FontStyle;
using DrawingColor = System.Drawing.Color;
using DrawingBitmap = System.Drawing.Bitmap;
using DrawingGraphics = System.Drawing.Graphics;
using DrawingSolidBrush = System.Drawing.SolidBrush;
using DrawingIcon = System.Drawing.Icon;
using DrawingFontFamily = System.Drawing.FontFamily;

namespace DeeMusic.Desktop.Services
{
    /// <summary>
    /// Service for managing system tray integration
    /// </summary>
    public class TrayService : IDisposable
    {
        private NotifyIcon? _notifyIcon;
        private readonly MainViewModel _mainViewModel;
        private bool _disposed;

        public TrayService(MainViewModel mainViewModel)
        {
            _mainViewModel = mainViewModel ?? throw new ArgumentNullException(nameof(mainViewModel));
        }

        /// <summary>
        /// Initializes the system tray icon and context menu
        /// </summary>
        public void Initialize()
        {
            if (_notifyIcon != null)
                return;

            _notifyIcon = new NotifyIcon
            {
                Icon = CreateTrayIcon(),
                Text = "DeeMusic",
                Visible = true
            };

            _notifyIcon.DoubleClick += OnTrayIconDoubleClick;
            _notifyIcon.ContextMenuStrip = CreateContextMenu();
        }

        /// <summary>
        /// Shows a notification balloon
        /// </summary>
        public void ShowNotification(string title, string message, ToolTipIcon icon = ToolTipIcon.Info)
        {
            if (_notifyIcon == null)
                return;

            _notifyIcon.ShowBalloonTip(3000, title, message, icon);
        }

        /// <summary>
        /// Shows a download completion notification
        /// </summary>
        public void ShowDownloadCompleted(string trackName)
        {
            ShowNotification("Download Complete", $"{trackName} has been downloaded successfully.", ToolTipIcon.Info);
        }

        /// <summary>
        /// Shows a download error notification
        /// </summary>
        public void ShowDownloadError(string trackName, string error)
        {
            ShowNotification("Download Failed", $"{trackName}: {error}", ToolTipIcon.Error);
        }

        /// <summary>
        /// Shows the main window and brings it to front
        /// </summary>
        public void ShowMainWindow()
        {
            var mainWindow = Application.Current.MainWindow;
            if (mainWindow != null)
            {
                mainWindow.Show();
                mainWindow.WindowState = WindowState.Normal;
                mainWindow.Activate();
            }
        }

        /// <summary>
        /// Hides the main window to tray
        /// </summary>
        public void HideToTray()
        {
            var mainWindow = Application.Current.MainWindow;
            if (mainWindow != null)
            {
                mainWindow.Hide();
            }
        }

        private DrawingIcon CreateTrayIcon()
        {
            // Create a simple icon with "DM" text
            // In production, you would load an actual .ico file
            var bitmap = new DrawingBitmap(16, 16);
            using (var graphics = DrawingGraphics.FromImage(bitmap))
            {
                graphics.Clear(DrawingColor.FromArgb(99, 102, 241)); // Indigo color
                using (var font = new DrawingFont(new DrawingFontFamily("Segoe UI"), 7, DrawingFontStyle.Bold))
                using (var brush = new DrawingSolidBrush(DrawingColor.White))
                {
                    graphics.DrawString("DM", font, brush, -1, 1);
                }
            }

            return DrawingIcon.FromHandle(bitmap.GetHicon());
        }

        private ContextMenuStrip CreateContextMenu()
        {
            var contextMenu = new ContextMenuStrip();

            // Open
            var openItem = new ToolStripMenuItem("Open", null, (s, e) => ShowMainWindow());
            openItem.Font = new DrawingFont(openItem.Font, DrawingFontStyle.Bold);
            contextMenu.Items.Add(openItem);

            contextMenu.Items.Add(new ToolStripSeparator());

            // Show Queue
            var queueItem = new ToolStripMenuItem("Show Queue", null, (s, e) =>
            {
                ShowMainWindow();
                _mainViewModel.NavigateToQueue();
            });
            contextMenu.Items.Add(queueItem);

            contextMenu.Items.Add(new ToolStripSeparator());

            // Pause All
            var pauseAllItem = new ToolStripMenuItem("Pause All", null, (s, e) =>
            {
                if (_mainViewModel.QueueViewModel.PauseAllCommand.CanExecute(null))
                {
                    _mainViewModel.QueueViewModel.PauseAllCommand.Execute(null);
                }
            });
            contextMenu.Items.Add(pauseAllItem);

            // Resume All
            var resumeAllItem = new ToolStripMenuItem("Resume All", null, (s, e) =>
            {
                if (_mainViewModel.QueueViewModel.ResumeAllCommand.CanExecute(null))
                {
                    _mainViewModel.QueueViewModel.ResumeAllCommand.Execute(null);
                }
            });
            contextMenu.Items.Add(resumeAllItem);

            contextMenu.Items.Add(new ToolStripSeparator());

            // Settings
            var settingsItem = new ToolStripMenuItem("Settings", null, (s, e) =>
            {
                ShowMainWindow();
                _mainViewModel.NavigateToSettings();
            });
            contextMenu.Items.Add(settingsItem);

            contextMenu.Items.Add(new ToolStripSeparator());

            // Exit
            var exitItem = new ToolStripMenuItem("Exit", null, (s, e) =>
            {
                Application.Current.Shutdown();
            });
            contextMenu.Items.Add(exitItem);

            return contextMenu;
        }

        private void OnTrayIconDoubleClick(object? sender, EventArgs e)
        {
            ShowMainWindow();
        }

        public void Dispose()
        {
            if (_disposed)
                return;

            if (_notifyIcon != null)
            {
                _notifyIcon.Visible = false;
                _notifyIcon.Dispose();
                _notifyIcon = null;
            }

            _disposed = true;
        }
    }
}
