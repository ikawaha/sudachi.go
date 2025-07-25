package main

import (
	"fmt"
	"golang.org/x/text/unicode/norm"
	"unicode"
)

func main() {
	// Test Roman numeral conversion
	upper := 'Ⅲ'
	lower := unicode.ToLower(upper)

	fmt.Printf("=== Unicode Roman Numeral Conversion ===\n")
	fmt.Printf("Original: %c (U+%04X)\n", upper, upper)
	fmt.Printf("ToLower:  %c (U+%04X)\n", lower, lower)
	fmt.Printf("IsUpper:  %t\n", unicode.IsUpper(upper))
	fmt.Printf("IsLower:  %t\n", unicode.IsLower(lower))

	// Test NFKC normalization on both forms
	upperStr := string(upper)
	lowerStr := string(lower)

	upperNFKC := norm.NFKC.String(upperStr)
	lowerNFKC := norm.NFKC.String(lowerStr)

	fmt.Printf("\n=== NFKC Normalization ===\n")
	fmt.Printf("Original %c -> NFKC: %q\n", upper, upperNFKC)
	fmt.Printf("Lower %c -> NFKC: %q\n", lower, lowerNFKC)
	fmt.Printf("Upper IsNormal: %t\n", norm.NFKC.IsNormalString(upperStr))
	fmt.Printf("Lower IsNormal: %t\n", norm.NFKC.IsNormalString(lowerStr))

	// Test the complete processing sequence
	fmt.Printf("\n=== Complete Processing Sequence ===\n")

	// Step 1: Original
	fmt.Printf("1. Input: %c (U+%04X)\n", upper, upper)

	// Step 2: Unicode ToLower
	step2 := unicode.ToLower(upper)
	fmt.Printf("2. ToLower: %c (U+%04X)\n", step2, step2)

	// Step 3: NFKC (if needed)
	step2Str := string(step2)
	if !norm.NFKC.IsNormalString(step2Str) {
		step3 := norm.NFKC.String(step2Str)
		fmt.Printf("3. NFKC: %q\n", step3)
	} else {
		fmt.Printf("3. NFKC: Not needed (already normalized)\n")
	}

	// Test range check for Roman numerals
	fmt.Printf("\n=== Roman Numeral Range Check ===\n")
	for r := 'Ⅰ'; r <= 'Ⅻ'; r++ {
		lower := unicode.ToLower(r)
		fmt.Printf("%c (U+%04X) -> %c (U+%04X)\n", r, r, lower, lower)
	}
}
