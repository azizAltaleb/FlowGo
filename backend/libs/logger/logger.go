package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Log levels
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Context keys for correlation
type ctxKey string

const (
	CorrelationIDKey ctxKey = "correlation_id"
	ComponentKey     ctxKey = "component"
)

// Logger is a structured logger with correlation ID support
type Logger struct {
	mu        sync.Mutex
	out       io.Writer
	level     Level
	component string
	fields    map[string]any
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp     string         `json:"timestamp"`
	Level         string         `json:"level"`
	Component     string         `json:"component,omitempty"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	Message       string         `json:"message"`
	Fields        map[string]any `json:"fields,omitempty"`
	Caller        string         `json:"caller,omitempty"`
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Default returns the default logger instance
func Default() *Logger {
	once.Do(func() {
		defaultLogger = New("app")
	})
	return defaultLogger
}

// New creates a new logger with the given component name
func New(component string) *Logger {
	levelStr := os.Getenv("LOG_LEVEL")
	level := LevelInfo
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		level = LevelDebug
	case "WARN":
		level = LevelWarn
	case "ERROR":
		level = LevelError
	}

	return &Logger{
		out:       os.Stdout,
		level:     level,
		component: component,
		fields:    make(map[string]any),
	}
}

// WithField returns a new logger with the given field added
func (l *Logger) WithField(key string, value any) *Logger {
	newFields := make(map[string]any, len(l.fields)+1)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &Logger{
		out:       l.out,
		level:     l.level,
		component: l.component,
		fields:    newFields,
	}
}

// WithFields returns a new logger with the given fields added
func (l *Logger) WithFields(fields map[string]any) *Logger {
	newFields := make(map[string]any, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		out:       l.out,
		level:     l.level,
		component: l.component,
		fields:    newFields,
	}
}

// WithComponent returns a new logger with the given component name
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		out:       l.out,
		level:     l.level,
		component: component,
		fields:    l.fields,
	}
}

// ContextWithCorrelationID adds a correlation ID to the context
func ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// CorrelationIDFromContext extracts the correlation ID from context
func CorrelationIDFromContext(ctx context.Context) string {
	if v := ctx.Value(CorrelationIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// log writes a log entry
func (l *Logger) log(ctx context.Context, level Level, msg string, fields map[string]any) {
	if level < l.level {
		return
	}

	// Merge fields
	allFields := make(map[string]any, len(l.fields)+len(fields))
	for k, v := range l.fields {
		allFields[k] = v
	}
	for k, v := range fields {
		allFields[k] = v
	}

	// Get caller info
	_, file, line, ok := runtime.Caller(2)
	caller := ""
	if ok {
		parts := strings.Split(file, "/")
		if len(parts) > 2 {
			caller = fmt.Sprintf("%s/%s:%d", parts[len(parts)-2], parts[len(parts)-1], line)
		} else {
			caller = fmt.Sprintf("%s:%d", file, line)
		}
	}

	entry := LogEntry{
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Level:         level.String(),
		Component:     l.component,
		CorrelationID: CorrelationIDFromContext(ctx),
		Message:       msg,
		Caller:        caller,
	}

	if len(allFields) > 0 {
		entry.Fields = allFields
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(l.out, `{"timestamp":"%s","level":"ERROR","message":"failed to marshal log entry: %v"}`+"\n",
			time.Now().UTC().Format(time.RFC3339Nano), err)
		return
	}

	fmt.Fprintln(l.out, string(data))
}

// Debug logs a debug message
func (l *Logger) Debug(ctx context.Context, msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(ctx, LevelDebug, msg, f)
}

// Info logs an info message
func (l *Logger) Info(ctx context.Context, msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(ctx, LevelInfo, msg, f)
}

// Warn logs a warning message
func (l *Logger) Warn(ctx context.Context, msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(ctx, LevelWarn, msg, f)
}

// Error logs an error message
func (l *Logger) Error(ctx context.Context, msg string, fields ...map[string]any) {
	f := mergeFields(fields)
	l.log(ctx, LevelError, msg, f)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(ctx context.Context, format string, args ...any) {
	l.log(ctx, LevelDebug, fmt.Sprintf(format, args...), nil)
}

// Infof logs a formatted info message
func (l *Logger) Infof(ctx context.Context, format string, args ...any) {
	l.log(ctx, LevelInfo, fmt.Sprintf(format, args...), nil)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(ctx context.Context, format string, args ...any) {
	l.log(ctx, LevelWarn, fmt.Sprintf(format, args...), nil)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(ctx context.Context, format string, args ...any) {
	l.log(ctx, LevelError, fmt.Sprintf(format, args...), nil)
}

func mergeFields(fields []map[string]any) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	result := make(map[string]any)
	for _, f := range fields {
		for k, v := range f {
			result[k] = v
		}
	}
	return result
}
