package input_text

import (
	"fmt"
	"testing"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
)

// buildMockSettingForYomigana creates test settings (matching Rust build_mock_setting)
func buildMockSettingForYomigana() map[string]any {
	return map[string]any{
		"leftBrackets":      []any{"（", "("},
		"rightBrackets":     []any{"）", ")"},
		"maxYomiganaLength": 4, // Match Rust test default
	}
}

// setupIgnoreYomiganaPlugin creates plugin for testing (matching Rust setup function)
func setupIgnoreYomiganaPlugin() (*dic.Grammar, *IgnoreYomiganaPlugin) {
	settings := buildMockSettingForYomigana()

	// Create a more complete grammar with character category
	zeroBytes := make([]byte, 100) // Larger buffer for character category
	grammar, err := dic.NewGrammar(zeroBytes, 0)
	if err != nil {
		panic("Failed to create grammar: " + err.Error())
	}

	plugin := NewIgnoreYomiganaPlugin()

	err = plugin.SetUp(settings, "", grammar)
	if err != nil {
		panic("Failed to setup plugin: " + err.Error())
	}

	return grammar, plugin
}

func TestIgnoreYomiganaPlugin_NewIgnoreYomiganaPlugin(t *testing.T) {
	plugin := NewIgnoreYomiganaPlugin()

	if plugin == nil {
		t.Fatal("NewIgnoreYomiganaPlugin should not return nil")
	}

	// Check default values
	if plugin.characterCategory != nil {
		t.Error("Expected characterCategory to be nil before setup")
	}

	if plugin.regex != nil {
		t.Error("Expected regex to be nil before setup")
	}

	if plugin.maxYomiganaLength != 0 {
		t.Errorf("Expected default maxYomiganaLength to be 0, got %d", plugin.maxYomiganaLength)
	}
}

func TestIgnoreYomiganaPlugin_GetName(t *testing.T) {
	plugin := NewIgnoreYomiganaPlugin()
	name := plugin.GetName()

	expected := "IgnoreYomiganaPlugin"
	if name != expected {
		t.Errorf("Expected name '%s', got '%s'", expected, name)
	}
}

func TestIgnoreYomiganaPlugin_SetUp(t *testing.T) {
	plugin := NewIgnoreYomiganaPlugin()

	// Create a basic grammar for testing
	zeroBytes := make([]byte, 100)
	grammar, err := dic.NewGrammar(zeroBytes, 0)
	if err != nil {
		t.Fatalf("Failed to create grammar: %v", err)
	}

	// Test with nil settings (should use defaults)
	err = plugin.SetUp(nil, "", grammar)
	if err != nil {
		t.Fatalf("SetUp with nil settings failed: %v", err)
	}

	// Should have default brackets
	if len(plugin.leftBracketSet) == 0 {
		t.Error("Expected default left brackets to be set")
	}

	if len(plugin.rightBracketSet) == 0 {
		t.Error("Expected default right brackets to be set")
	}

	// Should have default max length
	if plugin.maxYomiganaLength == 0 {
		t.Error("Expected default maxYomiganaLength to be set")
	}

	// Should have compiled regex
	if plugin.regex == nil {
		t.Error("Expected regex to be compiled after setup")
	}

	// Test with valid settings
	settings := map[string]any{
		"leftBrackets":      []any{"（", "["},
		"rightBrackets":     []any{"）", "]"},
		"maxYomiganaLength": 20,
	}

	plugin2 := NewIgnoreYomiganaPlugin()
	err = plugin2.SetUp(settings, "", grammar)
	if err != nil {
		t.Fatalf("SetUp with valid settings failed: %v", err)
	}

	// Verify settings were applied
	if len(plugin2.leftBracketSet) != 2 {
		t.Errorf("Expected 2 left brackets, got %d", len(plugin2.leftBracketSet))
	}

	if len(plugin2.rightBracketSet) != 2 {
		t.Errorf("Expected 2 right brackets, got %d", len(plugin2.rightBracketSet))
	}

	if plugin2.maxYomiganaLength != 20 {
		t.Errorf("Expected maxYomiganaLength to be 20, got %d", plugin2.maxYomiganaLength)
	}
}

func TestIgnoreYomiganaPlugin_SetUpWithoutGrammar(t *testing.T) {
	plugin := NewIgnoreYomiganaPlugin()

	// Test with nil grammar (should fail)
	err := plugin.SetUp(nil, "", nil)
	if err == nil {
		t.Error("Expected error when setting up without grammar")
	}
}

// TestIgnoreYomiganaAtMiddle matches Rust ignore_yomigana_at_middle test
func TestIgnoreYomiganaAtMiddle(t *testing.T) {
	original := "徳島（とくしま）に行く"
	normalized := "徳島に行く"

	_, plugin := setupIgnoreYomiganaPlugin()
	buffer := input.NewInputBuffer()

	err := buffer.StartBuild(original)
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}

	err = plugin.Rewrite(buffer)
	if err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	err = buffer.Build(zeroGrammarForProlongedSoundMark()) // Reuse the grammar function
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}

	if buffer.Original() != original {
		t.Errorf("Expected original '%s', got '%s'", original, buffer.Original())
	}

	if buffer.Modified() != normalized {
		t.Errorf("Expected normalized '%s', got '%s'", normalized, buffer.Modified())
	}
}

// TestIgnoreYomiganaAtEnd matches Rust ignore_yomigana_at_end test
func TestIgnoreYomiganaAtEnd(t *testing.T) {
	original := "徳島（とくしま）"
	normalized := "徳島"

	_, plugin := setupIgnoreYomiganaPlugin()
	buffer := input.NewInputBuffer()

	err := buffer.StartBuild(original)
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}

	err = plugin.Rewrite(buffer)
	if err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	err = buffer.Build(zeroGrammarForProlongedSoundMark())
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}

	if buffer.Original() != original {
		t.Errorf("Expected original '%s', got '%s'", original, buffer.Original())
	}

	if buffer.Modified() != normalized {
		t.Errorf("Expected normalized '%s', got '%s'", normalized, buffer.Modified())
	}
}

func TestIgnoreYomiganaPlugin_RewriteImpl(t *testing.T) {
	_, plugin := setupIgnoreYomiganaPlugin()

	testCases := []struct {
		name     string
		input    string
		expected string
		changed  bool
	}{
		{
			name:     "Basic yomigana removal (default max yomigana length = 4)",
			input:    "東京（とうきょう）",
			expected: "東京（とうきょう）",
			changed:  false,
		},
		{
			name:     "Basic yomigana removal",
			input:    "大阪（おおさか）",
			expected: "大阪",
			changed:  true,
		},
		{
			name:     "Multiple yomigana",
			input:    "東京（とうきょう）と大阪（おおさか）",
			expected: "東京（とうきょう）と大阪",
			changed:  true,
		},
		{
			name:     "No yomigana",
			input:    "東京に行く",
			expected: "東京に行く",
			changed:  false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
			changed:  false,
		},
		{
			name:     "Parentheses without kanji",
			input:    "（とうきょう）",
			expected: "（とうきょう）",
			changed:  false,
		},
		{
			name:     "Western parentheses",
			input:    "大阪(おおさか)",
			expected: "大阪",
			changed:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, changed := plugin.RewriteImpl(tc.input)

			if result != tc.expected {
				t.Errorf("Expected result '%s', got '%s'", tc.expected, result)
			}

			if changed != tc.changed {
				t.Errorf("Expected changed %v, got %v", tc.changed, changed)
			}
		})
	}
}

func TestIgnoreYomiganaPlugin_CharacterClassPatterns(t *testing.T) {
	_, plugin := setupIgnoreYomiganaPlugin()

	// Test kanji pattern
	kanjiPattern := plugin.kanjiPattern()
	if kanjiPattern == "" {
		t.Error("Expected non-empty kanji pattern")
	}

	// Test reading pattern
	readingPattern := plugin.readingPattern()
	if readingPattern == "" {
		t.Error("Expected non-empty reading pattern")
	}

	// Test bracket patterns
	leftBracketPattern := plugin.anyOfPattern(plugin.leftBracketSet)
	if leftBracketPattern == "" {
		t.Error("Expected non-empty left bracket pattern")
	}

	rightBracketPattern := plugin.anyOfPattern(plugin.rightBracketSet)
	if rightBracketPattern == "" {
		t.Error("Expected non-empty right bracket pattern")
	}
}

func TestIgnoreYomiganaPlugin_HelperFunctions(t *testing.T) {
	_, plugin := setupIgnoreYomiganaPlugin()

	// Test kanji detection
	testCases := []struct {
		char    rune
		isKanji bool
		isKana  bool
	}{
		{'東', true, false},  // Kanji
		{'京', true, false},  // Kanji
		{'と', false, true},  // Hiragana
		{'ト', false, true},  // Katakana
		{'a', false, false}, // ASCII
		{'１', false, false}, // Full-width number
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("char_%c", tc.char), func(t *testing.T) {
			isKanji := plugin.isKanji(tc.char)
			if isKanji != tc.isKanji {
				t.Errorf("Expected isKanji(%c) = %v, got %v", tc.char, tc.isKanji, isKanji)
			}

			isKana := plugin.isHiraganaOrKatakana(tc.char)
			if isKana != tc.isKana {
				t.Errorf("Expected isHiraganaOrKatakana(%c) = %v, got %v", tc.char, tc.isKana, isKana)
			}
		})
	}
}

// BenchmarkIgnoreYomiganaPlugin benchmarks plugin performance
func BenchmarkIgnoreYomiganaPlugin(b *testing.B) {
	_, plugin := setupIgnoreYomiganaPlugin()
	input := "東京（とうきょう）と大阪（おおさか）と名古屋（なごや）"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin.RewriteImpl(input)
	}
}
