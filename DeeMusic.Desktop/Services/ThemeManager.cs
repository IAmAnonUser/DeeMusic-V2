using System;
using System.Windows;
using System.Windows.Media.Animation;
using MaterialDesignThemes.Wpf;

namespace DeeMusic.Desktop.Services
{
    /// <summary>
    /// Manages application theme switching with smooth transitions
    /// </summary>
    public class ThemeManager
    {
        private static ThemeManager? _instance;
        private static readonly object _lock = new();
        private string _currentTheme = "dark";

        private ThemeManager()
        {
        }

        /// <summary>
        /// Gets the singleton instance of ThemeManager
        /// </summary>
        public static ThemeManager Instance
        {
            get
            {
                if (_instance == null)
                {
                    lock (_lock)
                    {
                        _instance ??= new ThemeManager();
                    }
                }
                return _instance;
            }
        }

        /// <summary>
        /// Gets the current theme name
        /// </summary>
        public string CurrentTheme => _currentTheme;

        /// <summary>
        /// Applies the specified theme to the application
        /// </summary>
        /// <param name="theme">Theme name: "dark" or "light"</param>
        /// <param name="animate">Whether to animate the transition</param>
        public void ApplyTheme(string theme, bool animate = true)
        {
            if (string.IsNullOrEmpty(theme))
                theme = "dark";

            theme = theme.ToLower();
            if (theme != "dark" && theme != "light")
                theme = "dark";

            if (_currentTheme == theme)
                return;

            _currentTheme = theme;

            Application.Current.Dispatcher.Invoke(() =>
            {
                try
                {
                    var resources = Application.Current.Resources;
                    var mergedDictionaries = resources.MergedDictionaries;

                    // Find and remove existing theme dictionary
                    ResourceDictionary? oldThemeDict = null;
                    foreach (var dict in mergedDictionaries)
                    {
                        if (dict.Source != null && 
                            (dict.Source.ToString().Contains("DarkTheme.xaml") || 
                             dict.Source.ToString().Contains("LightTheme.xaml")))
                        {
                            oldThemeDict = dict;
                            break;
                        }
                    }

                    // Create new theme dictionary
                    var newThemeDict = new ResourceDictionary
                    {
                        Source = new Uri($"pack://application:,,,/Resources/Styles/{(theme == "dark" ? "DarkTheme" : "LightTheme")}.xaml", UriKind.Absolute)
                    };

                    // Apply theme with optional animation
                    if (animate && oldThemeDict != null)
                    {
                        AnimateThemeTransition(() =>
                        {
                            if (oldThemeDict != null)
                                mergedDictionaries.Remove(oldThemeDict);
                            mergedDictionaries.Insert(0, newThemeDict);
                        });
                    }
                    else
                    {
                        if (oldThemeDict != null)
                            mergedDictionaries.Remove(oldThemeDict);
                        mergedDictionaries.Insert(0, newThemeDict);
                    }

                    // Update Material Design theme
                    var paletteHelper = new PaletteHelper();
                    var materialTheme = paletteHelper.GetTheme();
                    materialTheme.SetBaseTheme(theme == "dark" ? Theme.Dark : Theme.Light);
                    paletteHelper.SetTheme(materialTheme);
                }
                catch (Exception ex)
                {
                    System.Diagnostics.Debug.WriteLine($"Error applying theme: {ex.Message}");
                }
            });
        }

        /// <summary>
        /// Toggles between dark and light themes
        /// </summary>
        /// <param name="animate">Whether to animate the transition</param>
        /// <returns>The new theme name</returns>
        public string ToggleTheme(bool animate = true)
        {
            var newTheme = _currentTheme == "dark" ? "light" : "dark";
            ApplyTheme(newTheme, animate);
            return newTheme;
        }

        /// <summary>
        /// Animates the theme transition with a fade effect
        /// </summary>
        private void AnimateThemeTransition(Action applyThemeAction)
        {
            var mainWindow = Application.Current.MainWindow;
            if (mainWindow == null)
            {
                applyThemeAction();
                return;
            }

            // Create fade out animation
            var fadeOut = new DoubleAnimation
            {
                From = 1.0,
                To = 0.95,
                Duration = TimeSpan.FromMilliseconds(150),
                EasingFunction = new QuadraticEase { EasingMode = EasingMode.EaseOut }
            };

            // Create fade in animation
            var fadeIn = new DoubleAnimation
            {
                From = 0.95,
                To = 1.0,
                Duration = TimeSpan.FromMilliseconds(150),
                EasingFunction = new QuadraticEase { EasingMode = EasingMode.EaseIn }
            };

            fadeOut.Completed += (s, e) =>
            {
                // Apply theme at the midpoint of the animation
                applyThemeAction();

                // Fade back in
                mainWindow.BeginAnimation(UIElement.OpacityProperty, fadeIn);
            };

            // Start fade out
            mainWindow.BeginAnimation(UIElement.OpacityProperty, fadeOut);
        }

        /// <summary>
        /// Initializes the theme from settings
        /// </summary>
        /// <param name="theme">Theme name from settings</param>
        public void Initialize(string theme)
        {
            ApplyTheme(theme, animate: false);
        }
    }
}
