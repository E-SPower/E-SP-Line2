package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log   *zap.Logger
	sugar *zap.SugaredLogger
	once  sync.Once
)

// Init initializes the logger with the specified level
func Init(level string) {
	once.Do(func() {
		var zapLevel zapcore.Level
		switch level {
		case "debug":
			zapLevel = zapcore.DebugLevel
		case "info":
			zapLevel = zapcore.InfoLevel
		case "warn":
			zapLevel = zapcore.WarnLevel
		case "error":
			zapLevel = zapcore.ErrorLevel
		default:
			zapLevel = zapcore.InfoLevel
		}

		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			zapLevel,
		)

		log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
		sugar = log.Sugar()
	})
}

// Debug logs a debug message
func Debug(msg string, keysAndValues ...interface{}) {
	sugar.Debugw(msg, keysAndValues...)
}

// Info logs an info message
func Info(msg string, keysAndValues ...interface{}) {
	sugar.Infow(msg, keysAndValues...)
}

// Infof logs a formatted info message
func Infof(template string, args ...interface{}) {
	sugar.Infof(template, args...)
}

// Warn logs a warning message
func Warn(msg string, keysAndValues ...interface{}) {
	sugar.Warnw(msg, keysAndValues...)
}

// Error logs an error message
func Error(msg string, keysAndValues ...interface{}) {
	sugar.Errorw(msg, keysAndValues...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, keysAndValues ...interface{}) {
	sugar.Fatalw(msg, keysAndValues...)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(template string, args ...interface{}) {
	sugar.Fatalf(template, args...)
}

// Sync flushes any buffered log entries
func Sync() error {
	return log.Sync()
}

// String creates a key-value pair for logging
func String(key, value string) zap.Field {
	return zap.String(key, value)
}

// Int creates a key-value pair for logging
func Int(key string, value int) zap.Field {
	return zap.Int(key, value)
}

// Bool creates a key-value pair for logging
func Bool(key string, value bool) zap.Field {
	return zap.Bool(key, value)
}

// Error creates a key-value pair for logging
func ErrorField(key string, value error) zap.Field {
	return zap.Error(value)
}
