package scanner

import (
	"testing"
	"unsafe"
)

func TestScanner_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []uint32
	}{
		{
			name:     "simple object",
			input:    `{"key":"value"}`,
			expected: []uint32{0, 1, 5, 6, 7, 13, 14}, // { " : " }
		},
		{
			name:     "simple array", 
			input:    `[1,2,3]`,
			expected: []uint32{0, 1, 2, 3, 4, 5, 6}, // [ 1 , 2 , 3 ]
		},
		{
			name:     "nested structure",  
			input:    `{"a":[1,2],"b":true}`,
			expected: []uint32{0, 1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 13, 14, 15, 19}, // all structural elements
		},
		{
			name:     "empty object",
			input:    `{}`,
			expected: []uint32{0, 1},
		},
		{
			name:     "empty array",
			input:    `[]`,
			expected: []uint32{0, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			defer s.Release()

			err := s.Scan([]byte(tt.input))
			if err != nil {
				t.Fatalf("Scan failed: %v", err)
			}

			indices := s.GetStructuralIndices()
			if len(indices) != len(tt.expected) {
				t.Fatalf("Expected %d indices, got %d. Expected: %v, Got: %v",
					len(tt.expected), len(indices), tt.expected, indices)
			}

			for i, expected := range tt.expected {
				if indices[i] != expected {
					t.Errorf("Index %d: expected %d, got %d", i, expected, indices[i])
				}
			}
		})
	}
}

func TestScanner_SIMD(t *testing.T) {
	if !HasSIMD() {
		t.Skip("SIMD not available on this platform")
	}

	tests := []struct {
		name  string
		input string
	}{
		{"small json", `{"name":"test","value":42}`},
		{"array", `[1,2,3,4,5,6,7,8,9,10]`},
		{"nested", `{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]}`},
		{"large string", `{"data":"` + string(make([]byte, 100)) + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test scalar vs SIMD consistency
			scalarScanner := New()
			defer scalarScanner.Release()

			simdScanner := New()
			defer simdScanner.Release()

			// Scan with scalar
			err := scalarScanner.scanScalar()
			if err == nil {
				err = scalarScanner.Scan([]byte(tt.input))
			}
			if err != nil {
				t.Fatalf("Scalar scan failed: %v", err)
			}

			// Scan with SIMD
			err = simdScanner.ScanSIMD([]byte(tt.input))
			if err != nil {
				t.Fatalf("SIMD scan failed: %v", err)
			}

			// Compare results
			scalarIndices := scalarScanner.GetStructuralIndices()
			simdIndices := simdScanner.GetStructuralIndices()

			if len(scalarIndices) != len(simdIndices) {
				t.Fatalf("Index count mismatch: scalar=%d, simd=%d",
					len(scalarIndices), len(simdIndices))
			}

			for i := range scalarIndices {
				if scalarIndices[i] != simdIndices[i] {
					t.Errorf("Index %d mismatch: scalar=%d, simd=%d",
						i, scalarIndices[i], simdIndices[i])
				}
			}
		})
	}
}

func TestScanner_SIMDParseInteger(t *testing.T) {
	if !HasSIMD() {
		t.Skip("SIMD not available - SIMDParseInteger will always return false")
	}

	tests := []struct {
		name     string
		input    string
		expected int64
		valid    bool
	}{
		{"positive integer", "123", 123, true},
		{"negative integer", "-456", -456, true},
		{"zero", "0", 0, true},
		{"large positive", "9223372036854775807", 9223372036854775807, true},
		{"large negative", "-9223372036854775808", -9223372036854775808, true},
		{"invalid", "abc", 0, false},
		{"empty", "", 0, false},
		{"mixed", "123abc", 123, true}, // Should parse up to first non-digit
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			defer s.Release()

			result, valid := s.SIMDParseInteger([]byte(tt.input))
			if valid != tt.valid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.valid, valid)
			}

			if valid && result != tt.expected {
				t.Errorf("Expected result=%d, got result=%d", tt.expected, result)
			}
		})
	}
}

func TestScanner_SIMDQuoteMask(t *testing.T) {
	tests := []struct {
		name  string
		input string
		// We can't easily predict exact bitmasks, but we can test functionality
	}{
		{"simple string", `"hello"`},
		{"escaped quotes", `"say \"hello\""`},
		{"multiple strings", `"first" "second" "third"`},
		{"no quotes", "123 456 789"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			defer s.Release()

			masks, err := s.SIMDQuoteMask([]byte(tt.input))
			if err != nil {
				t.Fatalf("SIMDQuoteMask failed: %v", err)
			}

			// Basic sanity check - masks should be reasonable
			expectedMasks := (len(tt.input) + 31) / 32
			if len(masks) > expectedMasks {
				t.Errorf("Too many masks: expected <=%d, got %d", expectedMasks, len(masks))
			}
		})
	}
}

func TestScanner_SIMDValidateUTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{"ascii", []byte("hello world"), true},
		{"empty", []byte(""), true},
		{"utf8", []byte("hello 世界"), true},
		{"emoji", []byte("hello 😀"), true},
		// Note: Invalid UTF-8 tests would be more complex to construct
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			defer s.Release()

			result := s.SIMDValidateUTF8(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestScanner_Validation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid object", `{"key":"value"}`, true},
		{"valid array", `[1,2,3]`, true},
		{"valid nested", `{"a":[1,2],"b":{"c":3}}`, true},
		{"invalid - missing quote", `{"key:value}`, false},
		{"invalid - trailing comma", `{"key":"value",}`, false},
		{"invalid - unbalanced braces", `{"key":"value"`, false},
		{"invalid - unbalanced brackets", `[1,2,3`, false},
		{"invalid - double comma", `[1,,2]`, false},
		{"empty", ``, false},
		{"null", `null`, true},
		{"true", `true`, true},
		{"false", `false`, true},
		{"number", `42`, true},
		{"string", `"hello"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			defer s.Release()

			result := s.Validate([]byte(tt.input))
			if result != tt.expected {
				t.Errorf("Input: %s, Expected %v, got %v", tt.input, tt.expected, result)
			}
		})
	}
}

func TestScanner_LargeInput(t *testing.T) {
	// Generate large JSON
	large := `{"data":[`
	for i := 0; i < 1000; i++ {
		if i > 0 {
			large += ","
		}
		large += `{"id":` + string(rune('0'+i%10)) + `,"name":"item` + string(rune('0'+i%10)) + `"}`
	}
	large += `]}`

	s := New()
	defer s.Release()

	err := s.Scan([]byte(large))
	if err != nil {
		t.Fatalf("Failed to scan large input: %v", err)
	}

	indices := s.GetStructuralIndices()
	if len(indices) == 0 {
		t.Error("Expected structural indices for large input")
	}

	// Validate large input
	if !s.Validate([]byte(large)) {
		t.Error("Large input should be valid")
	}
}

func TestScanner_MemoryAlignment(t *testing.T) {
	testData := []byte(`{"test":"data","numbers":[1,2,3,4,5]}`)
	
	// Test aligned buffer
	aligned := NewAlignedBuffer(len(testData), 32)
	copy(aligned.Bytes(), testData)
	
	// Check alignment
	if !IsAligned(unsafe.Pointer(&aligned.Bytes()[0]), 32) {
		t.Error("Buffer should be 32-byte aligned")
	}
	
	s := New()
	defer s.Release()
	
	err := s.Scan(aligned.Bytes())
	if err != nil {
		t.Fatalf("Failed to scan aligned data: %v", err)
	}
	
	// Test with unaligned data
	unaligned := make([]byte, len(testData)+1)
	copy(unaligned[1:], testData)
	
	err = s.Scan(unaligned[1:])
	if err != nil {
		t.Fatalf("Failed to scan unaligned data: %v", err)
	}
}

func TestScanner_ThreadSafety(t *testing.T) {
	testData := []byte(`{"concurrent":"test"}`)
	
	// Test that separate scanner instances are thread-safe
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			s := New()
			defer s.Release()
			
			for j := 0; j < 100; j++ {
				err := s.Scan(testData)
				if err != nil {
					t.Errorf("Scan failed in goroutine: %v", err)
					return
				}
				
				if !s.Validate(testData) {
					t.Error("Validation failed in goroutine")
					return
				}
			}
		}()
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestScanner_Pool(t *testing.T) {
	// Test token pooling
	tokens1 := getTokenSlice()
	tokens2 := getTokenSlice()
	
	// Should get different slices initially
	if len(tokens1) != 0 || len(tokens2) != 0 {
		t.Error("New token slices should be empty")
	}
	
	// Test putting back and reusing
	tokens1 = append(tokens1, Token{Type: TokenString, Start: 0, End: 5})
	PutTokenSlice(tokens1)
	
	tokens3 := getTokenSlice()
	if len(tokens3) != 0 {
		t.Error("Recycled token slice should be reset to empty")
	}
	
	PutTokenSlice(tokens2)
	PutTokenSlice(tokens3)
}

// Benchmark tests to ensure SIMD performance
func BenchmarkScanner_ScalarVsSIMD(b *testing.B) {
	testData := []byte(`{"users":[{"id":1,"name":"Alice","email":"alice@example.com","active":true},{"id":2,"name":"Bob","email":"bob@example.com","active":false}],"count":2}`)
	
	b.Run("Scalar", func(b *testing.B) {
		s := New()
		defer s.Release()
		
		for i := 0; i < b.N; i++ {
			s.scanScalar()
		}
	})
	
	if HasSIMD() {
		b.Run("SIMD", func(b *testing.B) {
			s := New()
			defer s.Release()
			
			for i := 0; i < b.N; i++ {
				s.ScanSIMD(testData)
			}
		})
	}
}

func TestSimpleTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:  "simple object",
			input: `{"key":"value"}`,
			expected: []TokenType{
				TokenObjectBegin, TokenString, TokenColon, TokenString, TokenObjectEnd,
			},
		},
		{
			name:  "array with numbers",
			input: `[1,2,3]`,
			expected: []TokenType{
				TokenArrayBegin, TokenNumber, TokenComma, TokenNumber, TokenComma, TokenNumber, TokenArrayEnd,
			},
		},
		{
			name:  "boolean values",
			input: `{"flag":true,"other":false}`,
			expected: []TokenType{
				TokenObjectBegin, TokenString, TokenColon, TokenTrue, TokenComma, TokenString, TokenColon, TokenFalse, TokenObjectEnd,
			},
		},
		{
			name:  "null value",
			input: `{"value":null}`,
			expected: []TokenType{
				TokenObjectBegin, TokenString, TokenColon, TokenNull, TokenObjectEnd,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			defer s.Release()

			tokens, err := s.SimpleTokenize([]byte(tt.input))
			if err != nil {
				t.Fatalf("SimpleTokenize failed: %v", err)
			}

			if len(tokens) != len(tt.expected) {
				t.Fatalf("Expected %d tokens, got %d", len(tt.expected), len(tokens))
			}

			for i, expectedType := range tt.expected {
				if tokens[i].Type != expectedType {
					t.Errorf("Token %d: expected type %v, got %v", i, expectedType, tokens[i].Type)
				}
			}
		})
	}
}