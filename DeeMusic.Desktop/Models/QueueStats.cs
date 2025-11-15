using System.ComponentModel;
using System.Runtime.CompilerServices;
using System.Text.Json.Serialization;

namespace DeeMusic.Desktop.Models
{
    /// <summary>
    /// Represents queue statistics
    /// </summary>
    public class QueueStats : INotifyPropertyChanged
    {
        private int _total;
        private int _pending;
        private int _downloading;
        private int _completed;
        private int _failed;

        [JsonPropertyName("total")]
        public int Total
        {
            get => _total;
            set
            {
                if (_total != value)
                {
                    _total = value;
                    OnPropertyChanged();
                }
            }
        }

        [JsonPropertyName("pending")]
        public int Pending
        {
            get => _pending;
            set
            {
                if (_pending != value)
                {
                    _pending = value;
                    OnPropertyChanged();
                }
            }
        }

        [JsonPropertyName("downloading")]
        public int Downloading
        {
            get => _downloading;
            set
            {
                if (_downloading != value)
                {
                    _downloading = value;
                    OnPropertyChanged();
                }
            }
        }

        [JsonPropertyName("completed")]
        public int Completed
        {
            get => _completed;
            set
            {
                if (_completed != value)
                {
                    _completed = value;
                    OnPropertyChanged();
                }
            }
        }

        [JsonPropertyName("failed")]
        public int Failed
        {
            get => _failed;
            set
            {
                if (_failed != value)
                {
                    _failed = value;
                    OnPropertyChanged();
                }
            }
        }

        /// <summary>
        /// Gets the number of active downloads (pending + downloading)
        /// </summary>
        public int Active => Pending + Downloading;

        /// <summary>
        /// Gets a summary string of the queue stats
        /// </summary>
        public string Summary => $"{Downloading} downloading, {Pending} pending, {Completed} completed";

        public event PropertyChangedEventHandler? PropertyChanged;

        protected virtual void OnPropertyChanged([CallerMemberName] string? propertyName = null)
        {
            PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
        }
    }
}
