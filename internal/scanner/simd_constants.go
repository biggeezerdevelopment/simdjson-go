package scanner

// SIMD constants for JSON parsing
const (
	// Chunk sizes for different SIMD instruction sets
	AVX2ChunkSize = 32  // 256-bit vectors
	SSE4ChunkSize = 16  // 128-bit vectors
	NEONChunkSize = 16  // 128-bit vectors
	
	// Character classes for parallel classification
	CharClassStructural = 0x01  // {}[]:,
	CharClassWhitespace = 0x02  // space, tab, newline, carriage return
	CharClassQuote      = 0x04  // "
	CharClassBackslash  = 0x08  // \
	CharClassDigit      = 0x10  // 0-9
	CharClassSign       = 0x20  // +, -
	CharClassAlpha      = 0x40  // a-z, A-Z
	CharClassOther      = 0x80  // everything else
)

// Lookup tables for character classification (256 bytes, cache-friendly)
var (
	CharClassLookup = [256]uint8{
		// 0x00-0x0F
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassWhitespace, CharClassWhitespace, CharClassOther,
		CharClassOther, CharClassWhitespace, CharClassOther, CharClassOther,
		
		// 0x10-0x1F
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		
		// 0x20-0x2F (space !"#$%&'()*+,-./)
		CharClassWhitespace, CharClassOther, CharClassQuote, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassSign,
		CharClassStructural, CharClassSign, CharClassOther, CharClassOther,
		
		// 0x30-0x3F (0123456789:;<=>?)
		CharClassDigit, CharClassDigit, CharClassDigit, CharClassDigit,
		CharClassDigit, CharClassDigit, CharClassDigit, CharClassDigit,
		CharClassDigit, CharClassDigit, CharClassStructural, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		
		// 0x40-0x4F (@ABCDEFGHIJKLMNO)
		CharClassOther, CharClassAlpha, CharClassAlpha, CharClassAlpha,
		CharClassAlpha, CharClassAlpha, CharClassAlpha, CharClassAlpha,
		CharClassAlpha, CharClassAlpha, CharClassAlpha, CharClassAlpha,
		CharClassAlpha, CharClassAlpha, CharClassAlpha, CharClassAlpha,
		
		// 0x50-0x5F (PQRSTUVWXYZ[\]^_)
		CharClassAlpha, CharClassAlpha, CharClassAlpha, CharClassAlpha,
		CharClassAlpha, CharClassAlpha, CharClassAlpha, CharClassAlpha,
		CharClassAlpha, CharClassAlpha, CharClassAlpha, CharClassStructural,
		CharClassBackslash, CharClassStructural, CharClassOther, CharClassOther,
		
		// 0x60-0x6F (`abcdefghijklmno)
		CharClassOther, CharClassAlpha, CharClassAlpha, CharClassAlpha,
		CharClassAlpha, CharClassAlpha, CharClassAlpha, CharClassAlpha,
		CharClassAlpha, CharClassAlpha, CharClassAlpha, CharClassAlpha,
		CharClassAlpha, CharClassAlpha, CharClassAlpha, CharClassAlpha,
		
		// 0x70-0x7F (pqrstuvwxyz{|}~DEL)
		CharClassAlpha, CharClassAlpha, CharClassAlpha, CharClassAlpha,
		CharClassAlpha, CharClassAlpha, CharClassAlpha, CharClassAlpha,
		CharClassAlpha, CharClassAlpha, CharClassAlpha, CharClassStructural,
		CharClassOther, CharClassStructural, CharClassOther, CharClassOther,
		
		// 0x80-0xFF (extended ASCII - all classified as other)
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
		CharClassOther, CharClassOther, CharClassOther, CharClassOther,
	}
)