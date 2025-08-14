package benchmarks

import (
	"fmt"
	"testing"
	
	"github.com/biggeezerdevelopment/simdjson-go/internal/scanner"
)

var (
	// Test data for SIMD benchmarks
	structuralTestData = []byte(`{"users":[{"id":1,"name":"Alice","active":true},{"id":2,"name":"Bob","active":false}],"count":2}`)
	integerTestData    = []byte("1234567890")
	floatTestData      = []byte("123.456")
	stringTestData     = []byte(`"Hello, SIMD World!"`)
	mixedTestData      = make([]byte, 0, 1024)
)

func init() {
	// Generate mixed test data with various JSON elements
	for i := 0; i < 32; i++ {
		mixedTestData = append(mixedTestData, structuralTestData...)
	}
}

// Benchmark SIMD vs scalar structural scanning
func BenchmarkStructuralScanning_Scalar(b *testing.B) {
	s := scanner.New()
	defer s.Release()
	
	for i := 0; i < b.N; i++ {
		s.Scan(structuralTestData)
	}
}

func BenchmarkStructuralScanning_SIMD(b *testing.B) {
	s := scanner.New()
	defer s.Release()
	
	for i := 0; i < b.N; i++ {
		// Force SIMD path
		if scanner.HasSIMD() {
			s.ScanSIMD(structuralTestData)
		} else {
			s.Scan(structuralTestData)
		}
	}
}

// Benchmark SIMD integer parsing
func BenchmarkIntegerParsing_Scalar(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Simulate scalar integer parsing
		_ = len(integerTestData)
	}
}

func BenchmarkIntegerParsing_SIMD(b *testing.B) {
	s := scanner.New()
	defer s.Release()
	
	for i := 0; i < b.N; i++ {
		s.SIMDParseInteger(integerTestData)
	}
}

// Benchmark quote mask generation
func BenchmarkQuoteMask_Generation(b *testing.B) {
	s := scanner.New()
	defer s.Release()
	
	for i := 0; i < b.N; i++ {
		s.SIMDQuoteMask(stringTestData)
	}
}

// Benchmark UTF-8 validation
func BenchmarkUTF8Validation_SIMD(b *testing.B) {
	s := scanner.New()
	defer s.Release()
	
	for i := 0; i < b.N; i++ {
		s.SIMDValidateUTF8(stringTestData)
	}
}

// Benchmark large data processing with SIMD
func BenchmarkLargeDataProcessing_SIMD(b *testing.B) {
	s := scanner.New()
	defer s.Release()
	
	// Create large test data (1MB)
	largeData := make([]byte, 0, 1024*1024)
	for len(largeData) < cap(largeData) {
		largeData = append(largeData, mixedTestData...)
	}
	
	b.ResetTimer()
	b.SetBytes(int64(len(largeData)))
	
	for i := 0; i < b.N; i++ {
		s.Scan(largeData)
	}
}

// Benchmark memory alignment effects
func BenchmarkAlignedVsUnaligned(b *testing.B) {
	// Aligned data
	b.Run("Aligned", func(b *testing.B) {
		aligned := scanner.NewAlignedBuffer(len(structuralTestData), 32)
		copy(aligned.Bytes(), structuralTestData)
		
		s := scanner.New()
		defer s.Release()
		
		for i := 0; i < b.N; i++ {
			s.Scan(aligned.Bytes())
		}
	})
	
	// Unaligned data  
	b.Run("Unaligned", func(b *testing.B) {
		// Create unaligned data by offsetting by 1 byte
		unaligned := make([]byte, len(structuralTestData)+1)
		copy(unaligned[1:], structuralTestData)
		data := unaligned[1:]
		
		s := scanner.New()
		defer s.Release()
		
		for i := 0; i < b.N; i++ {
			s.Scan(data)
		}
	})
}

// Test different chunk sizes for SIMD processing
func BenchmarkChunkSizes(b *testing.B) {
	data := mixedTestData
	
	chunkSizes := []int{16, 32, 64, 128}
	
	for _, chunkSize := range chunkSizes {
		b.Run(fmt.Sprintf("ChunkSize%d", chunkSize), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				err := scanner.ProcessAligned(data, chunkSize, func(chunk []byte) error {
					// Simulate processing
					_ = len(chunk)
					return nil
				})
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
