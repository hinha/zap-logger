package zap_logger

import (
	"context"
	"reflect"
	"strconv"
	"sync"

	"go.uber.org/zap/zapcore"
)

func newMemCtx() *inmemCtx {
	return &inmemCtx{
		mapCtx: make(map[interface{}]context.Context),
		mu:     &sync.RWMutex{},
		caller: make(map[interface{}]zapcore.EntryCaller),
	}
}

type inmemCtx struct {
	mapCtx map[any]context.Context
	mu     *sync.RWMutex
	caller map[any]zapcore.EntryCaller
}

// Set sets the context entries associated with key to the
// single element value. It replaces any existing values
// associated with key. And captures a stack trace of the specified depth
func (c *inmemCtx) Set(key any, ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mapCtx[key] = ctx

	stackDepth := stacktraceFirst
	stack := captureStacktrace(1, stackDepth)
	defer stack.Free()

	frame, _ := stack.Next()
	c.caller[key] = zapcore.EntryCaller{
		Defined:  frame.PC != 0,
		PC:       frame.PC,
		File:     frame.File,
		Line:     frame.Line,
		Function: frame.Function,
	}

}

// Get gets the first value associated with the given key. If
// there are no values associated with the key
func (c *inmemCtx) Get(key any) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mapCtx[key]
}

// Prune delete all key and value of map Context
func (c *inmemCtx) Prune() {
	for key, _ := range c.mapCtx {
		delete(c.mapCtx, key)
	}
}

// Len Gets length key of map Context
func (c *inmemCtx) Len() int { return len(c.mapCtx) }

func (c *inmemCtx) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	c.mu.RLock()
	for key, ctx := range c.mapCtx {
		keyTypeOf := reflect.TypeOf(key)
		keyValOf := reflect.ValueOf(key)

		ctxKey := keyTypeOf.String()
		switch keyTypeOf.Kind() {
		case reflect.Int:
			ctxKey += "." + strconv.Itoa(int(keyValOf.Int()))
		case reflect.String:
			ctxKey += "." + keyValOf.String()
		}

		ctxValue := ctx.Value(key)
		if ctxValue == nil {
			continue
		}

		if err := enc.AddObject(ctxKey, contextLog{value: ctxValue, caller: c.caller[key]}); err != nil {
			continue
		}
	}
	c.mu.RUnlock()
	return nil
}
