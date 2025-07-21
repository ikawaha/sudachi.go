package oov

import (
	"strings"
	"testing"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
)

// zeroGrammar creates a minimal grammar for testing (equivalent to Rust zero_grammar)
func zeroGrammar() *dic.Grammar {
	// Create a minimal grammar structure with zero bytes
	zeroBytes := make([]byte, 6)
	grammar, err := dic.NewGrammar(zeroBytes, 0)
	if err != nil {
		panic("Failed to create zero grammar: " + err.Error())
	}
	return grammar
}

func TestMeCabOovPluginBasic(t *testing.T) {
	// Create mock character category
	charCategory := dic.NewCharacterCategory()

	// Add mock category definitions
	err := charCategory.LoadFromReader(strings.NewReader(`
DEFAULT 0 1 0
KANJI 0 0 2
HIRAGANA 0 1 2
KATAKANA 1 1 2
ALPHA 1 1 0

0x3041..0x309F HIRAGANA
0x30A1..0x30FF KATAKANA
0x4E00..0x9FFF KANJI
0x0041..0x005A ALPHA
0x0061..0x007A ALPHA
`))
	if err != nil {
		t.Fatalf("Failed to load character category: %v", err)
	}

	// Create mock unknown word definitions
	unkDefs := dic.NewUnknownWordDefinitions()
	err = unkDefs.LoadFromReader(strings.NewReader(`
DEFAULT,5968,5968,3857,補助記号,一般,*,*,*,*
KANJI,5139,5139,14657,名詞,普通名詞,一般,*,*,*
HIRAGANA,5139,5139,16012,名詞,普通名詞,一般,*,*,*
KATAKANA,5139,5139,10980,名詞,普通名詞,一般,*,*,*
ALPHA,5139,5139,11633,名詞,普通名詞,一般,*,*,*
`), charCategory)
	if err != nil {
		t.Fatalf("Failed to load unknown word definitions: %v", err)
	}

	// Create mock grammar
	grammar := zeroGrammar()

	// Create plugin
	plugin := NewMeCabOovPlugin(charCategory, unkDefs, grammar)

	// Test input buffer
	buffer := input.NewInputBuffer()
	err = buffer.StartBuild("テスト")
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}

	err = buffer.Build(grammar)
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}

	// Mock hasKnownWord slice (all false, indicating no known words)
	hasKnownWord := make([]bool, buffer.CharCount())

	// Generate OOV candidates
	candidates, err := plugin.ProvideOOVCandidates(buffer, hasKnownWord)
	if err != nil {
		t.Fatalf("Failed to provide OOV: %v", err)
	}

	// Verify candidates were generated
	if len(candidates) == 0 {
		t.Error("Expected OOV candidates to be generated")
	}

	// Check that all candidates have valid definitions
	for i, candidate := range candidates {
		if candidate.Definition == nil {
			t.Errorf("Candidate %d has nil definition", i)
		}
		if candidate.Begin < 0 || candidate.End <= candidate.Begin {
			t.Errorf("Candidate %d has invalid position: %d-%d", i, candidate.Begin, candidate.End)
		}
	}
}

func TestMeCabOovPluginFromResourceDir(t *testing.T) {
	resourceDir := "../sudachi.rs/resources"

	// Create mock grammar for testing
	grammar := zeroGrammar()

	plugin, err := NewMeCabOovPluginFromResourceDir(resourceDir, grammar)
	if err != nil {
		t.Skipf("Skipping test due to missing resource files: %v", err)
		return
	}

	// Test that plugin was created successfully
	if plugin == nil {
		t.Fatal("Plugin should not be nil")
	}

	if plugin.GetCharacterCategory() == nil {
		t.Error("Character category should not be nil")
	}

	if plugin.GetUnknownWordDefinitions() == nil {
		t.Error("Unknown word definitions should not be nil")
	}
}
