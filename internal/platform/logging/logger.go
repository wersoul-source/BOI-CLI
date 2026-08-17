package logging

import (
	"io"
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with convenience methods
type Logger struct {
	*slog.Logger
}

// New creates a new Logger at INFO level
func New() *Logger {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return &Logger{Logger: slog.New(handler)}
}

// NewWithLevel creates a Logger at the specified level
func NewWithLevel(levelStr string) *Logger {
	var level slog.Level
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	return &Logger{Logger: slog.New(handler)}
}

// NewSilent creates a Logger that discards all output
func NewSilent() *Logger {
	handler := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})
	return &Logger{Logger: slog.New(handler)}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}
