package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/google/uuid"
)

// CorrelationIDKey is the context key for correlation IDs
type CorrelationIDKey struct{}

// Logger type is now defined in types.go to avoid redeclaration

// LogEntry type is now defined in types.go to avoid redeclaration

// NewLogger creates a new structured logger for a component
func NewLogger(component string) *Logger {
	// Create JSON handler for structured logging
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize timestamp format
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   "timestamp",
					Value: slog.StringValue(a.Value.Time().Format(time.RFC3339Nano)),
				}
			}
			// Customize level format
			if a.Key == slog.LevelKey {
				return slog.Attr{
					Key:   "level",
					Value: slog.StringValue(a.Value.String()),
				}
			}
			// Customize message format
			if a.Key == slog.MessageKey {
				return slog.Attr{
					Key:   "message",
					Value: a.Value,
				}
			}
			// Customize source format
			if a.Key == slog.SourceKey {
				source := a.Value.Any().(*slog.Source)
				return slog.Attr{
					Key:   "caller",
					Value: slog.StringValue(fmt.Sprintf("%s:%d", source.File, source.Line)),
				}
			}
			return a
		},
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &Logger{
		Logger:    logger,
		component: component,
	}
}

// WithCorrelationID adds a correlation ID to the context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	if correlationID == "" {
		correlationID = uuid.New().String()
	}
	return context.WithValue(ctx, CorrelationIDKey{}, correlationID)
}


// WithContext creates a logger with context information
func (l *Logger) WithContext(ctx context.Context) *Logger {
	correlationID := GetCorrelationID(ctx)
	
	// Create a new logger with context attributes
	contextLogger := l.Logger.With(
		"component", l.component,
		"correlation_id", correlationID,
	)
	
	return &Logger{
		Logger:    contextLogger,
		component: l.component,
	}
}

// WithFields adds structured fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	attrs := make([]slog.Attr, 0, len(fields))
	for k, v := range fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	
	fieldLogger := l.Logger.With(attrs...)
	return &Logger{
		Logger:    fieldLogger,
		component: l.component,
	}
}

// WithError adds error information to the logger
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	
	errorLogger := l.Logger.With("error", err.Error())
	return &Logger{
		Logger:    errorLogger,
		component: l.component,
	}
}

// InfoCtx logs an info message with context
func (l *Logger) InfoCtx(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Info(msg, args...)
}

// WarnCtx logs a warning message with context
func (l *Logger) WarnCtx(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Warn(msg, args...)
}

// ErrorCtx logs an error message with context
func (l *Logger) ErrorCtx(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Error(msg, args...)
}

// DebugCtx logs a debug message with context
func (l *Logger) DebugCtx(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Debug(msg, args...)
}

// LogRequest logs HTTP request information
func (l *Logger) LogRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	l.WithContext(ctx).Info("HTTP request",
		"method", method,
		"path", path,
		"status_code", statusCode,
		"duration_ms", duration.Milliseconds(),
	)
}

// LogE2Operation logs E2 interface operations
func (l *Logger) LogE2Operation(ctx context.Context, operation, nodeID string, success bool, duration time.Duration) {
	level := slog.LevelInfo
	if !success {
		level = slog.LevelError
	}
	
	l.WithContext(ctx).Log(ctx, level, "E2 operation",
		"operation", operation,
		"node_id", nodeID,
		"success", success,
		"duration_ms", duration.Milliseconds(),
		"interface", "E2",
	)
}

// LogA1Operation logs A1 interface operations
func (l *Logger) LogA1Operation(ctx context.Context, operation, policyTypeID, policyInstanceID string, success bool, duration time.Duration) {
	level := slog.LevelInfo
	if !success {
		level = slog.LevelError
	}
	
	l.WithContext(ctx).Log(ctx, level, "A1 operation",
		"operation", operation,
		"policy_type_id", policyTypeID,
		"policy_instance_id", policyInstanceID,
		"success", success,
		"duration_ms", duration.Milliseconds(),
		"interface", "A1",
	)
}

// LogO1Operation logs O1 interface operations
func (l *Logger) LogO1Operation(ctx context.Context, operation, target string, success bool, duration time.Duration) {
	level := slog.LevelInfo
	if !success {
		level = slog.LevelError
	}
	
	l.WithContext(ctx).Log(ctx, level, "O1 operation",
		"operation", operation,
		"target", target,
		"success", success,
		"duration_ms", duration.Milliseconds(),
		"interface", "O1",
	)
}

// LogSubscriptionOperation logs subscription operations
func (l *Logger) LogSubscriptionOperation(ctx context.Context, operation, subscriptionID, nodeID string, success bool, duration time.Duration) {
	level := slog.LevelInfo
	if !success {
		level = slog.LevelError
	}
	
	l.WithContext(ctx).Log(ctx, level, "Subscription operation",
		"operation", operation,
		"subscription_id", subscriptionID,
		"node_id", nodeID,
		"success", success,
		"duration_ms", duration.Milliseconds(),
	)
}

// LogSecurityEvent logs security-related events
func (l *Logger) LogSecurityEvent(ctx context.Context, event, userID, resource string, success bool) {
	level := slog.LevelWarn
	if !success {
		level = slog.LevelError
	}
	
	l.WithContext(ctx).Log(ctx, level, "Security event",
		"event", event,
		"user_id", userID,
		"resource", resource,
		"success", success,
		"category", "security",
	)
}

// LogPerformanceMetric logs performance metrics
func (l *Logger) LogPerformanceMetric(ctx context.Context, metric string, value float64, unit string, tags map[string]string) {
	fields := map[string]interface{}{
		"metric": metric,
		"value":  value,
		"unit":   unit,
	}
	
	for k, v := range tags {
		fields[k] = v
	}
	
	l.WithContext(ctx).WithFields(fields).Info("Performance metric")
}

// GetCaller returns the caller information
func GetCaller(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown"
	}
	return fmt.Sprintf("%s:%d", file, line)
}

// Global logger instances for different components
var (
	DashboardLogger    = NewLogger("dashboard")
	E2Logger          = NewLogger("e2")
	A1Logger          = NewLogger("a1")
	O1Logger          = NewLogger("o1")
	SubscriptionLogger = NewLogger("subscription")
	SecurityLogger     = NewLogger("security")
	MetricsLogger      = NewLogger("metrics")
)