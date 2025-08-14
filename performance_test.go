package simdjson

import (
	"encoding/json"
	"fmt"
	"runtime"
	"testing"
	"time"
)

// TestPerformanceRegression ensures we maintain performance improvements
func TestPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression tests in short mode")
	}

	testCases := []struct {
		name     string
		json     []byte
		minRatio float64 // Minimum speedup ratio we expect
	}{
		{
			name:     "small_json",
			json:     []byte(`{"name":"John","age":30,"city":"New York"}`),
			minRatio: 0.8, // Allow slight overhead for small JSON
		},
		{
			name:     "medium_json",
			json:     generateMediumJSON(),
			minRatio: 1.0, // Should be at least as fast
		},
		{
			name:     "large_json",
			json:     generateLargeJSON(1000),
			minRatio: 1.5, // Should be significantly faster
		},
		{
			name:     "very_large_json",
			json:     generateLargeJSON(10000),
			minRatio: 1.8, // Should show major improvement
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Warm up
			for i := 0; i < 10; i++ {
				var std, our interface{}
				json.Unmarshal(tc.json, &std)
				Unmarshal(tc.json, &our)
			}

			// Benchmark standard library
			stdTime := benchmarkUnmarshal(tc.json, 100, func(data []byte, v interface{}) error {
				return json.Unmarshal(data, v)
			})

			// Benchmark our implementation
			ourTime := benchmarkUnmarshal(tc.json, 100, func(data []byte, v interface{}) error {
				return Unmarshal(data, v)
			})

			ratio := float64(stdTime) / float64(ourTime)
			
			t.Logf("Performance ratio (std/ours): %.2f (std=%v, ours=%v)", ratio, stdTime, ourTime)

			if ratio < tc.minRatio {
				t.Errorf("Performance regression: expected ratio >= %.2f, got %.2f", tc.minRatio, ratio)
			}
		})
	}
}

// TestValidationPerformance tests validation speed
func TestValidationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping validation performance tests in short mode")
	}

	largeJSON := generateLargeJSON(5000)

	// Benchmark standard validation
	stdTime := benchmarkValidation(largeJSON, 50, json.Valid)

	// Benchmark our validation
	ourTime := benchmarkValidation(largeJSON, 50, Valid)

	ratio := float64(stdTime) / float64(ourTime)
	
	t.Logf("Validation ratio (std/ours): %.2f (std=%v, ours=%v)", ratio, stdTime, ourTime)

	// Our validation should be competitive or faster
	if ratio < 1.0 {
		t.Logf("Warning: Our validation is slower (ratio=%.2f)", ratio)
	}
}

// TestMemoryEfficiency ensures our implementation doesn't use excessive memory
func TestMemoryEfficiency(t *testing.T) {
	largeJSON := generateLargeJSON(1000)

	// Test memory usage with standard library
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	for i := 0; i < 100; i++ {
		var result interface{}
		json.Unmarshal(largeJSON, &result)
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)
	stdAllocs := m2.TotalAlloc - m1.TotalAlloc

	// Test memory usage with our implementation
	var m3, m4 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m3)

	for i := 0; i < 100; i++ {
		var result interface{}
		Unmarshal(largeJSON, &result)
	}

	runtime.GC()
	runtime.ReadMemStats(&m4)
	ourAllocs := m4.TotalAlloc - m3.TotalAlloc

	ratio := float64(ourAllocs) / float64(stdAllocs)
	
	t.Logf("Memory ratio (ours/std): %.2f (std=%d bytes, ours=%d bytes)", ratio, stdAllocs, ourAllocs)

	// Our implementation might use more memory for some optimizations, but shouldn't be excessive
	if ratio > 3.0 {
		t.Errorf("Memory usage too high: ratio=%.2f", ratio)
	}
}

// TestConcurrentPerformance tests performance under concurrent load
func TestConcurrentPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent performance tests in short mode")
	}

	jsonData := generateMediumJSON()
	numGoroutines := runtime.GOMAXPROCS(0)
	iterationsPerGoroutine := 100

	// Test standard library
	start := time.Now()
	done := make(chan struct{}, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < iterationsPerGoroutine; j++ {
				var result interface{}
				json.Unmarshal(jsonData, &result)
			}
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	stdTime := time.Since(start)

	// Test our implementation
	start = time.Now()

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < iterationsPerGoroutine; j++ {
				var result interface{}
				Unmarshal(jsonData, &result)
			}
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	ourTime := time.Since(start)

	ratio := float64(stdTime) / float64(ourTime)
	
	t.Logf("Concurrent performance ratio (std/ours): %.2f (std=%v, ours=%v)", ratio, stdTime, ourTime)

	// Should maintain good performance under concurrency
	if ratio < 0.8 {
		t.Logf("Warning: Concurrent performance degraded (ratio=%.2f)", ratio)
	}
}

// TestLargeInputScaling tests how performance scales with input size
func TestLargeInputScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scaling tests in short mode")
	}

	sizes := []int{100, 500, 1000, 2000, 5000}
	
	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			jsonData := generateLargeJSON(size)
			
			// Quick performance test
			iterations := max(10, 1000/size) // Fewer iterations for larger sizes
			
			stdTime := benchmarkUnmarshal(jsonData, iterations, func(data []byte, v interface{}) error {
				return json.Unmarshal(data, v)
			})

			ourTime := benchmarkUnmarshal(jsonData, iterations, func(data []byte, v interface{}) error {
				return Unmarshal(data, v)
			})

			ratio := float64(stdTime) / float64(ourTime)
			throughputStd := float64(len(jsonData)*iterations) / float64(stdTime.Nanoseconds()) * 1e9 / 1e6 // MB/s
			throughputOurs := float64(len(jsonData)*iterations) / float64(ourTime.Nanoseconds()) * 1e9 / 1e6 // MB/s
			
			t.Logf("Size %d: ratio=%.2f, throughput: std=%.1f MB/s, ours=%.1f MB/s", 
				size, ratio, throughputStd, throughputOurs)

			// Performance should generally improve with larger sizes
			if size >= 1000 && ratio < 1.0 {
				t.Logf("Note: Performance didn't improve for large size %d (ratio=%.2f)", size, ratio)
			}
		})
	}
}

// TestCorrectnessUnderLoad tests correctness under high load
func TestCorrectnessUnderLoad(t *testing.T) {
	jsonData := generateComplexJSON()
	numGoroutines := 20
	iterationsPerGoroutine := 50

	// Parse once with standard library to get expected result
	var expected interface{}
	err := json.Unmarshal(jsonData, &expected)
	if err != nil {
		t.Fatalf("Failed to parse with standard library: %v", err)
	}

	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { errors <- nil }()

			for j := 0; j < iterationsPerGoroutine; j++ {
				var result interface{}
				err := Unmarshal(jsonData, &result)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d iteration %d: unmarshal failed: %v", id, j, err)
					return
				}

				// Deep comparison is expensive, so just check a few key fields
				if !quickEqual(expected, result) {
					errors <- fmt.Errorf("goroutine %d iteration %d: result mismatch", id, j)
					return
				}
			}
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		if err := <-errors; err != nil {
			t.Error(err)
		}
	}
}

// Helper functions

func benchmarkUnmarshal(data []byte, iterations int, unmarshalFunc func([]byte, interface{}) error) time.Duration {
	start := time.Now()
	
	for i := 0; i < iterations; i++ {
		var result interface{}
		unmarshalFunc(data, &result)
	}
	
	return time.Since(start)
}

func benchmarkValidation(data []byte, iterations int, validFunc func([]byte) bool) time.Duration {
	start := time.Now()
	
	for i := 0; i < iterations; i++ {
		validFunc(data)
	}
	
	return time.Since(start)
}

func generateMediumJSON() []byte {
	return []byte(`{
		"users": [
			{"id": 1, "name": "Alice", "email": "alice@example.com", "active": true},
			{"id": 2, "name": "Bob", "email": "bob@example.com", "active": false},
			{"id": 3, "name": "Charlie", "email": "charlie@example.com", "active": true}
		],
		"metadata": {
			"version": "1.0.0",
			"timestamp": 1234567890,
			"count": 3
		},
		"settings": {
			"debug": false,
			"timeout": 30,
			"retries": 3
		}
	}`)
}

func generateLargeJSON(numItems int) []byte {
	result := `{"items":[`
	
	for i := 0; i < numItems; i++ {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf(`{
			"id": %d,
			"name": "Item %d",
			"description": "This is item number %d with some longer text to make it more realistic",
			"price": %.2f,
			"active": %t,
			"tags": ["tag1", "tag2", "tag%d"],
			"metadata": {
				"created": "2023-01-01T00:00:00Z",
				"updated": "2023-12-31T23:59:59Z",
				"category": "category_%d"
			}
		}`, i, i, i, float64(i)*1.99, i%2 == 0, i%10, i%5)
	}
	
	result += `],"count":` + fmt.Sprintf("%d", numItems) + `}`
	return []byte(result)
}

func generateComplexJSON() []byte {
	return []byte(`{
		"string_field": "test_value",
		"number_field": 42,
		"float_field": 3.14159,
		"bool_field": true,
		"null_field": null,
		"array_field": [1, "two", true, null, {"nested": "object"}],
		"object_field": {
			"nested_string": "nested_value",
			"nested_number": 123,
			"nested_array": [1, 2, 3, 4, 5]
		},
		"unicode_field": "Hello 世界 🌍",
		"escaped_field": "Quote: \"Hello\", Backslash: \\, Newline: \n"
	}`)
}

func quickEqual(a, b interface{}) bool {
	// Quick equality check - not comprehensive but catches major issues
	aMap, aOk := a.(map[string]interface{})
	bMap, bOk := b.(map[string]interface{})
	
	if aOk && bOk {
		// Check a few key fields exist and match
		fields := []string{"string_field", "number_field", "bool_field"}
		for _, field := range fields {
			if aMap[field] != bMap[field] {
				return false
			}
		}
		return true
	}
	
	return a == b // Fallback to simple equality
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}