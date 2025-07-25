package main

import (
	"fmt"
	"log"
	"strings"

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

	fmt.Println("=== Comprehensive Roman Numeral Processing Test ===")
	fmt.Println()

	// Test all Roman numeral characters
	upperRomanNumerals := []rune{
		'Ⅰ', 'Ⅱ', 'Ⅲ', 'Ⅳ', 'Ⅴ', 'Ⅵ', 'Ⅶ', 'Ⅷ', 'Ⅸ', 'Ⅹ',
		'Ⅺ', 'Ⅻ', 'Ⅼ', 'Ⅽ', 'Ⅾ', 'Ⅿ',
	}

	lowerRomanNumerals := []rune{
		'ⅰ', 'ⅱ', 'ⅲ', 'ⅳ', 'ⅴ', 'ⅵ', 'ⅶ', 'ⅷ', 'ⅸ', 'ⅹ',
		'ⅺ', 'ⅻ', 'ⅼ', 'ⅽ', 'ⅾ', 'ⅿ',
	}

	// Test individual Roman numerals
	fmt.Println("--- Individual Roman Numeral Processing ---")
	testIndividualRomanNumerals(plugin, upperRomanNumerals, "Upper")
	testIndividualRomanNumerals(plugin, lowerRomanNumerals, "Lower")

	// Test mixed text scenarios
	fmt.Println("\n--- Mixed Text Scenarios ---")
	testMixedScenarios(plugin)

	// Test configuration-based processing
	fmt.Println("\n--- Configuration Mechanism Verification ---")
	testConfigurationMechanism(plugin)

	// Test edge cases
	fmt.Println("\n--- Edge Cases ---")
	testEdgeCases(plugin)
}

func testIndividualRomanNumerals(plugin *input_text.DefaultInputTextPlugin, numerals []rune, category string) {
	fmt.Printf("--- %s Roman Numerals ---\n", category)

	for _, ch := range numerals {
		input := string(ch)
		output, changed := plugin.RewriteImpl(input)

		// Check if in ignore set (should skip NFKC)
		nfkcResult := norm.NFKC.String(input)
		nfkcWouldChange := nfkcResult != input

		fmt.Printf("Input: %c (U+%04X)\n", ch, ch)
		fmt.Printf("  Output: %s", output)
		if changed {
			fmt.Printf(" (changed)")
		}
		fmt.Printf("\n")
		fmt.Printf("  NFKC would change: %t", nfkcWouldChange)
		if nfkcWouldChange {
			fmt.Printf(" -> %s", nfkcResult)
		}
		fmt.Printf("\n")
		fmt.Printf("  Expected behavior: ")
		if category == "Upper" {
			fmt.Printf("uppercase->lowercase conversion, skip NFKC\n")
		} else {
			fmt.Printf("skip both case conversion and NFKC\n")
		}
		fmt.Println()
	}
}

func testMixedScenarios(plugin *input_text.DefaultInputTextPlugin) {
	testCases := []struct {
		name  string
		input string
		desc  string
	}{
		{
			name:  "Mixed Roman numerals",
			input: "ⅠⅡⅢⅳⅴⅵ",
			desc:  "Mix of upper and lower Roman numerals",
		},
		{
			name:  "Roman with Japanese",
			input: "第Ⅲ章",
			desc:  "Roman numeral in Japanese context",
		},
		{
			name:  "Roman with Latin",
			input: "ChapterⅤ",
			desc:  "Roman numeral with Latin letters",
		},
		{
			name:  "Multiple Romans",
			input: "ⅠからⅩまで",
			desc:  "Multiple Roman numerals in sentence",
		},
		{
			name:  "Roman with symbols",
			input: "Ⅲ-Ⅳ",
			desc:  "Roman numerals with hyphen",
		},
	}

	for _, tc := range testCases {
		fmt.Printf("Test: %s\n", tc.name)
		fmt.Printf("Input: %s\n", tc.input)
		fmt.Printf("Description: %s\n", tc.desc)

		output, changed := plugin.RewriteImpl(tc.input)

		fmt.Printf("Output: %s", output)
		if changed {
			fmt.Printf(" (changed)")
		}
		fmt.Printf("\n")

		// Show character-by-character breakdown
		if changed {
			fmt.Println("Character breakdown:")
			inputRunes := []rune(tc.input)
			outputRunes := []rune(output)

			for i, inputChar := range inputRunes {
				if i < len(outputRunes) {
					outputChar := outputRunes[i]
					if inputChar != outputChar {
						fmt.Printf("  %c (U+%04X) -> %c (U+%04X)\n",
							inputChar, inputChar, outputChar, outputChar)
					} else {
						fmt.Printf("  %c (U+%04X) -> unchanged\n", inputChar, inputChar)
					}
				}
			}
		}
		fmt.Println()
	}
}

func testConfigurationMechanism(plugin *input_text.DefaultInputTextPlugin) {
	fmt.Println("Testing that Roman numerals are in ignore_normalize_set...")

	// Test some key Roman numerals
	testChars := []rune{'Ⅰ', 'Ⅴ', 'Ⅹ', 'ⅰ', 'ⅴ', 'ⅹ'}

	for _, ch := range testChars {
		input := string(ch)

		// Test NFKC behavior
		nfkcResult := norm.NFKC.String(input)
		nfkcWouldChange := nfkcResult != input

		// Test plugin behavior
		output, changed := plugin.RewriteImpl(input)

		fmt.Printf("Character: %c (U+%04X)\n", ch, ch)
		fmt.Printf("  NFKC would change: %t", nfkcWouldChange)
		if nfkcWouldChange {
			fmt.Printf(" -> %s", nfkcResult)
		}
		fmt.Printf("\n")
		fmt.Printf("  Plugin output: %s", output)
		if changed {
			fmt.Printf(" (changed)")
		}
		fmt.Printf("\n")

		// Verify expected behavior
		if ch >= 'Ⅰ' && ch <= 'Ⅿ' {
			// Uppercase Roman: should convert to lowercase
			expected := strings.ToLower(input)
			if output == expected {
				fmt.Printf("  ✓ Correct: uppercase Roman converted to lowercase\n")
			} else {
				fmt.Printf("  ✗ Error: expected %s, got %s\n", expected, output)
			}
		} else {
			// Lowercase Roman: should remain unchanged
			if output == input {
				fmt.Printf("  ✓ Correct: lowercase Roman unchanged\n")
			} else {
				fmt.Printf("  ✗ Error: lowercase Roman should not change\n")
			}
		}
		fmt.Println()
	}
}

func testEdgeCases(plugin *input_text.DefaultInputTextPlugin) {
	testCases := []struct {
		name  string
		input string
		desc  string
	}{
		{
			name:  "Empty string",
			input: "",
			desc:  "Empty input",
		},
		{
			name:  "Single space",
			input: " ",
			desc:  "Single space character",
		},
		{
			name:  "Roman at start",
			input: "Ⅰst",
			desc:  "Roman numeral at beginning",
		},
		{
			name:  "Roman at end",
			input: "ChapterⅤ",
			desc:  "Roman numeral at end",
		},
		{
			name:  "Only Romans",
			input: "ⅠⅡⅢⅣⅤ",
			desc:  "Only Roman numerals",
		},
		{
			name:  "Boundary test",
			input: "Ⅰ Ⅱ",
			desc:  "Roman numerals with space",
		},
	}

	for _, tc := range testCases {
		fmt.Printf("Test: %s\n", tc.name)
		fmt.Printf("Input: %q\n", tc.input)
		fmt.Printf("Description: %s\n", tc.desc)

		output, changed := plugin.RewriteImpl(tc.input)

		fmt.Printf("Output: %q", output)
		if changed {
			fmt.Printf(" (changed)")
		}
		fmt.Printf("\n")
		fmt.Println()
	}
}
