using System.Windows.Controls;

namespace DeeMusic.Desktop.Views
{
    /// <summary>
    /// Interaction logic for QueueView.xaml
    /// </summary>
    public partial class QueueView : UserControl
    {
        public QueueView()
        {
            Services.LoggingService.Instance.LogInfo("========================================");
            Services.LoggingService.Instance.LogInfo("QueueView constructor called!");
            Services.LoggingService.Instance.LogInfo("BUILD TIMESTAMP: 2025-11-15 05:15:00");
            Services.LoggingService.Instance.LogInfo("XAML SHOULD INCLUDE: STATIC TEST TEXT");
            Services.LoggingService.Instance.LogInfo("========================================");
            
            InitializeComponent();
            
            Services.LoggingService.Instance.LogInfo("QueueView InitializeComponent completed successfully");
        }
    }
}
