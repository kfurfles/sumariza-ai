package log

import (
	"context"
	"fmt"
	"runtime"
	"sync"
)

// Logger is the main logging interface.
type Logger struct {
	level      Level
	buffer     *Buffer
	baseFields map[string]any
	mu         sync.RWMutex
}

// New creates a new logger with the given minimum level and transporters.
func New(level Level, transporters ...Transporter) *Logger {
	return &Logger{
		level:      level,
		buffer:     NewBuffer(1000, transporters...),
		baseFields: make(map[string]any),
	}
}

// SetLevel changes the minimum log level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	l.level = level
	l.mu.Unlock()
}

// With creates a child logger with additional base fields.
func (l *Logger) With(keysAndValues ...any) *Logger {
	l.mu.RLock()
	newFields := make(map[string]any, len(l.baseFields))
	for k, v := range l.baseFields {
		newFields[k] = v
	}
	l.mu.RUnlock()

	for i := 0; i+1 < len(keysAndValues); i += 2 {
		if key, ok := keysAndValues[i].(string); ok {
			newFields[key] = keysAndValues[i+1]
		}
	}

	return &Logger{
		level:      l.level,
		buffer:     l.buffer,
		baseFields: newFields,
	}
}

// Close shuts down the logger and flushes remaining entries.
func (l *Logger) Close() {
	l.buffer.Close()
}

// log is the internal logging method.
func (l *Logger) log(level Level, ctx context.Context, msg string, keysAndValues ...any) {
	l.mu.RLock()
	minLevel := l.level
	l.mu.RUnlock()

	if !minLevel.Enables(level) {
		return
	}

	entry := NewEntry(level, msg)
	entry.Caller = getCaller(3)

	// Add base fields
	l.mu.RLock()
	for k, v := range l.baseFields {
		entry.Fields[k] = v
	}
	l.mu.RUnlock()

	// Extract from context
	if ctx != nil {
		entry.RequestID = RequestIDFromContext(ctx)
		ctxFields := FieldsFromContext(ctx)
		for k, v := range ctxFields {
			entry.Fields[k] = v
		}
	}

	// Add call-site fields
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		if key, ok := keysAndValues[i].(string); ok {
			entry.Fields[key] = keysAndValues[i+1]
		}
	}

	l.buffer.Send(*entry)
}

// getCaller returns the file:line of the caller.
func getCaller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}

	// Get just the filename, not full path
	short := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			break
		}
	}

	return fmt.Sprintf("%s:%d", short, line)
}

// Trace logs at Trace level.
func (l *Logger) Trace(msg string, keysAndValues ...any) {
	l.log(Trace, nil, msg, keysAndValues...)
}

// Debug logs at Debug level.
func (l *Logger) Debug(msg string, keysAndValues ...any) {
	l.log(Debug, nil, msg, keysAndValues...)
}

// Info logs at Info level.
func (l *Logger) Info(msg string, keysAndValues ...any) {
	l.log(Info, nil, msg, keysAndValues...)
}

// Warn logs at Warn level.
func (l *Logger) Warn(msg string, keysAndValues ...any) {
	l.log(Warn, nil, msg, keysAndValues...)
}

// Error logs at Error level.
func (l *Logger) Error(msg string, keysAndValues ...any) {
	l.log(Error, nil, msg, keysAndValues...)
}

// Fatal logs at Fatal level.
// Note: Does not actually exit - that's the caller's responsibility.
func (l *Logger) Fatal(msg string, keysAndValues ...any) {
	l.log(Fatal, nil, msg, keysAndValues...)
}

// TraceCtx logs at Trace level with context.
func (l *Logger) TraceCtx(ctx context.Context, msg string, keysAndValues ...any) {
	l.log(Trace, ctx, msg, keysAndValues...)
}

// DebugCtx logs at Debug level with context.
func (l *Logger) DebugCtx(ctx context.Context, msg string, keysAndValues ...any) {
	l.log(Debug, ctx, msg, keysAndValues...)
}

// InfoCtx logs at Info level with context.
func (l *Logger) InfoCtx(ctx context.Context, msg string, keysAndValues ...any) {
	l.log(Info, ctx, msg, keysAndValues...)
}

// WarnCtx logs at Warn level with context.
func (l *Logger) WarnCtx(ctx context.Context, msg string, keysAndValues ...any) {
	l.log(Warn, ctx, msg, keysAndValues...)
}

// ErrorCtx logs at Error level with context.
func (l *Logger) ErrorCtx(ctx context.Context, msg string, keysAndValues ...any) {
	l.log(Error, ctx, msg, keysAndValues...)
}

// FatalCtx logs at Fatal level with context.
func (l *Logger) FatalCtx(ctx context.Context, msg string, keysAndValues ...any) {
	l.log(Fatal, ctx, msg, keysAndValues...)
}

// --- Global Logger ---

var (
	globalLogger *Logger
	globalMu     sync.RWMutex
)

// SetDefault sets the global default logger.
func SetDefault(l *Logger) {
	globalMu.Lock()
	globalLogger = l
	globalMu.Unlock()
}

// Default returns the global logger, creating a no-op one if not set.
func Default() *Logger {
	globalMu.RLock()
	l := globalLogger
	globalMu.RUnlock()

	if l == nil {
		// Return a no-op logger
		return &Logger{
			level:      Fatal + 1, // Nothing will be logged
			buffer:     NewBuffer(1, &noopTransporter{}),
			baseFields: make(map[string]any),
		}
	}
	return l
}

type noopTransporter struct{}

func (n *noopTransporter) Name() string      { return "noop" }
func (n *noopTransporter) Write(Entry) error { return nil }
func (n *noopTransporter) Close() error      { return nil }

// Global convenience functions

// GlobalTrace logs at Trace level using the global logger.
func GlobalTrace(msg string, keysAndValues ...any) {
	Default().Trace(msg, keysAndValues...)
}

// GlobalDebug logs at Debug level using the global logger.
func GlobalDebug(msg string, keysAndValues ...any) {
	Default().Debug(msg, keysAndValues...)
}

// GlobalInfo logs at Info level using the global logger.
func GlobalInfo(msg string, keysAndValues ...any) {
	Default().Info(msg, keysAndValues...)
}

// GlobalWarn logs at Warn level using the global logger.
func GlobalWarn(msg string, keysAndValues ...any) {
	Default().Warn(msg, keysAndValues...)
}

// GlobalError logs at Error level using the global logger.
func GlobalError(msg string, keysAndValues ...any) {
	Default().Error(msg, keysAndValues...)
}

// GlobalFatal logs at Fatal level using the global logger.
func GlobalFatal(msg string, keysAndValues ...any) {
	Default().Fatal(msg, keysAndValues...)
}

// GlobalTraceCtx logs at Trace level with context using the global logger.
func GlobalTraceCtx(ctx context.Context, msg string, keysAndValues ...any) {
	Default().TraceCtx(ctx, msg, keysAndValues...)
}

// GlobalDebugCtx logs at Debug level with context using the global logger.
func GlobalDebugCtx(ctx context.Context, msg string, keysAndValues ...any) {
	Default().DebugCtx(ctx, msg, keysAndValues...)
}

// GlobalInfoCtx logs at Info level with context using the global logger.
func GlobalInfoCtx(ctx context.Context, msg string, keysAndValues ...any) {
	Default().InfoCtx(ctx, msg, keysAndValues...)
}

// GlobalWarnCtx logs at Warn level with context using the global logger.
func GlobalWarnCtx(ctx context.Context, msg string, keysAndValues ...any) {
	Default().WarnCtx(ctx, msg, keysAndValues...)
}

// GlobalErrorCtx logs at Error level with context using the global logger.
func GlobalErrorCtx(ctx context.Context, msg string, keysAndValues ...any) {
	Default().ErrorCtx(ctx, msg, keysAndValues...)
}

// GlobalFatalCtx logs at Fatal level with context using the global logger.
func GlobalFatalCtx(ctx context.Context, msg string, keysAndValues ...any) {
	Default().FatalCtx(ctx, msg, keysAndValues...)
}
