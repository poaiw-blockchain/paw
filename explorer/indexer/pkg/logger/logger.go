package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger provides structured logging with consistent fields.
type Logger struct {
	base zerolog.Logger
}

// NewLogger creates a logger with component metadata.
func NewLogger(component string) *Logger {
	l := zerolog.New(os.Stdout).With().
		Timestamp().
		Str("component", component).
		Logger().
		Level(zerolog.InfoLevel)
	zerolog.DurationFieldUnit = time.Millisecond
	return &Logger{base: l}
}

// Info logs informational messages with optional key/value pairs.
func (l *Logger) Info(msg string, keyvals ...interface{}) {
	l.base.Info().Fields(kvToMap(keyvals...)).Msg(msg)
}

// Warn logs warning messages with optional key/value pairs.
func (l *Logger) Warn(msg string, keyvals ...interface{}) {
	l.base.Warn().Fields(kvToMap(keyvals...)).Msg(msg)
}

// Error logs error messages with optional key/value pairs.
func (l *Logger) Error(msg string, keyvals ...interface{}) {
	l.base.Error().Fields(kvToMap(keyvals...)).Msg(msg)
}

// kvToMap converts a flat list of key/value pairs into a map for zerolog.
func kvToMap(kv ...interface{}) map[string]interface{} {
	fields := make(map[string]interface{})
	for i := 0; i < len(kv)-1; i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			continue
		}
		fields[key] = kv[i+1]
	}
	return fields
}
