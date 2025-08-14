//go:build arm64

package scanner

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"unsafe"
)

// TestARM64NEONSupport tests ARM64 NEON availability and features
func TestARM64NEONSupport(t *testing.T) {
	if runtime.GOARCH != "arm64" {
		t.Skip("ARM64 tests require arm64 architecture")
	}

	t.Run("NEON_Available", func(t *testing.T) {
		if !hasSIMD() {
			t.Error("NEON should be available on ARM64")
		}
	})

	t.Run("Feature_Detection", func(t *testing.T) {
		// Test that NEON is detected
		if !hasSIMD() {
			t.Error("hasSIMD() should return true on ARM64")
		}
	})
}

// TestARM64StructuralScanning tests NEON structural scanning
func TestARM64StructuralScanning(t *testing.T) {
	if runtime.GOARCH != "arm64" {
		t.Skip("ARM64 tests require arm64 architecture")
	}

	testCases := []struct {
		name  string
		json  string
		count int // Expected minimum structural indices
	}{
		{"simple_object", `{"key":"value"}`, 5},
		{"array", `[1,2,3,4,5]`, 9},
		{"nested", `{"a":{"b":[1,2]}}`, 9},
		{"complex", `{"users":[{"id":1,"name":"Alice"}],"count":2}`, 15},
		{"large_string", `{"data":"` + string(make([]byte, 100)) + `"}`, 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := New()
			defer s.Release()

			// Test NEON scanning
			err := s.ScanSIMD([]byte(tc.json))
			if err != nil {
				t.Fatalf("NEON scan failed: %v", err)
			}

			indices := s.GetStructuralIndices()
			if len(indices) < tc.count {
				t.Errorf("Expected at least %d indices, got %d", tc.count, len(indices))
			}
		})
	}
}

// TestARM64IntegerParsing tests NEON integer parsing
func TestARM64IntegerParsing(t *testing.T) {
	if runtime.GOARCH != "arm64" {
		t.Skip("ARM64 tests require arm64 architecture")
	}

	testCases := []struct {
		name     string
		input    string
		expected int64
		valid    bool
	}{
		{"zero", "0", 0, true},
		{"positive", "12345", 12345, true},
		{"negative", "-6789", -6789, true},
		{"max_int64", "9223372036854775807", 9223372036854775807, true},
		{"min_int64", "-9223372036854775808", -9223372036854775808, true},
		{"invalid", "abc", 0, false}, // parseIntegerScalar returns false for invalid input
		{"empty", "", 0, false},
		{"float_prefix", "123.45", 123, true}, // Should parse integer part
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := New()
			defer s.Release()

			result, valid := s.SIMDParseInteger([]byte(tc.input))

			if valid != tc.valid {
				t.Errorf("Valid mismatch: expected=%v, got=%v", tc.valid, valid)
			}

			if valid && result != tc.expected {
				t.Errorf("Result mismatch: expected=%d, got=%d", tc.expected, result)
			}
		})
	}
}

// TestARM64QuoteMasking tests NEON quote masking
func TestARM64QuoteMasking(t *testing.T) {
	if runtime.GOARCH != "arm64" {
		t.Skip("ARM64 tests require arm64 architecture")
	}

	testCases := []struct {
		name  string
		input string
	}{
		{"no_quotes", "123 456 789"},
		{"simple_string", `"hello world"`},
		{"multiple_strings", `"first" "second" "third"`},
		{"escaped_quotes", `"say \"hello\" to the world"`},
		{"complex_json", `{"message":"She said \"Hi there!\""}`},
		{"empty", ""},
		{"long_string", `"` + string(make([]byte, 200)) + `"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := New()
			defer s.Release()

			masks, err := s.SIMDQuoteMask([]byte(tc.input))
			if err != nil {
				t.Fatalf("NEON quote mask failed: %v", err)
			}

			// Basic validation - should not crash and return reasonable results
			maxExpectedMasks := (len(tc.input) + 63) / 64
			if len(masks) > maxExpectedMasks {
				t.Errorf("Too many masks: got %d, expected <=%d", len(masks), maxExpectedMasks)
			}
		})
	}
}

// TestARM64UTF8Validation tests NEON UTF-8 validation
func TestARM64UTF8Validation(t *testing.T) {
	if runtime.GOARCH != "arm64" {
		t.Skip("ARM64 tests require arm64 architecture")
	}

	testCases := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{"ascii", []byte("Hello, World!"), true},
		{"empty", []byte(""), true},
		{"utf8_2byte", []byte("café"), true},
		{"utf8_3byte", []byte("世界"), true},
		{"utf8_4byte", []byte("🌍🎉"), true},
		{"mixed", []byte("Hello 世界 🌍 from ARM64!"), true},
		{"long_ascii", make([]byte, 1000), true}, // Will be filled with zeros (valid ASCII)
		{"long_utf8", []byte(string([]rune(strings.Repeat("世界🌍", 100)))), true},
	}

	// Fill long_ascii with valid ASCII characters
	for i := range testCases[6].input {
		testCases[6].input[i] = byte('A' + (i % 26))
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := New()
			defer s.Release()

			result := s.SIMDValidateUTF8(tc.input)
			if result != tc.expected {
				t.Errorf("UTF-8 validation mismatch: expected=%v, got=%v", tc.expected, result)
			}
		})
	}
}

// TestARM64MemoryAlignment tests NEON memory alignment utilities
func TestARM64MemoryAlignment(t *testing.T) {
	if runtime.GOARCH != "arm64" {
		t.Skip("ARM64 tests require arm64 architecture")
	}

	alignments := []int{16, 32, 64}
	sizes := []int{16, 32, 64, 100, 1000}

	for _, alignment := range alignments {
		for _, size := range sizes {
			t.Run(fmt.Sprintf("align_%d_size_%d", alignment, size), func(t *testing.T) {
				buf := NewAlignedBuffer(size, alignment)

				// Check alignment
				ptr := unsafe.Pointer(&buf.Bytes()[0])
				if !IsAligned(ptr, alignment) {
					t.Errorf("Buffer not aligned to %d bytes", alignment)
				}

				// Check size
				if len(buf.Bytes()) < size {
					t.Errorf("Buffer size too small: got %d, expected at least %d", len(buf.Bytes()), size)
				}

				// Test write/read
				data := buf.Bytes()
				for i := 0; i < len(data) && i < size; i++ {
					data[i] = byte(i % 256)
				}

				for i := 0; i < len(data) && i < size; i++ {
					if data[i] != byte(i%256) {
						t.Errorf("Data corruption at index %d", i)
						break
					}
				}
			})
		}
	}
}

// TestARM64Performance compares NEON vs scalar performance
func TestARM64Performance(t *testing.T) {
	if runtime.GOARCH != "arm64" {
		t.Skip("ARM64 tests require arm64 architecture")
	}

	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	// Generate test data
	jsonData := []byte(`{
		"users": [
			{"id": 1, "name": "Alice", "email": "alice@example.com", "active": true},
			{"id": 2, "name": "Bob", "email": "bob@example.com", "active": false},
			{"id": 3, "name": "Charlie", "email": "charlie@example.com", "active": true}
		],
		"metadata": {
			"version": "1.0.0",
			"timestamp": 1234567890,
			"count": 3
		}
	}`)

	// Repeat to make it larger for meaningful comparison
	largeData := make([]byte, 0, len(jsonData)*100)
	for i := 0; i < 100; i++ {
		largeData = append(largeData, jsonData...)
	}

	t.Run("NEON_vs_Scalar", func(t *testing.T) {
		// Test NEON
		s1 := New()
		defer s1.Release()

		neonStart := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				s1.ScanSIMD(largeData)
			}
		})

		// Test scalar fallback
		s2 := New()
		defer s2.Release()

		scalarStart := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				s2.scanScalar()
			}
		})

		// Compare results (just log, don't fail - performance can vary)
		t.Logf("NEON: %v ops", neonStart.N)
		t.Logf("Scalar: %v ops", scalarStart.N)

		// Verify both produce same results
		s1.ScanSIMD(largeData)
		s2.scanScalar()

		neonIndices := s1.GetStructuralIndices()
		scalarIndices := s2.GetStructuralIndices()

		if len(neonIndices) != len(scalarIndices) {
			t.Errorf("Index count mismatch: NEON=%d, Scalar=%d", len(neonIndices), len(scalarIndices))
		}
	})
}

// TestARM64ConcurrentSafety tests thread safety of NEON operations
func TestARM64ConcurrentSafety(t *testing.T) {
	if runtime.GOARCH != "arm64" {
		t.Skip("ARM64 tests require arm64 architecture")
	}

	jsonData := []byte(`{"test":"concurrent","numbers":[1,2,3,4,5],"nested":{"value":42}}`)
	numGoroutines := 10
	numIterations := 100

	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					done <- &ARM64TestError{msg: "goroutine panicked", id: id, cause: r}
				} else {
					done <- nil
				}
			}()

			// Each goroutine has its own scanner
			s := New()
			defer s.Release()

			for j := 0; j < numIterations; j++ {
				err := s.ScanSIMD(jsonData)
				if err != nil {
					done <- &ARM64TestError{msg: "scan failed", id: id, cause: err}
					return
				}

				// Test other NEON operations
				s.SIMDValidateUTF8(jsonData)
				s.SIMDParseInteger([]byte("42"))
				s.SIMDQuoteMask(jsonData)
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

// TestARM64EdgeCases tests edge cases specific to ARM64/NEON
func TestARM64EdgeCases(t *testing.T) {
	if runtime.GOARCH != "arm64" {
		t.Skip("ARM64 tests require arm64 architecture")
	}

	t.Run("UnalignedData", func(t *testing.T) {
		// Test with unaligned data
		baseData := []byte(`{"unaligned":"test data"}`)
		
		// Create unaligned buffer
		buf := make([]byte, len(baseData)+8)
		copy(buf[3:], baseData) // Offset by 3 to make it unaligned
		unalignedData := buf[3:]

		s := New()
		defer s.Release()

		err := s.ScanSIMD(unalignedData)
		if err != nil {
			t.Errorf("NEON should handle unaligned data gracefully: %v", err)
		}
	})

	t.Run("SmallBuffers", func(t *testing.T) {
		// Test with buffers smaller than NEON width (16 bytes)
		smallInputs := []string{
			"{}",
			`{"a":1}`,
			`[1,2,3]`,
			"null",
			"true",
		}

		s := New()
		defer s.Release()

		for _, input := range smallInputs {
			err := s.ScanSIMD([]byte(input))
			if err != nil {
				t.Errorf("NEON should handle small input %q: %v", input, err)
			}
		}
	})

	t.Run("LargeBuffers", func(t *testing.T) {
		// Test with very large buffers
		largeJson := `{"data":[`
		for i := 0; i < 10000; i++ {
			if i > 0 {
				largeJson += ","
			}
			largeJson += `{"id":` + string(rune('0'+i%10)) + `}`
		}
		largeJson += `]}`

		s := New()
		defer s.Release()

		err := s.ScanSIMD([]byte(largeJson))
		if err != nil {
			t.Errorf("NEON should handle large buffers: %v", err)
		}
	})
}

// ARM64TestError provides detailed error information for ARM64 tests
type ARM64TestError struct {
	msg   string
	id    int
	cause interface{}
}

func (e *ARM64TestError) Error() string {
	return fmt.Sprintf("ARM64 test error (goroutine %d): %s - %v", e.id, e.msg, e.cause)
}

// BenchmarkARM64Operations benchmarks ARM64 NEON operations
func BenchmarkARM64Operations(b *testing.B) {
	if runtime.GOARCH != "arm64" {
		b.Skip("ARM64 benchmarks require arm64 architecture")
	}

	testData := []byte(`{"users":[{"id":1,"name":"Alice","active":true},{"id":2,"name":"Bob","active":false}],"count":2}`)

	b.Run("StructuralScanning", func(b *testing.B) {
		s := New()
		defer s.Release()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.ScanSIMD(testData)
		}
	})

	b.Run("IntegerParsing", func(b *testing.B) {
		s := New()
		defer s.Release()
		intData := []byte("123456789")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.SIMDParseInteger(intData)
		}
	})

	b.Run("UTF8Validation", func(b *testing.B) {
		s := New()
		defer s.Release()
		utf8Data := []byte("Hello 世界 🌍 from ARM64!")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.SIMDValidateUTF8(utf8Data)
		}
	})

	b.Run("QuoteMasking", func(b *testing.B) {
		s := New()
		defer s.Release()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			s.SIMDQuoteMask(testData)
		}
	})
}