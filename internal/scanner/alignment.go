package scanner

import (
	"unsafe"
)

const (
	// SIMD alignment requirements
	AVX2Alignment = 32  // 256-bit alignment for AVX2
	SSE4Alignment = 16  // 128-bit alignment for SSE4.2
	NEONAlignment = 16  // 128-bit alignment for NEON
	CacheLineSize = 64  // Typical cache line size
)

// AlignedBuffer represents a SIMD-aligned buffer
type AlignedBuffer struct {
	data     []byte
	aligned  []byte
	capacity int
}

// NewAlignedBuffer creates a new SIMD-aligned buffer
func NewAlignedBuffer(size int, alignment int) *AlignedBuffer {
	// Allocate extra space for alignment
	totalSize := size + alignment - 1
	data := make([]byte, totalSize)
	
	// Calculate aligned offset
	addr := uintptr(unsafe.Pointer(&data[0]))
	alignedAddr := (addr + uintptr(alignment-1)) &^ uintptr(alignment-1)
	offset := alignedAddr - addr
	
	// Create aligned slice
	aligned := data[offset : offset+uintptr(size)]
	
	return &AlignedBuffer{
		data:     data,
		aligned:  aligned,
		capacity: size,
	}
}

// Bytes returns the aligned byte slice
func (ab *AlignedBuffer) Bytes() []byte {
	return ab.aligned
}

// Resize resizes the aligned buffer
func (ab *AlignedBuffer) Resize(newSize int, alignment int) {
	if newSize <= ab.capacity {
		ab.aligned = ab.aligned[:newSize]
		return
	}
	
	// Need to reallocate
	*ab = *NewAlignedBuffer(newSize, alignment)
}

// IsAligned checks if a pointer is aligned to the specified boundary
func IsAligned(ptr unsafe.Pointer, alignment int) bool {
	addr := uintptr(ptr)
	return addr&uintptr(alignment-1) == 0
}

// GetOptimalAlignment returns the optimal alignment for the current CPU
func GetOptimalAlignment() int {
	return AVX2Alignment // Use largest alignment for safety
}

// PadToAlignment pads a slice to ensure SIMD-safe processing
func PadToAlignment(data []byte, alignment int) []byte {
	remainder := len(data) % alignment
	if remainder == 0 {
		return data
	}
	
	// Pad with zeros to alignment boundary
	padSize := alignment - remainder
	padded := make([]byte, len(data)+padSize)
	copy(padded, data)
	return padded
}

// ProcessAligned processes data in aligned chunks for optimal SIMD performance
func ProcessAligned(data []byte, chunkSize int, processor func([]byte) error) error {
	// Ensure data is aligned
	if !IsAligned(unsafe.Pointer(&data[0]), chunkSize) {
		// Copy to aligned buffer if needed
		aligned := NewAlignedBuffer(len(data), chunkSize)
		copy(aligned.Bytes(), data)
		data = aligned.Bytes()
	}
	
	// Process in aligned chunks
	for i := 0; i+chunkSize <= len(data); i += chunkSize {
		if err := processor(data[i : i+chunkSize]); err != nil {
			return err
		}
	}
	
	// Process remainder
	remainder := len(data) % chunkSize
	if remainder > 0 {
		// Pad remainder to chunk size for SIMD safety
		padded := make([]byte, chunkSize)
		copy(padded, data[len(data)-remainder:])
		if err := processor(padded[:remainder]); err != nil {
			return err
		}
	}
	
	return nil
}