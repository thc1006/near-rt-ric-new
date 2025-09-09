package dashboard

import (
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger variable is now defined in types.go as LogrusLogger to avoid redeclaration

// InitStructuredLogging initializes the structured logging system
func InitStructuredLogging() {
	LogrusLogger = logrus.New()
	LogrusLogger.SetOutput(os.Stdout)
	LogrusLogger.SetLevel(logrus.InfoLevel)
	LogrusLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
}

// LogInfo logs an info level message with structured fields
func LogInfo(message string, fields logrus.Fields) {
	if LogrusLogger == nil {
		InitStructuredLogging()
	}
	LogrusLogger.WithFields(fields).Info(message)
}

// LogError logs an error level message with structured fields
func LogError(message string, fields logrus.Fields, err error) {
	if LogrusLogger == nil {
		InitStructuredLogging()
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	LogrusLogger.WithFields(fields).Error(message)
}

// LogWarning logs a warning level message with structured fields
func LogWarning(message string, fields logrus.Fields) {
	if LogrusLogger == nil {
		InitStructuredLogging()
	}
	LogrusLogger.WithFields(fields).Warn(message)
}

// LogDebug logs a debug level message with structured fields
func LogDebug(message string, fields logrus.Fields) {
	if LogrusLogger == nil {
		InitStructuredLogging()
	}
	LogrusLogger.WithFields(fields).Debug(message)
}