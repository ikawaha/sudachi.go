package main

import (
	"fmt"
	"os"

	"github.com/ikawaha/sudachi.go/analysis"
	"github.com/ikawaha/sudachi.go/dic"
)

func main() {
	sentence := "大抵は十三四人｜漬ってるがたまには誰も居ない事がある。"

	fmt.Printf("Original sentence: %q\n", sentence)
	fmt.Printf("Sentence length: %d bytes\n", len(sentence))

	// Test with original bytes
	for i, b := range []byte(sentence) {
		fmt.Printf("  [%d] 0x%02x (%d)\n", i, b, b)
	}

	// Try to analyze
	loader := dic.NewDictionaryLoader()
	systemDict, err := loader.LoadSystemDictionary("resources/system.dic")
	if err != nil {
		fmt.Printf("Failed to load dictionary: %v\n", err)
		os.Exit(1)
	}

	tokenizer, err := analysis.NewTokenizer(systemDict)
	if err != nil {
		fmt.Printf("Failed to create tokenizer: %v\n", err)
		os.Exit(1)
	}

	morphemes, err := tokenizer.Tokenize(sentence)
	if err != nil {
		fmt.Printf("Failed to tokenize: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nAnalysis result:\n")
	for i := 0; i < morphemes.Size(); i++ {
		morpheme := morphemes.Get(i)
		surface := morpheme.Surface()
		fmt.Printf("  [%d] Surface: %q (len=%d)\n", i, surface, len(surface))
	}
}
