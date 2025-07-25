package main

import (
	"fmt"
	"golang.org/x/text/unicode/norm"
	"unicode"
)

func main() {
	// Test what Rust's to_lowercase().nfkc() should produce
	ch := 'Ⅲ'

	fmt.Printf("=== Simulating Rust's ch.to_lowercase().nfkc() ===\n")
	fmt.Printf("Original: %c (U+%04X)\n", ch, ch)

	// Step 1: to_lowercase()
	lowered := unicode.ToLower(ch)
	fmt.Printf("After to_lowercase(): %c (U+%04X)\n", lowered, lowered)

	// Step 2: nfkc() on the lowercased result
	loweredStr := string(lowered)
	nfkcResult := norm.NFKC.String(loweredStr)
	fmt.Printf("After NFKC: %q\n", nfkcResult)

	// Check NFKC on original character
	originalStr := string(ch)
	originalNFKC := norm.NFKC.String(originalStr)
	fmt.Printf("\nDirect NFKC on original: %q\n", originalNFKC)

	// Check if chars are considered uppercase/need normalization
	fmt.Printf("\nChar analysis:\n")
	fmt.Printf("Ⅲ is_uppercase: %t\n", unicode.IsUpper(ch))
	fmt.Printf("ⅲ is_uppercase: %t\n", unicode.IsUpper(lowered))
	fmt.Printf("Ⅲ needs NFKC: %t\n", !norm.NFKC.IsNormalString(originalStr))
	fmt.Printf("ⅲ needs NFKC: %t\n", !norm.NFKC.IsNormalString(loweredStr))
}
