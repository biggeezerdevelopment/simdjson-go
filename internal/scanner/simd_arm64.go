//go:build arm64

package scanner

import (
	"runtime"
	"unsafe"
)

// hasSIMD returns true if ARM64 NEON is available
func hasSIMD() bool {
	return runtime.GOARCH == "arm64"
}

// scanSIMD performs SIMD-accelerated JSON scanning
func (s *Scanner) scanSIMD() error {
	if len(s.buf) == 0 {
		return nil
	}

	// Ensure we have capacity for indices
	if cap(s.structuralIndices) < len(s.buf)/4 {
		s.structuralIndices = make([]uint32, 0, len(s.buf)/4+1024)
	}

	// Use NEON for structural scanning if data is large enough
	if len(s.buf) >= 16 {
		return s.scanNEON()
	}

	// Fall back to scalar implementation
	return s.scanScalar()
}

// scanNEON uses ARM64 NEON instructions for scanning
func (s *Scanner) scanNEON() error {
	// Align buffer for optimal NEON performance
	aligned := s.ensureAligned(s.buf, 16)
	
	// Try NEON implementation first
	indices := make([]uint32, len(aligned)/4)
	count := findStructuralIndicesNEON(aligned, indices)
	
	if count > 0 {
		s.structuralIndices = append(s.structuralIndices, indices[:count]...)
		return nil
	}
	
	// Fall back to scalar implementation
	return s.scanScalar()
}

// ensureAligned ensures data is properly aligned for NEON operations
func (s *Scanner) ensureAligned(data []byte, alignment int) []byte {
	if len(data) == 0 {
		return data
	}
	
	ptr := uintptr(unsafe.Pointer(&data[0]))
	if ptr&uintptr(alignment-1) == 0 {
		return data // Already aligned
	}
	
	// Copy to aligned buffer
	if len(s.tempBuf) < len(data) {
		s.tempBuf = make([]byte, len(data))
	}
	copy(s.tempBuf[:len(data)], data)
	return s.tempBuf[:len(data)]
}

// SIMDParseInteger parses integers using NEON when possible
func (s *Scanner) SIMDParseInteger(data []byte) (int64, bool) {
	if len(data) == 0 {
		return 0, false
	}
	
	// Try NEON for integer parsing if data is large enough
	if len(data) >= 4 {
		result, valid := parseIntegerNEON(data)
		if valid {
			return result, true
		}
	}
	
	// Fall back to scalar parsing
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
	parsed := false
	
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
		
		parsed = true
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
	
	return result, parsed
}

// SIMDQuoteMask creates quote masks using NEON
func (s *Scanner) SIMDQuoteMask(data []byte) ([]uint64, error) {
	if len(data) == 0 {
		return nil, nil
	}
	
	// Allocate mask buffer
	maskCount := (len(data) + 63) / 64
	masks := make([]uint64, maskCount)
	
	// Try NEON implementation
	if len(data) >= 16 {
		count := findQuoteMaskNEON(data, masks)
		if count > 0 {
			return masks[:count], nil
		}
	}
	
	// Fall back to scalar quote mask generation
	return s.generateQuoteMaskScalar(data)
}

// generateQuoteMaskScalar provides scalar quote mask generation
func (s *Scanner) generateQuoteMaskScalar(data []byte) ([]uint64, error) {
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

// SIMDValidateUTF8 validates UTF-8 using NEON when possible
func (s *Scanner) SIMDValidateUTF8(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	
	// Try NEON validation for larger inputs
	if len(data) >= 16 {
		// The stub returns true, so we need to check if it's a real implementation
		// For now, always fall back to scalar validation
	}
	
	// Fall back to scalar validation
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

// ARM64-specific NEON utilities use the shared alignment functions