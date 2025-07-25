package main

import (
	"fmt"
	"log"
	"unicode"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/plugin/input_text"
	"golang.org/x/text/unicode/norm"
)

func main() {
	// Load system dictionary
	loader := dic.NewDictionaryLoader()
	dict, err := loader.LoadSystemDictionary("resources/system.dic")
	if err != nil {
		log.Fatalf("Failed to load dictionary: %v", err)
	}

	// Create DefaultInputTextPlugin
	plugin := input_text.NewDefaultInputTextPlugin()
	err = plugin.SetUp(nil, "", dict.Grammar())
	if err != nil {
		log.Fatalf("Failed to set up plugin: %v", err)
	}

	fmt.Println("=== Debug Roman Numeral Processing ===")

	// Test specific Roman numeral
	testChar := 'Ⅲ'
	input := string(testChar)

	fmt.Printf("Testing character: %c (U+%04X)\n", testChar, testChar)
	fmt.Println()

	// Check individual components
	fmt.Println("--- Individual Component Checks ---")

	// 1. Check if it's uppercase Roman numeral (hardcoded function)
	isUpperRoman := testChar >= 'Ⅰ' && testChar <= 'Ⅿ'
	fmt.Printf("isUpperRomanNumeral (range check): %t\n", isUpperRoman)

	// 2. Check unicode.IsUpper
	isUpper := unicode.IsUpper(testChar)
	fmt.Printf("unicode.IsUpper: %t\n", isUpper)

	// 3. Check unicode.ToLower result
	lowered := unicode.ToLower(testChar)
	fmt.Printf("unicode.ToLower: %c (U+%04X)\n", lowered, lowered)

	// 4. Check NFKC behavior
	nfkcResult := norm.NFKC.String(input)
	nfkcWouldChange := nfkcResult != input
	fmt.Printf("NFKC would change: %t", nfkcWouldChange)
	if nfkcWouldChange {
		fmt.Printf(" -> %s", nfkcResult)
	}
	fmt.Printf("\n")

	// 5. Check if in ignore set (simulate configuration behavior)
	fmt.Println("--- Configuration Checks ---")

	fmt.Println()

	// Test plugin processing
	fmt.Println("--- Plugin Processing ---")
	output, changed := plugin.RewriteImpl(input)
	fmt.Printf("RewriteImpl result: %s", output)
	if changed {
		fmt.Printf(" (changed)")
	}
	fmt.Printf("\n")

	// Expected vs actual
	fmt.Println()
	fmt.Println("--- Expected vs Actual ---")
	expectedOutput := string(lowered)
	fmt.Printf("Expected: %s (convert to lowercase)\n", expectedOutput)
	fmt.Printf("Actual:   %s\n", output)
	if output == expectedOutput {
		fmt.Println("Result:   ✓ CORRECT")
	} else {
		fmt.Println("Result:   ✗ INCORRECT")
	}

	// Test the processing logic step by step
	fmt.Println()
	fmt.Println("--- Step-by-step Processing Simulation ---")

	// Simulate the replaceSlow logic
	needNFKC := !norm.NFKC.IsNormalString(input)
	needLowercase := unicode.IsUpper(testChar)

	fmt.Printf("needNFKC: %t\n", needNFKC)
	fmt.Printf("needLowercase: %t\n", needLowercase)

	if needNFKC || needLowercase {
		fmt.Println("Taking slow path...")

		// Simulate character processing
		needLowercaseChar := unicode.IsUpper(testChar)
		needNFKCChar := true // This would check shouldIgnore, but let's assume true for now

		fmt.Printf("Character level - needLowercase: %t, needNFKC: %t\n", needLowercaseChar, needNFKCChar)

		switch {
		case !needLowercaseChar && !needNFKCChar:
			fmt.Println("Case: (false, false) - no change")
		case needLowercaseChar && !needNFKCChar:
			fmt.Println("Case: (true, false) - lowercase only")
			result := unicode.ToLower(testChar)
			fmt.Printf("Result: %c\n", result)
		case !needLowercaseChar && needNFKCChar:
			fmt.Println("Case: (false, true) - NFKC only")
		case needLowercaseChar && needNFKCChar:
			fmt.Println("Case: (true, true) - both lowercase and NFKC")
			result := unicode.ToLower(testChar)
			nfkcResult := norm.NFKC.String(string(result))
			fmt.Printf("Lowercase: %c, then NFKC: %s\n", result, nfkcResult)
		}
	} else {
		fmt.Println("Taking fast path...")
	}
}
