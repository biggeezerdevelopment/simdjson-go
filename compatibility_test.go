package simdjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// TestCompatibilityWithStandardLibrary ensures our implementation matches encoding/json exactly
func TestCompatibilityWithStandardLibrary(t *testing.T) {
	testCases := []struct {
		name string
		json string
	}{
		// Basic types
		{"null", "null"},
		{"true", "true"},
		{"false", "false"},
		{"zero", "0"},
		{"positive_int", "42"},
		{"negative_int", "-123"},
		{"float", "3.14"},
		{"string", `"hello"`},
		{"empty_string", `""`},
		
		// Objects
		{"empty_object", "{}"},
		{"simple_object", `{"key":"value"}`},
		{"nested_object", `{"outer":{"inner":"value"}}`},
		
		// Arrays
		{"empty_array", "[]"},
		{"number_array", "[1,2,3]"},
		{"mixed_array", `[1,"two",true,null]`},
		
		// Complex structures
		{"complex", `{
			"name": "Alice",
			"age": 30,
			"active": true,
			"scores": [85, 92, 78],
			"address": {
				"street": "123 Main St",
				"city": "Boston",
				"zip": "02101"
			},
			"metadata": null
		}`},
		
		// Edge cases with whitespace
		{"whitespace", " \t\n{\n\t \"key\" \t:\n \"value\" \t\n} \n\t "},
		
		// Numbers
		{"large_int", "9223372036854775807"},
		{"scientific", "1.23e-10"},
		{"negative_scientific", "-1.23e+10"},
		
		// Unicode
		{"unicode", `{"text":"Hello 世界 🌍"}`},
		
		// Escaped characters
		{"escaped", `{"quote":"He said \"Hello\"","backslash":"path\\to\\file","newline":"line1\nline2"}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse with standard library
			var stdResult interface{}
			stdErr := json.Unmarshal([]byte(tc.json), &stdResult)
			
			// Parse with our implementation
			var ourResult interface{}
			ourErr := Unmarshal([]byte(tc.json), &ourResult)
			
			// Compare errors
			if (stdErr == nil) != (ourErr == nil) {
				t.Fatalf("Error mismatch: std=%v, ours=%v", stdErr, ourErr)
			}
			
			// If both succeeded, compare results
			if stdErr == nil {
				if !deepEqual(stdResult, ourResult) {
					t.Errorf("Result mismatch:\nStd:  %#v\nOurs: %#v", stdResult, ourResult)
				}
			}
		})
	}
}

// TestMarshalCompatibility tests marshalling compatibility
func TestMarshalCompatibility(t *testing.T) {
	testValues := []interface{}{
		nil,
		true,
		false,
		42,
		-123,
		3.14,
		"hello world",
		"",
		[]int{1, 2, 3},
		[]interface{}{1, "two", true, nil},
		map[string]interface{}{
			"name":   "Alice",
			"age":    30,
			"active": true,
		},
		map[string]interface{}{
			"nested": map[string]interface{}{
				"value": 42,
			},
		},
	}

	for i, val := range testValues {
		t.Run(fmt.Sprintf("case_%d", i), func(t *testing.T) {
			// Marshal with standard library
			stdBytes, stdErr := json.Marshal(val)
			
			// Marshal with our implementation
			ourBytes, ourErr := Marshal(val)
			
			// Compare errors
			if (stdErr == nil) != (ourErr == nil) {
				t.Fatalf("Error mismatch: std=%v, ours=%v", stdErr, ourErr)
			}
			
			if stdErr == nil {
				// Both should produce valid JSON
				var stdCheck, ourCheck interface{}
				if err := json.Unmarshal(stdBytes, &stdCheck); err != nil {
					t.Fatalf("Standard library produced invalid JSON: %v", err)
				}
				if err := json.Unmarshal(ourBytes, &ourCheck); err != nil {
					t.Fatalf("Our implementation produced invalid JSON: %v", err)
				}
				
				// Results should be equivalent
				if !deepEqual(stdCheck, ourCheck) {
					t.Errorf("Marshal results differ:\nStd:  %s -> %#v\nOurs: %s -> %#v",
						string(stdBytes), stdCheck, string(ourBytes), ourCheck)
				}
			}
		})
	}
}

// TestValidationCompatibility tests JSON validation
func TestValidationCompatibility(t *testing.T) {
	testCases := []struct {
		name string
		json string
	}{
		// Valid JSON
		{"valid_null", "null"},
		{"valid_bool", "true"},
		{"valid_number", "42"},
		{"valid_string", `"hello"`},
		{"valid_array", "[1,2,3]"},
		{"valid_object", `{"key":"value"}`},
		
		// Invalid JSON
		{"invalid_empty", ""},
		{"invalid_trailing_comma", `{"key":"value",}`},
		{"invalid_missing_quote", `{"key:value}`},
		{"invalid_unclosed_object", `{"key":"value"`},
		{"invalid_unclosed_array", `[1,2,3`},
		{"invalid_number", "12."},
		{"invalid_escape", `{"key":"val\ue"}`},
		{"invalid_unicode", `{"key":"\u12"}`},
		{"invalid_duplicate_comma", `[1,,2]`},
		{"invalid_leading_zero", `{"num":01}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stdValid := json.Valid([]byte(tc.json))
			ourValid := Valid([]byte(tc.json))
			
			if stdValid != ourValid {
				t.Errorf("Validation mismatch for %q: std=%v, ours=%v", tc.json, stdValid, ourValid)
			}
		})
	}
}

// TestStructUnmarshalling tests unmarshalling into structs
func TestStructUnmarshalling(t *testing.T) {
	type Person struct {
		Name    string `json:"name"`
		Age     int    `json:"age"`
		Active  bool   `json:"active"`
		Address struct {
			Street string `json:"street"`
			City   string `json:"city"`
		} `json:"address"`
		Scores []int `json:"scores"`
	}

	jsonData := `{
		"name": "Alice",
		"age": 30,
		"active": true,
		"address": {
			"street": "123 Main St",
			"city": "Boston"
		},
		"scores": [85, 92, 78]
	}`

	// Unmarshal with standard library
	var stdPerson Person
	stdErr := json.Unmarshal([]byte(jsonData), &stdPerson)
	
	// Unmarshal with our implementation
	var ourPerson Person
	ourErr := Unmarshal([]byte(jsonData), &ourPerson)
	
	// Compare errors and results
	if stdErr != nil || ourErr != nil {
		t.Fatalf("Unmarshal errors: std=%v, ours=%v", stdErr, ourErr)
	}
	
	if !reflect.DeepEqual(stdPerson, ourPerson) {
		t.Errorf("Struct unmarshal mismatch:\nStd:  %+v\nOurs: %+v", stdPerson, ourPerson)
	}
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		json        string
		shouldError bool
	}{
		// Valid edge cases
		{"deeply_nested", createDeeplyNested(10), false},
		{"large_array", createLargeArray(1000), false},
		{"unicode_keys", `{"键":"值","🔑":"🎁"}`, false},
		{"all_escapes", `{"test":"\"\\\/\b\f\n\r\t\u0041"}`, false},
		
		// Invalid edge cases  
		{"too_deep", createDeeplyNested(1000), true}, // Should hit recursion limit
		{"control_chars", "{\"key\":\"value\x00\"}", true},
		{"lone_surrogate", `{"test":"\uD800"}`, true},
		{"invalid_surrogate_pair", `{"test":"\uD800\u0041"}`, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var stdResult, ourResult interface{}
			
			stdErr := json.Unmarshal([]byte(tc.json), &stdResult)
			ourErr := Unmarshal([]byte(tc.json), &ourResult)
			
			stdFailed := stdErr != nil
			ourFailed := ourErr != nil
			
			if stdFailed != ourFailed {
				t.Errorf("Error expectation mismatch: std failed=%v, ours failed=%v", stdFailed, ourFailed)
				t.Logf("Standard error: %v", stdErr)
				t.Logf("Our error: %v", ourErr)
			}
			
			if !stdFailed && !ourFailed {
				if !deepEqual(stdResult, ourResult) {
					t.Errorf("Results differ for valid input")
				}
			}
		})
	}
}

// Property-based testing with random JSON generation
func TestRandomJSONCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping property-based test in short mode")
	}
	
	rand.Seed(time.Now().UnixNano())
	
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("random_%d", i), func(t *testing.T) {
			// Generate random JSON
			jsonData := generateRandomJSON(5, 10)
			
			var stdResult, ourResult interface{}
			
			stdErr := json.Unmarshal(jsonData, &stdResult)
			ourErr := Unmarshal(jsonData, &ourResult)
			
			// Errors should match
			if (stdErr == nil) != (ourErr == nil) {
				t.Errorf("Error mismatch for JSON: %s", string(jsonData))
				t.Logf("Standard error: %v", stdErr)
				t.Logf("Our error: %v", ourErr)
			}
			
			// Results should match if both succeeded
			if stdErr == nil && !deepEqual(stdResult, ourResult) {
				t.Errorf("Results differ for JSON: %s", string(jsonData))
				t.Logf("Standard result: %#v", stdResult)
				t.Logf("Our result: %#v", ourResult)
			}
		})
	}
}

// Roundtrip testing: Marshal -> Unmarshal should be identity
func TestRoundtripCompatibility(t *testing.T) {
	testValues := []interface{}{
		map[string]interface{}{
			"string":  "hello",
			"number":  42.5,
			"bool":    true,
			"null":    nil,
			"array":   []interface{}{1, 2, 3},
			"object":  map[string]interface{}{"nested": "value"},
		},
		[]interface{}{
			"mixed", 123, true, nil,
			map[string]interface{}{"key": "value"},
		},
	}

	for i, original := range testValues {
		t.Run(fmt.Sprintf("roundtrip_%d", i), func(t *testing.T) {
			// Standard library roundtrip
			stdBytes, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Standard marshal failed: %v", err)
			}
			
			var stdResult interface{}
			err = json.Unmarshal(stdBytes, &stdResult)
			if err != nil {
				t.Fatalf("Standard unmarshal failed: %v", err)
			}
			
			// Our implementation roundtrip
			ourBytes, err := Marshal(original)
			if err != nil {
				t.Fatalf("Our marshal failed: %v", err)
			}
			
			var ourResult interface{}
			err = Unmarshal(ourBytes, &ourResult)
			if err != nil {
				t.Fatalf("Our unmarshal failed: %v", err)
			}
			
			// Both roundtrips should produce equivalent results
			if !deepEqual(stdResult, ourResult) {
				t.Errorf("Roundtrip results differ")
				t.Logf("Original: %#v", original)
				t.Logf("Standard roundtrip: %#v", stdResult)
				t.Logf("Our roundtrip: %#v", ourResult)
			}
		})
	}
}

// Helper functions

func deepEqual(a, b interface{}) bool {
	return reflect.DeepEqual(normalizeNumbers(a), normalizeNumbers(b))
}

// normalizeNumbers converts all numbers to float64 for comparison
func normalizeNumbers(v interface{}) interface{} {
	switch val := v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		// Convert all integer types to float64 for comparison
		return float64(reflect.ValueOf(val).Convert(reflect.TypeOf(int64(0))).Int())
	case float32:
		return float64(val)
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = normalizeNumbers(item)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, item := range val {
			result[k] = normalizeNumbers(item)
		}
		return result
	default:
		return v
	}
}

func createDeeplyNested(depth int) string {
	var buf bytes.Buffer
	
	// Create nested objects
	for i := 0; i < depth; i++ {
		buf.WriteString(`{"level":`)
	}
	buf.WriteString("42")
	for i := 0; i < depth; i++ {
		buf.WriteString("}")
	}
	
	return buf.String()
}

func createLargeArray(size int) string {
	var buf bytes.Buffer
	buf.WriteString("[")
	
	for i := 0; i < size; i++ {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(fmt.Sprintf("%d", i))
	}
	
	buf.WriteString("]")
	return buf.String()
}

func generateRandomJSON(maxDepth, maxWidth int) []byte {
	return generateRandomValue(maxDepth, maxWidth)
}

func generateRandomValue(maxDepth, maxWidth int) []byte {
	if maxDepth <= 0 {
		// Generate leaf values
		switch rand.Intn(5) {
		case 0:
			return []byte("null")
		case 1:
			if rand.Intn(2) == 0 {
				return []byte("true")
			}
			return []byte("false")
		case 2:
			return []byte(fmt.Sprintf("%d", rand.Intn(1000)-500))
		case 3:
			return []byte(fmt.Sprintf("%.2f", rand.Float64()*1000-500))
		case 4:
			return []byte(fmt.Sprintf(`"string_%d"`, rand.Intn(100)))
		}
	}
	
	// Generate composite values
	switch rand.Intn(2) {
	case 0:
		// Array
		var buf bytes.Buffer
		buf.WriteString("[")
		
		width := rand.Intn(maxWidth) + 1
		for i := 0; i < width; i++ {
			if i > 0 {
				buf.WriteString(",")
			}
			buf.Write(generateRandomValue(maxDepth-1, maxWidth))
		}
		
		buf.WriteString("]")
		return buf.Bytes()
		
	case 1:
		// Object
		var buf bytes.Buffer
		buf.WriteString("{")
		
		width := rand.Intn(maxWidth) + 1
		for i := 0; i < width; i++ {
			if i > 0 {
				buf.WriteString(",")
			}
			buf.WriteString(fmt.Sprintf(`"key_%d":`, rand.Intn(100)))
			buf.Write(generateRandomValue(maxDepth-1, maxWidth))
		}
		
		buf.WriteString("}")
		return buf.Bytes()
	}
	
	return []byte("null")
}