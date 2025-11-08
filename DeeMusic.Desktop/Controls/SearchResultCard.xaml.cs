using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Media;
using System.Windows.Media.Imaging;

namespace DeeMusic.Desktop.Controls
{
    /// <summary>
    /// Search result card control for displaying tracks, albums, artists, and playlists
    /// </summary>
    public partial class SearchResultCard : UserControl
    {
        public static readonly DependencyProperty TitleProperty =
            DependencyProperty.Register("Title", typeof(string), typeof(SearchResultCard), 
                new PropertyMetadata(string.Empty, OnTitleChanged));

        public static readonly DependencyProperty ArtistProperty =
            DependencyProperty.Register("Artist", typeof(string), typeof(SearchResultCard), 
                new PropertyMetadata(string.Empty, OnArtistChanged));

        public static readonly DependencyProperty AlbumProperty =
            DependencyProperty.Register("Album", typeof(string), typeof(SearchResultCard), 
                new PropertyMetadata(string.Empty, OnAlbumChanged));

        public static readonly DependencyProperty DurationProperty =
            DependencyProperty.Register("Duration", typeof(string), typeof(SearchResultCard), 
                new PropertyMetadata(string.Empty, OnDurationChanged));

        public static readonly DependencyProperty YearProperty =
            DependencyProperty.Register("Year", typeof(string), typeof(SearchResultCard), 
                new PropertyMetadata(string.Empty, OnYearChanged));

        public static readonly DependencyProperty ArtworkUrlProperty =
            DependencyProperty.Register("ArtworkUrl", typeof(string), typeof(SearchResultCard), 
                new PropertyMetadata(string.Empty, OnArtworkUrlChanged));

        public static readonly DependencyProperty ResultTypeProperty =
            DependencyProperty.Register("ResultType", typeof(SearchResultType), typeof(SearchResultCard), 
                new PropertyMetadata(SearchResultType.Track, OnResultTypeChanged));

        public static readonly DependencyProperty DownloadCommandProperty =
            DependencyProperty.Register("DownloadCommand", typeof(ICommand), typeof(SearchResultCard), 
                new PropertyMetadata(null));

        public static readonly DependencyProperty InfoCommandProperty =
            DependencyProperty.Register("InfoCommand", typeof(ICommand), typeof(SearchResultCard), 
                new PropertyMetadata(null));

        public static readonly DependencyProperty ItemClickCommandProperty =
            DependencyProperty.Register("ItemClickCommand", typeof(ICommand), typeof(SearchResultCard), 
                new PropertyMetadata(null));

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

        public string Album
        {
            get => (string)GetValue(AlbumProperty);
            set => SetValue(AlbumProperty, value);
        }

        public string Duration
        {
            get => (string)GetValue(DurationProperty);
            set => SetValue(DurationProperty, value);
        }

        public string Year
        {
            get => (string)GetValue(YearProperty);
            set => SetValue(YearProperty, value);
        }

        public string ArtworkUrl
        {
            get => (string)GetValue(ArtworkUrlProperty);
            set => SetValue(ArtworkUrlProperty, value);
        }

        public SearchResultType ResultType
        {
            get => (SearchResultType)GetValue(ResultTypeProperty);
            set => SetValue(ResultTypeProperty, value);
        }

        public ICommand DownloadCommand
        {
            get => (ICommand)GetValue(DownloadCommandProperty);
            set => SetValue(DownloadCommandProperty, value);
        }

        public ICommand InfoCommand
        {
            get => (ICommand)GetValue(InfoCommandProperty);
            set => SetValue(InfoCommandProperty, value);
        }

        public ICommand ItemClickCommand
        {
            get => (ICommand)GetValue(ItemClickCommandProperty);
            set => SetValue(ItemClickCommandProperty, value);
        }

        public SearchResultCard()
        {
            InitializeComponent();
            
            // Wire up button events
            downloadButton.Click += (s, e) =>
            {
                e.Handled = true;
                DownloadCommand?.Execute(null);
            };
            
            infoButton.Click += (s, e) =>
            {
                e.Handled = true;
                InfoCommand?.Execute(null);
            };
        }

        private static void OnTitleChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
        {
            if (d is SearchResultCard card)
            {
                card.titleText.Text = e.NewValue?.ToString() ?? string.Empty;
            }
        }

        private static void OnArtistChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
        {
            if (d is SearchResultCard card)
            {
                card.artistText.Text = e.NewValue?.ToString() ?? string.Empty;
            }
        }

        private static void OnAlbumChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
        {
            if (d is SearchResultCard card)
            {
                var album = e.NewValue?.ToString() ?? string.Empty;
                card.albumText.Text = album;
                card.albumText.Visibility = string.IsNullOrEmpty(album) ? Visibility.Collapsed : Visibility.Visible;
            }
        }

        private static void OnDurationChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
        {
            if (d is SearchResultCard card)
            {
                var duration = e.NewValue?.ToString() ?? string.Empty;
                card.durationText.Text = duration;
                card.durationText.Visibility = string.IsNullOrEmpty(duration) ? Visibility.Collapsed : Visibility.Visible;
            }
        }

        private static void OnYearChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
        {
            if (d is SearchResultCard card)
            {
                var year = e.NewValue?.ToString() ?? string.Empty;
                card.yearText.Text = year;
                card.yearText.Visibility = string.IsNullOrEmpty(year) ? Visibility.Collapsed : Visibility.Visible;
            }
        }

        private static void OnArtworkUrlChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
        {
            if (d is SearchResultCard card)
            {
                var url = e.NewValue?.ToString();
                if (!string.IsNullOrEmpty(url))
                {
                    try
                    {
                        var bitmap = new BitmapImage();
                        bitmap.BeginInit();
                        bitmap.UriSource = new Uri(url, UriKind.Absolute);
                        bitmap.CacheOption = BitmapCacheOption.OnLoad;
                        bitmap.EndInit();
                        
                        card.artworkImage.Source = bitmap;
                        card.artworkImage.Visibility = Visibility.Visible;
                        card.artworkPlaceholder.Visibility = Visibility.Collapsed;
                    }
                    catch
                    {
                        // If image fails to load, keep placeholder visible
                        card.artworkImage.Visibility = Visibility.Collapsed;
                        card.artworkPlaceholder.Visibility = Visibility.Visible;
                    }
                }
                else
                {
                    card.artworkImage.Visibility = Visibility.Collapsed;
                    card.artworkPlaceholder.Visibility = Visibility.Visible;
                }
            }
        }

        private static void OnResultTypeChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
        {
            if (d is SearchResultCard card && e.NewValue is SearchResultType type)
            {
                card.UpdateForResultType(type);
            }
        }

        private void UpdateForResultType(SearchResultType type)
        {
            switch (type)
            {
                case SearchResultType.Track:
                    downloadButton.Content = "Download";
                    infoButton.Visibility = Visibility.Collapsed;
                    durationText.Visibility = Visibility.Visible;
                    break;

                case SearchResultType.Album:
                    downloadButton.Content = "Download Album";
                    infoButton.Visibility = Visibility.Visible;
                    infoButton.Content = "View";
                    artistText.Visibility = Visibility.Visible;
                    albumText.Visibility = Visibility.Collapsed;
                    break;

                case SearchResultType.Artist:
                    downloadButton.Visibility = Visibility.Collapsed;
                    infoButton.Visibility = Visibility.Visible;
                    infoButton.Content = "View Artist";
                    artistText.Visibility = Visibility.Collapsed;
                    durationText.Visibility = Visibility.Collapsed;
                    break;

                case SearchResultType.Playlist:
                    downloadButton.Content = "Download Playlist";
                    infoButton.Visibility = Visibility.Visible;
                    infoButton.Content = "View";
                    artistText.Visibility = Visibility.Visible;
                    albumText.Visibility = Visibility.Collapsed;
                    durationText.Visibility = Visibility.Collapsed;
                    break;
            }
        }

        private void Card_MouseLeftButtonDown(object sender, MouseButtonEventArgs e)
        {
            // Only trigger if not clicking on buttons
            if (e.OriginalSource is Button || 
                (e.OriginalSource is FrameworkElement element && element.TemplatedParent is Button))
            {
                return;
            }

            ItemClickCommand?.Execute(null);
        }
    }

    public enum SearchResultType
    {
        Track,
        Album,
        Artist,
        Playlist
    }
}
