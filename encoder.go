package simdjson

import (
	"encoding/base64"
	"errors"
	"math"
	"reflect"
	"strconv"
	"sync"
)

type encoder struct {
	buf    []byte
	scratch [64]byte
}

var encoderPool = sync.Pool{
	New: func() interface{} {
		return &encoder{
			buf: make([]byte, 0, 4096),
		}
	},
}

func newEncoder() *encoder {
	e := encoderPool.Get().(*encoder)
	e.buf = e.buf[:0]
	return e
}

func (e *encoder) release() {
	if cap(e.buf) > 64*1024 {
		e.buf = make([]byte, 0, 4096)
	}
	encoderPool.Put(e)
}

func (e *encoder) marshal(v interface{}) ([]byte, error) {
	if err := e.encode(reflect.ValueOf(v)); err != nil {
		return nil, err
	}
	
	result := make([]byte, len(e.buf))
	copy(result, e.buf)
	return result, nil
}

func (e *encoder) encode(v reflect.Value) error {
	if !v.IsValid() {
		e.buf = append(e.buf, "null"...)
		return nil
	}
	
	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			e.buf = append(e.buf, "null"...)
			return nil
		}
		v = v.Elem()
	}
	
	switch v.Kind() {
	case reflect.Bool:
		return e.encodeBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return e.encodeInt(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return e.encodeUint(v.Uint())
	case reflect.Float32, reflect.Float64:
		return e.encodeFloat(v.Float())
	case reflect.String:
		return e.encodeString(v.String())
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			// []byte - encode as base64 string
			return e.encodeBytes(v.Bytes())
		}
		return e.encodeArray(v)
	case reflect.Array:
		return e.encodeArray(v)
	case reflect.Map:
		return e.encodeMap(v)
	case reflect.Struct:
		return e.encodeStruct(v)
	case reflect.Interface:
		if v.IsNil() {
			e.buf = append(e.buf, "null"...)
			return nil
		}
		return e.encode(v.Elem())
	default:
		return errors.New("unsupported type: " + v.Type().String())
	}
}

func (e *encoder) encodeBool(b bool) error {
	if b {
		e.buf = append(e.buf, "true"...)
	} else {
		e.buf = append(e.buf, "false"...)
	}
	return nil
}

func (e *encoder) encodeInt(i int64) error {
	e.buf = strconv.AppendInt(e.buf, i, 10)
	return nil
}

func (e *encoder) encodeUint(u uint64) error {
	e.buf = strconv.AppendUint(e.buf, u, 10)
	return nil
}

func (e *encoder) encodeFloat(f float64) error {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return errors.New("unsupported float value")
	}
	
	// Use optimized float formatting
	e.buf = strconv.AppendFloat(e.buf, f, 'g', -1, 64)
	return nil
}

func (e *encoder) encodeString(s string) error {
	e.buf = append(e.buf, '"')
	
	// Fast path for strings without special characters
	if !needsEscape(s) {
		e.buf = append(e.buf, s...)
		e.buf = append(e.buf, '"')
		return nil
	}
	
	// Slow path with escaping
	e.buf = appendEscapedString(e.buf, s)
	e.buf = append(e.buf, '"')
	return nil
}

func needsEscape(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == '"' || r == '\\' {
			return true
		}
	}
	return false
}

func appendEscapedString(dst []byte, s string) []byte {
	for _, r := range s {
		switch r {
		case '"':
			dst = append(dst, '\\', '"')
		case '\\':
			dst = append(dst, '\\', '\\')
		case '\b':
			dst = append(dst, '\\', 'b')
		case '\f':
			dst = append(dst, '\\', 'f')
		case '\n':
			dst = append(dst, '\\', 'n')
		case '\r':
			dst = append(dst, '\\', 'r')
		case '\t':
			dst = append(dst, '\\', 't')
		default:
			if r < 0x20 {
				dst = append(dst, '\\', 'u')
				dst = append(dst, "0000"...)
				hex := strconv.FormatInt(int64(r), 16)
				copy(dst[len(dst)-len(hex):], hex)
			} else {
				dst = append(dst, string(r)...)
			}
		}
	}
	return dst
}

func (e *encoder) encodeBytes(b []byte) error {
	e.buf = append(e.buf, '"')
	
	// Use base64 encoding for byte slices
	encodedLen := base64.StdEncoding.EncodedLen(len(b))
	start := len(e.buf)
	e.buf = append(e.buf, make([]byte, encodedLen)...)
	base64.StdEncoding.Encode(e.buf[start:], b)
	
	e.buf = append(e.buf, '"')
	return nil
}

func (e *encoder) encodeArray(v reflect.Value) error {
	e.buf = append(e.buf, '[')
	
	n := v.Len()
	for i := 0; i < n; i++ {
		if i > 0 {
			e.buf = append(e.buf, ',')
		}
		if err := e.encode(v.Index(i)); err != nil {
			return err
		}
	}
	
	e.buf = append(e.buf, ']')
	return nil
}

func (e *encoder) encodeMap(v reflect.Value) error {
	if v.Type().Key().Kind() != reflect.String {
		return errors.New("map key must be string")
	}
	
	e.buf = append(e.buf, '{')
	
	keys := v.MapKeys()
	for i, key := range keys {
		if i > 0 {
			e.buf = append(e.buf, ',')
		}
		
		// Encode key
		if err := e.encodeString(key.String()); err != nil {
			return err
		}
		
		e.buf = append(e.buf, ':')
		
		// Encode value
		if err := e.encode(v.MapIndex(key)); err != nil {
			return err
		}
	}
	
	e.buf = append(e.buf, '}')
	return nil
}

func (e *encoder) encodeStruct(v reflect.Value) error {
	e.buf = append(e.buf, '{')
	
	typ := v.Type()
	first := true
	
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		structField := typ.Field(i)
		
		// Skip unexported fields
		if structField.PkgPath != "" {
			continue
		}
		
		// Handle JSON tags
		tag := structField.Tag.Get("json")
		if tag == "-" {
			continue
		}
		
		name := structField.Name
		omitempty := false
		
		if tag != "" {
			// Parse tag
			if idx := findComma(tag); idx != -1 {
				name = tag[:idx]
				if tag[idx+1:] == "omitempty" {
					omitempty = true
				}
			} else {
				name = tag
			}
		}
		
		// Skip empty fields if omitempty
		if omitempty && isEmptyValue(field) {
			continue
		}
		
		if !first {
			e.buf = append(e.buf, ',')
		}
		first = false
		
		// Encode field name
		if err := e.encodeString(name); err != nil {
			return err
		}
		
		e.buf = append(e.buf, ':')
		
		// Encode field value
		if err := e.encode(field); err != nil {
			return err
		}
	}
	
	e.buf = append(e.buf, '}')
	return nil
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

// SIMD-optimized string escaping (would be implemented in assembly)
func escapeStringSIMD(dst, src []byte) int {
	// This would use SIMD to scan for characters that need escaping
	// and process multiple bytes at once
	// For now, fallback to scalar implementation
	return len(appendEscapedString(dst, string(src)))
}