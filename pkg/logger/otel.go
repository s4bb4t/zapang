package logger

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// WithOtelContext extracts trace and span IDs from an OpenTelemetry context
// and returns a logger with those fields attached.
func WithOtelContext(ctx context.Context, log *zap.Logger) *zap.Logger {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return log
	}

	sc := span.SpanContext()
	return log.With(
		TraceID(sc.TraceID().String()),
		SpanID(sc.SpanID().String()),
	)
}

// OtelCore is a zapcore.Core wrapper that automatically adds trace context.
// Use this when you want all logs to automatically include trace IDs.
type OtelCore struct {
	zap.Logger
}

// FromOtelContext retrieves the logger from context and enriches it with
// OpenTelemetry trace information if available.
func FromOtelContext(ctx context.Context) *zap.Logger {
	log := FromContext(ctx)
	return WithOtelContext(ctx, log)
}

// LoggerWithSpan creates a new logger with span information attached.
// This is useful when you want to correlate logs with a specific span.
func LoggerWithSpan(log *zap.Logger, span trace.Span) *zap.Logger {
	if span == nil {
		return log
	}

	sc := span.SpanContext()
	if !sc.IsValid() {
		return log
	}

	return log.With(
		TraceID(sc.TraceID().String()),
		SpanID(sc.SpanID().String()),
	)
}

// TraceEvent logs a trace event as a zap log entry.
// This helps correlate application logs with distributed traces.
func TraceEvent(log *zap.Logger, span trace.Span, msg string, fields ...zap.Field) {
	if span == nil {
		log.Info(msg, fields...)
		return
	}

	sc := span.SpanContext()
	enrichedFields := append([]zap.Field{
		TraceID(sc.TraceID().String()),
		SpanID(sc.SpanID().String()),
	}, fields...)

	log.Info(msg, enrichedFields...)
}
