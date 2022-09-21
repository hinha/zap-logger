package zap_logger

import (
	"go.uber.org/zap/zapcore"
	"reflect"
	"unsafe"
)

type any = interface{}

func marshalInterface(value any) any {
	valTypeof := reflect.TypeOf(value)
	switch valTypeof.Kind() {
	case reflect.Struct:
		return value
	case reflect.Ptr:
		return getUnsafePtr(reflect.ValueOf(value).Elem())
	default:
	}
	return nil
}

func getUnsafePtr(field reflect.Value) interface{} {
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}

type contextFieldValue struct {
	value  interface{}
	caller zapcore.EntryCaller
}

func (c contextFieldValue) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("caller", c.caller.TrimmedPath())
	switch ctxValue := c.value.(type) {
	case string:
		enc.AddString("value", ctxValue)
	case int8:
		enc.AddInt8("value", ctxValue)
	case int:
		enc.AddInt("value", ctxValue)
	case int32:
		enc.AddInt32("value", ctxValue)
	case int64:
		enc.AddInt64("value", ctxValue)
	case bool:
		enc.AddBool("value", ctxValue)
	case interface{}:
		if ctxValue != nil {
			if err := enc.AddReflected("value", marshalInterface(ctxValue)); err != nil {
				return err
			}
		}
	}
	return nil
}
