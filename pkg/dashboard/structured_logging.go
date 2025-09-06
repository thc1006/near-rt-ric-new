package dashboard

import (
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger is the global structured logger instance
var Logger *logrus.Logger

// InitStructuredLogging initializes the structured logging system
func InitStructuredLogging() {
	Logger = logrus.New()
	Logger.SetOutput(os.Stdout)
	Logger.SetLevel(logrus.InfoLevel)
	Logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
}

// LogInfo logs an info level message with structured fields
func LogInfo(message string, fields logrus.Fields) {
	if Logger == nil {
		InitStructuredLogging()
	}
	Logger.WithFields(fields).Info(message)
}

// LogError logs an error level message with structured fields
func LogError(message string, fields logrus.Fields, err error) {
	if Logger == nil {
		InitStructuredLogging()
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	Logger.WithFields(fields).Error(message)
}

// LogWarning logs a warning level message with structured fields
func LogWarning(message string, fields logrus.Fields) {
	if Logger == nil {
		InitStructuredLogging()
	}
	Logger.WithFields(fields).Warn(message)
}

// LogDebug logs a debug level message with structured fields
func LogDebug(message string, fields logrus.Fields) {
	if Logger == nil {
		InitStructuredLogging()
	}
	Logger.WithFields(fields).Debug(message)
}