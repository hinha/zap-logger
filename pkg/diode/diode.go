// Package diode provides a thread-safe, lock-free, non-blocking io.Writer
// wrapper.
package diode

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/hinha/zap-logger/pkg/diode/internal"
)

var bufPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 500)
	},
}

type Alerter func(missed int)

type diodeFetcher interface {
	internal.Diode
	Next() internal.GenericDataType
}

// Writer is a io.Writer wrapper that uses a diode to make Write lock-free,
// non-blocking and thread safe.
type Writer struct {
	w    io.Writer
	d    diodeFetcher
	c    context.CancelFunc
	done chan struct{}
}

// NewWriter creates a writer wrapping w with a many-to-one diode in order to
// never block log producers and drop events if the writer can't keep up with
// the flow of data.
//
// If pollInterval is greater than 0, a poller is used otherwise a waiter is
// used.
//
// See code.cloudfoundry.org/go-diodes for more info on diode.
func NewWriter(w io.Writer, size int, pollInterval time.Duration, f Alerter) Writer {
	ctx, cancel := context.WithCancel(context.Background())
	dw := Writer{
		w:    w,
		c:    cancel,
		done: make(chan struct{}),
	}
	if f == nil {
		f = func(int) {}
	}
	d := internal.NewManyToOne(size, internal.AlertFunc(f))
	if pollInterval > 0 {
		dw.d = internal.NewPoller(d,
			internal.WithPollingInterval(pollInterval),
			internal.WithPollingContext(ctx))
	} else {
		dw.d = internal.NewWaiter(d, internal.WithWaiterContext(ctx))
	}
	go dw.poll()
	return dw
}

func (dw Writer) Write(p []byte) (n int, err error) {
	// p is pooled in zap, so we can't hold it passed this call, hence the
	// copy.
	p = append(bufPool.Get().([]byte), p...)
	dw.d.Set(internal.GenericDataType(&p))
	return len(p), nil
}

// Close releases the diode poller and call Close on the wrapped writer if
// io.Closer is implemented.
func (dw Writer) Close() error {
	dw.c()
	<-dw.done
	if w, ok := dw.w.(io.Closer); ok {
		return w.Close()
	}
	return nil
}

func (dw Writer) poll() {
	defer close(dw.done)
	for {
		d := dw.d.Next()
		if d == nil {
			return
		}
		p := *(*[]byte)(d)
		_, _ = dw.w.Write(p)

		// Proper usage of a sync.Pool requires each entry to have approximately
		// the same memory cost. To obtain this property when the stored type
		// contains a variably-sized buffer, we add a hard limit on the maximum buffer
		// to place back in the pool.
		//
		// See https://golang.org/issue/23199
		const maxSize = 1 << 16 // 64KiB
		if cap(p) <= maxSize {
			bufPool.Put(p[:0])
		}
	}
}
