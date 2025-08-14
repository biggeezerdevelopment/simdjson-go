package scanner

import (
	"errors"
)

// SimpleTokenize tokenizes JSON without complex structural scanning
func (s *Scanner) SimpleTokenize(data []byte) ([]Token, error) {
	s.buf = data
	if len(data) == 0 {
		return nil, errors.New("empty input")
	}
	
	tokens := getTokenSlice()
	if cap(tokens) < len(data)/4 {
		tokens = make([]Token, 0, len(data)/4)
	}
	i := 0
	
	for i < len(data) {
		// Skip whitespace
		for i < len(data) && isWhitespace(data[i]) {
			i++
		}
		
		if i >= len(data) {
			break
		}
		
		c := data[i]
		token := Token{Start: uint32(i)}
		
		// Check for invalid syntax: value expected but got structural character
		if len(tokens) > 0 {
			lastToken := tokens[len(tokens)-1]
			if lastToken.Type == TokenColon {
				// After colon, we need a value, not }, ], or ,
				if c == '}' || c == ']' || c == ',' {
					return nil, errors.New("expected value after colon")
				}
			}
			// Check for double comma
			if lastToken.Type == TokenComma && c == ',' {
				return nil, errors.New("unexpected comma")
			}
			// Check for trailing comma in object
			if lastToken.Type == TokenComma && c == '}' {
				return nil, errors.New("trailing comma in object")
			}
			// Check for trailing comma in array
			if lastToken.Type == TokenComma && c == ']' {
				return nil, errors.New("trailing comma in array")
			}
		}
		
		switch c {
		case '{':
			token.Type = TokenObjectBegin
			token.End = uint32(i + 1)
			i++
		case '}':
			token.Type = TokenObjectEnd
			token.End = uint32(i + 1)
			i++
		case '[':
			token.Type = TokenArrayBegin
			token.End = uint32(i + 1)
			i++
		case ']':
			token.Type = TokenArrayEnd
			token.End = uint32(i + 1)
			i++
		case ':':
			token.Type = TokenColon
			token.End = uint32(i + 1)
			i++
		case ',':
			token.Type = TokenComma
			token.End = uint32(i + 1)
			i++
		case '"':
			// Parse string
			i++ // Skip opening quote
			for i < len(data) {
				if data[i] == '"' {
					// Check if escaped
					escaped := false
					backslashes := 0
					for j := i - 1; j >= 0 && data[j] == '\\'; j-- {
						backslashes++
					}
					escaped = backslashes%2 == 1
					
					if !escaped {
						i++ // Skip closing quote
						break
					}
				}
				i++
			}
			token.Type = TokenString
			token.End = uint32(i)
		case 't':
			// true
			if i+4 <= len(data) && string(data[i:i+4]) == "true" {
				token.Type = TokenTrue
				token.End = uint32(i + 4)
				i += 4
			} else {
				return nil, errors.New("invalid token starting with 't'")
			}
		case 'f':
			// false
			if i+5 <= len(data) && string(data[i:i+5]) == "false" {
				token.Type = TokenFalse
				token.End = uint32(i + 5)
				i += 5
			} else {
				return nil, errors.New("invalid token starting with 'f'")
			}
		case 'n':
			// null
			if i+4 <= len(data) && string(data[i:i+4]) == "null" {
				token.Type = TokenNull
				token.End = uint32(i + 4)
				i += 4
			} else {
				return nil, errors.New("invalid token starting with 'n'")
			}
		default:
			if c == '-' || (c >= '0' && c <= '9') {
				// Parse number
				numStart := i
				if c == '-' {
					i++
					if i >= len(data) || !(data[i] >= '0' && data[i] <= '9') {
						return nil, errors.New("invalid number: missing digits after minus")
					}
				}
				
				// Must have at least one digit
				if i >= len(data) || !(data[i] >= '0' && data[i] <= '9') {
					return nil, errors.New("invalid number: no digits")
				}
				
				// Parse integer part
				if data[i] == '0' {
					i++
					// After 0, must be . or e/E or end
					if i < len(data) && data[i] >= '0' && data[i] <= '9' {
						return nil, errors.New("invalid number: leading zero")
					}
				} else {
					for i < len(data) && data[i] >= '0' && data[i] <= '9' {
						i++
					}
				}
				
				// Parse decimal part
				if i < len(data) && data[i] == '.' {
					i++
					if i >= len(data) || !(data[i] >= '0' && data[i] <= '9') {
						return nil, errors.New("invalid number: no digits after decimal")
					}
					for i < len(data) && data[i] >= '0' && data[i] <= '9' {
						i++
					}
				}
				
				// Parse exponent
				if i < len(data) && (data[i] == 'e' || data[i] == 'E') {
					i++
					if i < len(data) && (data[i] == '+' || data[i] == '-') {
						i++
					}
					if i >= len(data) || !(data[i] >= '0' && data[i] <= '9') {
						return nil, errors.New("invalid number: no digits in exponent")
					}
					for i < len(data) && data[i] >= '0' && data[i] <= '9' {
						i++
					}
				}
				
				// Validate we have a complete number
				if i <= numStart || (numStart == i-1 && data[numStart] == '-') {
					return nil, errors.New("invalid number")
				}
				
				token.Type = TokenNumber
				token.End = uint32(i)
			} else {
				return nil, errors.New("unexpected character: " + string(c))
			}
		}
		
		tokens = append(tokens, token)
	}
	
	return tokens, nil
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}