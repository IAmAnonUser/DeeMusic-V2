using System.Diagnostics;
using System.Windows.Controls;
using System.Windows.Navigation;

namespace DeeMusic.Desktop.Views
{
    /// <summary>
    /// Interaction logic for SettingsView.xaml
    /// </summary>
    public partial class SettingsView : UserControl
    {
        public SettingsView()
        {
            InitializeComponent();
        }

        private void Hyperlink_RequestNavigate(object sender, RequestNavigateEventArgs e)
        {
            Process.Start(new ProcessStartInfo(e.Uri.AbsoluteUri) { UseShellExecute = true });
            e.Handled = true;
        }
        
        private void ArlTextBox_TextChanged(object sender, TextChangedEventArgs e)
        {
            if (sender is TextBox textBox)
            {
                Debug.WriteLine($"ARL TextBox changed: '{textBox.Text}'");
                Debug.WriteLine($"Text length: {textBox.Text.Length}");
            }
        }
    }
}
