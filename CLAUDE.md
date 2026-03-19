# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Zapang is a Go structured logging library built on top of `go.uber.org/zap`. It provides environment-aware configuration (local/dev/prod), context propagation with trace IDs, HTTP middleware, OpenTelemetry integration, and pre-built field helpers.

Single-package library — all code lives in the root under package `zapang`.

## Commands

```bash
go build ./...    # Build
go test ./...     # Run tests
go vet ./...      # Vet
go mod tidy       # Tidy dependencies
```

No Makefile, CI config, or linter config exists.

## Architecture

**logger.go** — Core. Creates zap loggers with multi-core output (console + optional JSON export). Manages a thread-safe global singleton (`sync.RWMutex`). Provides context integration (`FromContext`/`WithContext`) and graceful shutdown on context cancellation. Custom caller encoder normalizes paths relative to project root (detected at `init()` by walking up to find `go.mod`). Human-readable time encoder (`02 Jan 15:04:05`) for console, RFC3339Nano for JSON export.

**encoder.go** — Custom encoder wrappers:
- `consoleEncoder` — wraps zap's console encoder. Intercepts `ErrorType` fields to extract verbose error traces from `go-faster/errors` (or any `fmt.Formatter`), renders them as a colored multi-line block (bold red for error messages, dim for stack frames). Reformats the JSON fields blob as `key=value` pairs.
- `exportEncoder` — wraps zap's JSON encoder. Strips `errorVerbose` field from output, keeping only the short error string for clean JSON export to aggregation systems.

**config.go** — `Config` and `SamplingConfig` structs with `yaml`/`json`/`mapstructure` tags. `DefaultLoggerConfig()` returns sensible defaults (info level, local env). Three environments change output behavior: `local` (console only), `dev`/`prod` (console + optional JSON export). `ExportWriter` (`io.Writer`) allows direct log export to any destination (Kafka, ClickHouse, etc.) in any environment, takes precedence over `ExportPath`.

**fields.go** — ~37 pre-built `zap.Field` helpers organized by domain: request metadata, tracing, user context, errors, database, cache, queue, gRPC, and general metadata.

**middleware.go** — `HTTPMiddleware` logs requests with status/latency/IPs (auto-extracts from X-Forwarded-For/X-Real-IP). Log level varies by status code (5xx→Error, 4xx→Warn). `RecoveryMiddleware` catches panics. Uses a response wrapper to capture status/size.

**otel.go** — OpenTelemetry trace/span ID extraction and correlation (`WithOtelContext`, `FromOtelContext`, `LoggerWithSpan`, `TraceEvent`).

## Key Patterns

- **Global + Context:** Logger accessed globally via `Global()` or per-request via `FromContext(ctx)`
- **Dual output:** Console (`key=value`, human time, colored verbose errors) + JSON export (RFC3339Nano, no errorVerbose) via `TeeCore`
- **`NewWithLevel()`** returns a `zap.AtomicLevel` for dynamic log level changes at runtime
- **`New()`** accepts an optional `io.Writer` parameter for test/debug output injection
- **`ExportWriter`** in Config accepts any `io.Writer` for direct log export (Kafka, ClickHouse HTTP, pipes, etc.)
- **Field helpers** are pure functions returning `zap.Field` — composable and zero-allocation
- **Encoder wrappers** intercept `ErrorType` fields at `EncodeEntry` level, replacing them with plain `String` fields to prevent the inner encoder from generating `errorVerbose`