package parser

import (
	"errors"
	"strconv"
	"unicode/utf8"
	"unsafe"
	
	"github.com/biggeezerdevelopment/simdjson-go/internal/scanner"
)

type Parser struct {
	scanner     *scanner.Scanner
	tokens      []scanner.Token
	pos         int
	data        []byte
	ownedTokens bool
}

func New() *Parser {
	return &Parser{
		scanner: scanner.New(),
	}
}

func (p *Parser) Parse(data []byte) (interface{}, error) {
	p.data = data
	p.pos = 0
	p.ownedTokens = true
	
	// Use simple tokenization
	tokens, err := p.scanner.SimpleTokenize(data)
	if err != nil {
		return nil, err
	}
	p.tokens = tokens
	
	if len(p.tokens) == 0 {
		return nil, errors.New("empty JSON")
	}
	
	result, err := p.parseValue()
	
	// Return tokens to pool after parsing
	if p.ownedTokens {
		scanner.PutTokenSlice(p.tokens)
		p.tokens = nil
		p.ownedTokens = false
	}
	
	return result, err
}

func (p *Parser) parseValue() (interface{}, error) {
	if p.pos >= len(p.tokens) {
		return nil, errors.New("unexpected end of JSON")
	}
	
	token := p.tokens[p.pos]
	
	switch token.Type {
	case scanner.TokenObjectBegin:
		return p.parseObject()
	case scanner.TokenArrayBegin:
		return p.parseArray()
	case scanner.TokenString:
		return p.parseString()
	case scanner.TokenNumber:
		return p.parseNumber()
	case scanner.TokenTrue:
		p.pos++
		return true, nil
	case scanner.TokenFalse:
		p.pos++
		return false, nil
	case scanner.TokenNull:
		p.pos++
		return nil, nil
	default:
		return nil, errors.New("unexpected token")
	}
}

func (p *Parser) parseObject() (map[string]interface{}, error) {
	obj := make(map[string]interface{})
	p.pos++ // Skip '{'
	
	// Empty object
	if p.pos < len(p.tokens) && p.tokens[p.pos].Type == scanner.TokenObjectEnd {
		p.pos++
		return obj, nil
	}
	
	for {
		// Parse key
		if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != scanner.TokenString {
			return nil, errors.New("expected string key")
		}
		
		key, err := p.parseString()
		if err != nil {
			return nil, err
		}
		
		// Expect colon
		if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != scanner.TokenColon {
			return nil, errors.New("expected colon after key")
		}
		p.pos++
		
		// Parse value
		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		
		obj[key.(string)] = value
		
		// Check for comma or end
		if p.pos >= len(p.tokens) {
			return nil, errors.New("unexpected end in object")
		}
		
		if p.tokens[p.pos].Type == scanner.TokenObjectEnd {
			p.pos++
			break
		}
		
		if p.tokens[p.pos].Type == scanner.TokenComma {
			p.pos++
			continue
		}
		
		return nil, errors.New("expected comma or object end")
	}
	
	return obj, nil
}

func (p *Parser) parseArray() ([]interface{}, error) {
	arr := make([]interface{}, 0, 8)
	p.pos++ // Skip '['
	
	// Empty array
	if p.pos < len(p.tokens) && p.tokens[p.pos].Type == scanner.TokenArrayEnd {
		p.pos++
		return arr, nil
	}
	
	for {
		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		
		arr = append(arr, value)
		
		// Check for comma or end
		if p.pos >= len(p.tokens) {
			return nil, errors.New("unexpected end in array")
		}
		
		if p.tokens[p.pos].Type == scanner.TokenArrayEnd {
			p.pos++
			break
		}
		
		if p.tokens[p.pos].Type == scanner.TokenComma {
			p.pos++
			continue
		}
		
		return nil, errors.New("expected comma or array end")
	}
	
	return arr, nil
}

func (p *Parser) parseString() (interface{}, error) {
	token := p.tokens[p.pos]
	p.pos++
	
	// Extract string content (without quotes)
	if token.Start+1 >= token.End-1 {
		return "", nil
	}
	
	str := p.data[token.Start+1 : token.End-1]
	
	// Fast path: no escapes - use zero-copy string conversion
	if !containsEscape(str) {
		return unsafeString(str), nil
	}
	
	// Slow path: handle escapes
	return p.unescapeString(str)
}

func containsEscape(b []byte) bool {
	for _, c := range b {
		if c == '\\' {
			return true
		}
	}
	return false
}

func (p *Parser) unescapeString(b []byte) (string, error) {
	buf := make([]byte, 0, len(b))
	
	for i := 0; i < len(b); i++ {
		if b[i] != '\\' {
			buf = append(buf, b[i])
			continue
		}
		
		if i+1 >= len(b) {
			return "", errors.New("invalid escape sequence")
		}
		
		i++
		switch b[i] {
		case '"', '\\', '/':
			buf = append(buf, b[i])
		case 'b':
			buf = append(buf, '\b')
		case 'f':
			buf = append(buf, '\f')
		case 'n':
			buf = append(buf, '\n')
		case 'r':
			buf = append(buf, '\r')
		case 't':
			buf = append(buf, '\t')
		case 'u':
			if i+4 >= len(b) {
				return "", errors.New("invalid unicode escape")
			}
			r, err := strconv.ParseUint(string(b[i+1:i+5]), 16, 16)
			if err != nil {
				return "", err
			}
			buf = append(buf, string(rune(r))...)
			i += 4
		default:
			return "", errors.New("invalid escape character")
		}
	}
	
	return string(buf), nil
}

func (p *Parser) parseNumber() (interface{}, error) {
	token := p.tokens[p.pos]
	p.pos++
	
	numBytes := p.data[token.Start:token.End]
	
	// Try SIMD integer parsing first if no float indicators
	if !containsFloatCharsBytes(numBytes) {
		if val, ok := p.scanner.SIMDParseInteger(numBytes); ok {
			return val, nil
		}
		
		// Fallback to standard integer parsing
		if val, err := strconv.ParseInt(unsafeString(numBytes), 10, 64); err == nil {
			return val, nil
		}
	}
	
	// Parse as float
	val, err := strconv.ParseFloat(unsafeString(numBytes), 64)
	if err != nil {
		return nil, err
	}
	
	return val, nil
}

func containsFloatChars(s string) bool {
	for _, c := range s {
		if c == '.' || c == 'e' || c == 'E' {
			return true
		}
	}
	return false
}

func containsFloatCharsBytes(b []byte) bool {
	for _, c := range b {
		if c == '.' || c == 'e' || c == 'E' {
			return true
		}
	}
	return false
}

// SIMD-optimized number parsing
func parseNumberSIMD(b []byte) (float64, error) {
	// This would use SIMD instructions to parse numbers faster
	// For now, fallback to standard parsing
	return strconv.ParseFloat(string(b), 64)
}

// SIMD-optimized string validation
func validateUTF8SIMD(b []byte) bool {
	// This would use SIMD to validate UTF-8 in parallel
	// For now, use standard validation
	return utf8.Valid(b)
}

// Zero-copy conversion from []byte to string
func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
