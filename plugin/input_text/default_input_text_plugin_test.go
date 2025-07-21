package input_text

import (
	"testing"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
)

// zeroGrammar creates a minimal grammar with zero bytes for testing
func zeroGrammar() *dic.Grammar {
	// Create a minimal grammar structure:
	// - 2 bytes for POS size (0)
	// - 2 bytes for left_id_size (0)
	// - 2 bytes for right_id_size (0)
	// - 0 bytes for connection matrix (0 x 0)
	zeroBytes := make([]byte, 6)
	// All zeros - will create empty grammar

	grammar, err := dic.NewGrammar(zeroBytes, 0)
	if err != nil {
		panic("Failed to create zero grammar: " + err.Error())
	}
	return grammar
}

func TestDefaultInputTextPlugin(t *testing.T) {
	plugin := NewDefaultInputTextPlugin()
	err := plugin.SetUpFromData()
	if err != nil {
		t.Skipf("Skipping test due to setup error: %v", err)
		return
	}

	testCases := []struct {
		input    string
		expected string
	}{
		// Half-width katakana with dakuten should be converted
		{"ｶﾞｷﾞｸﾞ", "ガギグ"},
		{"ｻﾞｼﾞｽﾞ", "ザジズ"},
		// Full-width characters should be normalized
		{"ＡＢＣ", "abc"},
		{"１２３", "123"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, applied := plugin.RewriteImpl(tc.input)
			if !applied {
				t.Error("Expected normalization to be applied")
			}
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestPluginInputBufferIntegration(t *testing.T) {
	plugin := NewDefaultInputTextPlugin()
	err := plugin.SetUpFromData()
	if err != nil {
		t.Skipf("Skipping test due to setup error: %v", err)
		return
	}

	original := "ｶﾞｷﾞｸﾞ"
	buffer, err := plugin.CreateNormalizedInputBuffer(original, zeroGrammar())
	if err != nil {
		t.Fatalf("Failed to create normalized input buffer: %v", err)
	}

	if buffer.Original() != original {
		t.Errorf("Original text should be preserved: expected '%s', got '%s'",
			original, buffer.Original())
	}

	expected := "ガギグ"
	if buffer.Modified() != expected {
		t.Errorf("Modified text should be normalized: expected '%s', got '%s'",
			expected, buffer.Modified())
	}
}

func TestPluginJapaneseText(t *testing.T) {
	plugin := NewDefaultInputTextPlugin()
	err := plugin.SetUpFromData()
	if err != nil {
		t.Skipf("Skipping test due to setup error: %v", err)
		return
	}

	// Test with Japanese text containing half-width katakana
	original := "こんにちはｶﾞｷﾞｸﾞ"
	buffer, err := plugin.CreateNormalizedInputBuffer(original, zeroGrammar())
	if err != nil {
		t.Fatalf("Failed to create normalized input buffer: %v", err)
	}

	// The normalized text should have full-width katakana
	expected := "こんにちはガギグ"
	if buffer.Modified() != expected {
		t.Errorf("Expected normalized text '%s', got '%s'", expected, buffer.Modified())
	}
}

func TestDefaultInputTextPluginNormalization(t *testing.T) {
	// Create plugin with default Sudachi rewrite.def (faithful port of Rust version)
	plugin := NewDefaultInputTextPlugin()
	err := plugin.SetUpFromData()
	if err != nil {
		t.Skipf("Skipping Rust compatibility test: %v", err)
		return
	}

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		// Roman numerals should be preserved (ignore normalize)
		{
			name:     "Roman numerals preservation",
			input:    "ⅠⅡⅢⅣⅤⅵⅶⅷⅸⅹ",
			expected: "ⅠⅡⅢⅣⅤⅵⅶⅷⅸⅹ",
		},
		// Half-width katakana with dakuten should be converted
		{
			name:     "Half-width katakana dakuten",
			input:    "ｶﾞｷﾞｸﾞｹﾞｺﾞ",
			expected: "ガギグゲゴ",
		},
		// Mixed case and full-width characters
		{
			name:     "Mixed normalization",
			input:    "ＴｅＳｔ１２３",
			expected: "test123",
		},
		// Kangxi radicals should be preserved (ignore normalize)
		{
			name:     "Kangxi radicals preservation",
			input:    "⺀⺁⺂⺃⺄",
			expected: "⺀⺁⺂⺃⺄",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, changed := plugin.RewriteImpl(tc.input)

			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}

			// For preserved characters, normalization should not report changes
			if tc.input == tc.expected && changed {
				t.Errorf("Expected no changes for preserved input '%s', but changes were reported", tc.input)
			}

			if tc.input != tc.expected && !changed {
				t.Errorf("Expected changes for input '%s' -> '%s', but no changes were reported", tc.input, tc.expected)
			}
		})
	}
}

// TestRewriteDefParser tests the rewrite.def parser functionality
func TestRewriteDefParser(t *testing.T) {
	// Test minimal rewrite.def parsing
	content := `# Test rewrite.def
# ignore normalize chars (single chars)
Ⅰ
Ⅱ
Ⅲ

# replace char list (before after pairs)
ｶﾞ	ガ
ｷﾞ	ギ
ｸﾞ	グ
`

	data, err := input.ParseRewriteDefFromBytes([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse rewrite.def: %v", err)
	}

	// Check ignore normalize characters
	expectedIgnore := []rune{'Ⅰ', 'Ⅱ', 'Ⅲ'}
	for _, r := range expectedIgnore {
		if !data.IgnoreNormalizeChars[r] {
			t.Errorf("Expected ignore normalize character '%c' not found", r)
		}
	}

	// Check replace rules
	expectedReplace := map[string]string{
		"ｶﾞ": "ガ",
		"ｷﾞ": "ギ",
		"ｸﾞ": "グ",
	}
	for before, after := range expectedReplace {
		if data.ReplaceRules[before] != after {
			t.Errorf("Expected replace rule '%s' -> '%s', got '%s'", before, after, data.ReplaceRules[before])
		}
	}
}

// TestStringReplacer tests the string replacer functionality
func TestStringReplacer(t *testing.T) {
	rules := map[string]string{
		"ｶﾞ":   "ガ",
		"ｷﾞ":   "ギ",
		"ｸﾞ":   "グ",
		"test": "TEST",
	}

	replacer := input.NewStringReplacer(rules)

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single replacement",
			input:    "ｶﾞ",
			expected: "ガ",
		},
		{
			name:     "Multiple replacements",
			input:    "ｶﾞｷﾞｸﾞ",
			expected: "ガギグ",
		},
		{
			name:     "Mixed with unreplaced",
			input:    "hello ｶﾞ world ｷﾞ",
			expected: "hello ガ world ギ",
		},
		{
			name:     "ASCII replacement",
			input:    "this is a test",
			expected: "this is a TEST",
		},
		{
			name:     "No replacements",
			input:    "no changes here",
			expected: "no changes here",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := replacer.Replace(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestPluginFastVsSlowPath tests that fast and slow paths work correctly
func TestPluginFastVsSlowPath(t *testing.T) {
	// Create plugin with rewrite.def rules
	plugin := NewDefaultInputTextPlugin()
	err := plugin.SetUpFromData()
	if err != nil {
		t.Skipf("Skipping test due to setup error: %v", err)
		return
	}

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Half-width katakana replacement",
			input:    "ｶﾞｷﾞｸﾞ",
			expected: "ガギグ",
		},
		{
			name:     "Mixed case and replacement",
			input:    "ＡＢｶﾞ",
			expected: "abガ", // Case folding + replacement
		},
		{
			name:     "No changes needed",
			input:    "hello world",
			expected: "hello world",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, _ := plugin.RewriteImpl(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s' for input '%s'", tc.expected, result, tc.input)
			}
		})
	}
}

// TestPluginIgnoreNormalizeChars tests that ignore normalize characters are preserved
func TestPluginIgnoreNormalizeChars(t *testing.T) {
	plugin := NewDefaultInputTextPlugin()
	err := plugin.SetUpFromData()
	if err != nil {
		t.Skipf("Skipping test due to setup error: %v", err)
		return
	}

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Preserved Roman numeral",
			input:    "Ⅰ",
			expected: "Ⅰ",
		},
		{
			name:     "Preserved Kangxi radical",
			input:    "⺀",
			expected: "⺀",
		},
		{
			name:     "Mixed with normal chars",
			input:    "testⅠ⺀TEST",
			expected: "testⅠ⺀test", // TEST should be lowercased but Ⅰ⺀ preserved
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, _ := plugin.RewriteImpl(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// BenchmarkPlugin benchmarks plugin performance
func BenchmarkPlugin(b *testing.B) {
	plugin := NewDefaultInputTextPlugin()
	err := plugin.SetUpFromData()
	if err != nil {
		b.Skipf("Skipping benchmark due to setup error: %v", err)
		return
	}

	testInput := "これはｶﾞｷﾞｸﾞのＴｅＳｔです。"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin.RewriteImpl(testInput)
	}
}

// BenchmarkStringReplacer benchmarks string replacer performance
func BenchmarkStringReplacer(b *testing.B) {
	rules := map[string]string{
		"ｶﾞ": "ガ", "ｷﾞ": "ギ", "ｸﾞ": "グ", "ｹﾞ": "ゲ", "ｺﾞ": "ゴ",
		"ｻﾞ": "ザ", "ｼﾞ": "ジ", "ｽﾞ": "ズ", "ｾﾞ": "ゼ", "ｿﾞ": "ゾ",
	}

	replacer := input.NewStringReplacer(rules)
	testInput := "ｶﾞｷﾞｸﾞｹﾞｺﾞｻﾞｼﾞｽﾞｾﾞｿﾞ"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		replacer.Replace(testInput)
	}
}
