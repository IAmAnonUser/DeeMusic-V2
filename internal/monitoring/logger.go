package monitoring

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogConfig represents logging configuration
type LogConfig struct {
	Level       string `json:"level" mapstructure:"level"`             // debug, info, warn, error
	Format      string `json:"format" mapstructure:"format"`           // json, console
	Output      string `json:"output" mapstructure:"output"`           // file, console, both
	FilePath    string `json:"file_path" mapstructure:"file_path"`     // log file path
	MaxSizeMB   int    `json:"max_size_mb" mapstructure:"max_size_mb"` // max size in MB before rotation
	MaxBackups  int    `json:"max_backups" mapstructure:"max_backups"` // max number of old log files
	MaxAgeDays  int    `json:"max_age_days" mapstructure:"max_age_days"` // max age in days
	Compress    bool   `json:"compress" mapstructure:"compress"`       // compress rotated files
}

// DefaultLogConfig returns default logging configuration
func DefaultLogConfig(dataDir string) *LogConfig {
	return &LogConfig{
		Level:      "info",
		Format:     "json",
		Output:     "file",
		FilePath:   filepath.Join(dataDir, "logs", "app.log"),
		MaxSizeMB:  100,
		MaxBackups: 3,
		MaxAgeDays: 30,
		Compress:   true,
	}
}

// NewLogger creates a new Zap logger with the given configuration
func NewLogger(cfg *LogConfig) (*zap.Logger, error) {
	// Parse log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	// Create encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// Create encoder based on format
	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create writers based on output configuration
	var writers []zapcore.WriteSyncer

	if cfg.Output == "file" || cfg.Output == "both" {
		// Ensure log directory exists
		logDir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		// Create file writer with rotation
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
			Compress:   cfg.Compress,
		}
		writers = append(writers, zapcore.AddSync(fileWriter))
	}

	if cfg.Output == "console" || cfg.Output == "both" {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	// Create core
	core := zapcore.NewCore(
		encoder,
		zapcore.NewMultiWriteSyncer(writers...),
		level,
	)

	// Create logger with caller information
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger, nil
}

// NewDevelopmentLogger creates a logger suitable for development
func NewDevelopmentLogger() (*zap.Logger, error) {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return cfg.Build()
}

// NewProductionLogger creates a logger suitable for production
func NewProductionLogger(dataDir string) (*zap.Logger, error) {
	cfg := DefaultLogConfig(dataDir)
	return NewLogger(cfg)
}

// LoggerWithContext adds context fields to a logger
func LoggerWithContext(logger *zap.Logger, fields ...zap.Field) *zap.Logger {
	return logger.With(fields...)
}
