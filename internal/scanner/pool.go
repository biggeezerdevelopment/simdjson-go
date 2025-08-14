package scanner

import "sync"

var tokenPool = sync.Pool{
	New: func() interface{} {
		return make([]Token, 0, 64)
	},
}

func getTokenSlice() []Token {
	return tokenPool.Get().([]Token)
}

func PutTokenSlice(tokens []Token) {
	if cap(tokens) > 1024 { // Don't pool very large slices
		return
	}
	tokens = tokens[:0]
	tokenPool.Put(tokens)
}