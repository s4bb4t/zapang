# zapang

Structured logging for Go services, built on [zap](https://github.com/uber-go/zap).

Human-readable console output with colored error traces + clean JSON export for log aggregation — out of the box.

## Install

```bash
go get github.com/s4bb4t/zapang
```

## Quick start

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

log := zapang.New(ctx, "my-service", zapang.Config{
    Level:       "info",
    Environment: "local",
}, nil)

log.Info("started", zap.String("addr", ":8080"))
```

## Console output

```
19 Mar 16:33:07  INFO  ./main.go:25  started  addr=:8080  service=my-service
```

Fields rendered as `key=value` pairs, not JSON. Timestamps in `02 Jan 15:04:05` format.

Errors from [go-faster/errors](https://github.com/go-faster/errors) (or any `fmt.Formatter` implementation) are rendered as a colored multi-line block:

```
19 Mar 16:33:07  ERROR  ./main.go:30  failed  error=handle request: parse: invalid input  service=my-service
handle request:                              ← bold red
    main.handleRequest
        ./handler.go:45                      ← dim
  - parse:                                   ← bold red
    main.parse
        ./parser.go:12                       ← dim
  - invalid input:                           ← bold red
    main.validate
        ./validator.go:8                     ← dim
```

## JSON export

For log aggregation (ClickHouse, Loki, ELK, etc.) — parallel JSON output without `errorVerbose`:

```go
// File
log := zapang.New(ctx, "svc", zapang.Config{
    Level:       "info",
    Environment: "prod",
    ExportPath:  "/var/log/app/svc.jsonl",
}, nil)

// Any io.Writer (works in any environment)
log = zapang.New(ctx, "svc", zapang.Config{
    Level:        "info",
    ExportWriter: myWriter, // Kafka, ClickHouse HTTP, pipe, etc.
}, nil)
```

Output:
```json
{"level":"error","timestamp":"2026-03-19T16:33:11.110086+03:00","caller":"./main.go:30","message":"failed","service":"svc","error":"handle request: parse: invalid input"}
```

Timestamps in RFC3339Nano. No `errorVerbose` — only the short error string.

## Configuration

```go
zapang.Config{
    Level:             "info",          // debug, info, warn, error, dpanic, panic, fatal
    Environment:       "local",         // local, dev, prod
    ExportPath:        "",              // file path, "stdout", "stderr" (dev/prod only)
    ExportWriter:      nil,             // io.Writer for JSON export (any env)
    DisableCaller:     false,           // hide caller file:line
    DisableStacktrace: false,           // disable stacktraces
    StacktraceLevel:   "error",         // min level for stacktraces
    Sampling: &zapang.SamplingConfig{
        Initial:    100,                // entries per second before sampling
        Thereafter: 100,                // keep every Nth entry after Initial
    },
}
```

`DefaultLoggerConfig()` returns sensible defaults (info level, local env, sampling 100/100).

## Context propagation

```go
// Attach logger to context
ctx = zapang.WithContext(ctx, log)

// Retrieve from context (falls back to global)
log = zapang.FromContext(ctx)
```

## Dynamic log level

```go
// At init
log, level := zapang.NewWithLevel(ctx, "svc", cfg, nil)

// At runtime
level.SetLevel(zapcore.DebugLevel)

// Or via global
zapang.SetGlobalLevel("debug")
```

## HTTP middleware

```go
mux := http.NewServeMux()
handler := zapang.HTTPMiddleware(log)(
    zapang.RecoveryMiddleware(log)(mux),
)
```

Logs method, path, status, latency, client IP, response size. Level by status: 5xx → Error, 4xx → Warn, rest → Info. Recovery middleware catches panics.

## OpenTelemetry

```go
// Enrich logger with trace/span IDs from context
log = zapang.WithOtelContext(ctx, log)

// Or get logger from context + OTel in one call
log = zapang.FromOtelContext(ctx)

// Attach to a specific span
log = zapang.LoggerWithSpan(log, span)

// Log a trace event
zapang.TraceEvent(log, span, "cache miss", zapang.CacheKey("user:42"))
```

## Field helpers

Pre-built `zap.Field` functions for structured logging:

| Domain | Fields |
|--------|--------|
| HTTP | `RequestID`, `Method`, `Path`, `StatusCode`, `Latency`, `LatencyMs`, `ClientIP`, `UserAgent`, `RequestSize`, `ResponseSize` |
| Tracing | `TraceID`, `SpanID`, `ParentSpanID` |
| User | `UserID`, `TenantID`, `SessionID` |
| Error | `Error`, `ErrorType`, `ErrorCode` |
| Database | `DBOperation`, `DBTable`, `DBDuration`, `RowsAffected` |
| Cache | `CacheHit`, `CacheKey` |
| Queue | `QueueName`, `MessageID` |
| gRPC | `GRPCMethod`, `GRPCService`, `GRPCCode` |
| Meta | `Component`, `Operation`, `Version`, `Environment` |