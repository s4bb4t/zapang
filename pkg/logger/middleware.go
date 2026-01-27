package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// responseWriter wraps http.ResponseWriter to capture status code and size.
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// HTTPMiddleware returns a middleware that logs HTTP requests.
// It captures method, path, status, latency, and request metadata.
func HTTPMiddleware(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newResponseWriter(w)

			// Extract trace ID if present
			traceID := r.Header.Get("X-Trace-ID")
			if traceID == "" {
				traceID = r.Header.Get("X-Request-ID")
			}

			// Create request-scoped logger
			reqLogger := log.With(
				Method(r.Method),
				Path(r.URL.Path),
				ClientIP(getClientIP(r)),
				UserAgent(r.UserAgent()),
			)

			if traceID != "" {
				reqLogger = reqLogger.With(TraceID(traceID))
			}

			// Store logger in context
			ctx := WithContext(r.Context(), reqLogger)
			r = r.WithContext(ctx)

			// Process request
			next.ServeHTTP(rw, r)

			// Calculate latency
			latency := time.Since(start)

			// Build log fields
			fields := []zap.Field{
				StatusCode(rw.status),
				LatencyMs(latency),
				ResponseSize(rw.size),
			}

			if r.ContentLength > 0 {
				fields = append(fields, RequestSize(r.ContentLength))
			}

			// Log at appropriate level based on status
			switch {
			case rw.status >= 500:
				reqLogger.Error("request completed", fields...)
			case rw.status >= 400:
				reqLogger.Warn("request completed", fields...)
			default:
				reqLogger.Info("request completed", fields...)
			}
		})
	}
}

// RecoveryMiddleware returns a middleware that recovers from panics and logs them.
func RecoveryMiddleware(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered",
						zap.Any("panic", rec),
						Method(r.Method),
						Path(r.URL.Path),
						zap.Stack("stacktrace"),
					)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func getClientIP(r *http.Request) string {
	// Check common proxy headers
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}
