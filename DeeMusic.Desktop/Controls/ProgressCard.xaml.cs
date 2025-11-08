using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Media;
using System.Windows.Media.Animation;

namespace DeeMusic.Desktop.Controls
{
    /// <summary>
    /// Progress card control for displaying download queue items with animated progress
    /// </summary>
    public partial class ProgressCard : UserControl
    {
        public static readonly DependencyProperty TitleProperty =
            DependencyProperty.Register("Title", typeof(string), typeof(ProgressCard), 
                new PropertyMetadata(string.Empty, OnTitleChanged));

        public static readonly DependencyProperty ArtistProperty =
            DependencyProperty.Register("Artist", typeof(string), typeof(ProgressCard), 
                new PropertyMetadata(string.Empty, OnArtistChanged));

        public static readonly DependencyProperty ProgressProperty =
            DependencyProperty.Register("Progress", typeof(double), typeof(ProgressCard), 
                new PropertyMetadata(0.0, OnProgressChanged));

        public static readonly DependencyProperty StatusProperty =
            DependencyProperty.Register("Status", typeof(string), typeof(ProgressCard), 
                new PropertyMetadata("Pending", OnStatusChanged));

        public static readonly DependencyProperty SpeedProperty =
            DependencyProperty.Register("Speed", typeof(string), typeof(ProgressCard), 
                new PropertyMetadata("0 MB/s", OnSpeedChanged));

        public static readonly DependencyProperty ETAProperty =
            DependencyProperty.Register("ETA", typeof(string), typeof(ProgressCard), 
                new PropertyMetadata("--", OnETAChanged));

        public static readonly DependencyProperty PauseCommandProperty =
            DependencyProperty.Register("PauseCommand", typeof(ICommand), typeof(ProgressCard), new PropertyMetadata(null));

        public static readonly DependencyProperty ResumeCommandProperty =
            DependencyProperty.Register("ResumeCommand", typeof(ICommand), typeof(ProgressCard), new PropertyMetadata(null));

        public static readonly DependencyProperty CancelCommandProperty =
            DependencyProperty.Register("CancelCommand", typeof(ICommand), typeof(ProgressCard), new PropertyMetadata(null));

        public static readonly DependencyProperty RetryCommandProperty =
            DependencyProperty.Register("RetryCommand", typeof(ICommand), typeof(ProgressCard), new PropertyMetadata(null));

        public string Title
        {
            get => (string)GetValue(TitleProperty);
            set => SetValue(TitleProperty, value);
        }

        public string Artist
        {
            get => (string)GetValue(ArtistProperty);
            set => SetValue(ArtistProperty, value);
        }

        public double Progress
        {
            get => (double)GetValue(ProgressProperty);
            set => SetValue(ProgressProperty, value);
        }

        public string Status
        {
            get => (string)GetValue(StatusProperty);
            set => SetValue(StatusProperty, value);
        }

        public string Speed
        {
            get => (string)GetValue(SpeedProperty);
            set => SetValue(SpeedProperty, value);
        }

        public string ETA
        {
            get => (string)GetValue(ETAProperty);
            set => SetValue(ETAProperty, value);
        }

        public ICommand PauseCommand
        {
            get => (ICommand)GetValue(PauseCommandProperty);
            set => SetValue(PauseCommandProperty, value);
        }

        public ICommand ResumeCommand
        {
            get => (ICommand)GetValue(ResumeCommandProperty);
            set => SetValue(ResumeCommandProperty, value);
        }

        public ICommand CancelCommand
        {
            get => (ICommand)GetValue(CancelCommandProperty);
            set => SetValue(CancelCommandProperty, value);
        }

        public ICommand RetryCommand
        {
            get => (ICommand)GetValue(RetryCommandProperty);
            set => SetValue(RetryCommandProperty, value);
        }

        public ProgressCard()
        {
            InitializeComponent();
            
            // Wire up button events
            pauseButton.Click += (s, e) => PauseCommand?.Execute(null);
            resumeButton.Click += (s, e) => ResumeCommand?.Execute(null);
            cancelButton.Click += (s, e) => CancelCommand?.Execute(null);
            retryButton.Click += (s, e) => RetryCommand?.Execute(null);
        }

        private static void OnTitleChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
        {
            if (d is ProgressCard card)
            {
                card.titleText.Text = e.NewValue?.ToString() ?? string.Empty;
            }
        }

        private static void OnArtistChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
        {
            if (d is ProgressCard card)
            {
                card.artistText.Text = e.NewValue?.ToString() ?? string.Empty;
            }
        }

        private static void OnProgressChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
        {
            if (d is ProgressCard card && e.NewValue is double progress)
            {
                // Animate progress bar
                var animation = new DoubleAnimation
                {
                    To = progress,
                    Duration = TimeSpan.FromMilliseconds(300),
                    EasingFunction = new CubicEase { EasingMode = EasingMode.EaseOut }
                };
                card.progressBar.BeginAnimation(ProgressBar.ValueProperty, animation);
                
                // Update progress text
                card.progressText.Text = $"{progress:F0}%";
            }
        }

        private static void OnStatusChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
        {
            if (d is ProgressCard card && e.NewValue is string status)
            {
                card.statusText.Text = status;
                card.UpdateStatusIndicator(status);
                card.UpdateControlButtons(status);
            }
        }

        private static void OnSpeedChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
        {
            if (d is ProgressCard card)
            {
                card.speedText.Text = e.NewValue?.ToString() ?? "0 MB/s";
            }
        }

        private static void OnETAChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
        {
            if (d is ProgressCard card)
            {
                card.etaText.Text = $"ETA: {e.NewValue?.ToString() ?? "--"}";
            }
        }

        private void UpdateStatusIndicator(string status)
        {
            var (color, text) = status.ToLower() switch
            {
                "downloading" => ("#10b981", "Downloading"),
                "pending" => ("#f59e0b", "Pending"),
                "paused" => ("#6b7280", "Paused"),
                "completed" => ("#10b981", "Completed"),
                "failed" => ("#ef4444", "Failed"),
                "cancelled" => ("#6b7280", "Cancelled"),
                _ => ("#9ca3af", status)
            };

            statusDot.Fill = new SolidColorBrush((Color)ColorConverter.ConvertFromString(color));
            statusText.Text = text;
        }

        private void UpdateControlButtons(string status)
        {
            switch (status.ToLower())
            {
                case "downloading":
                    pauseButton.Visibility = Visibility.Visible;
                    resumeButton.Visibility = Visibility.Collapsed;
                    retryButton.Visibility = Visibility.Collapsed;
                    cancelButton.Visibility = Visibility.Visible;
                    speedText.Visibility = Visibility.Visible;
                    etaText.Visibility = Visibility.Visible;
                    break;

                case "paused":
                    pauseButton.Visibility = Visibility.Collapsed;
                    resumeButton.Visibility = Visibility.Visible;
                    retryButton.Visibility = Visibility.Collapsed;
                    cancelButton.Visibility = Visibility.Visible;
                    speedText.Visibility = Visibility.Collapsed;
                    etaText.Visibility = Visibility.Collapsed;
                    break;

                case "failed":
                    pauseButton.Visibility = Visibility.Collapsed;
                    resumeButton.Visibility = Visibility.Collapsed;
                    retryButton.Visibility = Visibility.Visible;
                    cancelButton.Visibility = Visibility.Visible;
                    speedText.Visibility = Visibility.Collapsed;
                    etaText.Visibility = Visibility.Collapsed;
                    break;

                case "completed":
                case "cancelled":
                    pauseButton.Visibility = Visibility.Collapsed;
                    resumeButton.Visibility = Visibility.Collapsed;
                    retryButton.Visibility = Visibility.Collapsed;
                    cancelButton.Visibility = Visibility.Collapsed;
                    speedText.Visibility = Visibility.Collapsed;
                    etaText.Visibility = Visibility.Collapsed;
                    break;

                case "pending":
                default:
                    pauseButton.Visibility = Visibility.Collapsed;
                    resumeButton.Visibility = Visibility.Collapsed;
                    retryButton.Visibility = Visibility.Collapsed;
                    cancelButton.Visibility = Visibility.Visible;
                    speedText.Visibility = Visibility.Collapsed;
                    etaText.Visibility = Visibility.Collapsed;
                    break;
            }
        }
    }
}
