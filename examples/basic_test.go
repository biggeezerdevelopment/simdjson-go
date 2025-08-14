package examples

import (
	"encoding/json"
	"fmt"
	"testing"
	
	simdjson "github.com/biggeezerdevelopment/simdjson-go"
)

func TestBasicUsage(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
		City string `json:"city"`
	}
	
	// Test marshaling
	p := Person{Name: "John", Age: 30, City: "New York"}
	
	// Standard library
	stdData, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	
	// SimdJSON
	simdData, err := simdjson.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	
	fmt.Printf("Standard: %s\n", stdData)
	fmt.Printf("SimdJSON: %s\n", simdData)
	
	// Test unmarshaling
	jsonStr := `{"name":"Alice","age":25,"city":"Boston"}`
	
	var p1, p2 Person
	
	// Standard library
	err = json.Unmarshal([]byte(jsonStr), &p1)
	if err != nil {
		t.Fatal(err)
	}
	
	// SimdJSON
	err = simdjson.Unmarshal([]byte(jsonStr), &p2)
	if err != nil {
		t.Fatal(err)
	}
	
	if p1 != p2 {
		t.Errorf("Results don't match: %+v vs %+v", p1, p2)
	}
	
	fmt.Printf("Unmarshaled: %+v\n", p2)
}

func TestValidation(t *testing.T) {
	testCases := []struct {
		json  string
		valid bool
	}{
		{`{"name":"John","age":30}`, true},
		{`[1,2,3,4,5]`, true},
		{`true`, true},
		{`null`, true},
		{`{"name":"John","age":}`, false},
		{`[1,2,3,`, false},
		{`{`, false},
	}
	
	for _, tc := range testCases {
		stdValid := json.Valid([]byte(tc.json))
		simdValid := simdjson.Valid([]byte(tc.json))
		
		if stdValid != tc.valid {
			t.Errorf("Standard library validation failed for %s: expected %v, got %v", tc.json, tc.valid, stdValid)
		}
		
		if simdValid != tc.valid {
			t.Errorf("SimdJSON validation failed for %s: expected %v, got %v", tc.json, tc.valid, simdValid)
		}
		
		if stdValid != simdValid {
			t.Errorf("Validation mismatch for %s: std=%v, simd=%v", tc.json, stdValid, simdValid)
		}
	}
}

func ExampleMarshal() {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	
	p := Person{Name: "John", Age: 30}
	data, err := simdjson.Marshal(p)
	if err != nil {
		panic(err)
	}
	
	fmt.Println(string(data))
	// Output: {"name":"John","age":30}
}

func ExampleUnmarshal() {
	jsonData := []byte(`{"name":"Alice","age":25}`)
	
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	
	var p Person
	err := simdjson.Unmarshal(jsonData, &p)
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("Name: %s, Age: %d\n", p.Name, p.Age)
	// Output: Name: Alice, Age: 25
}
