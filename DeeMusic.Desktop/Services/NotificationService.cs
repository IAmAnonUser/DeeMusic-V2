using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media;
using System.Windows.Media.Animation;
using System.Windows.Threading;

namespace DeeMusic.Desktop.Services
{
    /// <summary>
    /// Service for showing modern toast notifications
    /// </summary>
    public class NotificationService
    {
        private static readonly Lazy<NotificationService> _instance = new(() => new NotificationService());
        public static NotificationService Instance => _instance.Value;

        private Grid? _notificationContainer;
        private readonly System.Collections.Generic.Dictionary<Border, DispatcherTimer> _activeNotifications;

        private NotificationService()
        {
            _activeNotifications = new System.Collections.Generic.Dictionary<Border, DispatcherTimer>();
        }

        /// <summary>
        /// Initialize the notification service with the main window's container
        /// </summary>
        public void Initialize(Grid container)
        {
            _notificationContainer = container;
        }

        /// <summary>
        /// Show a success notification
        /// </summary>
        public void ShowSuccess(string message, int durationMs = 3000)
        {
            ShowNotification(message, "#4CAF50", "CheckCircle", durationMs);
        }

        /// <summary>
        /// Show an info notification
        /// </summary>
        public void ShowInfo(string message, int durationMs = 3000)
        {
            ShowNotification(message, "#2196F3", "Information", durationMs);
        }

        /// <summary>
        /// Show a warning notification
        /// </summary>
        public void ShowWarning(string message, int durationMs = 3000)
        {
            ShowNotification(message, "#FF9800", "Alert", durationMs);
        }

        /// <summary>
        /// Show an error notification
        /// </summary>
        public void ShowError(string message, int durationMs = 4000)
        {
            ShowNotification(message, "#F44336", "AlertCircle", durationMs);
        }

        private void ShowNotification(string message, string colorHex, string iconKind, int durationMs)
        {
            if (_notificationContainer == null)
            {
                // Fallback to message box if not initialized
                MessageBox.Show(message);
                return;
            }

            Application.Current.Dispatcher.Invoke(() =>
            {
                // Create notification panel
                var notification = CreateNotificationPanel(message, colorHex, iconKind);

                // Set Grid positioning to span all rows and be on top
                Grid.SetRow(notification, 0);
                Grid.SetRowSpan(notification, 2);
                Grid.SetColumn(notification, 0);
                Grid.SetColumnSpan(notification, 2);
                Panel.SetZIndex(notification, 9999); // Ensure it's on top

                // Add to container
                _notificationContainer.Children.Add(notification);

                // Animate in
                AnimateIn(notification);

                // Schedule removal with a dedicated timer for this notification
                var timer = new DispatcherTimer
                {
                    Interval = TimeSpan.FromMilliseconds(durationMs)
                };
                timer.Tick += (s, e) => Timer_Tick(notification, timer);
                _activeNotifications[notification] = timer;
                timer.Start();
            });
        }

        private Border CreateNotificationPanel(string message, string colorHex, string iconKind)
        {
            var color = (Color)ColorConverter.ConvertFromString(colorHex);

            var border = new Border
            {
                Background = new SolidColorBrush(Color.FromArgb(240, color.R, color.G, color.B)),
                CornerRadius = new CornerRadius(8),
                Padding = new Thickness(16, 12, 16, 12),
                Margin = new Thickness(16, 80, 16, 16), // Top margin to avoid being cut off
                HorizontalAlignment = HorizontalAlignment.Right,
                VerticalAlignment = VerticalAlignment.Top,
                MaxWidth = 400,
                Opacity = 0,
                RenderTransform = new TranslateTransform(0, -20)
            };

            // Add shadow effect
            border.Effect = new System.Windows.Media.Effects.DropShadowEffect
            {
                Color = Colors.Black,
                Opacity = 0.3,
                BlurRadius = 10,
                ShadowDepth = 2
            };

            var stackPanel = new StackPanel
            {
                Orientation = Orientation.Horizontal
            };

            // Icon
            var icon = new MaterialDesignThemes.Wpf.PackIcon
            {
                Kind = (MaterialDesignThemes.Wpf.PackIconKind)Enum.Parse(
                    typeof(MaterialDesignThemes.Wpf.PackIconKind), iconKind),
                Width = 20,
                Height = 20,
                Foreground = Brushes.White,
                VerticalAlignment = VerticalAlignment.Center,
                Margin = new Thickness(0, 0, 12, 0)
            };

            // Message
            var textBlock = new TextBlock
            {
                Text = message,
                Foreground = Brushes.White,
                FontSize = 14,
                TextWrapping = TextWrapping.Wrap,
                VerticalAlignment = VerticalAlignment.Center
            };

            stackPanel.Children.Add(icon);
            stackPanel.Children.Add(textBlock);
            border.Child = stackPanel;

            return border;
        }

        private void AnimateIn(Border notification)
        {
            var fadeIn = new DoubleAnimation
            {
                From = 0,
                To = 1,
                Duration = TimeSpan.FromMilliseconds(300),
                EasingFunction = new CubicEase { EasingMode = EasingMode.EaseOut }
            };

            var slideIn = new DoubleAnimation
            {
                From = -20,
                To = 0,
                Duration = TimeSpan.FromMilliseconds(300),
                EasingFunction = new CubicEase { EasingMode = EasingMode.EaseOut }
            };

            notification.BeginAnimation(UIElement.OpacityProperty, fadeIn);
            ((TranslateTransform)notification.RenderTransform).BeginAnimation(
                TranslateTransform.YProperty, slideIn);
        }

        private void AnimateOut(Border notification, Action onComplete)
        {
            var fadeOut = new DoubleAnimation
            {
                From = 1,
                To = 0,
                Duration = TimeSpan.FromMilliseconds(200),
                EasingFunction = new CubicEase { EasingMode = EasingMode.EaseIn }
            };

            var slideOut = new DoubleAnimation
            {
                From = 0,
                To = -20,
                Duration = TimeSpan.FromMilliseconds(200),
                EasingFunction = new CubicEase { EasingMode = EasingMode.EaseIn }
            };

            fadeOut.Completed += (s, e) => onComplete();

            notification.BeginAnimation(UIElement.OpacityProperty, fadeOut);
            ((TranslateTransform)notification.RenderTransform).BeginAnimation(
                TranslateTransform.YProperty, slideOut);
        }

        private void Timer_Tick(Border notification, DispatcherTimer timer)
        {
            timer.Stop();
            _activeNotifications.Remove(notification);

            if (_notificationContainer != null)
            {
                AnimateOut(notification, () =>
                {
                    Application.Current.Dispatcher.Invoke(() =>
                    {
                        _notificationContainer.Children.Remove(notification);
                    });
                });
            }
        }

        #region Persistent Notification for Progress Updates

        private Border? _persistentNotification;
        private TextBlock? _persistentTextBlock;

        /// <summary>
        /// Show a persistent notification that stays open until dismissed
        /// </summary>
        public void ShowPersistentInfo(string message)
        {
            if (_notificationContainer == null) return;

            Application.Current.Dispatcher.Invoke(() =>
            {
                // If we already have a persistent notification, just update the text
                if (_persistentNotification != null && _persistentTextBlock != null)
                {
                    _persistentTextBlock.Text = message;
                    return;
                }

                // Create new persistent notification
                _persistentNotification = CreateNotificationPanel(message, "#2196F3", "Information");
                
                // Find the TextBlock in the notification to update later
                if (_persistentNotification.Child is StackPanel sp)
                {
                    foreach (var child in sp.Children)
                    {
                        if (child is TextBlock tb)
                        {
                            _persistentTextBlock = tb;
                            break;
                        }
                    }
                }

                // Set Grid positioning
                Grid.SetRow(_persistentNotification, 0);
                Grid.SetRowSpan(_persistentNotification, 2);
                Grid.SetColumn(_persistentNotification, 0);
                Grid.SetColumnSpan(_persistentNotification, 2);
                Panel.SetZIndex(_persistentNotification, 9999);

                // Add to container
                _notificationContainer.Children.Add(_persistentNotification);

                // Animate in
                AnimateIn(_persistentNotification);
            });
        }

        /// <summary>
        /// Update the text of the persistent notification
        /// </summary>
        public void UpdatePersistentInfo(string message)
        {
            if (_persistentTextBlock == null)
            {
                ShowPersistentInfo(message);
                return;
            }

            Application.Current.Dispatcher.Invoke(() =>
            {
                _persistentTextBlock.Text = message;
            });
        }

        /// <summary>
        /// Dismiss the persistent notification
        /// </summary>
        public void DismissPersistentNotification()
        {
            if (_persistentNotification == null || _notificationContainer == null) return;

            Application.Current.Dispatcher.Invoke(() =>
            {
                var notification = _persistentNotification;
                _persistentNotification = null;
                _persistentTextBlock = null;

                AnimateOut(notification, () =>
                {
                    Application.Current.Dispatcher.Invoke(() =>
                    {
                        _notificationContainer.Children.Remove(notification);
                    });
                });
            });
        }

        #endregion
    }
}
