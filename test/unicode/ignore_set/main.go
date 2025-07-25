package main

import (
	"fmt"
	"log"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/plugin/input_text"
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

	// Test ignore set for Roman numerals
	testChars := []rune{'Ⅲ', 'ⅲ', 'A', 'a'}

	fmt.Println("=== Ignore Set Test ===")
	for _, ch := range testChars {
		// Access the ignoreNormalizeSet through plugin methods
		// Note: We cannot access private fields directly, so we need to test behavior
		fmt.Printf("Character: %c (U+%04X)\n", ch, ch)
	}

	// Test actual processing
	fmt.Println("\n=== Processing Test ===")
	testInput := "Ⅲ"
	result, changed := plugin.RewriteImpl(testInput)
	fmt.Printf("Input: %q\n", testInput)
	fmt.Printf("Output: %q\n", result)
	fmt.Printf("Changed: %t\n", changed)
}
