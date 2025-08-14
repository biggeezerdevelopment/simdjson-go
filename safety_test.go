package simdjson

import (
	"fmt"
	"runtime"
	"testing"
	"unsafe"
	
	"github.com/biggeezerdevelopment/simdjson-go/internal/scanner"
)

// TestMemorySafety tests memory safety of SIMD operations
func TestMemorySafety(t *testing.T) {
	t.Run("AlignedMemory", testAlignedMemorySafety)
	t.Run("UnalignedMemory", testUnalignedMemorySafety)
	t.Run("BoundaryAccess", testBoundaryAccess)
	t.Run("ZeroLengthInput", testZeroLengthInput)
	t.Run("LargeInput", testLargeInputSafety)
	t.Run("ConcurrentAccess", testConcurrentMemoryAccess)
}

func testAlignedMemorySafety(t *testing.T) {
	alignments := []int{16, 32}
	sizes := []int{0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 100, 1000}

	for _, alignment := range alignments {
		for _, size := range sizes {
			t.Run(fmt.Sprintf("align_%d_size_%d", alignment, size), func(t *testing.T) {
				if size == 0 {
					return // Skip zero-size test for aligned buffers
				}

				buf := scanner.NewAlignedBuffer(size, alignment)
				
				// Verify alignment
				ptr := unsafe.Pointer(&buf.Bytes()[0])
				if !scanner.IsAligned(ptr, alignment) {
					t.Errorf("Buffer not properly aligned to %d bytes", alignment)
				}

				// Fill with valid JSON data
				jsonData := []byte(`{"test":"data"}`)
				copySize := min(size, len(jsonData))
				copy(buf.Bytes()[:copySize], jsonData[:copySize])

				// Test SIMD operations don't crash
				s := scanner.New()
				defer s.Release()

				// This should not crash even with partial/invalid JSON
				s.ScanSIMD(buf.Bytes()[:copySize])
				s.SIMDValidateUTF8(buf.Bytes()[:copySize])
				if copySize > 0 {
					s.SIMDParseInteger(buf.Bytes()[:copySize])
					s.SIMDQuoteMask(buf.Bytes()[:copySize])
				}
			})
		}
	}
}

func testUnalignedMemorySafety(t *testing.T) {
	// Test SIMD operations on unaligned data
	baseData := []byte(`{"unaligned":"test","number":42,"array":[1,2,3]}`)
	
	// Create unaligned versions by offsetting
	for offset := 1; offset < 8; offset++ {
		t.Run(fmt.Sprintf("offset_%d", offset), func(t *testing.T) {
			// Create buffer with offset
			buf := make([]byte, len(baseData)+offset)
			copy(buf[offset:], baseData)
			unalignedData := buf[offset:]

			// Verify it's unaligned
			ptr := unsafe.Pointer(&unalignedData[0])
			if scanner.IsAligned(ptr, 32) {
				t.Skip("Data happened to be aligned")
			}

			s := scanner.New()
			defer s.Release()

			// SIMD operations should handle unaligned data safely
			err := s.ScanSIMD(unalignedData)
			if err != nil {
				// Error is OK, but shouldn't crash
				t.Logf("SIMD scan failed safely: %v", err)
			}

			// Other SIMD operations
			s.SIMDValidateUTF8(unalignedData)
			s.SIMDParseInteger([]byte("123"))
			s.SIMDQuoteMask(unalignedData)
		})
	}
}

func testBoundaryAccess(t *testing.T) {
	// Test operations at buffer boundaries
	testCases := []struct {
		name string
		data []byte
	}{
		{"empty", []byte("")},
		{"single_byte", []byte("{")},
		{"two_bytes", []byte("{}")},
		{"just_under_16", make([]byte, 15)},
		{"exactly_16", make([]byte, 16)},
		{"just_over_16", make([]byte, 17)},
		{"just_under_32", make([]byte, 31)},
		{"exactly_32", make([]byte, 32)},
		{"just_over_32", make([]byte, 33)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Fill with valid JSON pattern if possible
			pattern := []byte(`{"a":1}`)
			for i := 0; i < len(tc.data); i++ {
				tc.data[i] = pattern[i%len(pattern)]
			}

			s := scanner.New()
			defer s.Release()

			// These operations should not crash regardless of input size
			s.ScanSIMD(tc.data)
			s.SIMDValidateUTF8(tc.data)
			if len(tc.data) > 0 {
				s.SIMDParseInteger(tc.data)
				s.SIMDQuoteMask(tc.data)
			}
		})
	}
}

func testZeroLengthInput(t *testing.T) {
	s := scanner.New()
	defer s.Release()

	// All SIMD operations should handle zero-length input safely
	emptyData := []byte("")

	err := s.ScanSIMD(emptyData)
	if err != nil {
		t.Logf("SIMD scan on empty data failed safely: %v", err)
	}

	valid := s.SIMDValidateUTF8(emptyData)
	if !valid {
		t.Error("Empty data should be valid UTF-8")
	}

	value, ok := s.SIMDParseInteger(emptyData)
	if ok {
		t.Errorf("Empty data should not parse as integer, got %d", value)
	}

	masks, err := s.SIMDQuoteMask(emptyData)
	if err != nil {
		t.Errorf("Quote mask on empty data failed: %v", err)
	}
	if len(masks) != 0 {
		t.Error("Empty data should produce no quote masks")
	}
}

func testLargeInputSafety(t *testing.T) {
	// Test with very large inputs to ensure no buffer overflows
	sizes := []int{1024, 10240, 102400}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			// Create large JSON
			data := make([]byte, size)
			
			// Fill with JSON pattern
			pattern := []byte(`{"key":"value","num":123,"arr":[1,2,3]},`)
			for i := 0; i < len(data); i++ {
				data[i] = pattern[i%len(pattern)]
			}
			
			// Make it valid JSON
			data[0] = '['
			data[len(data)-1] = ']'

			s := scanner.New()
			defer s.Release()

			// Should not crash or cause memory issues
			s.ScanSIMD(data)
			s.SIMDValidateUTF8(data)
			s.SIMDQuoteMask(data)
		})
	}
}

func testConcurrentMemoryAccess(t *testing.T) {
	// Test concurrent access to ensure thread safety
	testData := []byte(`{"concurrent":"test","data":[1,2,3,4,5]}`)
	numGoroutines := 10
	numIterations := 100

	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					done <- fmt.Errorf("goroutine %d panicked: %v", id, r)
				} else {
					done <- nil
				}
			}()

			// Each goroutine has its own scanner
			s := scanner.New()
			defer s.Release()

			for j := 0; j < numIterations; j++ {
				s.ScanSIMD(testData)
				s.SIMDValidateUTF8(testData)
				s.SIMDParseInteger([]byte("42"))
				s.SIMDQuoteMask(testData)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			t.Error(err)
		}
	}
}

// TestRaceConditions tests for race conditions in SIMD code
func TestRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition tests in short mode")
	}

	// This test should be run with -race flag
	testData := []byte(`{"race":"test","numbers":[1,2,3,4,5,6,7,8,9,10]}`)

	// Shared scanner (this would be unsafe, but we're testing individual methods)
	done := make(chan struct{}, 20)

	for i := 0; i < 20; i++ {
		go func() {
			defer func() { done <- struct{}{} }()

			// Each goroutine creates its own scanner to avoid races
			s := scanner.New()
			defer s.Release()

			for j := 0; j < 50; j++ {
				s.ScanSIMD(testData)
			}
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}

// TestMemoryLeaks tests for memory leaks in pooled objects
func TestMemoryLeaks(t *testing.T) {
	var m1, m2 runtime.MemStats
	
	// Baseline memory
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Create and release many scanners
	for i := 0; i < 1000; i++ {
		s := scanner.New()
		s.ScanSIMD([]byte(`{"test":"leak"}`))
		s.Release()
	}

	// Force garbage collection
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Memory growth should be minimal
	memGrowth := m2.HeapAlloc - m1.HeapAlloc
	t.Logf("Memory growth: %d bytes", memGrowth)

	// Allow some growth but not excessive
	if memGrowth > 1024*1024 { // 1MB threshold
		t.Errorf("Excessive memory growth detected: %d bytes", memGrowth)
	}
}

// TestBufferOverflow tests protection against buffer overflows
func TestBufferOverflow(t *testing.T) {
	// Test that SIMD operations don't read past buffer boundaries
	testCases := []int{1, 2, 4, 8, 15, 16, 17, 31, 32, 33}

	for _, size := range testCases {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			// Create buffer exactly at the size
			buf := make([]byte, size)
			
			// Fill with pattern
			for i := 0; i < size; i++ {
				buf[i] = byte('0' + (i % 10))
			}

			s := scanner.New()
			defer s.Release()

			// SIMD operations should not read beyond buffer
			// This is hard to test directly, but at least verify no crashes
			s.ScanSIMD(buf)
			s.SIMDValidateUTF8(buf)
			s.SIMDParseInteger(buf)
			s.SIMDQuoteMask(buf)
		})
	}
}

// TestInvalidPointers tests handling of edge case pointers
func TestInvalidPointers(t *testing.T) {
	s := scanner.New()
	defer s.Release()

	// Test nil slice (should be handled gracefully)
	var nilSlice []byte
	
	s.ScanSIMD(nilSlice) // Should not crash
	s.SIMDValidateUTF8(nilSlice)
	s.SIMDParseInteger(nilSlice)
	s.SIMDQuoteMask(nilSlice)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
