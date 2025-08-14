package simdjson

import (
	"errors"
	"io"
	
	"github.com/biggeezerdevelopment/simdjson-go/internal/scanner"
)

var (
	ErrInvalidJSON = errors.New("invalid JSON")
	ErrUnsupportedType = errors.New("unsupported type")
)

func Marshal(v interface{}) ([]byte, error) {
	e := newEncoder()
	defer e.release()
	
	return e.marshal(v)
}

func Unmarshal(data []byte, v interface{}) error {
	d := newDecoder(data)
	defer d.release()
	
	return d.unmarshal(v)
}

type Decoder struct {
	r       io.Reader
	buf     []byte
	scanner *scanner.Scanner
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r:       r,
		buf:     make([]byte, 0, 4096),
		scanner: scanner.New(),
	}
}

func (d *Decoder) Decode(v interface{}) error {
	if d.r != nil {
		data, err := io.ReadAll(d.r)
		if err != nil {
			return err
		}
		d.buf = data
	}
	
	dec := newDecoder(d.buf)
	defer dec.release()
	
	return dec.unmarshal(v)
}

type Encoder struct {
	w   io.Writer
	buf []byte
	enc *encoder
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w:   w,
		buf: make([]byte, 0, 4096),
		enc: newEncoder(),
	}
}

func (e *Encoder) Encode(v interface{}) error {
	data, err := e.enc.marshal(v)
	if err != nil {
		return err
	}
	
	_, err = e.w.Write(data)
	return err
}

func Valid(data []byte) bool {
	s := scanner.New()
	defer s.Release()
	
	return s.Validate(data)
}
