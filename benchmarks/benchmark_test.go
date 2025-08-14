package benchmarks

import (
	"encoding/json"
	"testing"
	
	simdjson "github.com/simdjson/simdjson-go"
)

var (
	smallJSON = []byte(`{"name":"John","age":30,"city":"New York"}`)
	
	mediumJSON = []byte(`{
		"users": [
			{"id": 1, "name": "Alice", "email": "alice@example.com", "active": true},
			{"id": 2, "name": "Bob", "email": "bob@example.com", "active": false},
			{"id": 3, "name": "Charlie", "email": "charlie@example.com", "active": true},
			{"id": 4, "name": "David", "email": "david@example.com", "active": true},
			{"id": 5, "name": "Eve", "email": "eve@example.com", "active": false}
		],
		"metadata": {
			"version": "1.0.0",
			"timestamp": 1234567890,
			"count": 5
		}
	}`)
	
	largeJSON []byte
)

func init() {
	// Generate large JSON (array of 1000 objects)
	largeJSON = []byte(`[`)
	for i := 0; i < 1000; i++ {
		if i > 0 {
			largeJSON = append(largeJSON, ',')
		}
		largeJSON = append(largeJSON, []byte(`{
			"id": 12345,
			"name": "User Name Here",
			"email": "user@example.com",
			"age": 25,
			"active": true,
			"tags": ["tag1", "tag2", "tag3"],
			"profile": {
				"bio": "This is a bio text",
				"location": "San Francisco, CA",
				"website": "https://example.com"
			}
		}`)...)
	}
	largeJSON = append(largeJSON, ']')
}

type SmallStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
	City string `json:"city"`
}

type MediumStruct struct {
	Users []struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Email  string `json:"email"`
		Active bool   `json:"active"`
	} `json:"users"`
	Metadata struct {
		Version   string `json:"version"`
		Timestamp int64  `json:"timestamp"`
		Count     int    `json:"count"`
	} `json:"metadata"`
}

// Unmarshal benchmarks

func BenchmarkUnmarshalSmall_StdLib(b *testing.B) {
	var s SmallStruct
	for i := 0; i < b.N; i++ {
		_ = json.Unmarshal(smallJSON, &s)
	}
}

func BenchmarkUnmarshalSmall_SimdJSON(b *testing.B) {
	var s SmallStruct
	for i := 0; i < b.N; i++ {
		_ = simdjson.Unmarshal(smallJSON, &s)
	}
}

func BenchmarkUnmarshalMedium_StdLib(b *testing.B) {
	var m MediumStruct
	for i := 0; i < b.N; i++ {
		_ = json.Unmarshal(mediumJSON, &m)
	}
}

func BenchmarkUnmarshalMedium_SimdJSON(b *testing.B) {
	var m MediumStruct
	for i := 0; i < b.N; i++ {
		_ = simdjson.Unmarshal(mediumJSON, &m)
	}
}

func BenchmarkUnmarshalLarge_StdLib(b *testing.B) {
	var data []interface{}
	for i := 0; i < b.N; i++ {
		_ = json.Unmarshal(largeJSON, &data)
	}
}

func BenchmarkUnmarshalLarge_SimdJSON(b *testing.B) {
	var data []interface{}
	for i := 0; i < b.N; i++ {
		_ = simdjson.Unmarshal(largeJSON, &data)
	}
}

// Marshal benchmarks

func BenchmarkMarshalSmall_StdLib(b *testing.B) {
	s := SmallStruct{Name: "John", Age: 30, City: "New York"}
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(s)
	}
}

func BenchmarkMarshalSmall_SimdJSON(b *testing.B) {
	s := SmallStruct{Name: "John", Age: 30, City: "New York"}
	for i := 0; i < b.N; i++ {
		_, _ = simdjson.Marshal(s)
	}
}

func BenchmarkMarshalMedium_StdLib(b *testing.B) {
	var m MediumStruct
	_ = json.Unmarshal(mediumJSON, &m)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(m)
	}
}

func BenchmarkMarshalMedium_SimdJSON(b *testing.B) {
	var m MediumStruct
	_ = simdjson.Unmarshal(mediumJSON, &m)
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, _ = simdjson.Marshal(m)
	}
}

// Validation benchmarks

func BenchmarkValidateSmall_StdLib(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = json.Valid(smallJSON)
	}
}

func BenchmarkValidateSmall_SimdJSON(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = simdjson.Valid(smallJSON)
	}
}

func BenchmarkValidateLarge_StdLib(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = json.Valid(largeJSON)
	}
}

func BenchmarkValidateLarge_SimdJSON(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = simdjson.Valid(largeJSON)
	}
}