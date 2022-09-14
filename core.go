package zap_logger

import (
	"io"
	"os"
	"time"

	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/hinha/zap-logger/pkg/diode"
)

const (
	//intervalWrite   = time.Duration(5) * time.Minute
	bufferSize      = 1024 * 1024 * 128
	bufferSizeDebug = 1024
)

// ZapCoreOption options of core
type ZapCoreOption func(*ZapLogger)

func WithWriter(filename string, days int) ZapCoreOption {
	return func(log *ZapLogger) {
		if log.Config.Encoding == "" || log.Config.Encoding == "console" {
			return
		}

		writer := newWriter(filename,
			days, log.Config.MaxSize, log.Config.MaxBackups, log.Config.LocalTime, log.Config.Interval)
		log.rotate = writer.Logger

		if log.Config.Development {
			w := zapcore.NewCore(zapcore.NewJSONEncoder(log.Config.EncoderConfig), &zapcore.BufferedWriteSyncer{
				WS:   zapcore.AddSync(writer.Writer),
				Size: bufferSizeDebug,
			}, log.Config.Level)
			log.cores = append(log.cores, w)

			return
		}
		w := zapcore.NewCore(zapcore.NewJSONEncoder(log.Config.EncoderConfig), &zapcore.BufferedWriteSyncer{
			WS:   zapcore.AddSync(writer.Writer),
			Size: bufferSize,
		}, log.Config.Level)
		log.cores = append(log.cores, w)
	}
}

type writer struct {
	io.Writer
	*lumberjack.Logger
}

func newWriter(filename string, days, size, backups int, local bool, interval time.Duration) writer {
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
	return writer{d, lg}
}

func getStdout(interval time.Duration) io.Writer {
	w := diode.NewWriter(os.Stdout, bufferSize, interval, func(missed int) {
		// fmt.Printf("Dropped %d messages\n", missed)
	})
	return w
}
