package zap_logger

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggerAtomicLevel(t *testing.T) {
	// Test that the dynamic level applies to all ancestors and descendants.
	dl := zap.NewAtomicLevel()

	withLogger(t, dl, nil, func(grandparent *Logger, _ *observer.ObservedLogs) {
		parent := grandparent.With(zap.Int("generation", 1))
		child := parent.With(zap.Int("generation", 2))

		tests := []struct {
			setLevel  zapcore.Level
			testLevel zapcore.Level
			enabled   bool
		}{
			{zap.DebugLevel, zap.DebugLevel, true},
			{zap.InfoLevel, zap.DebugLevel, false},
			{zap.WarnLevel, zap.PanicLevel, true},
		}

		for _, tt := range tests {
			dl.SetLevel(tt.setLevel)
			for _, logger := range []*Logger{grandparent, parent, child} {
				if tt.enabled {
					assert.NotNil(
						t,
						logger.Check(tt.testLevel, ""),
						"Expected level %s to be enabled after setting level %s.", tt.testLevel, tt.setLevel,
					)
				} else {
					assert.Nil(
						t,
						logger.Check(tt.testLevel, ""),
						"Expected level %s to be enabled after setting level %s.", tt.testLevel, tt.setLevel,
					)
				}
			}
		}
	})
}

func TestLoggerLevel(t *testing.T) {
	levels := []zapcore.Level{
		zap.DebugLevel,
		zap.InfoLevel,
		zap.WarnLevel,
		zap.ErrorLevel,
		zap.DPanicLevel,
		zap.PanicLevel,
		zap.FatalLevel,
	}

	for _, lvl := range levels {
		lvl := lvl
		t.Run(lvl.String(), func(t *testing.T) {
			t.Parallel()

			core, _ := observer.New(lvl)
			log := New(core, Config{})
			assert.Equal(t, lvl, log.Level())
		})
	}

	t.Run("Nop", func(t *testing.T) {
		assert.Equal(t, zapcore.InvalidLevel, NewNop().Level())
	})
}

func TestLoggerInitialFields(t *testing.T) {
	fieldOpts := opts(Fields(zap.Int("foo", 42), zap.String("bar", "baz")))
	withLogger(t, zap.DebugLevel, fieldOpts, func(logger *Logger, logs *observer.ObservedLogs) {
		logger.Info("")
		assert.Equal(
			t,
			observer.LoggedEntry{Context: []zap.Field{zap.Int("foo", 42), zap.String("bar", "baz")}},
			logs.AllUntimed()[0],
			"Unexpected output with initial fields set.",
		)
	})
}

func TestLoggerWith(t *testing.T) {
	fieldOpts := opts(Fields(zap.Int("foo", 42)))
	withLogger(t, zap.DebugLevel, fieldOpts, func(logger *Logger, logs *observer.ObservedLogs) {
		// Child loggers should have copy-on-write semantics, so two children
		// shouldn't stomp on each other's fields or affect the parent's fields.
		logger.With(zap.String("one", "two")).Info("")
		logger.With(zap.String("three", "four")).Info("")
		logger.Info("")

		assert.Equal(t, []observer.LoggedEntry{
			{Context: []zap.Field{zap.Int("foo", 42), zap.String("one", "two")}},
			{Context: []zap.Field{zap.Int("foo", 42), zap.String("three", "four")}},
			{Context: []zap.Field{zap.Int("foo", 42)}},
		}, logs.AllUntimed(), "Unexpected cross-talk between child loggers.")
	})
}

func TestLoggerContext(t *testing.T) {
	type ctxID string
	const (
		ContextID1 ctxID = "ContextID1"
		ContextID2 ctxID = "ContextID2"
	)

	ctx := context.TODO()
	ctx = context.WithValue(ctx, ContextID1, "c99c2ca0-37f1-11ed-a261-0242ac120002")
	fieldOpts := opts(AddContext(func(ctx context.Context, log *ZapLogger) {
		log.Ctx.Set(ContextID1, ctx)
		log.Ctx.Set(ContextID2, ctx)
	}))

	withLogger(t, zap.DebugLevel, fieldOpts, func(logger *Logger, logs *observer.ObservedLogs) {
		logger.InfoCtx(ctx, "")

		assert.Equal(t, []observer.LoggedEntry{
			{Context: []zap.Field{zap.Object("context", logger.Ctx.data())}},
		}, logs.AllUntimed(), "Unexpected context loggers.")
	})
}

func TestLoggerLogPanic(t *testing.T) {
	for _, tt := range []struct {
		do       func(*Logger)
		should   bool
		expected string
	}{
		{func(logger *Logger) { logger.Check(zap.PanicLevel, "foo").Write() }, true, "foo"},
		{func(logger *Logger) { logger.Log(zap.PanicLevel, "bar") }, true, "bar"},
		{func(logger *Logger) { logger.Panic("baz") }, true, "baz"},
	} {
		withLogger(t, zap.DebugLevel, nil, func(logger *Logger, logs *observer.ObservedLogs) {
			if tt.should {
				assert.Panics(t, func() { tt.do(logger) }, "Expected panic")
			} else {
				assert.NotPanics(t, func() { tt.do(logger) }, "Expected no panic")
			}

			output := logs.AllUntimed()
			assert.Equal(t, 1, len(output), "Unexpected number of logs.")
			assert.Equal(t, 0, len(output[0].Context), "Unexpected context on first log.")
			assert.Equal(
				t,
				zapcore.Entry{Message: tt.expected, Level: zap.PanicLevel},
				output[0].Entry,
				"Unexpected output from panic-level Log.",
			)
		})
	}
}

func TestLoggerNames(t *testing.T) {
	tests := []struct {
		names    []string
		expected string
	}{
		{nil, ""},
		{[]string{""}, ""},
		{[]string{"foo"}, "foo"},
		{[]string{"foo", ""}, "foo"},
		{[]string{"foo", "bar"}, "foo.bar"},
		{[]string{"foo.bar", "baz"}, "foo.bar.baz"},
		// Garbage in, garbage out.
		{[]string{"foo.", "bar"}, "foo..bar"},
		{[]string{"foo", ".bar"}, "foo..bar"},
		{[]string{"foo.", ".bar"}, "foo...bar"},
	}

	for _, tt := range tests {
		withLogger(t, zap.DebugLevel, nil, func(log *Logger, logs *observer.ObservedLogs) {
			for _, n := range tt.names {
				log = log.Named(n)
			}
			log.Info("")
			require.Equal(t, 1, logs.Len(), "Expected only one log entry to be written.")
			assert.Equal(t, tt.expected, logs.AllUntimed()[0].LoggerName, "Unexpected logger name.")
		})
	}
}

func TestLoggerAddCaller(t *testing.T) {
	tests := []struct {
		options []Option
		pat     string
	}{
		{opts(), `^undefined$`},
		{opts(WithCaller(false)), `^undefined$`},
		{opts(AddCaller()), `.+/logger_test.go:[\d]+$`},
		{opts(AddCaller(), WithCaller(false)), `^undefined$`},
		{opts(WithCaller(true)), `.+/logger_test.go:[\d]+$`},
		{opts(WithCaller(true), WithCaller(false)), `^undefined$`},
		{opts(AddCaller(), AddCallerSkip(1), AddCallerSkip(-1)), `.+/logger_test.go:[\d]+$`},
		{opts(AddCaller(), AddCallerSkip(1)), `.+/common_test.go:[\d]+$`},
		{opts(AddCaller(), AddCallerSkip(1), AddCallerSkip(3)), `.+/src/runtime/.*:[\d]+$`},
	}
	for _, tt := range tests {
		withLogger(t, zap.DebugLevel, tt.options, func(logger *Logger, logs *observer.ObservedLogs) {
			// Make sure that sugaring and desugaring resets caller skip properly.
			logger.Info("")
			output := logs.AllUntimed()
			assert.Equal(t, 1, len(output), "Unexpected number of logs written out.")
			assert.Regexp(
				t,
				tt.pat,
				output[0].Caller,
				"Expected to find package name and file name in output.",
			)
		})
	}
}
