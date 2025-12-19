// Package logging provides logging utilities for the MCP over SOCKS bridge.
package logging

import (
	"fmt"
	"io"
	"os"
	"time"
)

// LogLevel represents the severity of a log message.
type LogLevel int

const (
	// LogLevelError logs only errors.
	LogLevelError LogLevel = iota
	// LogLevelInfo logs errors and informational messages.
	LogLevelInfo
	// LogLevelDebug logs everything including debug messages.
	LogLevelDebug
)

// String returns the string representation of a LogLevel.
func (l LogLevel) String() string {
	switch l {
	case LogLevelError:
		return "ERROR"
	case LogLevelInfo:
		return "INFO"
	case LogLevelDebug:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel parses a string into a LogLevel.
func ParseLogLevel(s string) LogLevel {
	switch s {
	case "error":
		return LogLevelError
	case "info":
		return LogLevelInfo
	case "debug":
		return LogLevelDebug
	default:
		return LogLevelInfo
	}
}

// Logger is a simple logger that writes to stderr.
type Logger struct {
	level  LogLevel
	writer io.Writer
}

// New creates a new Logger with the specified log level.
func New(level LogLevel) *Logger {
	return &Logger{
		level:  level,
		writer: os.Stderr,
	}
}

// NewWithWriter creates a new Logger with a custom writer.
func NewWithWriter(level LogLevel, writer io.Writer) *Logger {
	return &Logger{
		level:  level,
		writer: writer,
	}
}

// SetLevel changes the log level.
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// log writes a log message if the level is enabled.
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level > l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.writer, "[%s] %s: %s\n", timestamp, level.String(), message)
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LogLevelError, format, args...)
}

// Info logs an informational message.
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LogLevelInfo, format, args...)
}

// Debug logs a debug message.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LogLevelDebug, format, args...)
}

