package zap_logger

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

type user struct {
	Name      string
	Email     string
	CreatedAt time.Time
}

func (u *user) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("name", u.Name)
	enc.AddString("email", u.Email)
	enc.AddInt64("created_at", u.CreatedAt.UnixNano())
	return nil
}

var _jane = &user{
	Name:      "Jane Doe",
	Email:     "jane@test.com",
	CreatedAt: time.Date(1980, 1, 1, 12, 0, 0, 0, time.UTC),
}

func withBenchedLogger(b *testing.B, f func(*Logger)) {
	logger := New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(NewProductionConfig().EncoderConfig),
			&Discarder{},
			zap.DebugLevel,
		), NewProductionConfig())
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			f(logger)
		}
	})
}

func BenchmarkNoContext(b *testing.B) {
	withBenchedLogger(b, func(log *Logger) {
		log.Info("No context.")
	})
}

func BenchmarkBoolField(b *testing.B) {
	withBenchedLogger(b, func(log *Logger) {
		log.Info("Boolean.", zap.Bool("foo", true))
	})
}

func BenchmarkByteStringField(b *testing.B) {
	val := []byte("bar")
	withBenchedLogger(b, func(log *Logger) {
		log.Info("ByteString.", zap.ByteString("foo", val))
	})
}

func BenchmarkFloat64Field(b *testing.B) {
	withBenchedLogger(b, func(log *Logger) {
		log.Info("Floating point.", zap.Float64("foo", 3.14))
	})
}

func BenchmarkIntField(b *testing.B) {
	withBenchedLogger(b, func(log *Logger) {
		log.Info("Integer.", zap.Int("foo", 42))
	})
}

func BenchmarkInt64Field(b *testing.B) {
	withBenchedLogger(b, func(log *Logger) {
		log.Info("64-bit integer.", zap.Int64("foo", 42))
	})
}

func BenchmarkStringField(b *testing.B) {
	withBenchedLogger(b, func(log *Logger) {
		log.Info("Strings.", zap.String("foo", "bar"))
	})
}

func BenchmarkStringerField(b *testing.B) {
	withBenchedLogger(b, func(log *Logger) {
		log.Info("Level.", zap.Stringer("foo", zap.InfoLevel))
	})
}

func BenchmarkTimeField(b *testing.B) {
	t := time.Unix(0, 0)
	withBenchedLogger(b, func(log *Logger) {
		log.Info("Time.", zap.Time("foo", t))
	})
}

func BenchmarkDurationField(b *testing.B) {
	withBenchedLogger(b, func(log *Logger) {
		log.Info("Duration", zap.Duration("foo", time.Second))
	})
}

func BenchmarkErrorField(b *testing.B) {
	err := errors.New("egad")
	withBenchedLogger(b, func(log *Logger) {
		log.Info("Error.", zap.Error(err))
	})
}

func BenchmarkErrorsField(b *testing.B) {
	errs := []error{
		errors.New("egad"),
		errors.New("oh no"),
		errors.New("dear me"),
		errors.New("such fail"),
	}
	withBenchedLogger(b, func(log *Logger) {
		log.Info("Errors.", zap.Errors("errors", errs))
	})
}

func BenchmarkStackField(b *testing.B) {
	withBenchedLogger(b, func(log *Logger) {
		log.Info("Error.", zap.Stack("stacktrace"))
	})
}

func BenchmarkObjectField(b *testing.B) {
	withBenchedLogger(b, func(log *Logger) {
		log.Info("Arbitrary ObjectMarshaler.", zap.Object("user", _jane))
	})
}

func BenchmarkReflectField(b *testing.B) {
	withBenchedLogger(b, func(log *Logger) {
		log.Info("Reflection-based serialization.", zap.Reflect("user", _jane))
	})
}

func BenchmarkAddCallerHook(b *testing.B) {
	logger := New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(NewProductionConfig().EncoderConfig),
			&Discarder{},
			zap.InfoLevel,
		),
		NewProductionConfig(),
		AddCaller(),
	)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("Caller.")
		}
	})
}

func BenchmarkAddCallerAndStacktrace(b *testing.B) {
	logger := New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(NewProductionConfig().EncoderConfig),
			&Discarder{},
			zap.InfoLevel,
		),
		NewProductionConfig(),
		AddCaller(),
		AddStacktrace(zap.WarnLevel),
	)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Warn("Caller and stacktrace.")
		}
	})
}

func BenchmarkAddContextAndAddCaller(b *testing.B) {
	type ctxID string
	const ContextID1 ctxID = "ContextID1"

	ctx := context.TODO()
	ctx = context.WithValue(ctx, ContextID1, "c99c2ca0-37f1-11ed-a261-0242ac120002")

	logger := New(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(NewProductionConfig().EncoderConfig),
			&Discarder{},
			zap.DebugLevel,
		),
		NewProductionConfig(),
		AddCaller(),
		AddContext(func(ctx context.Context, log *ZapLogger) {
			log.Ctx.Set(ContextID1, ctx)
		}),
	)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.InfoCtx(ctx, "Caller, context.")
		}
	})
}

func Benchmark10AddContextAndAddCaller(b *testing.B) {
	type ctxID string
	const (
		ContextID1  ctxID = "ContextID1"
		ContextID2  ctxID = "ContextID2"
		ContextID3  ctxID = "ContextID3"
		ContextID4  ctxID = "ContextID4"
		ContextID5  ctxID = "ContextID5"
		ContextID6  ctxID = "ContextID6"
		ContextID7  ctxID = "ContextID7"
		ContextID8  ctxID = "ContextID8"
		ContextID9  ctxID = "ContextID9"
		ContextID10 ctxID = "ContextID10"
	)

	ctx := context.TODO()
	ctx = context.WithValue(ctx, ContextID1, "c99c2ca0-37f1-11ed-a261-0242ac120002")
	ctx = context.WithValue(ctx, ContextID2, "3ea3239b-b3d1-4851-bdc8-8e983eab94d6")
	ctx = context.WithValue(ctx, ContextID3, "3ea3239b-b3d1-4851-bdc8-8e983eab94d6")
	ctx = context.WithValue(ctx, ContextID4, "cd81445a-04a9-41b3-bb27-36e9ba773353")
	ctx = context.WithValue(ctx, ContextID5, "3ea3239b-b3d1-4851-bdc8-8e983eab94d6")
	ctx = context.WithValue(ctx, ContextID6, "69053826-242a-4de7-b15f-f5eabe05930e")
	ctx = context.WithValue(ctx, ContextID7, "100fec28-3df5-4de7-9e48-9dbf54b00184")
	ctx = context.WithValue(ctx, ContextID8, "0ffc4918-d91a-43f2-a1b6-e23ce8338652")
	ctx = context.WithValue(ctx, ContextID9, "100fec28-3df5-4de7-9e48-9dbf54b0010a")
	ctx = context.WithValue(ctx, ContextID10, "3ea3239b-b3d1-4851-bdc8-8e983eab94d6")

	logger := New(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(NewProductionConfig().EncoderConfig),
			&Discarder{},
			zap.DebugLevel,
		),
		NewProductionConfig(),
		AddCaller(),
		AddContext(func(ctx context.Context, log *ZapLogger) {
			log.Ctx.Set(ContextID1, ctx)
			log.Ctx.Set(ContextID2, ctx)
			log.Ctx.Set(ContextID3, ctx)
			log.Ctx.Set(ContextID4, ctx)
			log.Ctx.Set(ContextID5, ctx)
			log.Ctx.Set(ContextID6, ctx)
			log.Ctx.Set(ContextID7, ctx)
			log.Ctx.Set(ContextID8, ctx)
			log.Ctx.Set(ContextID9, ctx)
			log.Ctx.Set(ContextID10, ctx)
		}),
	)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.InfoCtx(ctx, "Caller, context.")
		}
	})
}

func Benchmark10Fields(b *testing.B) {
	withBenchedLogger(b, func(log *Logger) {
		log.Info("Ten fields, passed at the log site.",
			zap.Int("one", 1),
			zap.Int("two", 2),
			zap.Int("three", 3),
			zap.Int("four", 4),
			zap.Int("five", 5),
			zap.Int("six", 6),
			zap.Int("seven", 7),
			zap.Int("eight", 8),
			zap.Int("nine", 9),
			zap.Int("ten", 10),
		)
	})
}

func Benchmark100Fields(b *testing.B) {
	const batchSize = 50
	logger := New(zapcore.NewCore(
		zapcore.NewJSONEncoder(NewProductionConfig().EncoderConfig),
		&Discarder{},
		zap.DebugLevel,
	), NewProductionConfig())

	// Don't include allocating these helper slices in the benchmark. Since
	// access to them isn't synchronized, we can't run the benchmark in
	// parallel.
	first := make([]zap.Field, batchSize)
	second := make([]zap.Field, batchSize)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for i := 0; i < batchSize; i++ {
			// We're duplicating keys, but that doesn't affect performance.
			first[i] = zap.Int("foo", i)
			second[i] = zap.Int("foo", i+batchSize)
		}
		logger.With(first...).Info("Child loggers with lots of context.", second...)
	}
}
