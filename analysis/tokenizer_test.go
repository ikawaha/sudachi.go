package analysis

import (
	"testing"

	"github.com/ikawaha/sudachi.go/dic"
)

// TestBasicTokenization tests basic Japanese text tokenization
// This matches the core functionality tested in Rust version
func TestBasicTokenization(t *testing.T) {
	// Load test dictionary (assuming resources are available)
	loader := dic.NewDictionaryLoader()
	systemDict, err := loader.LoadSystemDictionary("../resources/system.dic")
	if err != nil {
		t.Fatalf("Failed to load system dictionary: %v", err)
	}

	tokenizer, err := NewTokenizer(systemDict)
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}

	testCases := []struct {
		name     string
		input    string
		mode     Mode
		expected []string // Expected surface forms
	}{
		{
			name:     "Basic sentence ModeA",
			input:    "東京都に行く",
			mode:     ModeA,
			expected: []string{"東京", "都", "に", "行く"},
		},
		{
			name:     "Basic sentence ModeB",
			input:    "東京都に行く",
			mode:     ModeB,
			expected: []string{"東京都", "に", "行く"},
		},
		{
			name:     "Basic sentence ModeC",
			input:    "東京都に行く",
			mode:     ModeC,
			expected: []string{"東京都", "に", "行く"},
		},
		{
			name:     "Simple hiragana",
			input:    "あいうえお",
			mode:     ModeA,
			expected: []string{"あいうえお"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokenizer.SetMode(tc.mode)

			morphemes, err := tokenizer.Tokenize(tc.input)
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			if morphemes == nil {
				t.Fatal("Got nil morpheme list")
			}

			// Extract surface forms
			var surfaces []string
			for _, m := range morphemes.Results() {
				surfaces = append(surfaces, m.Surface())
			}

			// Compare lengths
			if len(surfaces) != len(tc.expected) {
				t.Errorf("Expected %d morphemes, got %d", len(tc.expected), len(surfaces))
				t.Errorf("Expected: %v", tc.expected)
				t.Errorf("Got: %v", surfaces)
				return
			}

			// Compare each morpheme
			for i, expected := range tc.expected {
				if i >= len(surfaces) {
					t.Errorf("Missing morpheme at index %d: expected %s", i, expected)
					continue
				}
				if surfaces[i] != expected {
					t.Errorf("Morpheme %d mismatch: expected %s, got %s", i, expected, surfaces[i])
				}
			}
		})
	}
}

// TestModeComparison tests different tokenization modes
func TestModeComparison(t *testing.T) {
	loader := dic.NewDictionaryLoader()
	systemDict, err := loader.LoadSystemDictionary("../resources/system.dic")
	if err != nil {
		t.Fatalf("Failed to load system dictionary: %v", err)
	}

	tokenizer, err := NewTokenizer(systemDict)
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	input := "労働者協同組合"

	modes := []Mode{ModeA, ModeB, ModeC}
	results := make(map[Mode][]string)

	// Tokenize with each mode
	for _, mode := range modes {
		tokenizer.SetMode(mode)
		morphemes, err := tokenizer.Tokenize(input)
		if err != nil {
			t.Fatalf("Tokenization failed for mode %s: %v", mode.String(), err)
		}

		var surfaces []string
		for _, m := range morphemes.Results() {
			surfaces = append(surfaces, m.Surface())
		}
		results[mode] = surfaces
	}

	// Verify that different modes produce different granularities
	// ModeA should have the most morphemes (finest granularity)
	// ModeC should have the fewest morphemes (coarsest granularity)
	if len(results[ModeA]) < len(results[ModeC]) {
		t.Errorf("Expected ModeA to have more morphemes than ModeC")
		t.Errorf("ModeA: %v", results[ModeA])
		t.Errorf("ModeC: %v", results[ModeC])
	}

	t.Logf("Mode comparison for '%s':", input)
	for _, mode := range modes {
		t.Logf("  %s: %v", mode.String(), results[mode])
	}
}

// TestPluginSystem tests the plugin system integration
func TestPluginSystem(t *testing.T) {
	loader := dic.NewDictionaryLoader()
	systemDict, err := loader.LoadSystemDictionary("../resources/system.dic")
	if err != nil {
		t.Fatalf("Failed to load system dictionary: %v", err)
	}

	tokenizer, err := NewTokenizer(systemDict)
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}
	pm := tokenizer.GetPluginManager()

	// Test that plugin manager is properly initialized
	if pm == nil {
		t.Fatal("Plugin manager is nil")
	}

	// Test plugin queries
	if pm.HasInputTextPlugins() {
		t.Log("Input text plugins are available")
	}
	if pm.HasOOVProviders() {
		t.Log("OOV provider plugins are available")
	}
	if pm.HasPathRewriters() {
		t.Log("Path rewrite plugins are available")
	}

	// Basic tokenization should work even without plugins
	morphemes, err := tokenizer.Tokenize("テスト")
	if err != nil {
		t.Fatalf("Basic tokenization failed: %v", err)
	}

	if morphemes == nil || len(morphemes.Results()) == 0 {
		t.Fatal("Expected at least one morpheme")
	}
}

// TestDebugMode tests debug mode functionality
func TestDebugMode(t *testing.T) {
	loader := dic.NewDictionaryLoader()
	systemDict, err := loader.LoadSystemDictionary("../resources/system.dic")
	if err != nil {
		t.Fatalf("Failed to load system dictionary: %v", err)
	}

	tokenizer, err := NewTokenizer(systemDict)
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}

	// Test debug mode toggle
	tokenizer.SetDebugMode(true)
	if !tokenizer.debugMode {
		t.Error("Debug mode should be enabled")
	}

	tokenizer.SetDebugMode(false)
	if tokenizer.debugMode {
		t.Error("Debug mode should be disabled")
	}

	// Test that tokenization works in debug mode
	tokenizer.SetDebugMode(true)
	morphemes, err := tokenizer.Tokenize("こんにちは")
	if err != nil {
		t.Fatalf("Tokenization failed in debug mode: %v", err)
	}

	if morphemes == nil {
		t.Fatal("Expected morphemes in debug mode")
	}
}

// TestEmptyInput tests handling of empty input
func TestEmptyInput(t *testing.T) {
	loader := dic.NewDictionaryLoader()
	systemDict, err := loader.LoadSystemDictionary("../resources/system.dic")
	if err != nil {
		t.Fatalf("Failed to load system dictionary: %v", err)
	}

	tokenizer, err := NewTokenizer(systemDict)
	if err != nil {
		t.Fatalf("Failed to create tokenizer: %v", err)
	}

	morphemes, err := tokenizer.Tokenize("")
	if err != nil {
		t.Fatalf("Empty input should not cause error: %v", err)
	}

	if morphemes == nil {
		t.Fatal("Expected non-nil morpheme list for empty input")
	}

	// Empty input should produce no morphemes (or only BOS/EOS)
	count := len(morphemes.Results())
	if count > 2 { // Allow for BOS/EOS
		t.Errorf("Empty input produced %d morphemes, expected 0-2", count)
	}
}
