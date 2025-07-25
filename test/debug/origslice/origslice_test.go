package origslice_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/ikawaha/sudachi.go/analysis"
	"github.com/ikawaha/sudachi.go/dic"
)

// TestOrigSliceDebug tests the actual OrigSlice issue found in CLI output
func TestOrigSliceDebug(t *testing.T) {
	// Load dictionary
	loader := dic.NewDictionaryLoader()
	dict, err := loader.LoadSystemDictionary("../../../resources/system.dic")
	if err != nil {
		t.Fatalf("Failed to load system dictionary: %v", err)
	}

	// Create tokenizer using LoadConfigFromResourceDir
	builder := analysis.NewTokenizerBuilder(dict)
	builder, err = builder.LoadConfigFromResourceDir("../../../resources")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	builder.SetDebug(true)

	tokenizer, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build tokenizer: %v", err)
	}

	tokenizer.SetMode(analysis.ModeC)
	tokenizer.SetDebugMode(true)

	// Test problematic input
	input := "ひＢら①がⅢな"

	// This should produce the same issue as the CLI
	morphemes, err := tokenizer.Tokenize(input)
	if err != nil {
		t.Fatalf("Tokenization failed: %v", err)
	}

	// Check each morpheme's surface form
	for i := 0; i < morphemes.Size(); i++ {
		morpheme := morphemes.Get(i)
		surface := morpheme.Surface()

		fmt.Printf("Morpheme %d: surface='%s' (len=%d, bytes=%v)\n",
			i, surface, len(surface), []byte(surface))

		// Check if we have garbled characters (�)
		for _, r := range surface {
			if r == '\uFFFD' { // Unicode replacement character
				t.Errorf("Found garbled character (�) in morpheme %d surface: %q", i, surface)
			}
		}
	}
}

// TestOrigSliceDirectCall tests OrigSlice directly to verify the mapping issue
func TestOrigSliceDirectCall(t *testing.T) {
	// Load dictionary
	loader := dic.NewDictionaryLoader()
	dict, err := loader.LoadSystemDictionary("../../../resources/system.dic")
	if err != nil {
		t.Fatalf("Failed to load system dictionary: %v", err)
	}

	// Create tokenizer using LoadConfigFromResourceDir
	builder := analysis.NewTokenizerBuilder(dict)
	builder, err = builder.LoadConfigFromResourceDir("../../../resources")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	builder.SetDebug(true)

	tokenizer, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build tokenizer: %v", err)
	}

	// Enable debug mode on tokenizer
	tokenizer.SetDebugMode(true)

	// Test with problematic input
	input := "ひＢら①がⅢな"

	// Enable debug for OS args checking
	if len(os.Args) == 0 {
		os.Args = []string{"test", "--debug"}
	}

	morphemes, err := tokenizer.Tokenize(input)
	if err != nil {
		t.Fatalf("Tokenization failed: %v", err)
	}

	t.Logf("Input: %q (%d bytes)", input, len(input))
	t.Logf("Found %d morphemes", morphemes.Size())

	// Check each morpheme
	for i := 0; i < morphemes.Size(); i++ {
		morpheme := morphemes.Get(i)
		surface := morpheme.Surface()

		t.Logf("Morpheme %d: range[%d,%d) surface=%q (%d bytes)",
			i, morpheme.Begin(), morpheme.End(), surface, len(surface))

		// Report any garbled characters
		hasGarbled := false
		for _, r := range surface {
			if r == '\uFFFD' {
				hasGarbled = true
				break
			}
		}

		if hasGarbled {
			t.Errorf("Morpheme %d has garbled characters: %q", i, surface)
		}
	}
}
