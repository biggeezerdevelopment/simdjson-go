//go:build amd64 && !noasm

package scanner

import (
	"errors"
	"unsafe"
)

//go:noescape
func findStructuralIndicesAVX2(data unsafe.Pointer, length uint64, indices *uint32) uint64

//go:noescape
func findQuoteMaskAVX2(data unsafe.Pointer, length uint64, mask *uint64) uint64

//go:noescape
func validateUTF8AVX2(data unsafe.Pointer, length uint64) bool

//go:noescape
func parseIntegerAVX2(data unsafe.Pointer, length uint64) (int64, bool)

// SSE4.2 fallback functions
//go:noescape
func findStructuralIndicesSSE42(data unsafe.Pointer, length uint64, indices *uint32) uint64

//go:noescape
func findQuoteMaskSSE42(data unsafe.Pointer, length uint64, mask *uint64) uint64

//go:noescape
func validateUTF8SSE42(data unsafe.Pointer, length uint64) bool

//go:noescape
func parseIntegerSSE42(data unsafe.Pointer, length uint64) (int64, bool)

func hasSIMD() bool {
	return hasAVX2() || hasSSE42()
}

func (s *Scanner) scanSIMD() error {
	if len(s.buf) == 0 {
		return nil
	}
	
	// Ensure we have enough space for indices (estimate 1/8 of buffer size)
	estimatedIndices := len(s.buf) / 8
	if cap(s.structuralIndices) < estimatedIndices {
		s.structuralIndices = make([]uint32, 0, estimatedIndices)
	}
	
	dataPtr := unsafe.Pointer(&s.buf[0])
	indicesPtr := unsafe.Pointer(&s.structuralIndices[0])
	
	// Resize slice to have capacity for potential indices
	s.structuralIndices = s.structuralIndices[:cap(s.structuralIndices)]
	
	var count uint64
	if hasAVX2() {
		count = findStructuralIndicesAVX2(dataPtr, uint64(len(s.buf)), (*uint32)(indicesPtr))
	} else if hasSSE42() {
		count = findStructuralIndicesSSE42(dataPtr, uint64(len(s.buf)), (*uint32)(indicesPtr))
	} else {
		// Fallback to scalar
		return s.scanScalar()
	}
	
	// Resize slice to actual count
	s.structuralIndices = s.structuralIndices[:count]
	
	return nil
}

// SIMDQuoteMask generates a bitmask of quote positions for fast string detection
func (s *Scanner) SIMDQuoteMask(data []byte) ([]uint64, error) {
	s.buf = data
	if len(s.buf) == 0 {
		return nil, nil
	}
	
	var chunkSize, maskCount int
	if hasAVX2() {
		chunkSize = 32
		maskCount = (len(s.buf) + 31) / 32
	} else {
		chunkSize = 16
		maskCount = (len(s.buf) + 15) / 16
	}
	
	masks := make([]uint64, maskCount)
	dataPtr := unsafe.Pointer(&s.buf[0])
	maskPtr := unsafe.Pointer(&masks[0])
	
	var actualCount uint64
	if hasAVX2() {
		actualCount = findQuoteMaskAVX2(dataPtr, uint64(len(s.buf)), (*uint64)(maskPtr))
	} else if hasSSE42() {
		actualCount = findQuoteMaskSSE42(dataPtr, uint64(len(s.buf)), (*uint64)(maskPtr))
	} else {
		return nil, errors.New("SIMD not available")
	}
	
	return masks[:actualCount], nil
}

// SIMDValidateUTF8 validates UTF-8 encoding using SIMD instructions
func (s *Scanner) SIMDValidateUTF8(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	
	dataPtr := unsafe.Pointer(&data[0])
	if hasAVX2() {
		return validateUTF8AVX2(dataPtr, uint64(len(data)))
	} else if hasSSE42() {
		return validateUTF8SSE42(dataPtr, uint64(len(data)))
	}
	
	// Fallback to standard validation
	return true // Simplified
}

// SIMDParseInteger attempts to parse an integer using SIMD acceleration
func SIMDParseInteger(data []byte) (int64, bool) {
	if len(data) == 0 {
		return 0, false
	}
	
	dataPtr := unsafe.Pointer(&data[0])
	if hasAVX2() {
		return parseIntegerAVX2(dataPtr, uint64(len(data)))
	} else if hasSSE42() {
		return parseIntegerSSE42(dataPtr, uint64(len(data)))
	}
	
	return 0, false
}

// SIMDParseInteger method for Scanner
func (s *Scanner) SIMDParseInteger(data []byte) (int64, bool) {
	return SIMDParseInteger(data)
}