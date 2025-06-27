package botcore

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a new Zap logger with appropriate configuration
func NewLogger(service string) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.StacktraceKey = ""
	config.EncoderConfig.MessageKey = "message"
	config.EncoderConfig.LevelKey = "level"
	config.EncoderConfig.CallerKey = "caller"

	// Add service name to all log entries
	config.InitialFields = map[string]interface{}{
		"service": service,
	}

	return config.Build()
}

// NewDevelopmentLogger creates a development logger with more verbose output
func NewDevelopmentLogger(service string) (*zap.Logger, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Add service name to all log entries
	config.InitialFields = map[string]interface{}{
		"service": service,
	}

	return config.Build()
}
