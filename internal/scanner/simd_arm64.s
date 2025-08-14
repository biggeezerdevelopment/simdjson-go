//go:build arm64 && !noasm && disabled

#include "textflag.h"

// ARM64 NEON implementations for JSON parsing

// findStructuralIndicesNEON processes 16 bytes at a time using NEON
// func findStructuralIndicesNEON(data unsafe.Pointer, length uint64, indices *uint32) uint64
TEXT ·findStructuralIndicesNEON(SB), NOSPLIT, $0-32
    MOVD    data+0(FP), R0        // Source data pointer
    MOVD    length+8(FP), R1      // Data length
    MOVD    indices+16(FP), R2    // Output indices pointer
    
    MOVD    $0, R3                // Current position
    MOVD    $0, R4                // Output index count
    
    // Load structural character constants into vector registers
    MOVD    $0x22, R5             // Quote character
    DUP     R5, V0.B16
    MOVD    $0x5C, R5             // Backslash
    DUP     R5, V1.B16
    MOVD    $0x7B, R5             // {
    DUP     R5, V2.B16
    MOVD    $0x7D, R5             // }
    DUP     R5, V3.B16
    MOVD    $0x5B, R5             // [
    DUP     R5, V4.B16
    MOVD    $0x5D, R5             // ]
    DUP     R5, V5.B16
    MOVD    $0x3A, R5             // :
    DUP     R5, V6.B16
    MOVD    $0x2C, R5             // ,
    DUP     R5, V7.B16

neon_main_loop:
    CMP     $16, R1               // Check if we have at least 16 bytes
    BLT     neon_remainder
    
    // Load 16 bytes of data
    VLD1.P  16(R0), [V8.B16]      // Load 16 bytes and post-increment
    
    // Compare with each structural character
    CMEQ    V0.B16, V8.B16, V16.B16    // Find quotes
    CMEQ    V1.B16, V8.B16, V17.B16    // Find backslashes
    CMEQ    V2.B16, V8.B16, V18.B16    // Find {
    CMEQ    V3.B16, V8.B16, V19.B16    // Find }
    CMEQ    V4.B16, V8.B16, V20.B16    // Find [
    CMEQ    V5.B16, V8.B16, V21.B16    // Find ]
    CMEQ    V6.B16, V8.B16, V22.B16    // Find :
    CMEQ    V7.B16, V8.B16, V23.B16    // Find ,
    
    // Combine all structural characters using OR
    ORR     V16.B16, V17.B16, V24.B16
    ORR     V18.B16, V19.B16, V25.B16
    ORR     V20.B16, V21.B16, V26.B16
    ORR     V22.B16, V23.B16, V27.B16
    ORR     V24.B16, V25.B16, V28.B16
    ORR     V26.B16, V27.B16, V29.B16
    ORR     V28.B16, V29.B16, V30.B16  // Final result
    
    // Extract mask - check each byte
    MOVD    $0, R7                // Bit counter
    MOVD    $0, R8                // Current byte index

neon_process_bytes:
    CMP     $16, R8
    BEQ     neon_next_chunk
    
    // Extract byte and check if non-zero
    UMOV    V30.B[0], R9
    CBZ     R9, neon_next_byte
    
    // Store index
    ADD     R3, R8, R10           // Calculate absolute index
    MOVW    R10, (R2)(R4<<2)      // Store index (4 bytes per uint32)
    ADD     $1, R4                // Increment output count

neon_next_byte:
    EXT     $1, V30.B16, V30.B16, V30.B16  // Shift vector by 1 byte
    ADD     $1, R8
    B       neon_process_bytes

neon_next_chunk:
    ADD     $16, R3               // Move to next 16-byte chunk
    SUB     $16, R1               // Decrease remaining length
    B       neon_main_loop

neon_remainder:
    // Process remaining bytes using scalar approach
    CBZ     R1, neon_done

neon_remainder_loop:
    LDRB.P  1(R0), R5             // Load byte and post-increment
    
    // Check if it's a structural character
    CMP     $0x22, R5             // "
    BEQ     neon_store_index
    CMP     $0x5C, R5             // \
    BEQ     neon_store_index
    CMP     $0x7B, R5             // {
    BEQ     neon_store_index
    CMP     $0x7D, R5             // }
    BEQ     neon_store_index
    CMP     $0x5B, R5             // [
    BEQ     neon_store_index
    CMP     $0x5D, R5             // ]
    BEQ     neon_store_index
    CMP     $0x3A, R5             // :
    BEQ     neon_store_index
    CMP     $0x2C, R5             // ,
    BEQ     neon_store_index
    B       neon_next_remainder

neon_store_index:
    MOVW    R3, (R2)(R4<<2)       // Store index
    ADD     $1, R4                // Increment output count

neon_next_remainder:
    ADD     $1, R3                // Move to next byte
    SUB     $1, R1                // Decrease remaining length
    CBNZ    R1, neon_remainder_loop

neon_done:
    MOVD    R4, ret+24(FP)        // Return number of indices found
    RET

// findQuoteMaskNEON creates a bitmask of quote positions
// func findQuoteMaskNEON(data unsafe.Pointer, length uint64, mask *uint64) uint64
TEXT ·findQuoteMaskNEON(SB), NOSPLIT, $0-32
    MOVD    data+0(FP), R0
    MOVD    length+8(FP), R1
    MOVD    mask+16(FP), R2
    
    MOVD    $0, R3                // Current position
    MOVD    $0, R4                // Mask count
    
    // Load quote character into vector register
    MOVD    $0x22, R5
    DUP     R5, V0.B16

neon_quote_loop:
    CMP     $16, R1
    BLT     neon_quote_remainder
    
    VLD1.P  16(R0), [V1.B16]
    
    // Find quotes
    CMEQ    V0.B16, V1.B16, V2.B16
    
    // Convert to bitmask (simplified - would use more efficient method)
    MOVD    $0, R7                // Result mask
    MOVD    $0, R8                // Bit position

neon_quote_extract_loop:
    CMP     $16, R8
    BEQ     neon_quote_store_mask
    
    UMOV    V2.B[0], R9
    CBZ     R9, neon_quote_next_bit
    
    MOVD    $1, R10
    LSL     R8, R10, R10          // Shift 1 left by bit position
    ORR     R7, R10, R7           // Set bit in mask

neon_quote_next_bit:
    EXT     $1, V2.B16, V2.B16, V2.B16
    ADD     $1, R8
    B       neon_quote_extract_loop

neon_quote_store_mask:
    MOVD    R7, (R2)(R4<<3)       // Store 64-bit mask
    ADD     $1, R4
    
    ADD     $16, R3
    SUB     $16, R1
    B       neon_quote_loop

neon_quote_remainder:
    CBZ     R1, neon_quote_done
    
    MOVD    $0, R7                // Partial mask
    MOVD    $0, R8                // Bit position

neon_quote_remainder_loop:
    LDRB.P  1(R0), R5
    CMP     $0x22, R5
    BNE     neon_quote_next_rem
    
    MOVD    $1, R9
    LSL     R8, R9, R9
    ORR     R7, R9, R7

neon_quote_next_rem:
    ADD     $1, R3
    ADD     $1, R8
    SUB     $1, R1
    CBNZ    R1, neon_quote_remainder_loop
    
    MOVD    R7, (R2)(R4<<3)
    ADD     $1, R4

neon_quote_done:
    MOVD    R4, ret+24(FP)
    RET

// validateUTF8NEON validates UTF-8 encoding using NEON
// func validateUTF8NEON(data unsafe.Pointer, length uint64) bool
TEXT ·validateUTF8NEON(SB), NOSPLIT, $0-24
    MOVD    data+0(FP), R0
    MOVD    length+8(FP), R1
    
    MOVD    $1, R2                // Assume valid
    MOVD    $0, R3                // Current position
    
    // ASCII threshold (0x80)
    MOVD    $0x80, R4
    DUP     R4, V0.B16

neon_utf8_loop:
    CMP     $16, R1
    BLT     neon_utf8_remainder
    
    VLD1.P  16(R0), [V1.B16]
    
    // Check if all bytes are < 0x80 (ASCII)
    CMHI    V1.B16, V0.B16, V2.B16  // 0x80 > data for ASCII bytes
    
    // If all are ASCII, continue (simplified check)
    // Real UTF-8 validation would be more complex
    
neon_utf8_next_chunk:
    ADD     $16, R3
    SUB     $16, R1
    B       neon_utf8_loop

neon_utf8_remainder:
    CBZ     R1, neon_utf8_valid

neon_utf8_remainder_loop:
    LDRB.P  1(R0), R4
    CMP     $0x80, R4
    BLT     neon_utf8_ascii
    
    // Multi-byte UTF-8 validation would go here
    // For now, assume valid

neon_utf8_ascii:
    ADD     $1, R3
    SUB     $1, R1
    CBNZ    R1, neon_utf8_remainder_loop

neon_utf8_valid:
    MOVD    R2, ret+16(FP)
    RET

// parseIntegerNEON parses integers using NEON digit detection
// func parseIntegerNEON(data unsafe.Pointer, length uint64) (int64, bool)
TEXT ·parseIntegerNEON(SB), NOSPLIT, $0-32
    MOVD    data+0(FP), R0
    MOVD    length+8(FP), R1
    
    MOVD    $0, R2                // Result accumulator
    MOVD    $0, R3                // Position
    MOVD    $1, R4                // Sign (positive)
    MOVD    $1, R5                // Valid flag
    
    CBZ     R1, neon_parse_invalid
    
    // Check for negative sign
    LDRB    (R0), R6
    CMP     $45, R6               // '-'
    BNE     neon_parse_digits
    NEG     R4, R4                // Set negative
    ADD     $1, R0                // Skip minus
    SUB     $1, R1

neon_parse_digits:
    CBZ     R1, neon_parse_invalid
    
    // Digit range constants
    MOVD    $0x30, R7             // '0'
    DUP     R7, V0.B16
    MOVD    $0x39, R7             // '9'
    DUP     R7, V1.B16

neon_parse_simd_loop:
    CMP     $16, R1
    BLT     neon_parse_scalar
    
    VLD1.P  16(R0), [V2.B16]
    
    // Check if all are digits (>= '0' and <= '9')
    CMHS    V2.B16, V0.B16, V3.B16    // data >= '0'
    CMHS    V1.B16, V2.B16, V4.B16    // '9' >= data
    AND     V3.B16, V4.B16, V5.B16    // Both conditions
    
    // For simplicity, fall back to scalar for actual parsing
    B       neon_parse_scalar

neon_parse_scalar:
    CBZ     R1, neon_parse_done

neon_parse_scalar_loop:
    LDRB.P  1(R0), R6
    SUB     $48, R6               // Convert ASCII to digit
    CMP     $9, R6
    BHI     neon_parse_done       // Not a digit
    
    // result = result * 10 + digit
    MOVD    $10, R7
    MUL     R7, R2, R2
    ADD     R6, R2
    
    ADD     $1, R3
    SUB     $1, R1
    CBNZ    R1, neon_parse_scalar_loop

neon_parse_done:
    CMP     $0, R4
    BGE     neon_parse_positive
    NEG     R2, R2                // Apply negative sign

neon_parse_positive:
    MOVD    R2, ret+16(FP)        // Return value
    MOVD    R5, ret+24(FP)        // Return success flag
    RET

neon_parse_invalid:
    MOVD    $0, ret+16(FP)
    MOVD    $0, ret+24(FP)
    RET

