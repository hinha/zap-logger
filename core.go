package zap_logger

import (
	"io"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/hinha/zap-logger/pkg/diode"
)

const (
	//intervalWrite   = time.Duration(5) * time.Minute
	bufferSize      = 1024 * 1024 * 128
	bufferSizeDebug = 1024
)

func newWriter(filename string, days, size, backups int, local bool, interval time.Duration) (io.Writer, *lumberjack.Logger) {
	lg := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    size,
		MaxAge:     days,
		MaxBackups: backups,
		LocalTime:  local,
	}

	d := diode.NewWriter(lg, bufferSize, interval, func(missed int) {
		// fmt.Printf("Dropped %d messages\n", missed)
	})
	return d, lg
}

func getStdout(interval time.Duration) io.Writer {
	w := diode.NewWriter(os.Stdout, bufferSize, interval, func(missed int) {
		// fmt.Printf("Dropped %d messages\n", missed)
	})
	return w
}

func jsonEncoder(w io.Writer, debug bool, cfg zapcore.EncoderConfig, lvl zap.AtomicLevel) zapcore.Core {
	if debug {
		return zapcore.NewCore(zapcore.NewJSONEncoder(cfg), &zapcore.BufferedWriteSyncer{
			WS:   zapcore.AddSync(w),
			Size: bufferSizeDebug,
		}, lvl)
	}
	return zapcore.NewCore(zapcore.NewJSONEncoder(cfg), &zapcore.BufferedWriteSyncer{
		WS:   zapcore.AddSync(w),
		Size: bufferSize,
	}, lvl)
}

func consoleEncoder(w io.Writer, cfg zapcore.EncoderConfig, lvl zap.AtomicLevel) zapcore.Core {
	return zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		zapcore.AddSync(w),
		lvl,
	)
}
