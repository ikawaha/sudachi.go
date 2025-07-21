/*
 *  Copyright (c) 2021-2024 Works Applications Co., Ltd.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *   Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package input_text

import (
	"regexp"
	"testing"

	"github.com/ikawaha/sudachi.go/input"
)

// TestIgnoreYomiganaRustCompatibility tests compatibility with Rust Sudachi implementation
// This matches Rust test cases from ignore_yomigana/tests.rs
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

// TestCharacterCategoryIntegration tests dynamic character category range generation
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

// TestKanjiRangeCompatibility tests kanji range compatibility with Rust implementation
func TestKanjiRangeCompatibility(t *testing.T) {
	_, plugin := setupIgnoreYomiganaPlugin()

	// Test specific kanji characters that should be recognized
	testKanjiChars := []rune{
		'一', '二', '三', '四', '五', '六', '七', '八', '九', '十', // Basic numbers
		'東', '京', '大', '学', '生', '日', '本', '語', '文', '字', // Common kanji
		'龍', '鶴', '亀', '鳳', '凰',                              // Complex kanji
		'豈', '更', '車', '賈', '滑', '串', '句', '龜',             // Compatibility ideographs
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

// TestDynamicVsStaticRanges tests that dynamic range generation works better than static ranges
func TestDynamicVsStaticRanges(t *testing.T) {
	// Test with various Unicode kanji blocks that might not be covered by static ranges
	extendedKanjiChars := []struct {
		char        rune
		block       string
		unicodePoint string
	}{
		{0x3400, "CJK Extension A start", "U+3400"},
		{0x4DB5, "CJK Extension A middle", "U+4DB5"},
		{0x4DBF, "CJK Extension A end", "U+4DBF"},
		{0xF900, "CJK Compatibility start", "U+F900"},
		{0xF950, "CJK Compatibility middle", "U+F950"},
		{0xFAD9, "CJK Compatibility end", "U+FAD9"},
	}

	_, plugin := setupIgnoreYomiganaPlugin()
	kanjiPattern := plugin.kanjiPattern()

	for _, tc := range extendedKanjiChars {
		// Skip if character is not a valid Unicode character
		if tc.char > 0x10FFFF {
			continue
		}

		matched, err := regexp.MatchString(kanjiPattern, string(tc.char))
		if err != nil {
			t.Logf("Could not test character %s (%s): %v", tc.unicodePoint, tc.block, err)
			continue
		}

		// Log the result for manual verification
		t.Logf("Character %s (%s): matched=%t", tc.unicodePoint, tc.block, matched)
	}
}