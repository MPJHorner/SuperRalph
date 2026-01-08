// Package log provides a structured logging wrapper around charmbracelet/log.
// It provides a consistent logging interface throughout the SuperRalph codebase.
package log

import (
	"io"
	"os"

	"github.com/charmbracelet/log"
)

// Level represents log levels
type Level = log.Level

// Level constants
const (
	DebugLevel = log.DebugLevel
	InfoLevel  = log.InfoLevel
	WarnLevel  = log.WarnLevel
	ErrorLevel = log.ErrorLevel
	FatalLevel = log.FatalLevel
)

// Logger is a structured logger instance
type Logger struct {
	*log.Logger
}

// Options configures a logger
type Options struct {
	Level           Level
	Prefix          string
	ReportCaller    bool
	ReportTimestamp bool
	Output          io.Writer
}

// DefaultOptions returns sensible default options
func DefaultOptions() Options {
	return Options{
		Level:           InfoLevel,
		ReportCaller:    false,
		ReportTimestamp: true,
		Output:          os.Stderr,
	}
}

// New creates a new logger with the given options
func New(opts Options) *Logger {
	output := opts.Output
	if output == nil {
		output = os.Stderr
	}

	l := log.NewWithOptions(output, log.Options{
		Level:           opts.Level,
		Prefix:          opts.Prefix,
		ReportCaller:    opts.ReportCaller,
		ReportTimestamp: opts.ReportTimestamp,
	})

	return &Logger{Logger: l}
}

// Default returns the default logger (writes to stderr with INFO level)
var defaultLogger = New(DefaultOptions())

// Default returns the default logger instance
func Default() *Logger {
	return defaultLogger
}

// SetDefault sets the default logger
func SetDefault(l *Logger) {
	defaultLogger = l
}

// SetLevel sets the log level for the default logger
func SetLevel(level Level) {
	defaultLogger.SetLevel(level)
}

// With returns a new logger with additional context
func (l *Logger) With(keyvals ...interface{}) *Logger {
	return &Logger{Logger: l.Logger.With(keyvals...)}
}

// WithPrefix returns a new logger with the given prefix
func (l *Logger) WithPrefix(prefix string) *Logger {
	newLogger := *l.Logger
	newLogger.SetPrefix(prefix)
	return &Logger{Logger: &newLogger}
}

// Package-level convenience functions that use the default logger

// Debug logs a debug message
func Debug(msg interface{}, keyvals ...interface{}) {
	defaultLogger.Debug(msg, keyvals...)
}

// Info logs an info message
func Info(msg interface{}, keyvals ...interface{}) {
	defaultLogger.Info(msg, keyvals...)
}

// Warn logs a warning message
func Warn(msg interface{}, keyvals ...interface{}) {
	defaultLogger.Warn(msg, keyvals...)
}

// Error logs an error message
func Error(msg interface{}, keyvals ...interface{}) {
	defaultLogger.Error(msg, keyvals...)
}

// Fatal logs a fatal message and exits
func Fatal(msg interface{}, keyvals ...interface{}) {
	defaultLogger.Fatal(msg, keyvals...)
}

// Debugf logs a formatted debug message
func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

// Infof logs a formatted info message
func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(format string, args ...interface{}) {
	defaultLogger.Fatalf(format, args...)
}

// Print logs a message at the default level (Info)
func Print(msg interface{}, keyvals ...interface{}) {
	defaultLogger.Info(msg, keyvals...)
}

// Printf logs a formatted message at the default level (Info)
func Printf(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}
