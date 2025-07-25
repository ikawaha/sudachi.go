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

	fmt.Println("=== WordID 6653 Investigation ===")

	// Create WordID for (0, 6653) - system dictionary, word 6653
	wordId := dic.New(0, 6653)
	fmt.Printf("WordID: %s\n", wordId.String())

	// Get word info from lexicon set
	wordInfo, err := dict.LexiconSet().GetWordInfo(wordId)
	if err != nil {
		log.Fatalf("Failed to get word info for WordID 6653: %v", err)
	}

	fmt.Printf("Surface: %q\n", wordInfo.Surface)
	fmt.Printf("DictionaryForm: %q\n", wordInfo.DictionaryForm)
	fmt.Printf("GetDictionaryForm(): %q\n", wordInfo.GetDictionaryForm())
	fmt.Printf("ReadingForm: %q\n", wordInfo.ReadingForm)
	fmt.Printf("GetReadingForm(): %q\n", wordInfo.GetReadingForm())
	fmt.Printf("NormalizedForm: %q\n", wordInfo.NormalizedForm)

	fmt.Println("\n=== WordID 6618 Investigation (for comparison) ===")

	// Also check WordID 6618 that was listed in the lattice
	wordId6618 := dic.New(0, 6618)
	fmt.Printf("WordID: %s\n", wordId6618.String())

	wordInfo6618, err := dict.LexiconSet().GetWordInfo(wordId6618)
	if err != nil {
		log.Fatalf("Failed to get word info for WordID 6618: %v", err)
	}

	fmt.Printf("Surface: %q\n", wordInfo6618.Surface)
	fmt.Printf("DictionaryForm: %q\n", wordInfo6618.DictionaryForm)
	fmt.Printf("GetDictionaryForm(): %q\n", wordInfo6618.GetDictionaryForm())
	fmt.Printf("ReadingForm: %q\n", wordInfo6618.ReadingForm)
	fmt.Printf("GetReadingForm(): %q\n", wordInfo6618.GetReadingForm())
	fmt.Printf("NormalizedForm: %q\n", wordInfo6618.NormalizedForm)
}
