// Package logger provides structured logging for FoGit using Go's log/slog.
//
// This package wraps slog to provide a consistent logging interface across
// the application. It separates diagnostic logging (debug, info, warn, error)
// from user-facing output which continues to use fmt for direct terminal display.
//
// Usage:
//
//	// Initialize at startup
//	logger.Init(logger.Options{Level: logger.LevelDebug})
//
//	// Use throughout the application
//	logger.Debug("processing feature", "id", feature.ID)
//	logger.Info("feature created", "name", feature.Name)
//	logger.Warn("config not found, using defaults", "path", configPath)
//	logger.Error("failed to save", "error", err)
//
//	// With context
//	logger.With("command", "feature").Info("starting command")
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

// Level represents the logging level.
type Level = slog.Level

// Log levels matching slog levels.
const (
	LevelDebug = slog.LevelDebug // -4
	LevelInfo  = slog.LevelInfo  // 0
	LevelWarn  = slog.LevelWarn  // 4
	LevelError = slog.LevelError // 8
)

// Format represents the output format for logs.
type Format string

const (
	// FormatText produces human-readable text output.
	FormatText Format = "text"
	// FormatJSON produces structured JSON output.
	FormatJSON Format = "json"
)

// Options configures the logger.
type Options struct {
	// Level is the minimum level to log. Default: LevelWarn
	Level Level

	// Format is the output format. Default: FormatText
	Format Format

	// Output is where logs are written. Default: os.Stderr
	Output io.Writer

	// AddSource adds source file and line to log entries.
	// Only enabled when Level is LevelDebug.
	AddSource bool
}

var (
	// defaultLogger is the package-level logger instance.
	defaultLogger *slog.Logger
	loggerMu      sync.RWMutex

	// currentLevel tracks the configured level for Enabled() checks.
	currentLevel Level = LevelWarn

	// initOnce ensures the logger is initialized exactly once.
	initOnce sync.Once
)

// ensureInit guarantees the logger is initialized before use.
// This prevents race conditions if logging functions are called
// before or during package initialization.
func ensureInit() {
	initOnce.Do(func() {
		initLogger(Options{})
	})
}

// initLogger performs the actual logger initialization.
// Must be called with appropriate locking or via ensureInit.
func initLogger(opts Options) {
	// Apply defaults
	if opts.Output == nil {
		opts.Output = os.Stderr
	}
	if opts.Format == "" {
		opts.Format = FormatText
	}

	currentLevel = opts.Level

	// Create handler options
	handlerOpts := &slog.HandlerOptions{
		Level:     opts.Level,
		AddSource: opts.AddSource || opts.Level <= LevelDebug,
	}

	// Create handler based on format
	var handler slog.Handler
	switch opts.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(opts.Output, handlerOpts)
	default:
		handler = slog.NewTextHandler(opts.Output, handlerOpts)
	}

	defaultLogger = slog.New(handler)
}

// Init initializes the package-level logger with the given options.
// This should be called early in main() before any logging occurs.
// If called multiple times, subsequent calls will reconfigure the logger.
func Init(opts Options) {
	// Ensure default initialization has happened
	ensureInit()

	loggerMu.Lock()
	defer loggerMu.Unlock()

	initLogger(opts)
}

// SetLevel changes the logging level at runtime.
func SetLevel(level Level) {
	ensureInit()

	loggerMu.Lock()
	defer loggerMu.Unlock()

	currentLevel = level

	// Reinitialize with new level
	initLogger(Options{
		Level:  level,
		Output: os.Stderr,
	})
}

// Enabled returns true if the given level is enabled.
func Enabled(level Level) bool {
	ensureInit()
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return level >= currentLevel
}

// Debug logs a message at debug level.
func Debug(msg string, args ...any) {
	ensureInit()
	loggerMu.RLock()
	l := defaultLogger
	loggerMu.RUnlock()
	l.Debug(msg, args...)
}

// Info logs a message at info level.
func Info(msg string, args ...any) {
	ensureInit()
	loggerMu.RLock()
	l := defaultLogger
	loggerMu.RUnlock()
	l.Info(msg, args...)
}

// Warn logs a message at warn level.
func Warn(msg string, args ...any) {
	ensureInit()
	loggerMu.RLock()
	l := defaultLogger
	loggerMu.RUnlock()
	l.Warn(msg, args...)
}

// Error logs a message at error level.
func Error(msg string, args ...any) {
	ensureInit()
	loggerMu.RLock()
	l := defaultLogger
	loggerMu.RUnlock()
	l.Error(msg, args...)
}

// With returns a new Logger with the given attributes added.
func With(args ...any) *slog.Logger {
	ensureInit()
	loggerMu.RLock()
	l := defaultLogger
	loggerMu.RUnlock()
	return l.With(args...)
}

// WithGroup returns a new Logger with the given group name.
func WithGroup(name string) *slog.Logger {
	ensureInit()
	loggerMu.RLock()
	l := defaultLogger
	loggerMu.RUnlock()
	return l.WithGroup(name)
}

// Logger returns the underlying slog.Logger for advanced usage.
func Logger() *slog.Logger {
	ensureInit()
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return defaultLogger
}

// --- Context-aware logging ---

type contextKey struct{}

// NewContext returns a new context with the logger attached.
func NewContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext returns the logger from the context, or the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(contextKey{}).(*slog.Logger); ok {
		return l
	}
	ensureInit()
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return defaultLogger
}

// DebugContext logs at debug level with context.
func DebugContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).DebugContext(ctx, msg, args...)
}

// InfoContext logs at info level with context.
func InfoContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).InfoContext(ctx, msg, args...)
}

// WarnContext logs at warn level with context.
func WarnContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).WarnContext(ctx, msg, args...)
}

// ErrorContext logs at error level with context.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).ErrorContext(ctx, msg, args...)
}

// --- Helper functions for common patterns ---

// Err is a helper that returns a slog.Attr for an error.
// Usage: logger.Error("operation failed", logger.Err(err))
func Err(err error) slog.Attr {
	return slog.Any("error", err)
}

// String is a helper that returns a slog.Attr for a string.
func String(key, value string) slog.Attr {
	return slog.String(key, value)
}

// Int is a helper that returns a slog.Attr for an int.
func Int(key string, value int) slog.Attr {
	return slog.Int(key, value)
}

// Bool is a helper that returns a slog.Attr for a bool.
func Bool(key string, value bool) slog.Attr {
	return slog.Bool(key, value)
}

// Any is a helper that returns a slog.Attr for any value.
func Any(key string, value any) slog.Attr {
	return slog.Any(key, value)
}
