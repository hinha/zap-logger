package zap_logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
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
		Config:      config,
		errorOutput: zapcore.Lock(os.Stderr),
		addStack:    zapcore.FatalLevel + 1,
		clock:       zapcore.DefaultClock,
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

	l.core = zapcore.NewTee(l.cores...)
	l.Log = zap.New(l.core)
	return l
}

type ZapLogger struct {
	Log    *zap.Logger
	Config Config

	core zapcore.Core

	development bool
	addCaller   bool
	onFatal     zapcore.CheckWriteHook // default is WriteThenFatal

	name        string
	errorOutput zapcore.WriteSyncer

	addStack zapcore.LevelEnabler

	callerSkip int

	clock zapcore.Clock

	cores  []zapcore.Core
	rotate *lumberjack.Logger
}

// Sync wrap sync
func (log *ZapLogger) Sync() {
	_ = log.Log.Sync()
	if log.rotate != nil {
		_ = log.rotate.Rotate()
	}
	_ = log.Log.Sync()
}

// Named  name logger
func (log *ZapLogger) Named(s string) *zap.Logger {
	return log.Log.Named(s)
}

func (log *ZapLogger) With(fields ...zap.Field) *ZapLogger {
	if len(fields) == 0 {
		return log
	}
	l := log.clone()
	l.core = l.core.With(fields)
	return l
}

func checkFields(fields ...zap.Field) []zap.Field {
	return fields
}

// Debug logs a message at level DebugMode on the ZapLogger.
func (log *ZapLogger) Debug(msg string, fields ...zap.Field) {
	if ce := log.check(zap.DebugLevel, msg); ce != nil {
		ce.Write(checkFields(fields...)...)
	}
}

// Info logs a message at level Info on the ZapLogger.
func (log *ZapLogger) Info(msg string, fields ...zap.Field) {
	if ce := log.check(zap.InfoLevel, msg); ce != nil {
		ce.Write(checkFields(fields...)...)
	}
}

// Warn logs a message at level Warn on the ZapLogger.
func (log *ZapLogger) Warn(msg string, fields ...zap.Field) {
	if ce := log.check(zap.WarnLevel, msg); ce != nil {
		ce.Write(checkFields(fields...)...)
	}
}

// Error logs a message at level Error on the ZapLogger.
func (log *ZapLogger) Error(msg string, fields ...zap.Field) {
	if ce := log.check(zap.ErrorLevel, msg); ce != nil {
		ce.Write(checkFields(fields...)...)
	}
}

// Fatal logs a message at level Fatal on the ZapLogger.
func (log *ZapLogger) Fatal(msg string, fields ...zap.Field) {
	if ce := log.check(zap.FatalLevel, msg); ce != nil {
		ce.Write(checkFields(fields...)...)
	}
}

// Panic logs a message at level Panic on the ZapLogger.
func (log *ZapLogger) Panic(msg string, fields ...zap.Field) {
	if ce := log.check(zap.PanicLevel, msg); ce != nil {
		ce.Write(checkFields(fields...)...)
	}
}

// DPanic logs a message at level DPanic on the ZapLogger.
func (log *ZapLogger) DPanic(msg string, fields ...zap.Field) {
	if ce := log.check(zap.DPanicLevel, msg); ce != nil {
		ce.Write(checkFields(fields...)...)
	}

}

// WithField return a log with an extra field.
func (log *ZapLogger) WithField(k string, v interface{}) *ZapLogger {
	return log.With(zap.Any(k, v))
}

// WithFields return a log with extra fields.
func (log *ZapLogger) WithFields(fields map[string]interface{}) *ZapLogger {
	clog := log
	i := 0
	for k, v := range fields {
		if i == 0 {
			clog = log.WithField(k, v)
		} else {
			clog = clog.WithField(k, v)
		}
		i++
	}
	return clog
}
