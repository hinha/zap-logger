package zap_logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LoggerI interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	Panic(msg string, fields ...zap.Field)
	DPanic(msg string, fields ...zap.Field)
}

func New(config Config, options ...ZapCoreOption) *ZapLogger {
	l := &ZapLogger{
		Config: config,
	}

	for _, opt := range options {
		opt(l)
	}

	console := zapcore.NewCore(
		zapcore.NewConsoleEncoder(l.Config.EncoderConfig),
		zapcore.AddSync(getStdout(l.Config.Interval)),
		l.Config.Level)
	if l.Config.Encoding == "" || l.Config.Encoding == "console" || l.Config.Encoding == "all" {
		l.cores = append(l.cores, console)
	}

	core := zapcore.NewTee(l.cores...)
	l.Log = zap.New(core)
	return l
}

type ZapLogger struct {
	Log    *zap.Logger
	Config Config

	addCaller bool

	addStack zapcore.LevelEnabler

	callerSkip int

	cores  []zapcore.Core
	rotate *lumberjack.Logger
}

// Sync wrap sync
func (l *ZapLogger) Sync() {
	_ = l.Log.Sync()
	if l.rotate != nil {
		_ = l.rotate.Rotate()
	}
	_ = l.Log.Sync()
}

// Named  name logger
func (l *ZapLogger) Named(s string) *zap.Logger {
	return l.Log.Named(s)
}

func checkFields(fields ...zap.Field) []zap.Field {
	return fields
}

// Debug logs a message at level DebugMode on the ZapLogger.
func (l *ZapLogger) Debug(msg string, fields ...zap.Field) {
	l.Log.Debug(msg, checkFields(fields...)...)
}

// Info logs a message at level Info on the ZapLogger.
func (l *ZapLogger) Info(msg string, fields ...zap.Field) {
	l.Log.Info(msg, checkFields(fields...)...)
}

// Warn logs a message at level Warn on the ZapLogger.
func (l *ZapLogger) Warn(msg string, fields ...zap.Field) {
	l.Log.Warn(msg, checkFields(fields...)...)
}

// Error logs a message at level Error on the ZapLogger.
func (l *ZapLogger) Error(msg string, fields ...zap.Field) {
	l.Log.Error(msg, checkFields(fields...)...)
}

// Fatal logs a message at level Fatal on the ZapLogger.
func (l *ZapLogger) Fatal(msg string, fields ...zap.Field) {
	l.Log.Fatal(msg, checkFields(fields...)...)
}

// Panic logs a message at level Panic on the ZapLogger.
func (l *ZapLogger) Panic(msg string, fields ...zap.Field) {
	l.Log.Panic(msg, checkFields(fields...)...)
}

// DPanic logs a message at level DPanic on the ZapLogger.
func (l *ZapLogger) DPanic(msg string, fields ...zap.Field) {
	l.Log.DPanic(msg, checkFields(fields...)...)
}

// WithField return a log with an extra field.
func (l *ZapLogger) WithField(k string, v interface{}) *ZapLogger {
	l.Log.With(zap.Any(k, v))
	return l
}

// WithFields return a log with extra fields.
func (l *ZapLogger) WithFields(fields map[string]interface{}) *ZapLogger {
	clog := l
	i := 0
	for k, v := range fields {
		if i == 0 {
			clog = l.WithField(k, v)
		} else {
			clog = clog.WithField(k, v)
		}
		i++
	}
	return clog
}

func (l *ZapLogger) With(fields ...zap.Field) {
	l.Log.With(fields...)
}
