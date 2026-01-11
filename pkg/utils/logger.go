package utils

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level      string
	Format     string // "json" or "console"
	OutputPath string
	ErrorPath  string
}

// NewLogger creates a new zap logger based on configuration
func NewLogger(cfg LoggerConfig) (*zap.Logger, error) {
	// Parse log level
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", cfg.Level, err)
	}

	// Configure encoder
	var encoderConfig zapcore.EncoderConfig
	if cfg.Format == "console" {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		encoderConfig = zap.NewProductionEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderConfig.MessageKey = "message"
		encoderConfig.LevelKey = "level"
		encoderConfig.TimeKey = "timestamp"
		encoderConfig.CallerKey = "caller"
		encoderConfig.StacktraceKey = "stacktrace"
	}

	// Create encoder
	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Configure outputs
	var outputWriteSyncer zapcore.WriteSyncer
	if cfg.OutputPath == "stdout" || cfg.OutputPath == "" {
		outputWriteSyncer = zapcore.AddSync(os.Stdout)
	} else {
		file, err := os.OpenFile(cfg.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open output file %s: %w", cfg.OutputPath, err)
		}
		outputWriteSyncer = zapcore.AddSync(file)
	}

	var errorWriteSyncer zapcore.WriteSyncer
	if cfg.ErrorPath == "stderr" || cfg.ErrorPath == "" {
		errorWriteSyncer = zapcore.AddSync(os.Stderr)
	} else {
		file, err := os.OpenFile(cfg.ErrorPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open error file %s: %w", cfg.ErrorPath, err)
		}
		errorWriteSyncer = zapcore.AddSync(file)
	}

	// Create core
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, outputWriteSyncer, level),
		zapcore.NewCore(encoder, errorWriteSyncer, zapcore.ErrorLevel),
	)

	// Create logger with caller and stacktrace
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger, nil
}

// NewDevelopmentLogger creates a logger suitable for development
func NewDevelopmentLogger() (*zap.Logger, error) {
	return NewLogger(LoggerConfig{
		Level:      "debug",
		Format:     "console",
		OutputPath: "stdout",
		ErrorPath:  "stderr",
	})
}

// NewProductionLogger creates a logger suitable for production
func NewProductionLogger() (*zap.Logger, error) {
	return NewLogger(LoggerConfig{
		Level:      "info",
		Format:     "json",
		OutputPath: "stdout",
		ErrorPath:  "stderr",
	})
}

// WithFields adds structured fields to a logger
func WithFields(logger *zap.Logger, fields map[string]interface{}) *zap.Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for key, value := range fields {
		zapFields = append(zapFields, zap.Any(key, value))
	}
	return logger.With(zapFields...)
}

// LoggerFromConfig creates a logger from config struct
func LoggerFromConfig(level, format, outputPath, errorPath string) (*zap.Logger, error) {
	return NewLogger(LoggerConfig{
		Level:      level,
		Format:     format,
		OutputPath: outputPath,
		ErrorPath:  errorPath,
	})
}
