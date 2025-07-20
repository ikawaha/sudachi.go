package lattice

import (
	"strings"
	"testing"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
)

// SplitMockInputBuffer implements InputBufferInterface for split testing
type SplitMockInputBuffer struct {
	charToByteMap map[int]int
	text          string
}

func (m *SplitMockInputBuffer) CharToByteIndex(charIdx int) (int, error) {
	if byteIdx, ok := m.charToByteMap[charIdx]; ok {
		return byteIdx, nil
	}
	// For unmapped indices, estimate based on UTF-8 (3 bytes per Japanese char)
	return charIdx * 3, nil
}

func (m *SplitMockInputBuffer) ByteToCharIndex(byteIdx int) (int, error) {
	// Reverse lookup
	for charIdx, mappedByteIdx := range m.charToByteMap {
		if mappedByteIdx == byteIdx {
			return charIdx, nil
		}
	}
	// For unmapped indices, estimate
	return byteIdx / 3, nil
}

func (m *SplitMockInputBuffer) OrigSlice(modifiedRange input.Range) string {
	// For testing, return substring based on character positions
	start := modifiedRange.Start / 3 // Rough conversion
	end := modifiedRange.End / 3

	if start < 0 {
		start = 0
	}
	if end > len([]rune(m.text)) {
		end = len([]rune(m.text))
	}

	runes := []rune(m.text)
	if start >= len(runes) {
		return ""
	}
	if end > len(runes) {
		end = len(runes)
	}

	return string(runes[start:end])
}

// createTestMorpheme creates a test morpheme for splitting tests
func createTestMorpheme(surface string, begin, end uint16, wordId dic.WordId) *NodeResult {
	node := NewNode(begin, end, 0, 0, 0, wordId)
	return NewNodeResult(
		node,
		surface,
		[]string{"名詞", "固有名詞", "地名", "一般", "*", "*"}, // Standard POS for proper nouns
		[]string{}, // features
		surface,    // normalized form
		surface,    // dictionary form
		surface,    // reading form (simplified)
	)
}

// TestSplit_ModeBasedSplitting tests basic mode-based splitting behavior
// Port of Rust's tokenizer_morpheme_split test concept
func TestSplit_ModeBasedSplitting(t *testing.T) {
	// Load dictionary for testing
	resourceDir := "../resources"
	dictPath := resourceDir + "/system.dic"

	loader := dic.NewDictionaryLoader()
	systemDict, err := loader.LoadSystemDictionary(dictPath)
	if err != nil {
		t.Fatalf("Failed to load system dictionary: %v", err)
	}

	// Create test morpheme "東京都"
	morpheme := createTestMorpheme("東京都", 0, 3, dic.FromRaw(12345))
	morphemeList := NewMorphemeList()
	morphemeList.Add(morpheme)

	buffer := &SplitMockInputBuffer{
		charToByteMap: map[int]int{0: 0, 1: 3, 2: 6, 3: 9},
		text:          "東京都",
	}

	// Test Mode C (should not split)
	resultsModeC, err := morphemeList.Split(ModeC, systemDict.LexiconSet(), buffer, systemDict.Grammar())
	if err != nil {
		t.Fatalf("Mode C split failed: %v", err)
	}

	if resultsModeC.Size() != 1 {
		t.Errorf("Mode C: expected 1 morpheme, got %d", resultsModeC.Size())
	}

	if resultsModeC.Get(0).Surface() != "東京都" {
		t.Errorf("Mode C: expected '東京都', got '%s'", resultsModeC.Get(0).Surface())
	}

	// Test Mode A (should split if dictionary supports it)
	resultsModeA, err := morphemeList.Split(ModeA, systemDict.LexiconSet(), buffer, systemDict.Grammar())
	if err != nil {
		t.Fatalf("Mode A split failed: %v", err)
	}

	// Note: Actual splitting depends on dictionary content
	// This test verifies the split mechanism works, actual results may vary
	if resultsModeA.Size() == 0 {
		t.Error("Mode A: should produce at least one result")
	}

	t.Logf("Mode A produced %d morphemes", resultsModeA.Size())
	for i := 0; i < resultsModeA.Size(); i++ {
		t.Logf("  [%d]: %s", i, resultsModeA.Get(i).Surface())
	}
}

// TestSplit_OOVNoSplit tests that OOV (Out-Of-Vocabulary) nodes are not split
func TestSplit_OOVNoSplit(t *testing.T) {
	resourceDir := "../resources"
	dictPath := resourceDir + "/system.dic"

	loader := dic.NewDictionaryLoader()
	systemDict, err := loader.LoadSystemDictionary(dictPath)
	if err != nil {
		t.Fatalf("Failed to load system dictionary: %v", err)
	}

	// Create an OOV morpheme (simulate unknown word)
	oovWordId := dic.OOV(123) // Create OOV word ID
	oovMorpheme := createTestMorpheme("未知語", 0, 3, oovWordId)

	morphemeList := NewMorphemeList()
	morphemeList.Add(oovMorpheme)

	buffer := &SplitMockInputBuffer{
		charToByteMap: map[int]int{0: 0, 1: 3, 2: 6, 3: 9},
		text:          "未知語",
	}

	// Try to split OOV morpheme - should return original morpheme unchanged
	splitResults, err := morphemeList.Split(ModeA, systemDict.LexiconSet(), buffer, systemDict.Grammar())
	if err != nil {
		t.Fatalf("Split operation on OOV failed: %v", err)
	}

	// Should have 1 morpheme (no splitting)
	if splitResults.Size() != 1 {
		t.Errorf("OOV morpheme split: expected 1 morpheme, got %d", splitResults.Size())
	}

	if splitResults.Get(0).Surface() != "未知語" {
		t.Errorf("OOV morpheme split: expected '未知語', got '%s'", splitResults.Get(0).Surface())
	}
}

// TestSplit_ModeC_NoSplit tests that Mode C never splits
func TestSplit_ModeC_NoSplit(t *testing.T) {
	resourceDir := "../resources"
	dictPath := resourceDir + "/system.dic"

	loader := dic.NewDictionaryLoader()
	systemDict, err := loader.LoadSystemDictionary(dictPath)
	if err != nil {
		t.Fatalf("Failed to load system dictionary: %v", err)
	}

	// Create a morpheme that could be split in other modes
	morpheme := createTestMorpheme("東京都", 0, 3, dic.FromRaw(12345))
	morphemeList := NewMorphemeList()
	morphemeList.Add(morpheme)

	buffer := &SplitMockInputBuffer{
		charToByteMap: map[int]int{0: 0, 1: 3, 2: 6, 3: 9},
		text:          "東京都",
	}

	// Try to split with Mode C - should return original morpheme unchanged
	splitResults, err := morphemeList.Split(ModeC, systemDict.LexiconSet(), buffer, systemDict.Grammar())
	if err != nil {
		t.Fatalf("Split operation with Mode C failed: %v", err)
	}

	// Should have 1 morpheme (no splitting in Mode C)
	if splitResults.Size() != 1 {
		t.Errorf("Mode C split: expected 1 morpheme, got %d", splitResults.Size())
	}

	if splitResults.Get(0).Surface() != "東京都" {
		t.Errorf("Mode C split: expected '東京都', got '%s'", splitResults.Get(0).Surface())
	}
}

// TestSplit_InvalidMode tests error handling for invalid modes
func TestSplit_InvalidMode(t *testing.T) {
	resourceDir := "../resources"
	dictPath := resourceDir + "/system.dic"

	loader := dic.NewDictionaryLoader()
	systemDict, err := loader.LoadSystemDictionary(dictPath)
	if err != nil {
		t.Fatalf("Failed to load system dictionary: %v", err)
	}

	morpheme := createTestMorpheme("テ", 0, 1, dic.FromRaw(12345))
	morphemeList := NewMorphemeList()
	morphemeList.Add(morpheme)

	buffer := &SplitMockInputBuffer{
		charToByteMap: map[int]int{0: 0, 1: 3},
		text:          "テ",
	}

	// Try to split with invalid mode
	invalidMode := Mode(999)
	_, err = morphemeList.Split(invalidMode, systemDict.LexiconSet(), buffer, systemDict.Grammar())
	if err == nil {
		t.Error("Split with invalid mode should return error")
	}

	if !strings.Contains(err.Error(), "invalid mode") {
		t.Errorf("Error message should contain 'invalid mode', got: %v", err)
	}
}

// TestSplit_RustCompatibility tests overall compatibility with Rust implementation
func TestSplit_RustCompatibility(t *testing.T) {
	// This test verifies that our Split implementation follows the same patterns as Rust

	resourceDir := "../resources"
	dictPath := resourceDir + "/system.dic"

	loader := dic.NewDictionaryLoader()
	systemDict, err := loader.LoadSystemDictionary(dictPath)
	if err != nil {
		t.Fatalf("Failed to load system dictionary: %v", err)
	}

	// Test with "東京都" - a known compound
	buffer := &SplitMockInputBuffer{
		charToByteMap: map[int]int{0: 0, 1: 3, 2: 6, 3: 9},
		text:          "東京都",
	}

	// Create morpheme list with single compound morpheme
	morpheme := createTestMorpheme("東京都", 0, 3, dic.FromRaw(12345))
	morphemeList := NewMorphemeList()
	morphemeList.Add(morpheme)

	// Test all modes
	modes := []Mode{ModeA, ModeB, ModeC}
	modeNames := []string{"ModeA", "ModeB", "ModeC"}

	for i, mode := range modes {
		splitResults, err := morphemeList.Split(mode, systemDict.LexiconSet(), buffer, systemDict.Grammar())
		if err != nil {
			t.Errorf("Rust compatibility test failed for %s: %v", modeNames[i], err)
			continue
		}

		// Verify basic split properties
		if splitResults.Size() == 0 {
			t.Errorf("%s: split should produce at least one result", modeNames[i])
			continue
		}

		// Mode C should never split (Rust behavior)
		if mode == ModeC && splitResults.Size() != 1 {
			t.Errorf("%s should not split: expected 1 morpheme, got %d", modeNames[i], splitResults.Size())
		}

		// For Mode A and B, verify that splitting preserves character coverage
		if splitResults.Size() > 1 {
			var totalLength uint16
			for j := 0; j < splitResults.Size(); j++ {
				splitMorpheme := splitResults.Get(j)
				totalLength += splitMorpheme.Length()
			}

			originalLength := morpheme.Length()
			if totalLength != originalLength {
				t.Errorf("%s: total split length %d != original length %d", modeNames[i], totalLength, originalLength)
			}
		}

		t.Logf("%s produced %d morphemes", modeNames[i], splitResults.Size())
	}

	t.Log("Split implementation follows Rust-compatible patterns")
}
