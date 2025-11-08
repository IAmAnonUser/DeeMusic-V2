using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Media;
using System.Windows.Media.Animation;
using System.Windows.Shapes;

namespace DeeMusic.Desktop.Controls
{
    /// <summary>
    /// Modern button control with ripple animation and multiple style variants
    /// </summary>
    public partial class ModernButton : UserControl
    {
        public static readonly DependencyProperty TextProperty =
            DependencyProperty.Register("Text", typeof(string), typeof(ModernButton), new PropertyMetadata(string.Empty));

        public static readonly DependencyProperty IconProperty =
            DependencyProperty.Register("Icon", typeof(string), typeof(ModernButton), new PropertyMetadata(string.Empty));

        public static readonly DependencyProperty ButtonStyleProperty =
            DependencyProperty.Register("ButtonStyle", typeof(ButtonStyleType), typeof(ModernButton), 
                new PropertyMetadata(ButtonStyleType.Primary, OnButtonStyleChanged));

        public static readonly DependencyProperty CommandProperty =
            DependencyProperty.Register("Command", typeof(ICommand), typeof(ModernButton), new PropertyMetadata(null));

        public static readonly DependencyProperty CommandParameterProperty =
            DependencyProperty.Register("CommandParameter", typeof(object), typeof(ModernButton), new PropertyMetadata(null));

        public string Text
        {
            get => (string)GetValue(TextProperty);
            set => SetValue(TextProperty, value);
        }

        public string Icon
        {
            get => (string)GetValue(IconProperty);
            set => SetValue(IconProperty, value);
        }

        public ButtonStyleType ButtonStyle
        {
            get => (ButtonStyleType)GetValue(ButtonStyleProperty);
            set => SetValue(ButtonStyleProperty, value);
        }

        public ICommand Command
        {
            get => (ICommand)GetValue(CommandProperty);
            set => SetValue(CommandProperty, value);
        }

        public object CommandParameter
        {
            get => GetValue(CommandParameterProperty);
            set => SetValue(CommandParameterProperty, value);
        }

        public ModernButton()
        {
            InitializeComponent();
            button.Click += Button_Click;
            button.PreviewMouseLeftButtonDown += Button_PreviewMouseLeftButtonDown;
            
            // Set initial content
            UpdateContent();
        }

        private static void OnButtonStyleChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
        {
            if (d is ModernButton modernButton)
            {
                modernButton.ApplyButtonStyle();
            }
        }

        private void ApplyButtonStyle()
        {
            var style = ButtonStyle switch
            {
                ButtonStyleType.Primary => (Style)FindResource("PrimaryButtonStyle"),
                ButtonStyleType.Secondary => (Style)FindResource("SecondaryButtonStyle"),
                ButtonStyleType.Icon => (Style)FindResource("IconButtonStyle"),
                _ => (Style)FindResource("PrimaryButtonStyle")
            };

            button.Style = style;
            UpdateContent();
        }

        private void UpdateContent()
        {
            var stackPanel = new StackPanel
            {
                Orientation = Orientation.Horizontal,
                HorizontalAlignment = HorizontalAlignment.Center,
                VerticalAlignment = VerticalAlignment.Center
            };

            // Add icon if provided
            if (!string.IsNullOrEmpty(Icon))
            {
                var iconText = new TextBlock
                {
                    Text = Icon,
                    FontSize = ButtonStyle == ButtonStyleType.Icon ? 18 : 16,
                    VerticalAlignment = VerticalAlignment.Center,
                    Margin = string.IsNullOrEmpty(Text) ? new Thickness(0) : new Thickness(0, 0, 8, 0)
                };
                stackPanel.Children.Add(iconText);
            }

            // Add text if provided
            if (!string.IsNullOrEmpty(Text))
            {
                var textBlock = new TextBlock
                {
                    Text = Text,
                    VerticalAlignment = VerticalAlignment.Center
                };
                stackPanel.Children.Add(textBlock);
            }

            button.Content = stackPanel;
        }

        private void Button_Click(object sender, RoutedEventArgs e)
        {
            // Execute command if available
            if (Command?.CanExecute(CommandParameter) == true)
            {
                Command.Execute(CommandParameter);
            }

            // Raise click event
            RaiseEvent(new RoutedEventArgs(Button.ClickEvent));
        }

        private void Button_PreviewMouseLeftButtonDown(object sender, MouseButtonEventArgs e)
        {
            // Create ripple effect
            CreateRippleEffect(e.GetPosition(button));
        }

        private void CreateRippleEffect(Point position)
        {
            var template = button.Template;
            if (template == null) return;

            var border = template.FindName("border", button) as Border;
            var rippleCanvas = template.FindName("rippleCanvas", button) as Canvas;
            var ripple = template.FindName("ripple", button) as Ellipse;

            if (border == null || rippleCanvas == null || ripple == null) return;

            // Calculate ripple size (should cover the entire button)
            double maxSize = Math.Max(border.ActualWidth, border.ActualHeight) * 2;

            // Position ripple at click point
            Canvas.SetLeft(ripple, position.X);
            Canvas.SetTop(ripple, position.Y);

            // Animate ripple
            var sizeAnimation = new DoubleAnimation
            {
                From = 0,
                To = maxSize,
                Duration = TimeSpan.FromMilliseconds(600),
                EasingFunction = new CubicEase { EasingMode = EasingMode.EaseOut }
            };

            var opacityAnimation = new DoubleAnimation
            {
                From = 0.3,
                To = 0,
                Duration = TimeSpan.FromMilliseconds(600),
                EasingFunction = new CubicEase { EasingMode = EasingMode.EaseOut }
            };

            ripple.BeginAnimation(Ellipse.WidthProperty, sizeAnimation);
            ripple.BeginAnimation(Ellipse.HeightProperty, sizeAnimation);
            ripple.BeginAnimation(Ellipse.OpacityProperty, opacityAnimation);
        }
    }

    public enum ButtonStyleType
    {
        Primary,
        Secondary,
        Icon
    }
}
