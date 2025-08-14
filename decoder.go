package simdjson

import (
	"errors"
	"reflect"
	"sync"
	
	"github.com/simdjson/simdjson-go/internal/parser"
	internalScanner "github.com/simdjson/simdjson-go/internal/scanner"
)

type decoder struct {
	parser  *parser.Parser
	scanner *internalScanner.Scanner
	data    []byte
}

var decoderPool = sync.Pool{
	New: func() interface{} {
		return &decoder{
			parser:  parser.New(),
			scanner: internalScanner.New(),
		}
	},
}

func newDecoder(data []byte) *decoder {
	d := decoderPool.Get().(*decoder)
	d.data = data
	return d
}

func (d *decoder) release() {
	d.data = nil
	if d.scanner != nil {
		d.scanner.Release()
	}
	decoderPool.Put(d)
}

func (d *decoder) unmarshal(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("unmarshal requires non-nil pointer")
	}
	
	// Parse JSON into intermediate representation
	parsed, err := d.parser.Parse(d.data)
	if err != nil {
		return err
	}
	
	// Decode into target value
	return d.decode(parsed, rv.Elem())
}

func (d *decoder) decode(src interface{}, dst reflect.Value) error {
	if src == nil {
		dst.Set(reflect.Zero(dst.Type()))
		return nil
	}
	
	// Handle pointer types
	if dst.Kind() == reflect.Ptr {
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		return d.decode(src, dst.Elem())
	}
	
	// Handle interface{} type
	if dst.Kind() == reflect.Interface && dst.Type().NumMethod() == 0 {
		dst.Set(reflect.ValueOf(src))
		return nil
	}
	
	switch v := src.(type) {
	case bool:
		return d.decodeBool(v, dst)
	case float64:
		return d.decodeNumber(v, dst)
	case int64:
		return d.decodeInt(v, dst)
	case string:
		return d.decodeString(v, dst)
	case []interface{}:
		return d.decodeArray(v, dst)
	case map[string]interface{}:
		return d.decodeObject(v, dst)
	default:
		return errors.New("unexpected value type")
	}
}

func (d *decoder) decodeBool(src bool, dst reflect.Value) error {
	switch dst.Kind() {
	case reflect.Bool:
		dst.SetBool(src)
		return nil
	case reflect.Interface:
		if dst.Type().NumMethod() == 0 {
			dst.Set(reflect.ValueOf(src))
			return nil
		}
	}
	return errors.New("cannot unmarshal bool into " + dst.Type().String())
}

func (d *decoder) decodeNumber(src float64, dst reflect.Value) error {
	switch dst.Kind() {
	case reflect.Float32:
		dst.SetFloat(src)
		return nil
	case reflect.Float64:
		dst.SetFloat(src)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		dst.SetInt(int64(src))
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		dst.SetUint(uint64(src))
		return nil
	case reflect.Interface:
		if dst.Type().NumMethod() == 0 {
			dst.Set(reflect.ValueOf(src))
			return nil
		}
	}
	return errors.New("cannot unmarshal number into " + dst.Type().String())
}

func (d *decoder) decodeInt(src int64, dst reflect.Value) error {
	switch dst.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		dst.SetInt(src)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		dst.SetUint(uint64(src))
		return nil
	case reflect.Float32, reflect.Float64:
		dst.SetFloat(float64(src))
		return nil
	case reflect.Interface:
		if dst.Type().NumMethod() == 0 {
			dst.Set(reflect.ValueOf(src))
			return nil
		}
	}
	return errors.New("cannot unmarshal int into " + dst.Type().String())
}

func (d *decoder) decodeString(src string, dst reflect.Value) error {
	switch dst.Kind() {
	case reflect.String:
		dst.SetString(src)
		return nil
	case reflect.Interface:
		if dst.Type().NumMethod() == 0 {
			dst.Set(reflect.ValueOf(src))
			return nil
		}
	}
	return errors.New("cannot unmarshal string into " + dst.Type().String())
}

func (d *decoder) decodeArray(src []interface{}, dst reflect.Value) error {
	switch dst.Kind() {
	case reflect.Slice:
		// Create or resize slice
		if dst.IsNil() || dst.Len() < len(src) {
			dst.Set(reflect.MakeSlice(dst.Type(), len(src), len(src)))
		}
		
		for i, v := range src {
			if err := d.decode(v, dst.Index(i)); err != nil {
				return err
			}
		}
		return nil
		
	case reflect.Array:
		if dst.Len() < len(src) {
			return errors.New("array too small")
		}
		
		for i, v := range src {
			if err := d.decode(v, dst.Index(i)); err != nil {
				return err
			}
		}
		return nil
		
	case reflect.Interface:
		if dst.Type().NumMethod() == 0 {
			dst.Set(reflect.ValueOf(src))
			return nil
		}
	}
	
	return errors.New("cannot unmarshal array into " + dst.Type().String())
}

func (d *decoder) decodeObject(src map[string]interface{}, dst reflect.Value) error {
	switch dst.Kind() {
	case reflect.Map:
		// Create map if nil
		if dst.IsNil() {
			dst.Set(reflect.MakeMap(dst.Type()))
		}
		
		keyType := dst.Type().Key()
		elemType := dst.Type().Elem()
		
		for k, v := range src {
			keyVal := reflect.New(keyType).Elem()
			if keyType.Kind() == reflect.String {
				keyVal.SetString(k)
			} else {
				return errors.New("map key must be string")
			}
			
			elemVal := reflect.New(elemType).Elem()
			if err := d.decode(v, elemVal); err != nil {
				return err
			}
			
			dst.SetMapIndex(keyVal, elemVal)
		}
		return nil
		
	case reflect.Struct:
		return d.decodeStruct(src, dst)
		
	case reflect.Interface:
		if dst.Type().NumMethod() == 0 {
			dst.Set(reflect.ValueOf(src))
			return nil
		}
	}
	
	return errors.New("cannot unmarshal object into " + dst.Type().String())
}

func (d *decoder) decodeStruct(src map[string]interface{}, dst reflect.Value) error {
	typ := dst.Type()
	
	// Build field map
	fields := make(map[string]int)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		
		// Get JSON tag
		tag := field.Tag.Get("json")
		if tag == "-" {
			continue
		}
		
		name := field.Name
		if tag != "" {
			// Parse tag (simplified - doesn't handle all cases)
			if idx := findComma(tag); idx != -1 {
				name = tag[:idx]
			} else {
				name = tag
			}
		}
		
		fields[name] = i
	}
	
	// Set struct fields
	for k, v := range src {
		if idx, ok := fields[k]; ok {
			field := dst.Field(idx)
			if field.CanSet() {
				if err := d.decode(v, field); err != nil {
					return err
				}
			}
		}
	}
	
	return nil
}

func findComma(s string) int {
	for i, c := range s {
		if c == ',' {
			return i
		}
	}
	return -1
}