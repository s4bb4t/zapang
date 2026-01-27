package logger

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Request fields for HTTP request logging.
func RequestID(id string) zap.Field {
	return zap.String("request_id", id)
}

func Method(method string) zap.Field {
	return zap.String("http_method", method)
}

func Path(path string) zap.Field {
	return zap.String("http_path", path)
}

func StatusCode(code int) zap.Field {
	return zap.Int("http_status", code)
}

func Latency(d time.Duration) zap.Field {
	return zap.Duration("latency", d)
}

func LatencyMs(d time.Duration) zap.Field {
	return zap.Float64("latency_ms", float64(d.Nanoseconds())/1e6)
}

func ClientIP(ip string) zap.Field {
	return zap.String("client_ip", ip)
}

func UserAgent(ua string) zap.Field {
	return zap.String("user_agent", ua)
}

func RequestSize(size int64) zap.Field {
	return zap.Int64("request_size", size)
}

func ResponseSize(size int) zap.Field {
	return zap.Int("response_size", size)
}

// Tracing fields for distributed tracing correlation.
func TraceID(id string) zap.Field {
	return zap.String("trace_id", id)
}

func SpanID(id string) zap.Field {
	return zap.String("span_id", id)
}

func ParentSpanID(id string) zap.Field {
	return zap.String("parent_span_id", id)
}

// User fields for user context.
func UserID(id string) zap.Field {
	return zap.String("user_id", id)
}

func TenantID(id string) zap.Field {
	return zap.String("tenant_id", id)
}

func SessionID(id string) zap.Field {
	return zap.String("session_id", id)
}

// Error fields for error logging.
func Error(err error) zap.Field {
	return zap.Error(err)
}

func ErrorType(err error) zap.Field {
	return zap.String("error_type", fmt.Sprintf("%T", err))
}

func ErrorCode(code string) zap.Field {
	return zap.String("error_code", code)
}

// Database fields for database operation logging.
func DBOperation(op string) zap.Field {
	return zap.String("db_operation", op)
}

func DBTable(table string) zap.Field {
	return zap.String("db_table", table)
}

func DBDuration(d time.Duration) zap.Field {
	return zap.Duration("db_duration", d)
}

func RowsAffected(n int64) zap.Field {
	return zap.Int64("rows_affected", n)
}

// Cache fields for cache operation logging.
func CacheHit(hit bool) zap.Field {
	return zap.Bool("cache_hit", hit)
}

func CacheKey(key string) zap.Field {
	return zap.String("cache_key", key)
}

// Queue fields for message queue logging.
func QueueName(name string) zap.Field {
	return zap.String("queue_name", name)
}

func MessageID(id string) zap.Field {
	return zap.String("message_id", id)
}

// gRPC fields for gRPC request logging.
func GRPCMethod(method string) zap.Field {
	return zap.String("grpc_method", method)
}

func GRPCService(service string) zap.Field {
	return zap.String("grpc_service", service)
}

func GRPCCode(code string) zap.Field {
	return zap.String("grpc_code", code)
}

// Component identifies the component generating the log.
func Component(name string) zap.Field {
	return zap.String("component", name)
}

// Operation identifies the operation being performed.
func Operation(name string) zap.Field {
	return zap.String("operation", name)
}

// Version for service versioning.
func Version(v string) zap.Field {
	return zap.String("version", v)
}

// Environment for deployment environment.
func Environment(env string) zap.Field {
	return zap.String("environment", env)
}
