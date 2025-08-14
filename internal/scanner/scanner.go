package scanner

import (
	"errors"
	"sync"
)

const (
	StructuralQuote     uint64 = 1 << 0
	StructuralBackslash uint64 = 1 << 1
	StructuralColon     uint64 = 1 << 2
	StructuralComma     uint64 = 1 << 3
	StructuralLBrace    uint64 = 1 << 4
	StructuralRBrace    uint64 = 1 << 5
	StructuralLBracket  uint64 = 1 << 6
	StructuralRBracket  uint64 = 1 << 7
	StructuralWhitespace uint64 = 1 << 8
)

type Scanner struct {
	buf              []byte
	structuralIndices []uint32
	stringMask       []uint64
	pos              int
	
	// Reusable buffers
	tempBuf          []byte
	charClassifier   [256]uint64
}

var scannerPool = sync.Pool{
	New: func() interface{} {
		s := &Scanner{
			structuralIndices: make([]uint32, 0, 1024),
			tempBuf:          make([]byte, 64),
		}
		s.initCharClassifier()
		return s
	},
}

func New() *Scanner {
	return scannerPool.Get().(*Scanner)
}

func (s *Scanner) Release() {
	s.buf = nil
	s.structuralIndices = s.structuralIndices[:0]
	s.stringMask = s.stringMask[:0]
	s.pos = 0
	scannerPool.Put(s)
}

func (s *Scanner) initCharClassifier() {
	// Initialize character classifier lookup table
	s.charClassifier['"'] = StructuralQuote
	s.charClassifier['\\'] = StructuralBackslash
	s.charClassifier[':'] = StructuralColon
	s.charClassifier[','] = StructuralComma
	s.charClassifier['{'] = StructuralLBrace
	s.charClassifier['}'] = StructuralRBrace
	s.charClassifier['['] = StructuralLBracket
	s.charClassifier[']'] = StructuralRBracket
	s.charClassifier[' '] = StructuralWhitespace
	s.charClassifier['\t'] = StructuralWhitespace
	s.charClassifier['\n'] = StructuralWhitespace
	s.charClassifier['\r'] = StructuralWhitespace
}

func (s *Scanner) Scan(data []byte) error {
	s.buf = data
	s.structuralIndices = s.structuralIndices[:0]
	
	if hasSIMD() {
		return s.scanSIMD()
	}
	return s.scanScalar()
}

// ScanSIMD forces SIMD scanning (exported for benchmarks)
func (s *Scanner) ScanSIMD(data []byte) error {
	s.buf = data
	s.structuralIndices = s.structuralIndices[:0]
	return s.scanSIMD()
}

// HasSIMD returns true if SIMD instructions are available
func HasSIMD() bool {
	return hasSIMD()
}

func (s *Scanner) scanScalar() error {
	inString := false
	escaped := false
	
	for i := 0; i < len(s.buf); i++ {
		c := s.buf[i]
		
		if escaped {
			escaped = false
			continue
		}
		
		if c == '\\' && inString {
			escaped = true
			continue
		}
		
		if c == '"' {
			inString = !inString
			s.structuralIndices = append(s.structuralIndices, uint32(i))
			continue
		}
		
		if !inString {
			class := s.charClassifier[c]
			if class != 0 && class != StructuralWhitespace {
				s.structuralIndices = append(s.structuralIndices, uint32(i))
			} else if class == 0 && (c >= '0' && c <= '9') || c == '-' || c == 't' || c == 'f' || c == 'n' {
				// Start of a value (number, true, false, null)
				if i == 0 || s.charClassifier[s.buf[i-1]] != 0 || s.buf[i-1] == ' ' || s.buf[i-1] == '\t' || s.buf[i-1] == '\n' || s.buf[i-1] == '\r' {
					s.structuralIndices = append(s.structuralIndices, uint32(i))
				}
			}
		}
	}
	
	return nil
}

func (s *Scanner) GetStructuralIndices() []uint32 {
	return s.structuralIndices
}

func (s *Scanner) Validate(data []byte) bool {
	// Try to tokenize - if it fails, JSON is invalid
	tokens, err := s.SimpleTokenize(data)
	if err != nil {
		return false
	}
	
	// Basic structural validation
	depth := 0
	stack := make([]TokenType, 0, 32)
	
	for _, token := range tokens {
		switch token.Type {
		case TokenObjectBegin:
			depth++
			stack = append(stack, TokenObjectBegin)
		case TokenArrayBegin:
			depth++
			stack = append(stack, TokenArrayBegin)
		case TokenObjectEnd:
			depth--
			if len(stack) == 0 || stack[len(stack)-1] != TokenObjectBegin {
				return false
			}
			stack = stack[:len(stack)-1]
		case TokenArrayEnd:
			depth--
			if len(stack) == 0 || stack[len(stack)-1] != TokenArrayBegin {
				return false
			}
			stack = stack[:len(stack)-1]
		}
		
		if depth < 0 {
			return false
		}
	}
	
	return depth == 0 && len(stack) == 0
}

type TokenType uint8

const (
	TokenNone TokenType = iota
	TokenObjectBegin
	TokenObjectEnd
	TokenArrayBegin
	TokenArrayEnd
	TokenString
	TokenNumber
	TokenTrue
	TokenFalse
	TokenNull
	TokenColon
	TokenComma
)

type Token struct {
	Type  TokenType
	Start uint32
	End   uint32
}

func (s *Scanner) Tokenize() ([]Token, error) {
	if len(s.structuralIndices) == 0 {
		return nil, nil
	}
	
	tokens := make([]Token, 0, len(s.structuralIndices))
	
	for i := 0; i < len(s.structuralIndices); i++ {
		idx := s.structuralIndices[i]
		c := s.buf[idx]
		
		var token Token
		token.Start = idx
		
		switch c {
		case '{':
			token.Type = TokenObjectBegin
			token.End = idx + 1
		case '}':
			token.Type = TokenObjectEnd
			token.End = idx + 1
		case '[':
			token.Type = TokenArrayBegin
			token.End = idx + 1
		case ']':
			token.Type = TokenArrayEnd
			token.End = idx + 1
		case ':':
			token.Type = TokenColon
			token.End = idx + 1
		case ',':
			token.Type = TokenComma
			token.End = idx + 1
		case '"':
			// Find matching quote
			token.Type = TokenString
			// Simple approach: find the next quote that's not escaped
			found := false
			for j := idx + 1; j < uint32(len(s.buf)); j++ {
				if s.buf[j] == '"' {
					// Check if it's escaped by counting preceding backslashes
					escaped := false
					backslashes := 0
					for k := j - 1; k >= 0 && s.buf[k] == '\\'; k-- {
						backslashes++
					}
					escaped = backslashes%2 == 1
					
					if !escaped {
						token.End = j + 1
						found = true
						break
					}
				}
			}
			if !found {
				return nil, errors.New("unterminated string")
			}
		default:
			// Could be number, true, false, or null
			token.Type = s.detectValueType(idx)
			token.End = s.findValueEnd(idx)
		}
		
		tokens = append(tokens, token)
	}
	
	return tokens, nil
}

func (s *Scanner) detectValueType(start uint32) TokenType {
	if start >= uint32(len(s.buf)) {
		return TokenNone
	}
	
	switch s.buf[start] {
	case 't':
		if start+4 <= uint32(len(s.buf)) && string(s.buf[start:start+4]) == "true" {
			return TokenTrue
		}
	case 'f':
		if start+5 <= uint32(len(s.buf)) && string(s.buf[start:start+5]) == "false" {
			return TokenFalse
		}
	case 'n':
		if start+4 <= uint32(len(s.buf)) && string(s.buf[start:start+4]) == "null" {
			return TokenNull
		}
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return TokenNumber
	}
	
	return TokenNone
}

func (s *Scanner) findValueEnd(start uint32) uint32 {
	for i := start; i < uint32(len(s.buf)); i++ {
		c := s.buf[i]
		switch c {
		case ' ', '\t', '\n', '\r', ',', '}', ']':
			return i
		}
	}
	return uint32(len(s.buf))
}