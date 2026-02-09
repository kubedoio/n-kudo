package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var log *logrus.Logger

// Init initializes the logger with the specified format and level
func Init(format, level string) {
	log = logrus.New()
	log.SetOutput(os.Stdout)

	if format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05Z",
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05Z",
		})
	}

	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	log.SetLevel(lvl)
}

// InitWithComponent initializes the logger with a component field
func InitWithComponent(format, level, component string) {
	Init(format, level)
	if component != "" {
		log = log.WithField("component", component).Logger
	}
}

// WithFields returns a new entry with the specified fields
func WithFields(fields map[string]interface{}) *logrus.Entry {
	if log == nil {
		Init("text", "info")
	}
	return log.WithFields(fields)
}

// WithComponent returns a new entry with the component field set
func WithComponent(component string) *logrus.Entry {
	if log == nil {
		Init("text", "info")
	}
	return log.WithField("component", component)
}

// Info logs an info message
func Info(msg string) {
	if log == nil {
		Init("text", "info")
	}
	log.Info(msg)
}

// Infof logs a formatted info message
func Infof(format string, args ...interface{}) {
	if log == nil {
		Init("text", "info")
	}
	log.Infof(format, args...)
}

// Error logs an error message
func Error(msg string) {
	if log == nil {
		Init("text", "info")
	}
	log.Error(msg)
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
	if log == nil {
		Init("text", "info")
	}
	log.Errorf(format, args...)
}

// Debug logs a debug message
func Debug(msg string) {
	if log == nil {
		Init("text", "info")
	}
	log.Debug(msg)
}

// Debugf logs a formatted debug message
func Debugf(format string, args ...interface{}) {
	if log == nil {
		Init("text", "info")
	}
	log.Debugf(format, args...)
}

// Warn logs a warning message
func Warn(msg string) {
	if log == nil {
		Init("text", "info")
	}
	log.Warn(msg)
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...interface{}) {
	if log == nil {
		Init("text", "info")
	}
	log.Warnf(format, args...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string) {
	if log == nil {
		Init("text", "info")
	}
	log.Fatal(msg)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(format string, args ...interface{}) {
	if log == nil {
		Init("text", "info")
	}
	log.Fatalf(format, args...)
}

// GetLogger returns the underlying logger instance
func GetLogger() *logrus.Logger {
	if log == nil {
		Init("text", "info")
	}
	return log
}
