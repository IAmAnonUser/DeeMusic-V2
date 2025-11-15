using System;
using System.Diagnostics;
using System.Runtime.InteropServices;
using System.Threading;
using System.Threading.Tasks;

namespace DeeMusic.Desktop.Services
{
    /// <summary>
    /// Monitors application performance metrics
    /// </summary>
    public class PerformanceMonitor
    {
        private static PerformanceMonitor? _instance;
        private readonly Process _currentProcess;
        private readonly Stopwatch _startupTimer;
        private bool _startupComplete;
        
        public static PerformanceMonitor Instance => _instance ??= new PerformanceMonitor();

        private PerformanceMonitor()
        {
            _currentProcess = Process.GetCurrentProcess();
            _startupTimer = Stopwatch.StartNew();
        }

        /// <summary>
        /// Mark startup as complete and log startup time
        /// </summary>
        public void MarkStartupComplete()
        {
            if (!_startupComplete)
            {
                _startupComplete = true;
                _startupTimer.Stop();
                var startupTime = _startupTimer.Elapsed.TotalSeconds;
                
                LoggingService.Instance.LogInfo($"Startup completed in {startupTime:F2} seconds");
                
                if (startupTime > 2.0)
                {
                    LoggingService.Instance.LogWarning($"Startup time exceeded target: {startupTime:F2}s > 2.0s");
                }
            }
        }

        /// <summary>
        /// Get current memory usage in MB
        /// </summary>
        public double GetMemoryUsageMB()
        {
            _currentProcess.Refresh();
            return _currentProcess.WorkingSet64 / 1024.0 / 1024.0;
        }

        /// <summary>
        /// Get private memory usage in MB
        /// </summary>
        public double GetPrivateMemoryMB()
        {
            _currentProcess.Refresh();
            return _currentProcess.PrivateMemorySize64 / 1024.0 / 1024.0;
        }

        /// <summary>
        /// Get managed memory usage in MB
        /// </summary>
        public double GetManagedMemoryMB()
        {
            return GC.GetTotalMemory(false) / 1024.0 / 1024.0;
        }

        /// <summary>
        /// Get CPU usage percentage (averaged over 1 second)
        /// </summary>
        public async Task<double> GetCpuUsageAsync()
        {
            _currentProcess.Refresh();
            var startTime = DateTime.UtcNow;
            var startCpuUsage = _currentProcess.TotalProcessorTime;

            await Task.Delay(1000);

            _currentProcess.Refresh();
            var endTime = DateTime.UtcNow;
            var endCpuUsage = _currentProcess.TotalProcessorTime;

            var cpuUsedMs = (endCpuUsage - startCpuUsage).TotalMilliseconds;
            var totalMsPassed = (endTime - startTime).TotalMilliseconds;
            var cpuUsageTotal = cpuUsedMs / (Environment.ProcessorCount * totalMsPassed);

            return cpuUsageTotal * 100;
        }

        /// <summary>
        /// Get thread count
        /// </summary>
        public int GetThreadCount()
        {
            _currentProcess.Refresh();
            return _currentProcess.Threads.Count;
        }

        /// <summary>
        /// Get handle count
        /// </summary>
        public int GetHandleCount()
        {
            _currentProcess.Refresh();
            return _currentProcess.HandleCount;
        }

        /// <summary>
        /// Log current performance metrics
        /// </summary>
        public void LogCurrentMetrics(string context = "")
        {
            var memory = GetMemoryUsageMB();
            var privateMemory = GetPrivateMemoryMB();
            var managedMemory = GetManagedMemoryMB();
            var threads = GetThreadCount();
            var handles = GetHandleCount();

            var message = $"Performance Metrics{(string.IsNullOrEmpty(context) ? "" : $" ({context})")}: " +
                         $"Memory={memory:F2}MB, Private={privateMemory:F2}MB, Managed={managedMemory:F2}MB, " +
                         $"Threads={threads}, Handles={handles}";

            LoggingService.Instance.LogInfo(message);

            // Check against targets
            if (memory > 350)
            {
                LoggingService.Instance.LogWarning($"Memory usage high: {memory:F2}MB > 350MB target");
            }
        }

        /// <summary>
        /// Start continuous monitoring (for testing)
        /// </summary>
        public async Task MonitorContinuouslyAsync(TimeSpan duration, TimeSpan interval, CancellationToken cancellationToken = default)
        {
            var stopwatch = Stopwatch.StartNew();
            
            while (stopwatch.Elapsed < duration && !cancellationToken.IsCancellationRequested)
            {
                LogCurrentMetrics($"T+{stopwatch.Elapsed.TotalSeconds:F1}s");
                await Task.Delay(interval, cancellationToken);
            }
        }

        /// <summary>
        /// Measure UI frame rate (approximate)
        /// </summary>
        public class FrameRateMonitor
        {
            private int _frameCount;
            private readonly Stopwatch _stopwatch = new();

            public void Start()
            {
                _frameCount = 0;
                _stopwatch.Restart();
            }

            public void RecordFrame()
            {
                _frameCount++;
            }

            public double GetAverageFPS()
            {
                var elapsed = _stopwatch.Elapsed.TotalSeconds;
                return elapsed > 0 ? _frameCount / elapsed : 0;
            }
        }
    }
}
