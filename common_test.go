package zap_logger

import (
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"sync"
	"testing"
)

func opts(opts ...Option) []Option {
	return opts
}

// Here specifically to introduce an easily-identifiable filename for testing
// stack traces and caller skips.
func withLogger(t testing.TB, e zapcore.LevelEnabler, opts []Option, f func(*Logger, *observer.ObservedLogs)) {
	fac, logs := observer.New(e)
	log := New(fac, Config{}, opts...)
	f(log, logs)
}

func runConcurrently(goroutines, iterations int, wg *sync.WaitGroup, f func()) {
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				f()
			}
		}()
	}
}
