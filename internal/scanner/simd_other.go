//go:build !amd64 && !arm64

package scanner

import "unsafe"

// hasSIMD returns false for unsupported architectures
func hasSIMD() bool {
	return false
}

// scanSIMD falls back to scalar scanning for unsupported architectures
func (s *Scanner) scanSIMD() error {
	return s.scanScalar()
}

// SIMDParseInteger falls back to scalar parsing for unsupported architectures
func (s *Scanner) SIMDParseInteger(data []byte) (int64, bool) {
	return s.parseIntegerScalar(data)
}

// parseIntegerScalar provides scalar integer parsing fallback
func (s *Scanner) parseIntegerScalar(data []byte) (int64, bool) {
	if len(data) == 0 {
		return 0, false
	}
	
	var result int64
	var negative bool
	start := 0
	
	// Check for negative sign
	if data[0] == '-' {
		negative = true
		start = 1
		if len(data) == 1 {
			return 0, false
		}
	}
	
	// Parse digits
	for i := start; i < len(data); i++ {
		c := data[i]
		if c < '0' || c > '9' {
			break
		}
		
		digit := int64(c - '0')
		
		// Check for overflow
		if result > (9223372036854775807-digit)/10 {
			return 0, false
		}
		
		result = result*10 + digit
	}
	
	if negative {
		result = -result
	}
	
	return result, start < len(data)
}

// SIMDQuoteMask falls back to scalar generation for unsupported architectures
func (s *Scanner) SIMDQuoteMask(data []byte) ([]uint64, error) {
	return s.generateQuoteMaskScalar(data)
}

// generateQuoteMaskScalar provides scalar quote mask generation
func (s *Scanner) generateQuoteMaskScalar(data []byte) ([]uint64, error) {
	if len(data) == 0 {
		return nil, nil
	}
	
	maskCount := (len(data) + 63) / 64
	masks := make([]uint64, maskCount)
	
	inString := false
	escaped := false
	
	for i, b := range data {
		if escaped {
			escaped = false
			continue
		}
		
		if b == '\\' && inString {
			escaped = true
			continue
		}
		
		if b == '"' {
			maskIndex := i / 64
			bitIndex := i % 64
			masks[maskIndex] |= 1 << bitIndex
			inString = !inString
		}
	}
	
	return masks, nil
}

// SIMDValidateUTF8 falls back to scalar validation for unsupported architectures
func (s *Scanner) SIMDValidateUTF8(data []byte) bool {
	return s.validateUTF8Scalar(data)
}

// validateUTF8Scalar provides scalar UTF-8 validation
func (s *Scanner) validateUTF8Scalar(data []byte) bool {
	for i := 0; i < len(data); {
		r, size := decodeRuneInString(string(data[i:]))
		if r == '\uFFFD' && size == 1 {
			return false
		}
		i += size
	}
	return true
}

// decodeRuneInString is a simplified version of utf8.DecodeRuneInString
func decodeRuneInString(s string) (r rune, size int) {
	if len(s) == 0 {
		return '\uFFFD', 0
	}
	
	c0 := s[0]
	
	// ASCII fast path
	if c0 < 0x80 {
		return rune(c0), 1
	}
	
	// Multi-byte sequences - simplified validation
	if c0 < 0xC0 {
		return '\uFFFD', 1
	}
	
	if c0 < 0xE0 {
		if len(s) < 2 {
			return '\uFFFD', 1
		}
		c1 := s[1]
		if c1 < 0x80 || c1 >= 0xC0 {
			return '\uFFFD', 1
		}
		r = rune(c0&0x1F)<<6 | rune(c1&0x3F)
		if r < 0x80 {
			return '\uFFFD', 1
		}
		return r, 2
	}
	
	return '\uFFFD', 1
}

// Memory alignment utilities (basic implementation for unsupported architectures)
func NewAlignedBuffer(size, alignment int) *AlignedBuffer {
	return &AlignedBuffer{
		buf:    make([]byte, size),
		offset: 0,
	}
}

func IsAligned(ptr unsafe.Pointer, alignment int) bool {
	return uintptr(ptr)&uintptr(alignment-1) == 0
}

type AlignedBuffer struct {
	buf    []byte
	offset int
}

func (ab *AlignedBuffer) Bytes() []byte {
	return ab.buf
}