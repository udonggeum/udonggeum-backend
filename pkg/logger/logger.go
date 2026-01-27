package logger

import (
	"io"
	"os"
	"runtime"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Logger wraps zerolog.Logger with additional context
type Logger struct {
	logger zerolog.Logger
}

// Config holds logger configuration
type Config struct {
	Level       string // debug, info, warn, error, fatal
	Format      string // json, console
	Output      io.Writer
	EnableColor bool
}

var globalLogger *Logger

// Initialize initializes the global logger with the given configuration
func Initialize(cfg Config) {
	// Set log level
	level := parseLogLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	// Configure output
	var output io.Writer = os.Stdout
	if cfg.Output != nil {
		output = cfg.Output
	}

	// Configure format
	var logger zerolog.Logger
	if cfg.Format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
			NoColor:    !cfg.EnableColor,
		}
		logger = zerolog.New(output).With().Timestamp().Logger()
	} else {
		// JSON format (default)
		logger = zerolog.New(output).With().Timestamp().Logger()
	}

	globalLogger = &Logger{logger: logger}
	log.Logger = logger
}

// parseLogLevel converts string level to zerolog.Level
func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// Get returns the global logger instance
func Get() *Logger {
	if globalLogger == nil {
		// Initialize with default config if not initialized
		Initialize(Config{
			Level:       "info",
			Format:      "console",
			EnableColor: true,
		})
	}
	return globalLogger
}

// WithContext returns a logger with additional context fields
func (l *Logger) WithContext(fields map[string]interface{}) *Logger {
	ctx := l.logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{logger: ctx.Logger()}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	pc, file, line, _ := runtime.Caller(1)
	event := l.logger.Debug().Str("caller", zerolog.CallerMarshalFunc(pc, file, line))
	if len(fields) > 0 {
		for k, v := range fields[0] {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	pc, file, line, _ := runtime.Caller(1)
	event := l.logger.Info().Str("caller", zerolog.CallerMarshalFunc(pc, file, line))
	if len(fields) > 0 {
		for k, v := range fields[0] {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	pc, file, line, _ := runtime.Caller(1)
	event := l.logger.Warn().Str("caller", zerolog.CallerMarshalFunc(pc, file, line))
	if len(fields) > 0 {
		for k, v := range fields[0] {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

// Error logs an error message
func (l *Logger) Error(msg string, err error, fields ...map[string]interface{}) {
	pc, file, line, _ := runtime.Caller(1)
	event := l.logger.Error().Err(err).Str("caller", zerolog.CallerMarshalFunc(pc, file, line))
	if len(fields) > 0 {
		for k, v := range fields[0] {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, err error, fields ...map[string]interface{}) {
	pc, file, line, _ := runtime.Caller(1)
	event := l.logger.Fatal().Err(err).Str("caller", zerolog.CallerMarshalFunc(pc, file, line))
	if len(fields) > 0 {
		for k, v := range fields[0] {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

// Package-level convenience functions

// Debug logs a debug message using the global logger
func Debug(msg string, fields ...map[string]interface{}) {
	l := Get()
	pc, file, line, _ := runtime.Caller(1)
	event := l.logger.Debug().Str("caller", zerolog.CallerMarshalFunc(pc, file, line))
	if len(fields) > 0 {
		for k, v := range fields[0] {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

// Info logs an info message using the global logger
func Info(msg string, fields ...map[string]interface{}) {
	l := Get()
	pc, file, line, _ := runtime.Caller(1)
	event := l.logger.Info().Str("caller", zerolog.CallerMarshalFunc(pc, file, line))
	if len(fields) > 0 {
		for k, v := range fields[0] {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

// Warn logs a warning message using the global logger
func Warn(msg string, fields ...map[string]interface{}) {
	l := Get()
	pc, file, line, _ := runtime.Caller(1)
	event := l.logger.Warn().Str("caller", zerolog.CallerMarshalFunc(pc, file, line))
	if len(fields) > 0 {
		for k, v := range fields[0] {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

// Error logs an error message using the global logger
func Error(msg string, err error, fields ...map[string]interface{}) {
	l := Get()
	pc, file, line, _ := runtime.Caller(1)
	event := l.logger.Error().Err(err).Str("caller", zerolog.CallerMarshalFunc(pc, file, line))
	if len(fields) > 0 {
		for k, v := range fields[0] {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

// Fatal logs a fatal message using the global logger and exits
func Fatal(msg string, err error, fields ...map[string]interface{}) {
	l := Get()
	pc, file, line, _ := runtime.Caller(1)
	event := l.logger.Fatal().Err(err).Str("caller", zerolog.CallerMarshalFunc(pc, file, line))
	if len(fields) > 0 {
		for k, v := range fields[0] {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

// WithContext returns a logger with additional context fields
func WithContext(fields map[string]interface{}) *Logger {
	return Get().WithContext(fields)
}
