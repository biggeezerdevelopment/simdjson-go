//go:build arm64

package scanner

// ARM64 NEON stub implementations in Go
// These are placeholders that indicate NEON is not available and should fall back to scalar

// findStructuralIndicesNEON finds structural characters (stub)
func findStructuralIndicesNEON(data []byte, indices []uint32) int {
	// Return 0 to indicate no NEON implementation available
	return 0
}

// findQuoteMaskNEON creates quote masks (stub)
func findQuoteMaskNEON(data []byte, masks []uint64) int {
	// Return 0 to indicate no NEON implementation available
	return 0
}

// validateUTF8NEON validates UTF-8 (stub)
func validateUTF8NEON(data []byte) bool {
	// Return true to indicate should fall back to Go validation
	return true
}

// parseIntegerNEON parses integers (stub)
func parseIntegerNEON(data []byte) (int64, bool) {
	// Return false to indicate should fall back to Go parsing
	return 0, false
}