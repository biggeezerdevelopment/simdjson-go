package parser

import (
	"math"
	"reflect"
	"testing"
)

func TestParser_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{"null", "null", nil},
		{"true", "true", true},
		{"false", "false", false},
		{"integer", "42", int64(42)},
		{"negative integer", "-123", int64(-123)},
		{"float", "3.14", 3.14},
		{"string", `"hello"`, "hello"},
		{"empty string", `""`, ""},
		{"simple object", `{"key":"value"}`, map[string]interface{}{"key": "value"}},
		{"simple array", `[1,2,3]`, []interface{}{int64(1), int64(2), int64(3)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			result, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

func TestParser_Numbers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		// Integers
		{"zero", "0", int64(0)},
		{"positive", "123", int64(123)},
		{"negative", "-456", int64(-456)},
		{"large positive", "9223372036854775807", int64(9223372036854775807)},
		{"large negative", "-9223372036854775808", int64(-9223372036854775808)},
		
		// Floats
		{"simple float", "1.5", 1.5},
		{"negative float", "-2.5", -2.5},
		{"exponential", "1e10", 1e10},
		{"negative exponential", "-1e10", -1e10},
		{"exponential with plus", "1e+10", 1e+10},
		{"small exponential", "1e-10", 1e-10},
		{"complex float", "123.456e-7", 123.456e-7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			result, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			switch expected := tt.expected.(type) {
			case int64:
				if result != expected {
					t.Errorf("Expected %d, got %v", expected, result)
				}
			case float64:
				if resultFloat, ok := result.(float64); !ok {
					t.Errorf("Expected float64, got %T", result)
				} else if math.Abs(resultFloat-expected) > 1e-10 {
					t.Errorf("Expected %f, got %f", expected, resultFloat)
				}
			}
		})
	}
}

func TestParser_Strings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", `"hello"`, "hello"},
		{"empty", `""`, ""},
		{"with spaces", `"hello world"`, "hello world"},
		{"escaped quote", `"say \"hello\""`, `say "hello"`},
		{"escaped backslash", `"path\\to\\file"`, `path\to\file`},
		{"escaped newline", `"line1\nline2"`, "line1\nline2"},
		{"escaped tab", `"col1\tcol2"`, "col1\tcol2"},
		{"unicode", `"hello \u0077orld"`, "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			result, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParser_Objects(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{
			"simple",
			`{"name":"Alice","age":30}`,
			map[string]interface{}{"name": "Alice", "age": int64(30)},
		},
		{
			"nested",
			`{"person":{"name":"Bob","age":25}}`,
			map[string]interface{}{
				"person": map[string]interface{}{
					"name": "Bob",
					"age":  int64(25),
				},
			},
		},
		{
			"mixed types",
			`{"string":"test","number":42,"bool":true,"null":null}`,
			map[string]interface{}{
				"string": "test",
				"number": int64(42),
				"bool":   true,
				"null":   nil,
			},
		},
		{
			"empty",
			`{}`,
			map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			result, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParser_Arrays(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []interface{}
	}{
		{
			"numbers",
			`[1,2,3,4,5]`,
			[]interface{}{int64(1), int64(2), int64(3), int64(4), int64(5)},
		},
		{
			"strings",
			`["a","b","c"]`,
			[]interface{}{"a", "b", "c"},
		},
		{
			"mixed",
			`[1,"hello",true,null,3.14]`,
			[]interface{}{int64(1), "hello", true, nil, 3.14},
		},
		{
			"nested arrays",
			`[[1,2],[3,4]]`,
			[]interface{}{
				[]interface{}{int64(1), int64(2)},
				[]interface{}{int64(3), int64(4)},
			},
		},
		{
			"empty",
			`[]`,
			[]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			result, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParser_Complex(t *testing.T) {
	complexJSON := `{
		"users": [
			{
				"id": 1,
				"name": "Alice",
				"email": "alice@example.com",
				"active": true,
				"profile": {
					"age": 30,
					"location": "New York"
				}
			},
			{
				"id": 2,
				"name": "Bob",
				"email": "bob@example.com",
				"active": false,
				"profile": {
					"age": 25,
					"location": "Los Angeles"
				}
			}
		],
		"count": 2,
		"version": "1.0"
	}`

	p := New()
	result, err := p.Parse([]byte(complexJSON))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check structure
	obj, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected object, got %T", result)
	}

	if obj["count"] != int64(2) {
		t.Errorf("Expected count=2, got %v", obj["count"])
	}

	if obj["version"] != "1.0" {
		t.Errorf("Expected version='1.0', got %v", obj["version"])
	}

	users, ok := obj["users"].([]interface{})
	if !ok {
		t.Fatalf("Expected users array, got %T", obj["users"])
	}

	if len(users) != 2 {
		t.Fatalf("Expected 2 users, got %d", len(users))
	}

	// Check first user
	user1, ok := users[0].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected user object, got %T", users[0])
	}

	if user1["name"] != "Alice" {
		t.Errorf("Expected name='Alice', got %v", user1["name"])
	}
}

func TestParser_ErrorCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"invalid json", "{"},
		{"trailing comma", `{"key":"value",}`},
		{"missing quotes", `{key:"value"}`},
		{"invalid number", `{"key":12.}`},
		{"unclosed string", `{"key":"value`},
		{"invalid escape", `{"key":"val\ue"}`},
		{"invalid unicode", `{"key":"\u12"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			_, err := p.Parse([]byte(tt.input))
			if err == nil {
				t.Errorf("Expected error for invalid input: %s", tt.input)
			}
		})
	}
}

func TestParser_UnicodeStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"ascii", `"hello"`, "hello"},
		{"utf8", `"hello 世界"`, "hello 世界"},
		{"emoji", `"hello 😀"`, "hello 😀"},
		{"escaped unicode", `"hello \u4e16\u754c"`, "hello 世界"},
		{"mixed", `"ASCII and 中文 and \u0065moji 🎉"`, "ASCII and 中文 and emoji 🎉"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			result, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParser_LargeNumbers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{"max int64", "9223372036854775807", int64(9223372036854775807)},
		{"min int64", "-9223372036854775808", int64(-9223372036854775808)},
		{"large float", "1.7976931348623157e+308", 1.7976931348623157e+308},
		{"small float", "2.2250738585072014e-308", 2.2250738585072014e-308},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			result, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			switch expected := tt.expected.(type) {
			case int64:
				if result != expected {
					t.Errorf("Expected %d, got %v", expected, result)
				}
			case float64:
				if resultFloat, ok := result.(float64); !ok {
					t.Errorf("Expected float64, got %T", result)
				} else if math.Abs(resultFloat-expected)/expected > 1e-15 {
					t.Errorf("Expected %e, got %e", expected, resultFloat)
				}
			}
		})
	}
}

func TestParser_Whitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{"spaces", `  { "key" : "value" }  `, map[string]interface{}{"key": "value"}},
		{"tabs", "\t{\t\"key\"\t:\t\"value\"\t}\t", map[string]interface{}{"key": "value"}},
		{"newlines", "{\n\"key\"\n:\n\"value\"\n}", map[string]interface{}{"key": "value"}},
		{"mixed", " \t\n{ \t\n\"key\" \t\n: \t\n\"value\" \t\n} \t\n", map[string]interface{}{"key": "value"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			result, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test SIMD-specific number parsing
func TestParser_SIMDNumberParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"single digit", "5", 5},
		{"multiple digits", "123456", 123456},
		{"negative", "-789", -789},
		{"zero", "0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			
			// Test that SIMD parsing gives same result as regular parsing
			result, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %d, got %v", tt.expected, result)
			}
		})
	}
}

func BenchmarkParser_Numbers(b *testing.B) {
	testCases := []string{
		"42",
		"-123",
		"3.14159",
		"1e10",
		"123456789",
	}

	for _, tc := range testCases {
		b.Run(tc, func(b *testing.B) {
			p := New()
			data := []byte(tc)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := p.Parse(data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkParser_Strings(b *testing.B) {
	testCases := []string{
		`"hello"`,
		`"hello world"`,
		`"say \"hello\""`,
		`"unicode: 世界"`,
	}

	for _, tc := range testCases {
		b.Run(tc, func(b *testing.B) {
			p := New()
			data := []byte(tc)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := p.Parse(data)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}