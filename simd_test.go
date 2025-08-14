package simdjson

import (
	"bytes"
	"fmt"
	"testing"
	"time"
	"unsafe"
	
	"github.com/simdjson/simdjson-go/internal/scanner"
)

// TestSIMDAlgorithms tests SIMD-specific functionality
func TestSIMDAlgorithms(t *testing.T) {
	if !scanner.HasSIMD() {
		t.Skip("SIMD not available on this platform")
	}

	t.Run("StructuralScanning", testSIMDStructuralScanning)
	t.Run("IntegerParsing", testSIMDIntegerParsing)
	t.Run("QuoteMasking", testSIMDQuoteMasking)
	t.Run("UTF8Validation", testSIMDUTF8Validation)
	t.Run("MemoryAlignment", testSIMDMemoryAlignment)
}

func testSIMDStructuralScanning(t *testing.T) {
	testCases := []struct {
		name string
		json string
	}{
		{"simple", `{"key":"value"}`},
		{"array", `[1,2,3,4,5]`},
		{"nested", `{"a":{"b":[1,2]}}`},
		{"complex", `{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}],"count":2}`},
		{"large_string", `{"data":"` + string(make([]byte, 1000)) + `"}`},
		{"many_elements", generateManyElements(100)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := scanner.New()
			defer s.Release()

			// Test SIMD scanning
			err := s.ScanSIMD([]byte(tc.json))
			if err != nil {
				t.Fatalf("SIMD scan failed: %v", err)
			}

			simdIndices := s.GetStructuralIndices()
			
			// Compare with scalar scanning
			s2 := scanner.New()
			defer s2.Release()
			
			err = s2.Scan([]byte(tc.json))
			if err != nil {
				t.Fatalf("Scalar scan failed: %v", err)
			}

			scalarIndices := s2.GetStructuralIndices()

			// Results should be identical
			if len(simdIndices) != len(scalarIndices) {
				t.Fatalf("Index count mismatch: SIMD=%d, Scalar=%d", 
					len(simdIndices), len(scalarIndices))
			}

			for i := range simdIndices {
				if simdIndices[i] != scalarIndices[i] {
					t.Errorf("Index %d mismatch: SIMD=%d, Scalar=%d", 
						i, simdIndices[i], scalarIndices[i])
				}
			}
		})
	}
}

func testSIMDIntegerParsing(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected int64
		valid    bool
	}{
		{"zero", "0", 0, true},
		{"positive", "123", 123, true},
		{"negative", "-456", -456, true},
		{"max_int64", "9223372036854775807", 9223372036854775807, true},
		{"min_int64", "-9223372036854775808", -9223372036854775808, true},
		{"leading_zeros", "000123", 123, true},
		{"negative_zero", "-0", 0, true},
		{"empty", "", 0, false},
		{"non_numeric", "abc", 0, false},
		{"float", "123.45", 123, true}, // Should parse integer part
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := scanner.New()
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

func testSIMDQuoteMasking(t *testing.T) {
	testCases := []struct {
		name  string
		input string
	}{
		{"no_quotes", "123 456 789"},
		{"simple_string", `"hello"`},
		{"multiple_strings", `"first" "second" "third"`},
		{"escaped_quotes", `"say \"hello\""`},
		{"complex_json", `{"name":"Alice","message":"She said \"Hi\""}`},
		{"empty", ""},
		{"long_string", `"` + string(make([]byte, 1000)) + `"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := scanner.New()
			defer s.Release()

			masks, err := s.SIMDQuoteMask([]byte(tc.input))
			if err != nil {
				t.Fatalf("SIMDQuoteMask failed: %v", err)
			}

			// Basic validation - masks should be reasonable
			if len(tc.input) == 0 && len(masks) > 0 {
				t.Error("Empty input should produce no masks")
			}

			// The exact mask values depend on SIMD implementation details,
			// but we can verify they don't cause crashes and are reasonable
			maxExpectedMasks := (len(tc.input) + 31) / 32 + 1
			if len(masks) > maxExpectedMasks {
				t.Errorf("Too many masks: got %d, expected <=%d", len(masks), maxExpectedMasks)
			}
		})
	}
}

func testSIMDUTF8Validation(t *testing.T) {
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
		{"mixed", []byte("Hello 世界 🌍"), true},
		{"long_ascii", bytes_repeat([]byte("a"), 1000), true},
		{"long_utf8", bytes_repeat([]byte("世"), 500), true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := scanner.New()
			defer s.Release()

			result := s.SIMDValidateUTF8(tc.input)
			if result != tc.expected {
				t.Errorf("UTF8 validation mismatch: expected=%v, got=%v", tc.expected, result)
			}
		})
	}
}

func testSIMDMemoryAlignment(t *testing.T) {
	// Test various alignment scenarios
	alignments := []int{16, 32}
	sizes := []int{15, 16, 17, 31, 32, 33, 63, 64, 65, 100, 1000}

	for _, alignment := range alignments {
		for _, size := range sizes {
			t.Run(fmt.Sprintf("align_%d_size_%d", alignment, size), func(t *testing.T) {
				// Create aligned buffer
				buf := scanner.NewAlignedBuffer(size, alignment)
				
				// Check alignment
				if !scanner.IsAligned(unsafe.Pointer(&buf.Bytes()[0]), alignment) {
					t.Errorf("Buffer not aligned to %d bytes", alignment)
				}

				// Fill with test data
				testData := generateTestJSON(size)
				if len(testData) > size {
					testData = testData[:size]
				}
				copy(buf.Bytes()[:len(testData)], testData)

				// Test SIMD operations on aligned data
				s := scanner.New()
				defer s.Release()

				err := s.ScanSIMD(buf.Bytes()[:len(testData)])
				if err != nil && len(testData) > 0 {
					// Only expect success for non-empty valid JSON
					valid := Valid(testData)
					if valid {
						t.Errorf("SIMD scan failed on valid aligned data: %v", err)
					}
				}
			})
		}
	}
}

// Test SIMD performance characteristics
func TestSIMDPerformanceCharacteristics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	if !scanner.HasSIMD() {
		t.Skip("SIMD not available")
	}

	// Test that SIMD scales well with input size
	sizes := []int{100, 1000, 10000}
	
	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			// Generate test data
			testJSON := generateLargeTestJSON(size)
			
			// Measure SIMD performance
			s := scanner.New()
			defer s.Release()
			
			start := time.Now()
			err := s.ScanSIMD(testJSON)
			simdTime := time.Since(start)
			
			if err != nil {
				t.Fatalf("SIMD scan failed: %v", err)
			}
			
			simdIndices := len(s.GetStructuralIndices())
			
			// Measure scalar performance
			s2 := scanner.New()
			defer s2.Release()
			
			start = time.Now()
			err = s2.Scan(testJSON)
			scalarTime := time.Since(start)
			
			if err != nil {
				t.Fatalf("Scalar scan failed: %v", err)
			}
			
			scalarIndices := len(s2.GetStructuralIndices())
			
			// Results should match
			if simdIndices != scalarIndices {
				t.Errorf("Index count mismatch: SIMD=%d, Scalar=%d", simdIndices, scalarIndices)
			}
			
			// SIMD should generally be faster or at least competitive
			ratio := float64(scalarTime) / float64(simdTime)
			t.Logf("Size %d: SIMD/Scalar time ratio = %.2f", size, ratio)
			
			// For very large inputs, SIMD should show benefits
			if size >= 10000 && ratio < 0.5 {
				t.Logf("Warning: SIMD significantly slower than scalar for size %d (ratio=%.2f)", size, ratio)
			}
		})
	}
}

// Test concurrent SIMD operations
func TestSIMDConcurrency(t *testing.T) {
	if !scanner.HasSIMD() {
		t.Skip("SIMD not available")
	}

	testJSON := []byte(`{"test":"concurrent","numbers":[1,2,3,4,5],"nested":{"value":42}}`)
	numGoroutines := 10
	numIterations := 100

	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- nil }()

			s := scanner.New()
			defer s.Release()

			for j := 0; j < numIterations; j++ {
				err := s.ScanSIMD(testJSON)
				if err != nil {
					done <- fmt.Errorf("goroutine %d iteration %d failed: %v", id, j, err)
					return
				}

				// Verify results
				indices := s.GetStructuralIndices()
				if len(indices) == 0 {
					done <- fmt.Errorf("goroutine %d iteration %d got no indices", id, j)
					return
				}
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

// Helper functions

func generateManyElements(count int) string {
	var buf bytes.Buffer
	buf.WriteString("[")
	
	for i := 0; i < count; i++ {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(fmt.Sprintf(`{"id":%d,"value":"item_%d"}`, i, i))
	}
	
	buf.WriteString("]")
	return buf.String()
}

func generateTestJSON(targetSize int) []byte {
	json := `{"test":"data","number":42}`
	
	// Repeat to reach target size
	for len(json) < targetSize {
		json = fmt.Sprintf(`[%s,%s]`, json, json)
		if len(json) > targetSize*2 {
			break // Avoid exponential growth
		}
	}
	
	return []byte(json[:min(len(json), targetSize)])
}

func generateLargeTestJSON(targetSize int) []byte {
	var buf bytes.Buffer
	buf.WriteString(`{"data":[`)
	
	elementSize := 50 // Approximate size per element
	numElements := targetSize / elementSize
	
	for i := 0; i < numElements; i++ {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(fmt.Sprintf(`{"id":%d,"name":"item_%d","value":%d}`, i, i, i*2))
	}
	
	buf.WriteString(`],"count":`)
	buf.WriteString(fmt.Sprintf("%d", numElements))
	buf.WriteString("}")
	
	return buf.Bytes()
}

func bytes_repeat(b []byte, count int) []byte {
	result := make([]byte, 0, len(b)*count)
	for i := 0; i < count; i++ {
		result = append(result, b...)
	}
	return result
}

