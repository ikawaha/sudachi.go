package lattice

import (
	"reflect"
	"testing"

	"github.com/ikawaha/sudachi.go/dic"
)

// TestNewNodeResult tests the single NodeResult constructor (Rust-compatible pattern)
func TestNewNodeResult(t *testing.T) {
	// Create test node
	wordId := dic.FromRaw(12345)
	node := NewNode(1, 5, 100, 200, 300, wordId)

	// Test data
	surface := "テスト"
	pos := []string{"名詞", "普通名詞", "一般", "*", "*", "*"}
	features := []string{"feature1", "feature2"}
	normalizedForm := "正規化形"
	dictionaryForm := "辞書形"
	readingForm := "テスト"

	// Create NodeResult
	result := NewNodeResult(node, surface, pos, features, normalizedForm, dictionaryForm, readingForm)

	// Verify all fields are set correctly
	if result.Surface() != surface {
		t.Errorf("Surface() = %q, want %q", result.Surface(), surface)
	}

	if !reflect.DeepEqual(result.POS(), pos) {
		t.Errorf("POS() = %v, want %v", result.POS(), pos)
	}

	if !reflect.DeepEqual(result.Features(), features) {
		t.Errorf("Features() = %v, want %v", result.Features(), features)
	}

	if result.NormalizedForm() != normalizedForm {
		t.Errorf("NormalizedForm() = %q, want %q", result.NormalizedForm(), normalizedForm)
	}

	if result.DictionaryForm() != dictionaryForm {
		t.Errorf("DictionaryForm() = %q, want %q", result.DictionaryForm(), dictionaryForm)
	}

	if result.Reading() != readingForm {
		t.Errorf("Reading() = %q, want %q", result.Reading(), readingForm)
	}

	// Verify node delegation
	if result.Begin() != node.Begin() {
		t.Errorf("Begin() = %d, want %d", result.Begin(), node.Begin())
	}

	if result.End() != node.End() {
		t.Errorf("End() = %d, want %d", result.End(), node.End())
	}

	if result.Length() != node.Length() {
		t.Errorf("Length() = %d, want %d", result.Length(), node.Length())
	}

	if result.Node() != node {
		t.Error("Node() does not return the original node")
	}
}

// TestNodeResult_OOVDetection tests OOV detection functionality
func TestNodeResult_OOVDetection(t *testing.T) {
	tests := []struct {
		name     string
		wordId   dic.WordId
		expected bool
	}{
		{"System dictionary word", dic.FromRaw(0x00001234), false},
		{"User dictionary word", dic.FromRaw(0x10001234), false},
		{"OOV word", dic.FromRaw(0xF0001234), true},
		{"BOS", BOSWordID, true},
		{"EOS", EOSWordID, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := NewNode(0, 1, 0, 0, 0, tt.wordId)
			result := NewNodeResult(node, "test", []string{}, []string{}, "test", "test", "test")

			if result.IsOOV() != tt.expected {
				t.Errorf("IsOOV() = %t, want %t", result.IsOOV(), tt.expected)
			}
		})
	}
}

// TestNodeResult_DictionaryId tests dictionary ID extraction (Rust-compatible)
func TestNodeResult_DictionaryId(t *testing.T) {
	tests := []struct {
		name       string
		wordId     dic.WordId
		expectedId int
	}{
		{"System dictionary", dic.FromRaw(0x00001234), 0},
		{"User dictionary 1", dic.FromRaw(0x10001234), 1},
		{"User dictionary 14", dic.FromRaw(0xE0001234), 14},
		{"OOV word", dic.FromRaw(0xF0001234), -1},
		{"BOS", BOSWordID, -1},
		{"EOS", EOSWordID, -1},
		{"Invalid", dic.Invalid, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := NewNode(0, 1, 0, 0, 0, tt.wordId)
			result := NewNodeResult(node, "test", []string{}, []string{}, "test", "test", "test")

			if result.DictionaryId() != tt.expectedId {
				t.Errorf("DictionaryId() = %d, want %d", result.DictionaryId(), tt.expectedId)
			}
		})
	}
}

// TestNodeResult_String tests string representation
func TestNodeResult_String(t *testing.T) {
	wordId := dic.FromRaw(12345)
	node := NewNode(1, 5, 100, 200, 300, wordId)
	pos := []string{"名詞", "普通名詞", "一般"}

	result := NewNodeResult(node, "テスト", pos, []string{}, "正規化", "辞書形", "テスト")

	str := result.String()
	expected := "NodeResult{surface='テスト', pos=[名詞 普通名詞 一般], begin=1, end=5}"

	if str != expected {
		t.Errorf("String() = %q, want %q", str, expected)
	}
}

// TestMorphemeList tests the morpheme list functionality
func TestMorphemeList(t *testing.T) {
	ml := NewMorphemeList()

	// Test empty list
	if !ml.IsEmpty() {
		t.Error("IsEmpty() = false, want true for new list")
	}

	if ml.Size() != 0 {
		t.Errorf("Size() = %d, want 0 for new list", ml.Size())
	}

	if ml.Get(0) != nil {
		t.Error("Get(0) should return nil for empty list")
	}

	// Add results
	wordId1 := dic.FromRaw(111)
	node1 := NewNode(0, 2, 0, 0, 0, wordId1)
	result1 := NewNodeResult(node1, "テスト", []string{"名詞"}, []string{}, "テスト", "テスト", "テスト")

	wordId2 := dic.FromRaw(222)
	node2 := NewNode(2, 4, 0, 0, 0, wordId2)
	result2 := NewNodeResult(node2, "です", []string{"助動詞"}, []string{}, "です", "だ", "デス")

	ml.Add(result1)
	ml.Add(result2)

	// Test populated list
	if ml.IsEmpty() {
		t.Error("IsEmpty() = true, want false for populated list")
	}

	if ml.Size() != 2 {
		t.Errorf("Size() = %d, want 2", ml.Size())
	}

	if ml.Get(0) != result1 {
		t.Error("Get(0) does not return first result")
	}

	if ml.Get(1) != result2 {
		t.Error("Get(1) does not return second result")
	}

	if ml.Get(2) != nil {
		t.Error("Get(2) should return nil for out of bounds")
	}

	if ml.Get(-1) != nil {
		t.Error("Get(-1) should return nil for negative index")
	}

	// Test Results()
	results := ml.Results()
	if len(results) != 2 {
		t.Errorf("Results() length = %d, want 2", len(results))
	}

	// Test Clear()
	ml.Clear()
	if !ml.IsEmpty() {
		t.Error("IsEmpty() = false, want true after Clear()")
	}

	if ml.Size() != 0 {
		t.Errorf("Size() = %d, want 0 after Clear()", ml.Size())
	}
}

// TestMorphemeList_String tests the string representation of morpheme list
func TestMorphemeList_String(t *testing.T) {
	ml := NewMorphemeList()

	// Test empty list
	emptyStr := ml.String()
	if emptyStr != "MorphemeList{empty}" {
		t.Errorf("Empty list string = %q, want %q", emptyStr, "MorphemeList{empty}")
	}

	// Test populated list
	wordId := dic.FromRaw(12345)
	node := NewNode(0, 2, 0, 0, 0, wordId)
	result := NewNodeResult(node, "テスト", []string{"名詞"}, []string{}, "テスト", "テスト", "テスト")
	ml.Add(result)

	str := ml.String()
	// Should contain the basic structure
	if !contains(str, "MorphemeList{") {
		t.Errorf("String should contain 'MorphemeList{', got: %s", str)
	}

	if !contains(str, "[0]:") {
		t.Errorf("String should contain '[0]:', got: %s", str)
	}

	if !contains(str, "テスト") {
		t.Errorf("String should contain 'テスト', got: %s", str)
	}
}

// TestNodeResult_RustCompatibility tests patterns that match Rust's ResultNode
func TestNodeResult_RustCompatibility(t *testing.T) {
	// Test that our single constructor pattern matches Rust's approach
	// Rust: ResultNode::new(inner, total_cost, begin_bytes, end_bytes, word_info)
	// Go: NewNodeResult(node, surface, pos, features, normalizedForm, dictionaryForm, readingForm)

	wordId := dic.FromRaw(0x12345)
	node := NewNode(5, 10, 150, 250, 400, wordId)

	// All fields explicitly specified (matching Rust's requirement for complete initialization)
	result := NewNodeResult(
		node,
		"表層形",                // surface
		[]string{"名詞", "一般"}, // pos
		[]string{"feature"},  // features
		"正規化形",               // normalizedForm
		"辞書形",                // dictionaryForm
		"ヒョウソウケイ",            // readingForm
	)

	// Verify all fields are independently set (not defaulting to surface)
	if result.Surface() == result.NormalizedForm() && result.Surface() == result.DictionaryForm() {
		// This would be a problem if they were supposed to be different
		// But in this test, we explicitly want them to be different to verify independence
	}

	// Test that normalized/dictionary/reading forms are independently stored
	if result.NormalizedForm() != "正規化形" {
		t.Errorf("NormalizedForm independence test failed: got %q", result.NormalizedForm())
	}

	if result.DictionaryForm() != "辞書形" {
		t.Errorf("DictionaryForm independence test failed: got %q", result.DictionaryForm())
	}

	if result.Reading() != "ヒョウソウケイ" {
		t.Errorf("Reading independence test failed: got %q", result.Reading())
	}

	// Verify node properties are correctly delegated
	if result.Begin() != 5 || result.End() != 10 {
		t.Errorf("Node delegation failed: begin=%d, end=%d", result.Begin(), result.End())
	}
}

// contains is a helper function for string testing
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || (len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				func() bool {
					for i := 0; i <= len(s)-len(substr); i++ {
						if s[i:i+len(substr)] == substr {
							return true
						}
					}
					return false
				}())))
}
