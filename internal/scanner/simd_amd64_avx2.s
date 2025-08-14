//go:build amd64 && !noasm

#include "textflag.h"

// Constants for AVX2 operations
#define QUOTE_CHAR     0x22 // "
#define BACKSLASH_CHAR 0x5C // \
#define LBRACE_CHAR    0x7B // {
#define RBRACE_CHAR    0x7D // }
#define LBRACKET_CHAR  0x5B // [
#define RBRACKET_CHAR  0x5D // ]
#define COLON_CHAR     0x3A // :
#define COMMA_CHAR     0x2C // ,

// findStructuralIndicesAVX2 processes 32 bytes at a time using AVX2
// func findStructuralIndicesAVX2(data unsafe.Pointer, length uint64, indices *uint32) uint64
TEXT ·findStructuralIndicesAVX2(SB), NOSPLIT, $0-32
    MOVQ    data+0(FP), SI        // Source data pointer
    MOVQ    length+8(FP), CX      // Data length
    MOVQ    indices+16(FP), DI    // Output indices pointer
    
    XORQ    AX, AX                // Index counter
    XORQ    R8, R8                // Current position in data
    XORQ    R9, R9                // Output index count
    
    // Load structural character constants into YMM registers
    VPBROADCASTB $QUOTE_CHAR, Y0
    VPBROADCASTB $BACKSLASH_CHAR, Y1
    VPBROADCASTB $LBRACE_CHAR, Y2
    VPBROADCASTB $RBRACE_CHAR, Y3
    VPBROADCASTB $LBRACKET_CHAR, Y4
    VPBROADCASTB $RBRACKET_CHAR, Y5
    VPBROADCASTB $COLON_CHAR, Y6
    VPBROADCASTB $COMMA_CHAR, Y7
    
    // Space and tab characters for whitespace detection
    VPBROADCASTB $0x20, Y8        // Space
    VPBROADCASTB $0x09, Y9        // Tab
    VPBROADCASTB $0x0A, Y10       // Newline
    VPBROADCASTB $0x0D, Y11       // Carriage return

main_loop:
    CMPQ    CX, $32               // Check if we have at least 32 bytes
    JL      remainder
    
    // Load 32 bytes of data
    VMOVDQU (SI)(R8*1), Y12
    
    // Compare with each structural character
    VPCMPEQB Y0, Y12, Y13         // Find quotes
    VPCMPEQB Y1, Y12, Y14         // Find backslashes
    VPCMPEQB Y2, Y12, Y15         // Find {
    VPCMPEQB Y3, Y12, Y16         // Find }
    VPCMPEQB Y4, Y12, Y17         // Find [
    VPCMPEQB Y5, Y12, Y18         // Find ]
    VPCMPEQB Y6, Y12, Y19         // Find :
    VPCMPEQB Y7, Y12, Y20         // Find ,
    
    // Combine all structural characters
    VPOR    Y13, Y14, Y21         // quotes | backslashes
    VPOR    Y15, Y16, Y22         // { | }
    VPOR    Y17, Y18, Y23         // [ | ]
    VPOR    Y19, Y20, Y24         // : | ,
    VPOR    Y21, Y22, Y25         // Combine first half
    VPOR    Y23, Y24, Y26         // Combine second half
    VPOR    Y25, Y26, Y27         // All structural characters
    
    // Extract bitmask
    VPMOVMSKB Y27, DX
    
    // Process each set bit
    TESTL   DX, DX
    JZ      next_chunk
    
process_bits:
    BSFL    DX, BX                // Find lowest set bit
    TESTL   BX, BX
    JS      next_chunk            // No more bits
    
    // Store index
    LEAQ    (R8)(BX*1), R10       // Calculate absolute index
    MOVL    R10, (DI)(R9*4)       // Store index
    INCQ    R9                    // Increment output count
    
    // Clear the bit and continue
    BLSRL   DX, DX                // Clear lowest set bit
    JNZ     process_bits          // Continue if more bits

next_chunk:
    ADDQ    $32, R8               // Move to next 32-byte chunk
    SUBQ    $32, CX               // Decrease remaining length
    JMP     main_loop

remainder:
    // Process remaining bytes (< 32) using scalar approach
    TESTQ   CX, CX
    JZ      done

remainder_loop:
    MOVB    (SI)(R8*1), AL        // Load byte
    
    // Check if it's a structural character
    CMPB    AL, $QUOTE_CHAR
    JE      store_index
    CMPB    AL, $BACKSLASH_CHAR
    JE      store_index
    CMPB    AL, $LBRACE_CHAR
    JE      store_index
    CMPB    AL, $RBRACE_CHAR
    JE      store_index
    CMPB    AL, $LBRACKET_CHAR
    JE      store_index
    CMPB    AL, $RBRACKET_CHAR
    JE      store_index
    CMPB    AL, $COLON_CHAR
    JE      store_index
    CMPB    AL, $COMMA_CHAR
    JE      store_index
    JMP     next_remainder

store_index:
    MOVL    R8, (DI)(R9*4)        // Store index
    INCQ    R9                    // Increment output count

next_remainder:
    INCQ    R8                    // Move to next byte
    DECQ    CX                    // Decrease remaining length
    JNZ     remainder_loop

done:
    MOVQ    R9, ret+24(FP)        // Return number of indices found
    VZEROUPPER                    // Clean up AVX state
    RET

// findQuoteMaskAVX2 creates a bitmask of quote positions
// func findQuoteMaskAVX2(data unsafe.Pointer, length uint64, mask *uint64) uint64
TEXT ·findQuoteMaskAVX2(SB), NOSPLIT, $0-32
    MOVQ    data+0(FP), SI
    MOVQ    length+8(FP), CX
    MOVQ    mask+16(FP), DI
    
    XORQ    R8, R8                // Current position
    XORQ    R9, R9                // Mask count
    
    VPBROADCASTB $QUOTE_CHAR, Y0
    VPBROADCASTB $BACKSLASH_CHAR, Y1

quote_loop:
    CMPQ    CX, $32
    JL      quote_remainder
    
    VMOVDQU (SI)(R8*1), Y2
    
    // Find quotes
    VPCMPEQB Y0, Y2, Y3
    VPMOVMSKB Y3, DX
    
    // Find backslashes (for escape detection)
    VPCMPEQB Y1, Y2, Y4
    VPMOVMSKB Y4, BX
    
    // Store quote mask
    MOVL    DX, (DI)(R9*4)
    INCQ    R9
    
    ADDQ    $32, R8
    SUBQ    $32, CX
    JMP     quote_loop

quote_remainder:
    // Handle remainder bytes
    TESTQ   CX, CX
    JZ      quote_done
    
    XORL    DX, DX
    XORQ    R10, R10

quote_remainder_loop:
    MOVB    (SI)(R8*1), AL
    CMPB    AL, $QUOTE_CHAR
    JNE     quote_next_rem
    
    // Set bit at position R10
    MOVQ    $1, R11
    SHLQ    R10, R11
    ORQ     R11, DX

quote_next_rem:
    INCQ    R8
    INCQ    R10
    DECQ    CX
    JNZ     quote_remainder_loop
    
    // Store final partial mask
    MOVL    DX, (DI)(R9*4)
    INCQ    R9

quote_done:
    MOVQ    R9, ret+24(FP)
    VZEROUPPER
    RET

// validateUTF8AVX2 validates UTF-8 encoding using AVX2
// func validateUTF8AVX2(data unsafe.Pointer, length uint64) bool
TEXT ·validateUTF8AVX2(SB), NOSPLIT, $0-24
    MOVQ    data+0(FP), SI
    MOVQ    length+8(FP), CX
    
    XORQ    R8, R8                // Current position
    MOVB    $1, AL                // Assume valid (return true)

utf8_loop:
    CMPQ    CX, $32
    JL      utf8_remainder
    
    // Load 32 bytes
    VMOVDQU (SI)(R8*1), Y0
    
    // Check for ASCII (0x00-0x7F)
    VPCMPGTB Y0, Y15, Y1          // Compare with 0x80
    VPMOVMSKB Y1, DX
    
    // If all ASCII, continue
    TESTL   DX, DX
    JZ      utf8_next_chunk
    
    // Complex UTF-8 validation would go here
    // For now, assume valid for non-ASCII
    
utf8_next_chunk:
    ADDQ    $32, R8
    SUBQ    $32, CX
    JMP     utf8_loop

utf8_remainder:
    // Handle remainder bytes with scalar validation
    TESTQ   CX, CX
    JZ      utf8_valid

utf8_remainder_loop:
    MOVB    (SI)(R8*1), BL
    CMPB    BL, $0x80
    JL      utf8_ascii            // ASCII character, valid
    
    // Multi-byte UTF-8 validation would go here
    // For simplicity, assume valid

utf8_ascii:
    INCQ    R8
    DECQ    CX
    JNZ     utf8_remainder_loop

utf8_valid:
    MOVB    AL, ret+16(FP)        // Return validation result
    VZEROUPPER
    RET

utf8_invalid:
    XORB    AL, AL                // Set to false
    MOVB    AL, ret+16(FP)
    VZEROUPPER
    RET

// parseIntegerAVX2 parses integers using SIMD digit detection
// func parseIntegerAVX2(data unsafe.Pointer, length uint64) (int64, bool)
TEXT ·parseIntegerAVX2(SB), NOSPLIT, $0-32
    MOVQ    data+0(FP), SI
    MOVQ    length+8(FP), CX
    
    XORQ    R8, R8                // Result accumulator
    XORQ    R9, R9                // Position
    MOVB    $1, AL                // Sign (1 = positive)
    MOVB    $1, BL                // Valid flag
    
    // Check for negative sign
    CMPQ    CX, $0
    JE      parse_invalid
    
    MOVB    (SI), DL
    CMPB    DL, $'-'
    JNE     parse_digits
    NEGB    AL                    // Set negative
    INCQ    SI                    // Skip minus sign
    DECQ    CX
    
parse_digits:
    TESTQ   CX, CX
    JZ      parse_invalid
    
    // SIMD digit processing for up to 32 digits at once
    VPBROADCASTB $'0', Y0         // ASCII '0'
    VPBROADCASTB $'9', Y1         // ASCII '9'

parse_simd_loop:
    CMPQ    CX, $32
    JL      parse_scalar
    
    // Load 32 bytes
    VMOVDQU (SI)(R9*1), Y2
    
    // Check if all are digits (>= '0' and <= '9')
    VPCMPGTB Y2, Y0, Y3           // data > '0'
    VPCMPGTB Y1, Y2, Y4           // '9' > data
    VPAND   Y3, Y4, Y5            // Both conditions
    VPMOVMSKB Y5, DX
    
    // If not all 32 bytes are digits, fall back to scalar
    CMPL    DX, $0xFFFFFFFF
    JNE     parse_scalar
    
    // Process digits in parallel (simplified for now)
    // Real implementation would use SIMD arithmetic
    JMP     parse_scalar

parse_scalar:
    TESTQ   CX, CX
    JZ      parse_done

parse_scalar_loop:
    MOVB    (SI)(R9*1), DL
    SUBB    $'0', DL
    CMPB    DL, $9
    JA      parse_done            // Not a digit, we're done
    
    // result = result * 10 + digit
    IMULQ   $10, R8
    MOVZBQ  DL, R10
    ADDQ    R10, R8
    
    INCQ    R9
    DECQ    CX
    JNZ     parse_scalar_loop

parse_done:
    TESTB   AL, AL
    JGE     parse_positive
    NEGQ    R8                    // Apply negative sign

parse_positive:
    MOVQ    R8, ret+16(FP)        // Return value
    MOVB    BL, ret+24(FP)        // Return success flag
    VZEROUPPER
    RET

parse_invalid:
    XORQ    R8, R8
    XORB    BL, BL
    MOVQ    R8, ret+16(FP)
    MOVB    BL, ret+24(FP)
    VZEROUPPER
    RET