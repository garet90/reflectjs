//go:build js && wasm

package reflectjs

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)
import "syscall/js"

var (
	valueGlobal           = js.Global()
	arrayConstructor      = valueGlobal.Get("Array")
	objectConstructor     = valueGlobal.Get("Object")
	uint8ArrayConstructor = valueGlobal.Get("Uint8Array")
)

var (
	reflectValue = reflect.TypeOf(reflect.Value{})
	mapStringAny = reflect.TypeOf(map[string]any{})
	byteSlice    = reflect.TypeOf([]byte{})
)

func tryJsValue(v reflect.Value) js.Value {
	if v.Type() == reflectValue {
		return tryJsValue(v.Interface().(reflect.Value))
	}

	switch v.Kind() {
	case reflect.Bool:
		return js.ValueOf(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return js.ValueOf(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return js.ValueOf(v.Uint())
	case reflect.Float32, reflect.Float64:
		return js.ValueOf(v.Float())
	//case reflect.Complex64:
	//case reflect.Complex128:
	case reflect.Array:
		vLen := v.Len()
		jv := arrayConstructor.New(vLen)
		for i := 0; i < vLen; i++ {
			jv.SetIndex(i, tryJsValue(v.Index(i)))
		}

		return jv
	// case reflect.Chan:
	case reflect.Func:
		return js.ValueOf(funcOf(v))
	case reflect.Interface:
		return tryJsValue(v.Elem())
	case reflect.Map:
		jv := objectConstructor.New()
		keys := v.MapKeys()
		for _, key := range keys {
			if key.Kind() != reflect.String {
				panic("map keys must be strings")
			}
			jv.Set(key.String(), tryJsValue(v.MapIndex(key)))
		}

		return jv
	case reflect.Pointer:
		if v.IsNil() {
			return js.Null()
		}

		return tryJsValue(v.Elem())
	case reflect.Slice:
		vLen := v.Len()
		if v.Type() == byteSlice {
			// fast path, use js.CopyBytesToJS
			uint8Array := uint8ArrayConstructor.New(vLen)
			js.CopyBytesToJS(uint8Array, v.Interface().([]byte))
			jv := arrayConstructor.Call("from", uint8Array)

			return jv
		}
		jv := arrayConstructor.New(vLen)
		for i := 0; i < vLen; i++ {
			jv.SetIndex(i, tryJsValue(v.Index(i)))
		}

		return jv
	case reflect.String:
		return js.ValueOf(v.String())
	case reflect.Struct:
		jv := objectConstructor.New()
		numField := v.NumField()

		vty := v.Type()

		for i := 0; i < numField; i++ {
			field := vty.Field(i)
			if !field.IsExported() {
				continue
			}
			name := field.Name
			tag, ok := field.Tag.Lookup("json")
			if ok {
				name, _, _ = strings.Cut(tag, ",")
			}

			jv.Set(name, tryJsValue(v.Field(i)))
		}

		return jv
	case reflect.UnsafePointer:
		return js.ValueOf(uintptr(v.UnsafePointer()))
	default:
		panic("invalid reflect type")
	}
}

func tryJsValues(v []reflect.Value) js.Value {
	if len(v) == 0 {
		return js.Undefined()
	}
	if len(v) == 1 {
		return tryJsValue(v[0])
	}

	// make array and return array of results
	jv := arrayConstructor.New(len(v))
	for i := 0; i < len(v); i++ {
		jv.SetIndex(i, tryJsValue(v[i]))
	}
	return jv
}

func tryJsAny(v js.Value) reflect.Value {
	switch v.Type() {
	case js.TypeUndefined, js.TypeNull:
		return reflect.ValueOf(nil)
	case js.TypeBoolean:
		return reflect.ValueOf(v.Bool())
	case js.TypeNumber:
		return reflect.ValueOf(v.Float())
	case js.TypeString:
		return reflect.ValueOf(v.String())
	case js.TypeSymbol:
		return reflect.ValueOf(v)
	case js.TypeObject:
		return tryReflectValue(mapStringAny, v)
	case js.TypeFunction:
		return reflect.ValueOf(func(args ...any) any {
			return tryJsAny(v.Invoke(args))
		})
	default:
		panic("unknown js type")
	}
}

func tryReflectValue(ty reflect.Type, v js.Value) reflect.Value {
	switch ty.Kind() {
	case reflect.Bool:
		return reflect.ValueOf(v.Bool())
	case reflect.Int:
		return reflect.ValueOf(v.Int())
	case reflect.Int8:
		return reflect.ValueOf(int8(v.Int()))
	case reflect.Int16:
		return reflect.ValueOf(int16(v.Int()))
	case reflect.Int32:
		return reflect.ValueOf(int32(v.Int()))
	case reflect.Int64:
		return reflect.ValueOf(int64(v.Int()))
	case reflect.Uint:
		return reflect.ValueOf(uint(v.Int()))
	case reflect.Uint8:
		return reflect.ValueOf(uint8(v.Int()))
	case reflect.Uint16:
		return reflect.ValueOf(uint16(v.Int()))
	case reflect.Uint32:
		return reflect.ValueOf(uint32(v.Int()))
	case reflect.Uint64:
		return reflect.ValueOf(uint64(v.Int()))
	case reflect.Uintptr:
		return reflect.ValueOf(uintptr(v.Int()))
	case reflect.Float32:
		return reflect.ValueOf(float32(v.Float()))
	case reflect.Float64:
		return reflect.ValueOf(v.Float())
	//case reflect.Complex64:
	//case reflect.Complex128:
	case reflect.Array:
		rvp := reflect.New(ty)
		rv := rvp.Elem()

		tyLen := ty.Len()
		elem := ty.Elem()
		for i := 0; i < tyLen; i++ {
			rv.Index(i).Set(tryReflectValue(elem, v.Index(i)))
		}

		return rv
	// case reflect.Chan:
	case reflect.Func:
		return reflect.MakeFunc(ty, func(args []reflect.Value) (results []reflect.Value) {
			ja := make([]any, len(args))
			for i, arg := range args {
				ja[i] = tryJsValue(arg)
			}
			jresult := v.Invoke(ja...)
			out := ty.NumOut()
			if out == 0 {
				return
			}
			if out == 1 {
				results = []reflect.Value{
					tryReflectValue(ty.Out(0), jresult),
				}
				return
			}

			results = make([]reflect.Value, out)
			for i := 0; i < out; i++ {
				results[i] = tryReflectValue(ty.Out(i), jresult.Index(i))
			}
			return
		})
	case reflect.Interface:
		rv := reflect.ValueOf(tryJsAny(v))
		rvt := rv.Type()

		if !rvt.Implements(ty) {
			panic("passed in js value does not implement type")
		}

		return rv
	case reflect.Map:
		key := ty.Key()
		if key.Kind() != reflect.String {
			panic("js keys are always strings")
		}
		value := ty.Elem()

		// []string
		keys := objectConstructor.Call("keys", v)
		keysLength := keys.Get("length").Int()

		rv := reflect.MakeMapWithSize(ty, keysLength)
		for i := 0; i < keysLength; i++ {
			keyName := keys.Index(i).String()
			rv.SetMapIndex(reflect.ValueOf(keyName), tryReflectValue(value, v.Get(keyName)))
		}

		return rv
	case reflect.Pointer:
		tyElem := ty.Elem()
		rvp := reflect.New(tyElem)

		if !v.IsNull() && !v.IsUndefined() {
			rv := rvp.Elem()

			rv.Set(tryReflectValue(tyElem, v))
		}

		return rvp
	case reflect.Slice:
		if v.IsNull() || v.IsUndefined() {
			return reflect.MakeSlice(ty, 0, 0)
		}

		// basically same as array, but we need to find the length first
		length := v.Get("length").Int()

		if ty == byteSlice {
			// fast path using js.CopyBytesToGo
			rv := make([]byte, length)
			uint8Array := uint8ArrayConstructor.Call("from", v)
			js.CopyBytesToGo(rv, uint8Array)

			return reflect.ValueOf(rv)
		}

		rv := reflect.MakeSlice(ty, length, length)

		elem := ty.Elem()
		for i := 0; i < length; i++ {
			rv.Index(i).Set(tryReflectValue(elem, v.Index(i)))
		}

		return rv
	case reflect.String:
		return reflect.ValueOf(v.String())
	case reflect.Struct:
		// try to parse js.Value to struct
		rvp := reflect.New(ty)
		rv := rvp.Elem()

		for i := 0; i < ty.NumField(); i++ {
			field := ty.Field(i)
			if !field.IsExported() {
				continue
			}
			name := field.Name
			tag, ok := field.Tag.Lookup("json")
			if ok {
				name, _, _ = strings.Cut(tag, ",")
			}

			rv.Field(i).Set(tryReflectValue(field.Type, v.Get(name)))
		}

		return rv
	case reflect.UnsafePointer:
		return reflect.ValueOf(unsafe.Pointer(uintptr(v.Int())))
	default:
		panic("invalid reflect type")
	}
}

func funcOf(f reflect.Value) js.Func {
	if f.Kind() != reflect.Func {
		panic("FuncOf expects a function argument")
	}

	fty := f.Type()

	return js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) != fty.NumIn() {
			panic(fmt.Sprint("Call expected ", fty.NumIn(), " arguments but got ", len(args)))
		}

		convertedArgs := make([]reflect.Value, len(args))
		for i, arg := range args {
			convertedArgs[i] = tryReflectValue(fty.In(i), arg)
		}
		if fty.IsVariadic() {
			return tryJsValues(f.CallSlice(convertedArgs))
		}
		return tryJsValues(f.Call(convertedArgs))
	})
}

func FuncOf(f any) js.Func {
	return funcOf(reflect.ValueOf(f))
}
