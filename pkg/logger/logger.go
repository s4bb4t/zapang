// Package logger provides a production-ready structured logging solution built on Uber's Zap.
// It supports environment-aware configuration, context propagation with trace IDs,
// log sampling, and graceful shutdown.
package logger

import (
	"context"
	"io"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ctxKey struct{}

var (
	globalLogger *zap.Logger
	globalMu     sync.RWMutex
)

// Logger wraps zap.Logger with additional functionality.
type Logger struct {
	*zap.Logger
	level zap.AtomicLevel
}

// New creates a new Logger based on the provided configuration.
// The serviceName is added as a permanent field to all log entries.
// If w is provided, logs will also be written to it (useful for testing or custom outputs).
func New(ctx context.Context, serviceName string, cfg Config, w io.Writer) *Logger {
	level := parseLevel(cfg.Level)
	atomicLevel := zap.NewAtomicLevelAt(level)

	encoderConfig := newEncoderConfig(cfg.Environment)

	var cores []zapcore.Core

	// Build cores for each output path
	for _, path := range cfg.OutputPaths {
		core := buildCore(path, atomicLevel, encoderConfig, cfg.Environment)
		if core != nil {
			cores = append(cores, core)
		}
	}

	// Add custom writer if provided
	if w != nil {
		encoder := newEncoder(cfg.Environment, encoderConfig)
		core := zapcore.NewCore(encoder, zapcore.AddSync(w), atomicLevel)
		cores = append(cores, core)
	}

	// Fallback to stdout if no cores configured
	if len(cores) == 0 {
		encoder := newEncoder(cfg.Environment, encoderConfig)
		core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), atomicLevel)
		cores = append(cores, core)
	}

	combinedCore := zapcore.NewTee(cores...)

	// Apply sampling if configured
	if cfg.Sampling != nil && cfg.Sampling.Initial > 0 {
		combinedCore = zapcore.NewSamplerWithOptions(
			combinedCore,
			time.Second,
			cfg.Sampling.Initial,
			cfg.Sampling.Thereafter,
		)
	}

	// Build options
	opts := buildOptions(cfg, serviceName)

	logger := zap.New(combinedCore, opts...)

	l := &Logger{
		Logger: logger,
		level:  atomicLevel,
	}

	// Set as global logger
	globalMu.Lock()
	globalLogger = logger
	globalMu.Unlock()

	// Register shutdown on context cancellation
	go func() {
		<-ctx.Done()
		_ = l.Sync()
	}()

	return l
}

// FromContext retrieves the logger from context, or returns the global logger.
func FromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		return l
	}
	return Global()
}

// WithContext returns a new context with the logger attached.
func WithContext(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// Global returns the global logger instance.
func Global() *zap.Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if globalLogger == nil {
		return zap.NewNop()
	}
	return globalLogger
}

// SetLevel dynamically changes the logging level.
func (l *Logger) SetLevel(level string) {
	l.level.SetLevel(parseLevel(level))
}

// GetLevel returns the current logging level.
func (l *Logger) GetLevel() zapcore.Level {
	return l.level.Level()
}

// WithTraceID returns a new logger with trace and span IDs attached.
func (l *Logger) WithTraceID(traceID, spanID string) *zap.Logger {
	return l.With(
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
	)
}

// WithFields returns a new logger with the given fields attached.
func (l *Logger) WithFields(fields ...zap.Field) *zap.Logger {
	return l.With(fields...)
}

// WithError returns a new logger with the error attached.
func (l *Logger) WithError(err error) *zap.Logger {
	return l.With(zap.Error(err))
}

// Sugar returns a sugared logger for printf-style logging.
func (l *Logger) Sugar() *zap.SugaredLogger {
	return l.Logger.Sugar()
}

func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func newEncoderConfig(env string) zapcore.EncoderConfig {
	if env == "development" {
		return zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}
	}

	// Production encoder config optimized for log aggregation (ELK, Splunk, DataDog, etc.)
	return zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    "function",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

func newEncoder(env string, cfg zapcore.EncoderConfig) zapcore.Encoder {
	if env == "development" {
		return zapcore.NewConsoleEncoder(cfg)
	}
	return zapcore.NewJSONEncoder(cfg)
}

func buildCore(path string, level zap.AtomicLevel, cfg zapcore.EncoderConfig, env string) zapcore.Core {
	var ws zapcore.WriteSyncer

	switch path {
	case "stdout":
		ws = zapcore.AddSync(os.Stdout)
	case "stderr":
		ws = zapcore.AddSync(os.Stderr)
	default:
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil
		}
		ws = zapcore.AddSync(file)
	}

	encoder := newEncoder(env, cfg)
	return zapcore.NewCore(encoder, ws, level)
}

func buildOptions(cfg Config, serviceName string) []zap.Option {
	opts := []zap.Option{
		zap.Fields(zap.String("service", serviceName)),
	}

	if !cfg.DisableCaller {
		opts = append(opts, zap.AddCaller())
	}

	if !cfg.DisableStacktrace {
		stackLevel := parseLevel(cfg.StacktraceLevel)
		if cfg.StacktraceLevel == "" {
			stackLevel = zapcore.ErrorLevel
		}
		opts = append(opts, zap.AddStacktrace(stackLevel))
	}

	return opts
}
