package input_text

import (
	"fmt"
	"regexp"
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

	// Load character category from real resources for testing
	charCategory, err := dic.LoadCharacterCategoryFromResourceDir("../../resources")
	if err != nil {
		panic("Failed to load character category: " + err.Error())
	}

	// Create a more complete grammar with character category
	zeroBytes := make([]byte, 100) // Larger buffer for character category
	grammar, err := dic.NewGrammar(zeroBytes, 0)
	if err != nil {
		panic("Failed to create grammar: " + err.Error())
	}

	// Set character category in grammar so SetUp can use it
	grammar.CharacterCategory = charCategory

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

	// Load character category from real resources for testing
	charCategory, err := dic.LoadCharacterCategoryFromResourceDir("../../resources")
	if err != nil {
		t.Fatalf("Error test due to missing resources: %v", err)
	}

	// Create a basic grammar for testing
	zeroBytes := make([]byte, 100)
	grammar, err := dic.NewGrammar(zeroBytes, 0)
	if err != nil {
		t.Fatalf("Failed to create grammar: %v", err)
	}

	// Set character category in grammar so SetUp can use it
	grammar.CharacterCategory = charCategory

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

// TestIgnoreYomiganaRustCompatibility tests compatibility with Rust Sudachi implementation
// This replaces individual TestIgnoreYomiganaAtMiddle/AtEnd with comprehensive Rust test cases
func TestIgnoreYomiganaRustCompatibility(t *testing.T) {
	_, plugin := setupIgnoreYomiganaPlugin()

	tests := []struct {
		name        string
		original    string
		expected    string
		description string
	}{
		{
			name:        "ignore_yomigana_at_middle",
			original:    "徳島（とくしま）に行く",
			expected:    "徳島に行く",
			description: "Rust test: ignore_yomigana_at_middle",
		},
		{
			name:        "ignore_yomigana_at_end",
			original:    "徳島（とくしま）",
			expected:    "徳島",
			description: "Rust test: ignore_yomigana_at_end",
		},
		{
			name:        "ignore_yomigana_multiple",
			original:    "徳島（とくしま）に行（い）く",
			expected:    "徳島に行く",
			description: "Rust test: ignore_yomigana_multiple",
		},
		{
			name:        "ignore_yomigana_multiple_brace_types",
			original:    "徳島(とくしま)に行（い）く",
			expected:    "徳島に行く",
			description: "Rust test: ignore_yomigana_multiple_brace_types",
		},
		{
			name:        "dont_ignore_not_yomigana",
			original:    "徳島に（よく）行く",
			expected:    "徳島に（よく）行く",
			description: "Rust test: dont_ignore_not_yomigana",
		},
		{
			name:        "dont_ignore_too_long",
			original:    "徳島（ながいよみ）に行く",
			expected:    "徳島（ながいよみ）に行く",
			description: "Rust test: dont_ignore_too_long (maxYomiganaLength=4)",
		},
		{
			name:        "ignore_hiragana",
			original:    "徳島(とくしま)に行（い）く",
			expected:    "徳島に行く",
			description: "Rust test: ignore_hiragana",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create input buffer
			buffer := input.NewInputBuffer()
			err := buffer.StartBuild(tt.original)
			if err != nil {
				t.Fatalf("Failed to start buffer build: %v", err)
			}

			// Apply rewrite before building (matching plugin usage pattern)
			err = plugin.Rewrite(buffer)
			if err != nil {
				t.Fatalf("Failed to rewrite: %v", err)
			}

			// Build buffer with grammar
			grammar, _ := setupIgnoreYomiganaPlugin()
			err = buffer.Build(grammar)
			if err != nil {
				t.Fatalf("Failed to build buffer: %v", err)
			}

			// Check results
			if buffer.Modified() != tt.expected {
				t.Errorf("%s: got %q, want %q", tt.description, buffer.Modified(), tt.expected)
			}

			// Verify original is preserved
			if buffer.Original() != tt.original {
				t.Errorf("Original text changed: got %q, want %q", buffer.Original(), tt.original)
			}
		})
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

// TestCharacterCategoryIntegration tests dynamic character category range generation
// This integrates and expands the original CharacterClassPatterns test
func TestCharacterCategoryIntegration(t *testing.T) {
	_, plugin := setupIgnoreYomiganaPlugin()

	// Test that character category ranges are being used when available
	kanjiPattern := plugin.kanjiPattern()
	readingPattern := plugin.readingPattern()

	// Verify patterns are not empty
	if kanjiPattern == "[]" {
		t.Error("Kanji pattern should not be empty")
	}
	if readingPattern == "[]" {
		t.Error("Reading pattern should not be empty")
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

	// Test with various kanji characters including extended ranges
	testChars := []struct {
		char        rune
		category    string
		shouldMatch bool
	}{
		{'一', "kanji", true},   // U+4E00 - CJK Unified Ideographs start
		{'龯', "kanji", true},   // U+9FEF - CJK Unified Ideographs end
		{'豈', "kanji", true},   // U+F900 - CJK Compatibility Ideographs
		{'鶴', "kanji", true},   // U+9DB4 - within CJK range
		{'ぁ', "reading", true}, // Hiragana
		{'ゟ', "reading", true}, // Hiragana end
		{'ァ', "reading", true}, // Katakana
		{'ヿ', "reading", true}, // Katakana end
	}

	for _, tc := range testChars {
		var pattern string
		switch tc.category {
		case "kanji":
			pattern = kanjiPattern
		case "reading":
			pattern = readingPattern
		}

		matched, err := regexp.MatchString(pattern, string(tc.char))
		if err != nil {
			t.Fatalf("Failed to match pattern for char %c: %v", tc.char, err)
		}

		if matched != tc.shouldMatch {
			t.Errorf("Character %c (%U) in %s pattern: got %t, want %t",
				tc.char, tc.char, tc.category, matched, tc.shouldMatch)
		}
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

// TestKanjiRangeCompatibility tests kanji range compatibility with Rust implementation
func TestKanjiRangeCompatibility(t *testing.T) {
	_, plugin := setupIgnoreYomiganaPlugin()

	// Test specific kanji characters that should be recognized
	testKanjiChars := []rune{
		'一', '二', '三', '四', '五', '六', '七', '八', '九', '十', // Basic numbers
		'東', '京', '大', '学', '生', '日', '本', '語', '文', '字', // Common kanji
		'龍', '鶴', '亀', '鳳', '凰', // Complex kanji
		'豈', '更', '車', '賈', '滑', '串', '句', '龜', // Compatibility ideographs
	}

	kanjiPattern := plugin.kanjiPattern()
	t.Logf("Generated kanji pattern: %s", kanjiPattern)

	for _, char := range testKanjiChars {
		matched, err := regexp.MatchString(kanjiPattern, string(char))
		if err != nil {
			t.Fatalf("Failed to match kanji pattern for char %c: %v", char, err)
		}

		if !matched {
			t.Errorf("Kanji character %c (%U) should match kanji pattern", char, char)
		}
	}

	// Test non-kanji characters that should NOT match
	nonKanjiChars := []rune{
		'あ', 'い', 'う', 'え', 'お', // Hiragana
		'ア', 'イ', 'ウ', 'エ', 'オ', // Katakana
		'a', 'b', 'c', 'A', 'B', 'C', // Latin
		'1', '2', '3', '０', '１', '２', // Numbers
		'!', '@', '#', '$', '%', // Symbols
	}

	for _, char := range nonKanjiChars {
		matched, err := regexp.MatchString(kanjiPattern, string(char))
		if err != nil {
			t.Fatalf("Failed to match kanji pattern for non-kanji char %c: %v", char, err)
		}

		if matched {
			t.Errorf("Non-kanji character %c (%U) should NOT match kanji pattern", char, char)
		}
	}
}

// TestRegexPatternGeneration tests the regex pattern generation
func TestRegexPatternGeneration(t *testing.T) {
	_, plugin := setupIgnoreYomiganaPlugin()

	// Test that the generated regex pattern works correctly
	regex := plugin.regex
	if regex == nil {
		t.Fatal("Plugin regex should not be nil after setup")
	}

	// Test cases matching Rust behavior
	testCases := []struct {
		input       string
		shouldMatch bool
		description string
	}{
		{"徳島（とくしま）", true, "Basic kanji with hiragana yomigana"},
		{"東京(とう)", true, "Kanji with western parentheses"},
		{"漢字（かんじ）", true, "Standard yomigana pattern"},
		{"徳島に（よく）", false, "Parentheses not directly after kanji"},
		{"（とくしま）", false, "Yomigana without preceding kanji"},
		{"abc（def）", false, "Non-kanji with parentheses"},
		{"漢字（かんじかんじかんじかんじ）", false, "Yomigana too long (>4 chars)"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			matches := regex.FindAllStringSubmatch(tc.input, -1)
			hasMatch := len(matches) > 0

			if hasMatch != tc.shouldMatch {
				t.Errorf("Input %q: expected match=%t, got match=%t", tc.input, tc.shouldMatch, hasMatch)
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
