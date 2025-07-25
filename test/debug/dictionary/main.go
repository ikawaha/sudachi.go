package main

import (
	"fmt"
	"log"

	"github.com/ikawaha/sudachi.go/dic"
)

func main() {
	// Load system dictionary
	loader := dic.NewDictionaryLoader()
	dict, err := loader.LoadSystemDictionary("resources/system.dic")
	if err != nil {
		log.Fatalf("Failed to load dictionary: %v", err)
	}

	fmt.Println("=== Dictionary Investigation ===")

	// Check if we can search for entries
	fmt.Println("\n1. Searching for entries containing '3':")
	searchDictionary(dict, "3")

	fmt.Println("\n2. Searching for entries containing 'Ⅲ':")
	searchDictionary(dict, "Ⅲ")

	fmt.Println("\n3. Searching for entries containing 'III':")
	searchDictionary(dict, "III")

	// Try to access specific word ID 16438 mentioned in Rust output
	fmt.Println("\n4. Trying to access word ID 16438:")
	if word, err := getWordInfo(dict, 16438); err == nil {
		fmt.Printf("Word ID 16438: %+v\n", word)
	} else {
		fmt.Printf("Failed to access word ID 16438: %v\n", err)
	}

	// Search for entries with 数詞 (numeral) POS
	fmt.Println("\n5. Searching for numeral entries:")
	searchNumerals(dict)
}

func searchDictionary(dict *dic.SystemDictionary, surface string) {
	// Try to search for surface form
	// Note: This is a simplified search - actual implementation may differ
	fmt.Printf("Searching for surface: %s\n", surface)

	// Since we don't have direct search capability, we'll try to iterate
	// This is a placeholder - actual dictionary access methods need to be determined
}

func getWordInfo(dict *dic.SystemDictionary, wordId int) (interface{}, error) {
	// Try to get word information by ID
	// This needs proper implementation based on available methods
	return nil, fmt.Errorf("not implemented")
}

func searchNumerals(dict *dic.SystemDictionary) {
	// Search for entries with numeral part-of-speech
	fmt.Println("Searching for numeral entries...")
	// This needs proper implementation
}
