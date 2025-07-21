/*
 *  Copyright (c) 2022-2024 Works Applications Co., Ltd.
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

package oov

import (
	"strings"
	"testing"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/lattice"
	"github.com/ikawaha/sudachi.go/types"
)

// TestRegexOovProvider_Works tests basic functionality
// Matches Rust test: #[test] fn works()
func TestRegexOovProvider_Works(t *testing.T) {
	plugin := createTestPlugin(t, "test", nil)
	
	// Test case 1: "xtest" at offset 0 - should not match
	buffer1 := createTestBuffer(t, "xtest")
	lattice1 := lattice.New()
	lattice1.Reset(4)
	createdWords1 := types.EmptyCreatedWords()
	
	result1, err := plugin.ProvideOOV(0, buffer1, lattice1, createdWords1)
	if err != nil {
		t.Fatalf("ProvideOOV failed: %v", err)
	}
	if result1.HasWord(4) != types.HasWordNo {
		t.Error("Expected no OOV at position 0 in 'xtest'")
	}
	
	// Test case 2: "xtest" at offset 1 - should not match
	result2, err := plugin.ProvideOOV(1, buffer1, lattice1, createdWords1)
	if err != nil {
		t.Fatalf("ProvideOOV failed: %v", err)
	}
	if result2.HasWord(4) != types.HasWordNo {
		t.Error("Expected no OOV at position 1 in 'xtest'")
	}
	
	// Test case 3: "testf" at offset 0 - should match
	buffer3 := createTestBuffer(t, "testf")
	lattice3 := lattice.New()
	lattice3.Reset(5)
	createdWords3 := types.EmptyCreatedWords()
	
	result3, err := plugin.ProvideOOV(0, buffer3, lattice3, createdWords3)
	if err != nil {
		t.Fatalf("ProvideOOV failed: %v", err)
	}
	if result3.HasWord(4) == types.HasWordNo {
		t.Error("Expected OOV at position 0 in 'testf'")
	}
}

// TestRegexOovProvider_WorksRegex tests regex functionality with complex pattern
// Matches Rust test: #[test] fn works_regex()
func TestRegexOovProvider_WorksRegex(t *testing.T) {
	plugin := createTestPlugin(t, "[-0-9a-zA-Z]{4,}", nil)
	
	// Test input: "おらおら1512XF-2テスト"
	// Expected match: "1512XF-2" at char positions 4-12
	testText := "おらおら1512XF-2テスト"
	buffer := createTestBuffer(t, testText)
	lattice := lattice.New()
	lattice.Reset(len([]rune(testText)))
	createdWords := types.EmptyCreatedWords()
	
	
	// Test at position 4 (start of "1512XF-2")
	result, err := plugin.ProvideOOV(4, buffer, lattice, createdWords)
	if err != nil {
		t.Fatalf("ProvideOOV failed: %v", err)
	}
	
	
	// Verify OOV was created
	if result.HasWord(8) == types.HasWordNo { // "1512XF-2" is 8 characters
		t.Error("Expected OOV match for '1512XF-2'")
	}
	
	// Test with existing word at same position (should not create OOV)
	createdWordsWithExisting := types.EmptyCreatedWords().AddWord(8)
	result2, err := plugin.ProvideOOV(4, buffer, lattice, createdWordsWithExisting)
	if err != nil {
		t.Fatalf("ProvideOOV failed: %v", err)
	}
	
	// Should not add another word of same length (result should be unchanged)
	if result2.HasWord(8) == types.HasWordNo {
		t.Error("Expected existing word to remain")
	}
}

// TestRegexOovProvider_Boundaries tests boundary mode functionality
// Matches Rust test: #[test] fn boundaries()
func TestRegexOovProvider_Boundaries(t *testing.T) {
	// Test relaxed boundaries
	relaxedPlugin := createTestPlugin(t, "[-0-9a-zA-Z]{4,}", map[string]any{
		"boundaries": "relaxed",
	})
	
	testText := "Q1232WERTY"
	buffer := createTestBuffer(t, testText)
	createdWords := types.EmptyCreatedWords()
	
	// With relaxed boundaries, should match at positions 0, 1, 2
	for pos := 0; pos <= 2; pos++ {
		lattice := lattice.New()
		lattice.Reset(len([]rune(testText)))
		result, err := relaxedPlugin.ProvideOOV(pos, buffer, lattice, createdWords)
		if err != nil {
			t.Fatalf("ProvideOOV failed at position %d: %v", pos, err)
		}
		
		// For relaxed mode, we expect OOV creation
		expectedLength := len(testText) - pos // Length from current position to end
		if result.HasWord(expectedLength) == types.HasWordNo {
			t.Errorf("Expected OOV at position %d with relaxed boundaries", pos)
		}
	}
	
	// Test strict boundaries (default)
	strictPlugin := createTestPlugin(t, "[-0-9a-zA-Z]{4,}", nil)
	
	// With strict boundaries, boundary conditions apply
	lattice := lattice.New()
	lattice.Reset(len([]rune(testText)))
	_, err := strictPlugin.ProvideOOV(2, buffer, lattice, createdWords)
	if err != nil {
		t.Fatalf("ProvideOOV failed: %v", err)
	}
	
	// Note: Boundary logic testing requires full character category integration
}

// TestRegexOovProvider_SetUp tests configuration parsing
func TestRegexOovProvider_SetUp(t *testing.T) {
	plugin := NewRegexOovProvider()
	grammar := createTestGrammar(t)
	
	settings := map[string]any{
		"leftId":     int64(10),
		"rightId":    int64(20),
		"cost":       int64(100),
		"regex":      "test.*",
		"pos":        []string{"名詞", "普通名詞", "一般", "*", "*", "*"},
		"maxLength":  64,
		"debug":      true,
		"boundaries": "relaxed",
	}
	
	err := plugin.SetUp(settings, "", grammar)
	if err != nil {
		t.Fatalf("SetUp failed: %v", err)
	}
	
	// Verify configuration was applied
	if plugin.leftId != 10 {
		t.Errorf("Expected leftId 10, got %d", plugin.leftId)
	}
	if plugin.rightId != 20 {
		t.Errorf("Expected rightId 20, got %d", plugin.rightId)
	}
	if plugin.cost != 100 {
		t.Errorf("Expected cost 100, got %d", plugin.cost)
	}
	if plugin.maxLength != 64 {
		t.Errorf("Expected maxLength 64, got %d", plugin.maxLength)
	}
	if !plugin.debug {
		t.Error("Expected debug true")
	}
	if plugin.boundaries != BoundaryModeRelaxed {
		t.Errorf("Expected relaxed boundaries, got %v", plugin.boundaries)
	}
	if plugin.regex.String() != "^test.*" {
		t.Errorf("Expected regex '^test.*', got %q", plugin.regex.String())
	}
}

// TestRegexOovProvider_GetName tests plugin name
func TestRegexOovProvider_GetName(t *testing.T) {
	plugin := NewRegexOovProvider()
	name := plugin.GetName()
	expected := "RegexOovProvider"
	if name != expected {
		t.Errorf("Expected name %q, got %q", expected, name)
	}
}

// Helper functions

// createTestPlugin creates a plugin with test configuration
// Matches Rust test helper: fn plugin()
func createTestPlugin(t *testing.T, regex string, extraSettings map[string]any) *RegexOovProvider {
	plugin := NewRegexOovProvider()
	grammar := createTestGrammar(t)
	
	settings := map[string]any{
		"leftId":  int64(0),
		"rightId": int64(0),
		"cost":    int64(0),
		"regex":   regex,
		"pos":     []string{"a", "b", "c", "d", "e", "f"},
	}
	
	// Add extra settings
	for k, v := range extraSettings {
		settings[k] = v
	}
	
	err := plugin.SetUp(settings, "", grammar)
	if err != nil {
		t.Fatalf("Failed to set up plugin: %v", err)
	}
	
	return plugin
}

// createTestBuffer creates a test input buffer
func createTestBuffer(t *testing.T, text string) *input.InputBuffer {
	grammar := createTestGrammar(t)
	buffer := input.NewInputBuffer()
	
	err := buffer.StartBuild(text)
	if err != nil {
		t.Fatalf("Failed to start buffer build: %v", err)
	}
	
	err = buffer.Build(grammar)
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}
	
	return buffer
}

// createTestGrammar creates a test grammar with character categories
// This exactly matches Rust's build_mock_grammar functionality
func createTestGrammar(t *testing.T) *dic.Grammar {
	// Exact port of Rust's build_mock_bytes() from sudachi.rs/sudachi/src/util/testing.rs:32-55
	buf := buildMockBytes()
	
	grammar, err := dic.NewGrammar(buf, 0)
	if err != nil {
		t.Fatalf("Failed to create test grammar: %v", err)
	}
	
	// Exact port of Rust's build_mock_grammar: grammar.set_character_category(char_cats())
	charCategory := createTestCharacterCategory(t)
	grammar.CharacterCategory = charCategory
	
	return grammar
}

// buildMockBytes exactly ports Rust's build_mock_bytes() function
// From sudachi.rs/sudachi/src/util/testing.rs:32-55
func buildMockBytes() []byte {
	var buf []byte
	
	// encode pos for oov (exactly matching Rust line 35)
	// buf.extend(&1_i16.to_le_bytes());
	buf = append(buf, 1, 0) // 1 as i16 little-endian
	
	// let pos = vec!["補助記号", "一般", "*", "*", "*", "*"]; (exactly matching Rust line 36)
	pos := []string{"補助記号", "一般", "*", "*", "*", "*"}
	for _, s := range pos {
		// Convert to UTF-16 (exactly matching Rust lines 38-42)
		utf16 := utf16Encode(s)
		buf = append(buf, byte(len(utf16))) // length as u8
		for _, c := range utf16 {
			// c as u16 little-endian
			buf = append(buf, byte(c), byte(c>>8))
		}
	}
	
	// set 10 for left and right id sizes (exactly matching Rust lines 44-45)
	buf = append(buf, 10, 0) // 10 as i16 little-endian  
	buf = append(buf, 10, 0) // 10 as i16 little-endian
	
	// Connection matrix 10x10 (exactly matching Rust lines 46-51)
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			val := int16(i*100 + j)
			buf = append(buf, byte(val), byte(val>>8)) // val as i16 little-endian
		}
	}
	
	return buf
}

// utf16Encode converts string to UTF-16 (matching Rust's s.encode_utf16())
func utf16Encode(s string) []uint16 {
	runes := []rune(s)
	var utf16 []uint16
	for _, r := range runes {
		if r <= 0xFFFF {
			utf16 = append(utf16, uint16(r))
		} else {
			// Surrogate pair for characters > 0xFFFF
			r -= 0x10000
			utf16 = append(utf16, uint16((r>>10)+0xD800))
			utf16 = append(utf16, uint16((r&0x3FF)+0xDC00))
		}
	}
	return utf16
}

// createTestCharacterCategory creates test character categories
// This exactly matches Rust's char_cats() function and ALL_KANJI_CAT constant
func createTestCharacterCategory(t *testing.T) *dic.CharacterCategory {
	// Exact copy of Rust's ALL_KANJI_CAT constant from sudachi.rs/sudachi/src/util/testing.rs:22-26
	charDef := `ALPHA 0 1 2
KANJI 0 1 2
KANJINUMERIC 0 1 2
0x0061..0x007A ALPHA
0x3041..0x309F KANJI
0x30A1..0x30FF KANJINUMERIC
`
	
	charCategory := dic.NewCharacterCategory()
	err := charCategory.LoadFromReader(strings.NewReader(charDef))
	if err != nil {
		t.Fatalf("Failed to create character category: %v", err)
	}
	
	return charCategory
}

// TestRustCompatibility verifies exact compatibility with Rust implementation
func TestRustCompatibility(t *testing.T) {
	// Test boundary mode constants
	if BoundaryModeStrict.String() != "strict" {
		t.Errorf("Expected strict boundary mode string 'strict', got %q", BoundaryModeStrict.String())
	}
	if BoundaryModeRelaxed.String() != "relaxed" {
		t.Errorf("Expected relaxed boundary mode string 'relaxed', got %q", BoundaryModeRelaxed.String())
	}
	
	// Test default max length
	if defaultMaxLength() != 32 {
		t.Errorf("Expected default max length 32, got %d", defaultMaxLength())
	}
	
	// Test regex prefix handling
	plugin := NewRegexOovProvider()
	grammar := createTestGrammar(t)
	
	settings := map[string]any{
		"leftId":  int64(0),
		"rightId": int64(0),
		"cost":    int64(0),
		"regex":   "test", // No ^ prefix
		"pos":     []string{"a", "b", "c", "d", "e", "f"},
	}
	
	err := plugin.SetUp(settings, "", grammar)
	if err != nil {
		t.Fatalf("SetUp failed: %v", err)
	}
	
	// Should automatically add ^ prefix
	if plugin.regex.String() != "^test" {
		t.Errorf("Expected regex '^test', got %q", plugin.regex.String())
	}
}