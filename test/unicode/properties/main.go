package main

import (
	"fmt"
	"unicode"
)

func main() {
	fmt.Println("=== Unicode Properties Test ===")

	// Test various characters
	testChars := []rune{
		'A', 'a', // Regular Latin
		'Ⅰ', 'ⅰ', // Roman numerals I
		'Ⅲ', 'ⅲ', // Roman numerals III
		'Ⅴ', 'ⅴ', // Roman numerals V
		'Ⅹ', 'ⅹ', // Roman numerals X
		'Ⅿ', 'ⅿ', // Roman numerals M
	}

	fmt.Printf("%-8s %-8s %-8s %-8s %-8s\n", "Char", "Upper?", "Lower?", "ToUpper", "ToLower")
	fmt.Println("--------------------------------------------")

	for _, ch := range testChars {
		isUpper := unicode.IsUpper(ch)
		isLower := unicode.IsLower(ch)
		toUpper := unicode.ToUpper(ch)
		toLower := unicode.ToLower(ch)

		fmt.Printf("%-8c %-8t %-8t %-8c %-8c\n",
			ch, isUpper, isLower, toUpper, toLower)
	}

	fmt.Println()
	fmt.Println("=== Roman Numeral Range Analysis ===")

	fmt.Println("Upper Roman Numerals (U+2160-U+216F):")
	for ch := rune(0x2160); ch <= 0x216F; ch++ {
		isUpper := unicode.IsUpper(ch)
		isLower := unicode.IsLower(ch)
		toLower := unicode.ToLower(ch)

		fmt.Printf("  %c (U+%04X): IsUpper=%t, IsLower=%t, ToLower=%c (U+%04X)\n",
			ch, ch, isUpper, isLower, toLower, toLower)
	}

	fmt.Println()
	fmt.Println("Lower Roman Numerals (U+2170-U+217F):")
	for ch := rune(0x2170); ch <= 0x217F; ch++ {
		isUpper := unicode.IsUpper(ch)
		isLower := unicode.IsLower(ch)
		toUpper := unicode.ToUpper(ch)

		fmt.Printf("  %c (U+%04X): IsUpper=%t, IsLower=%t, ToUpper=%c (U+%04X)\n",
			ch, ch, isUpper, isLower, toUpper, toUpper)
	}
}
