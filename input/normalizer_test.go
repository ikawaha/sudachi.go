package input

import (
	"testing"
)

func TestNormalizerBasic(t *testing.T) {
	normalizer := NewNormalizer()

	// Test no normalization needed
	result, applied := normalizer.Normalize("hello")
	if applied {
		t.Error("Expected no normalization for ASCII lowercase text")
	}
	if result != "hello" {
		t.Errorf("Expected 'hello', got '%s'", result)
	}

	// Test case folding
	result, applied = normalizer.Normalize("Hello")
	if !applied {
		t.Error("Expected normalization for mixed case text")
	}
	if result != "hello" {
		t.Errorf("Expected 'hello', got '%s'", result)
	}
}

func TestNormalizerNFKC(t *testing.T) {
	normalizer := NewNormalizer()

	// Test NFKC normalization (full-width to half-width)
	input := "Ｈｅｌｌｏ" // Full-width Hello
	result, applied := normalizer.Normalize(input)
	if !applied {
		t.Error("Expected normalization for full-width characters")
	}
	expected := "hello" // Should be normalized and case-folded
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestNormalizerReplacements(t *testing.T) {
	normalizer := NewNormalizer()

	// Add custom replacements
	normalizer.AddReplacement('ａ', "a")
	normalizer.AddReplacement('ｂ', "b")

	result, applied := normalizer.Normalize("ａｂｃ")
	if !applied {
		t.Error("Expected normalization for characters with replacements")
	}
	expected := "abc" // ａ->a, ｂ->b, ｃ normalized by NFKC
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestNormalizerWithInfo(t *testing.T) {
	normalizer := NewNormalizer()
	normalizer.AddReplacement('！', "!")

	input := "Ｈｅｌｌｏ！"
	result, info := normalizer.NormalizeWithInfo(input)

	if !info.Applied {
		t.Error("Expected normalization to be applied")
	}

	if !info.NFKCApplied {
		t.Error("Expected NFKC normalization to be applied")
	}

	if !info.CaseFoldApplied {
		t.Error("Expected case folding to be applied")
	}

	if info.ReplacementsApplied != 1 {
		t.Errorf("Expected 1 replacement, got %d", info.ReplacementsApplied)
	}

	expected := "hello!"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestNormalizerOptions(t *testing.T) {
	// Test with NFKC disabled
	normalizer := NewNormalizerWithOptions(false, true)

	input := "Ｈｅｌｌｏ" // Full-width Hello
	result, applied := normalizer.Normalize(input)

	// Should only apply case folding, not NFKC
	if !applied {
		t.Error("Expected case folding to be applied")
	}

	// Should remain full-width but lowercase
	if result == "hello" {
		t.Error("NFKC should not have been applied")
	}

	// Test with case folding disabled
	normalizer2 := NewNormalizerWithOptions(true, false)
	input2 := "Hello"
	result2, applied2 := normalizer2.Normalize(input2)

	if applied2 {
		t.Error("Expected no normalization when only NFKC is enabled for ASCII")
	}
	if result2 != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", result2)
	}
}


func TestNormalizerReplacementManagement(t *testing.T) {
	normalizer := NewNormalizer()

	// Add replacements
	normalizer.AddReplacement('ａ', "a")
	normalizer.AddReplacement('ｂ', "b")

	result, _ := normalizer.Normalize("ａｂ")
	if result != "ab" {
		t.Errorf("Expected 'ab', got '%s'", result)
	}

	// Remove a replacement
	normalizer.RemoveReplacement('ａ')
	result, _ = normalizer.Normalize("ａｂ")
	if result != "ab" { // ａ should be normalized by NFKC, ｂ by replacement
		t.Errorf("Expected 'ab', got '%s'", result)
	}

	// Clear all replacements
	normalizer.ClearReplacements()
	result, _ = normalizer.Normalize("ａｂ")
	if result != "ab" { // Both should be normalized by NFKC
		t.Errorf("Expected 'ab', got '%s'", result)
	}
}

func TestNormalizerEmpty(t *testing.T) {
	normalizer := NewNormalizer()

	result, applied := normalizer.Normalize("")
	if applied {
		t.Error("Expected no normalization for empty string")
	}
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}

	result, info := normalizer.NormalizeWithInfo("")
	if info.Applied {
		t.Error("Expected no normalization for empty string")
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

	data, err := ParseRewriteDefFromBytes([]byte(content))
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

	replacer := NewStringReplacer(rules)

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


// BenchmarkStringReplacer benchmarks string replacer performance
func BenchmarkStringReplacer(b *testing.B) {
	rules := map[string]string{
		"ｶﾞ": "ガ", "ｷﾞ": "ギ", "ｸﾞ": "グ", "ｹﾞ": "ゲ", "ｺﾞ": "ゴ",
		"ｻﾞ": "ザ", "ｼﾞ": "ジ", "ｽﾞ": "ズ", "ｾﾞ": "ゼ", "ｿﾞ": "ゾ",
	}

	replacer := NewStringReplacer(rules)
	testInput := "ｶﾞｷﾞｸﾞｹﾞｺﾞｻﾞｼﾞｽﾞｾﾞｿﾞ"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		replacer.Replace(testInput)
	}
}
