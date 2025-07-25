package main

import (
	"fmt"
	"golang.org/x/text/unicode/norm"
	"unicode"
)

func main() {
	fmt.Println("=== Detailed Normalization Analysis ===")

	roman := "Ⅲ"
	fmt.Printf("Original: %s (U+%04X)\n", roman, []rune(roman)[0])

	// NFKC normalization
	nfkc := norm.NFKC.String(roman)
	fmt.Printf("NFKC result: %s\n", nfkc)
	for i, r := range nfkc {
		fmt.Printf("  [%d] %c (U+%04X) - IsLetter: %t, IsUpper: %t\n", i, r, r, unicode.IsLetter(r), unicode.IsUpper(r))
	}

	// Check if there's a small roman numeral equivalent
	smallRoman := "ⅲ" // U+2172
	fmt.Printf("\nSmall roman numeral: %s (U+%04X)\n", smallRoman, []rune(smallRoman)[0])

	smallNfkc := norm.NFKC.String(smallRoman)
	fmt.Printf("Small roman NFKC: %s\n", smallNfkc)
	for i, r := range smallNfkc {
		fmt.Printf("  [%d] %c (U+%04X) - IsLetter: %t, IsLower: %t\n", i, r, r, unicode.IsLetter(r), unicode.IsLower(r))
	}

	// Test what Rust seems to be doing - converting to lowercase roman numeral
	fmt.Println("\n=== Conversion Analysis ===")

	// Direct character mapping approach
	romanToSmall := map[rune]rune{
		'Ⅰ': 'ⅰ', // U+2160 -> U+2170
		'Ⅱ': 'ⅱ', // U+2161 -> U+2171
		'Ⅲ': 'ⅲ', // U+2162 -> U+2172
		'Ⅳ': 'ⅳ', // U+2163 -> U+2173
		'Ⅴ': 'ⅴ', // U+2164 -> U+2174
	}

	for large, small := range romanToSmall {
		fmt.Printf("%c (U+%04X) -> %c (U+%04X)\n", large, large, small, small)

		// Test what NFKC does to each
		largeNfkc := norm.NFKC.String(string(large))
		smallNfkc := norm.NFKC.String(string(small))
		fmt.Printf("  NFKC: %c -> %s, %c -> %s\n", large, largeNfkc, small, smallNfkc)
	}

	// Unicode case conversion
	fmt.Println("\n=== Unicode Case Conversion ===")
	fmt.Printf("unicode.ToLower('Ⅲ'): %c (U+%04X)\n", unicode.ToLower('Ⅲ'), unicode.ToLower('Ⅲ'))
}
