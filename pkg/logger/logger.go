// Package logger provides a production-ready structured logging solution built on Uber's Zap.
// It supports environment-aware configuration, context propagation with trace IDs,
// log sampling, and graceful shutdown.
package logger

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	EnvLocal = "local"
	EnvProd  = "prod"
	EnvDev   = "dev"
)

type ctxKey struct{}

var (
	globalLogger *zap.Logger
	globalLevel  zap.AtomicLevel
	globalMu     sync.RWMutex
	projectRoot  string
)

func init() {
	// Determine project root at init time by finding the directory containing go.mod
	_, file, _, ok := runtime.Caller(0)
	if ok {
		dir := filepath.Dir(file)
		for dir != "/" && dir != "." {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				projectRoot = dir
				break
			}
			dir = filepath.Dir(dir)
		}
	}
}

// rootRelativeCallerEncoder encodes caller path relative to project root for clickable terminal links.
func rootRelativeCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	if !caller.Defined {
		enc.AppendString("undefined")
		return
	}

	path := caller.File
	if projectRoot != "" && strings.HasPrefix(path, projectRoot) {
		path = "." + strings.TrimPrefix(path, projectRoot)
	}

	enc.AppendString(path + ":" + strconv.Itoa(caller.Line))
}

// New creates a new *zap.Logger based on the provided configuration.
// The serviceName is added as a permanent field to all log entries.
// If w is provided, logs will also be written to it (useful for testing).
//
// Output behavior:
//   - All environments: Human-readable console output to stdout
//   - Dev/Prod with ExportPath: Additional JSON output for log aggregation
func New(ctx context.Context, serviceName string, cfg Config, w io.Writer) *zap.Logger {
	logger, level := NewWithLevel(ctx, serviceName, cfg, w)

	// Set as global logger
	globalMu.Lock()
	globalLogger = logger
	globalLevel = level
	globalMu.Unlock()

	return logger
}

// NewWithLevel creates a new *zap.Logger and returns its AtomicLevel for dynamic level control.
// Use this when you need to change the log level at runtime.
func NewWithLevel(ctx context.Context, serviceName string, cfg Config, w io.Writer) (*zap.Logger, zap.AtomicLevel) {
	level := parseLevel(cfg.Level)
	atomicLevel := zap.NewAtomicLevelAt(level)

	var cores []zapcore.Core

	// Always add human-readable console output to stdout
	consoleCore := buildConsoleCore(atomicLevel)
	cores = append(cores, consoleCore)

	// Add JSON export core for dev/prod if ExportPath is configured
	if cfg.ExportPath != "" && (cfg.Environment == EnvDev || cfg.Environment == EnvProd) {
		if exportCore := buildJSONExportCore(cfg.ExportPath, atomicLevel); exportCore != nil {
			cores = append(cores, exportCore)
		}
	}

	// Add custom writer if provided (useful for testing)
	if w != nil {
		encoder := zapcore.NewConsoleEncoder(consoleEncoderConfig())
		core := zapcore.NewCore(encoder, zapcore.AddSync(w), atomicLevel)
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

	// Register shutdown on context cancellation
	go func() {
		<-ctx.Done()
		_ = logger.Sync()
	}()

	return logger, atomicLevel
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

// GlobalLevel returns the global logger's AtomicLevel for dynamic level control.
func GlobalLevel() zap.AtomicLevel {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalLevel
}

// SetGlobalLevel dynamically changes the global logger's level.
func SetGlobalLevel(level string) {
	globalMu.RLock()
	defer globalMu.RUnlock()
	globalLevel.SetLevel(parseLevel(level))
}

// WithTraceID returns a new logger with trace and span IDs attached.
func WithTraceID(l *zap.Logger, traceID, spanID string) *zap.Logger {
	return l.With(
		zap.String("trace_id", traceID),
		zap.String("span_id", spanID),
	)
}

// WithError returns a new logger with the error attached.
func WithError(l *zap.Logger, err error) *zap.Logger {
	return l.With(zap.Error(err))
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

// consoleEncoderConfig returns encoder config for human-readable output.
func consoleEncoderConfig() zapcore.EncoderConfig {
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
		EncodeCaller:   rootRelativeCallerEncoder,
	}
}

// jsonEncoderConfig returns encoder config for JSON export (log aggregation systems).
func jsonEncoderConfig() zapcore.EncoderConfig {
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
		EncodeCaller:   rootRelativeCallerEncoder,
	}
}

// buildConsoleCore creates a human-readable console core that writes to stdout.
func buildConsoleCore(level zap.AtomicLevel) zapcore.Core {
	encoder := zapcore.NewConsoleEncoder(consoleEncoderConfig())
	return zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
}

// buildJSONExportCore creates a JSON core for log export/aggregation.
func buildJSONExportCore(path string, level zap.AtomicLevel) zapcore.Core {
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

	encoder := zapcore.NewJSONEncoder(jsonEncoderConfig())
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
