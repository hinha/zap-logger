package zap_logger

import (
	"io"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// A ZapLogger provides fast, leveled, structured logging. All methods are safe
// for concurrent use.
type ZapLogger struct {
	//Log    *zap.Logger
	config Config

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

type LoggerI interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	Panic(msg string, fields ...zap.Field)
	DPanic(msg string, fields ...zap.Field)
}

// New constructs a new ZapLogger from the provided zapcore.Core and Options. If
// the passed zapcore.Core is nil, it falls back to using a no-op
// implementation.
func New(core zapcore.Core, config Config, opts ...Option) *ZapLogger {
	if core == nil {
		return NewNop()
	}

	log := &ZapLogger{
		config:      config,
		core:        core,
		errorOutput: zapcore.Lock(os.Stderr),
		addStack:    zapcore.FatalLevel + 1,
		clock:       zapcore.DefaultClock,
	}

	return log.WithOptions(opts...)
}

// NewNop returns a no-op Logger. It never writes out logs or internal errors,
// and it never runs user-defined hooks.
//
// Using WithOptions to replace the Core or error output of a no-op Logger can
// re-enable logging.
func NewNop() *Logger {
	return &Logger{
		core:        zapcore.NewNopCore(),
		errorOutput: zapcore.AddSync(io.Discard),
		addStack:    zapcore.FatalLevel + 1,
		clock:       zapcore.DefaultClock,
	}
}

func NewLogger(config Config, opts ...Option) *ZapLogger {
	core := make([]zapcore.Core, 0)

	cslEncoder := consoleEncoder(getStdout(config.Interval), config.EncoderConfig, config.Level)
	if config.Encoding == "all" {
		logfile, lb := config.writer()
		opts = append(opts, addRotate(lb))
		fileEncoder := jsonEncoder(logfile, config.Development, config.EncoderConfig, config.Level)
		core = append(core, fileEncoder, cslEncoder)
	} else if config.Encoding == "json" {
		logfile, lb := config.writer()
		opts = append(opts, addRotate(lb))
		fileEncoder := jsonEncoder(logfile, config.Development, config.EncoderConfig, config.Level)
		core = append(core, fileEncoder)
	} else {
		core = append(core, cslEncoder)
	}

	return New(zapcore.NewTee(core...), config, opts...)
}

// Sync wrap sync
func (log *ZapLogger) Sync() error {
	err := log.core.Sync()
	if err != nil {
		return err
	}

	if log.rotate != nil {
		return log.rotate.Rotate()
	}
	return log.core.Sync()
}

// Named  name logger
func (log *ZapLogger) Named(s string) *ZapLogger {
	if s == "" {
		return log
	}
	l := log.clone()
	if log.name == "" {
		l.name = s
	} else {
		l.name = strings.Join([]string{l.name, s}, ".")
	}
	return l
}

func (log *ZapLogger) With(fields ...zap.Field) *ZapLogger {
	if len(fields) == 0 {
		return log
	}
	l := log.clone()
	l.core = l.core.With(fields)
	return l
}

// Level reports the minimum enabled level for this logger.
//
// For NopLoggers, this is [zapcore.InvalidLevel].
func (log *Logger) Level() zapcore.Level {
	return zapcore.LevelOf(log.core)
}

// Check returns a CheckedEntry if logging a message at the specified level
// is enabled. It's a completely optional optimization; in high-performance
// applications, Check can help avoid allocating a slice to hold fields.
func (log *Logger) Check(lvl zapcore.Level, msg string) *zapcore.CheckedEntry {
	return log.check(lvl, msg)
}

// Log logs a message at the specified level. The message includes any fields
// passed at the log site, as well as any fields accumulated on the logger.
func (log *Logger) Log(lvl zapcore.Level, msg string, fields ...zap.Field) {
	if ce := log.check(lvl, msg); ce != nil {
		ce.Write(fields...)
	}
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
