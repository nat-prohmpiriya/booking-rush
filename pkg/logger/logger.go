package logger

import (
	"context"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ContextKey is the type for context keys used by the logger
type ContextKey string

const (
	// TraceIDKey is the context key for trace ID
	TraceIDKey ContextKey = "trace_id"
	// SpanIDKey is the context key for span ID
	SpanIDKey ContextKey = "span_id"
	// ServiceKey is the context key for service name
	ServiceKey ContextKey = "service"
	// RequestIDKey is the context key for request ID
	RequestIDKey ContextKey = "request_id"
)

// Logger wraps zap.Logger with additional functionality
type Logger struct {
	*zap.Logger
	serviceName string
}

var (
	globalLogger *Logger
	once         sync.Once
)

// Config holds logger configuration
type Config struct {
	Level       string // debug, info, warn, error
	ServiceName string
	Development bool   // if true, uses console encoder; if false, uses JSON encoder
	OutputPath  string // stdout, stderr, or file path
	// OTLP configuration for exporting logs to OTel Collector
	OTLPEnabled   bool
	OTLPEndpoint  string        // e.g., "otel-collector:4317"
	OTLPInsecure  bool          // Use insecure connection (no TLS)
	OTLPTimeout   time.Duration // Timeout for OTLP export
	BatchSize     int           // Batch size for log export
	BatchInterval time.Duration // Interval for batch export
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:         "info",
		ServiceName:   "booking-rush",
		Development:   false,
		OutputPath:    "stdout",
		OTLPEnabled:   false,
		OTLPEndpoint:  "localhost:4317",
		OTLPInsecure:  true,
		OTLPTimeout:   5 * time.Second,
		BatchSize:     100,
		BatchInterval: 1 * time.Second,
	}
}

// parseLevel converts string level to zapcore.Level
func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// New creates a new Logger with the given configuration
func New(cfg *Config) (*Logger, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	level := parseLevel(cfg.Level)

	// Configure encoder for JSON output (structured logging)
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	if cfg.Development {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Configure output
	var output zapcore.WriteSyncer
	switch cfg.OutputPath {
	case "stdout", "":
		output = zapcore.AddSync(os.Stdout)
	case "stderr":
		output = zapcore.AddSync(os.Stderr)
	default:
		file, err := os.OpenFile(cfg.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		output = zapcore.AddSync(file)
	}

	// Create cores - always include stdout/file output
	cores := []zapcore.Core{
		zapcore.NewCore(encoder, output, level),
	}

	// Add OTLP core if enabled
	if cfg.OTLPEnabled && cfg.OTLPEndpoint != "" {
		otlpCore := NewOTLPCore(cfg, level)
		if otlpCore != nil {
			cores = append(cores, otlpCore)
		}
	}

	// Combine cores using Tee
	core := zapcore.NewTee(cores...)

	// Add caller skip for wrapper methods
	zapLogger := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	// Add service name as a default field
	zapLogger = zapLogger.With(zap.String("service", cfg.ServiceName))

	return &Logger{
		Logger:      zapLogger,
		serviceName: cfg.ServiceName,
	}, nil
}

// Init initializes the global logger
func Init(cfg *Config) error {
	var err error
	once.Do(func() {
		globalLogger, err = New(cfg)
	})
	return err
}

// Get returns the global logger instance
func Get() *Logger {
	if globalLogger == nil {
		// Initialize with default config if not initialized
		_ = Init(DefaultConfig())
	}
	return globalLogger
}

// WithContext returns a logger with context fields (trace_id, span_id, request_id)
// It extracts trace_id from OTel span context automatically
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if ctx == nil {
		return l
	}

	fields := make([]zap.Field, 0, 3)

	// Extract trace_id and span_id from OTel context (priority)
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		if span.SpanContext().HasTraceID() {
			fields = append(fields, zap.String("trace_id", span.SpanContext().TraceID().String()))
		}
		if span.SpanContext().HasSpanID() {
			fields = append(fields, zap.String("span_id", span.SpanContext().SpanID().String()))
		}
	} else {
		// Fallback to context value if OTel span not available
		if traceID, ok := ctx.Value(TraceIDKey).(string); ok && traceID != "" {
			fields = append(fields, zap.String("trace_id", traceID))
		}
		if spanID, ok := ctx.Value(SpanIDKey).(string); ok && spanID != "" {
			fields = append(fields, zap.String("span_id", spanID))
		}
	}

	if requestID, ok := ctx.Value(RequestIDKey).(string); ok && requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}

	if len(fields) == 0 {
		return l
	}

	return &Logger{
		Logger:      l.Logger.With(fields...),
		serviceName: l.serviceName,
	}
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields ...zap.Field) *Logger {
	return &Logger{
		Logger:      l.Logger.With(fields...),
		serviceName: l.serviceName,
	}
}

// WithService returns a logger with a different service name
func (l *Logger) WithService(serviceName string) *Logger {
	return &Logger{
		Logger:      l.Logger.With(zap.String("service", serviceName)),
		serviceName: serviceName,
	}
}

// Debug logs a debug message with context
func (l *Logger) DebugContext(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithContext(ctx).Debug(msg, fields...)
}

// Info logs an info message with context
func (l *Logger) InfoContext(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithContext(ctx).Info(msg, fields...)
}

// Warn logs a warning message with context
func (l *Logger) WarnContext(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithContext(ctx).Warn(msg, fields...)
}

// Error logs an error message with context
func (l *Logger) ErrorContext(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithContext(ctx).Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *Logger) FatalContext(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithContext(ctx).Fatal(msg, fields...)
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// --- Package-level convenience functions ---

// Debug logs a debug message using the global logger
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Info logs an info message using the global logger
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn logs a warning message using the global logger
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error logs an error message using the global logger
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal logs a fatal message using the global logger
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// DebugCtx logs a debug message with context using the global logger
func DebugCtx(ctx context.Context, msg string, fields ...zap.Field) {
	Get().DebugContext(ctx, msg, fields...)
}

// InfoCtx logs an info message with context using the global logger
func InfoCtx(ctx context.Context, msg string, fields ...zap.Field) {
	Get().InfoContext(ctx, msg, fields...)
}

// WarnCtx logs a warning message with context using the global logger
func WarnCtx(ctx context.Context, msg string, fields ...zap.Field) {
	Get().WarnContext(ctx, msg, fields...)
}

// ErrorCtx logs an error message with context using the global logger
func ErrorCtx(ctx context.Context, msg string, fields ...zap.Field) {
	Get().ErrorContext(ctx, msg, fields...)
}

// FatalCtx logs a fatal message with context using the global logger
func FatalCtx(ctx context.Context, msg string, fields ...zap.Field) {
	Get().FatalContext(ctx, msg, fields...)
}

// WithContext returns a logger with context fields from the global logger
func WithContext(ctx context.Context) *Logger {
	return Get().WithContext(ctx)
}

// WithFields returns a logger with additional fields from the global logger
func WithFields(fields ...zap.Field) *Logger {
	return Get().WithFields(fields...)
}

// Sync flushes any buffered log entries from the global logger
func Sync() error {
	return Get().Sync()
}
