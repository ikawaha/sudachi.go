package lattice

import (
	"fmt"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
)

// Mode represents the unit to split text
type Mode int

const (
	ModeA Mode = iota + 1 // Short unit tokenization
	ModeB                 // Middle unit tokenization (word-like)
	ModeC                 // Long unit tokenization (named entity)
)

// Split splits the morpheme list according to the specified mode
// This is a faithful port of Rust's MorphemeList.split_into() method
func (ml *MorphemeList) Split(mode Mode, lexiconSet *dic.LexiconSet,
	buffer InputBufferInterface, grammar *dic.Grammar) (*MorphemeList, error) {

	result := NewMorphemeList()

	for i := 0; i < ml.Size(); i++ {
		morpheme := ml.Get(i)

		// Rust implementation: get num_splits for the mode (Mode::C returns 0 splits)
		splitIds, err := getSplitIds(morpheme, mode, lexiconSet)
		if err != nil {
			return nil, err
		}

		if len(splitIds) == 0 {
			// No splits needed (including Mode::C), add original morpheme
			result.Add(morpheme)
		} else {
			// Rust implementation: split the morpheme using NodeSplitIterator logic
			splitMorphemes, err := createSplitMorphemes(splitIds, morpheme, lexiconSet, buffer, grammar)
			if err != nil {
				return nil, err
			}

			for _, sm := range splitMorphemes {
				result.Add(sm)
			}
		}
	}

	return result, nil
}

// getSplitIds returns the split word IDs for the given mode
// This matches Rust's num_splits() and split() methods exactly
func getSplitIds(morpheme *NodeResult, mode Mode, lexiconSet *dic.LexiconSet) ([]dic.WordId, error) {
	// Rust implementation: OOV nodes are not split
	if morpheme.Node().IsOOV() {
		return nil, nil
	}

	wordInfo, err := lexiconSet.GetWordInfo(morpheme.Node().WordId())
	if err != nil {
		return nil, err
	}

	// Rust implementation: exact mode matching
	switch mode {
	case ModeA:
		return wordInfo.AUnitSplit, nil
	case ModeB:
		return wordInfo.BUnitSplit, nil
	case ModeC:
		return nil, nil // Mode C never splits
	default:
		return nil, fmt.Errorf("invalid mode: Mode(%d)", mode)
	}
}

// createSplitMorphemes creates split morphemes from word IDs
// This is a faithful port of Rust's NodeSplitIterator logic
func createSplitMorphemes(splitIds []dic.WordId, originalMorpheme *NodeResult,
	lexiconSet *dic.LexiconSet, buffer InputBufferInterface, grammar *dic.Grammar) ([]*NodeResult, error) {

	var result []*NodeResult

	// Start with the original morpheme's byte and char ranges
	charOffset := originalMorpheme.Node().Begin()
	charEnd := originalMorpheme.Node().End()

	// Get the original morpheme's byte range from the InputBuffer
	// This matches Rust's ResultNode.bytes_range() functionality
	originalByteStart, err := buffer.CharToByteIndex(int(charOffset))
	if err != nil {
		return nil, err
	}
	originalByteEnd, err := buffer.CharToByteIndex(int(charEnd))
	if err != nil {
		return nil, err
	}

	byteOffset := originalByteStart

	for i, wordId := range splitIds {
		wordInfo, err := lexiconSet.GetWordInfo(wordId)
		if err != nil {
			return nil, err
		}

		// Store the start positions for this iteration (matching Rust: char_start, byte_start)
		charStart := charOffset
		byteStart := byteOffset

		// Rust implementation: calculate char_end and byte_end exactly as in NodeSplitIterator
		var splitCharEnd uint16
		var splitByteEnd int

		if i+1 == len(splitIds) {
			// Last split: use the original morpheme's end (matching Rust: self.char_end, self.byte_end)
			splitCharEnd = charEnd
			splitByteEnd = originalByteEnd
		} else {
			// Intermediate split: use head_word_length (matching Rust: byte_start + word_info.head_word_length())
			splitByteEnd = byteStart + int(wordInfo.HeadWordLength)
			// Convert byte index back to char index (matching Rust: self.text.ch_idx(byte_end))
			charEndIdx, err := buffer.ByteToCharIndex(splitByteEnd)
			if err != nil {
				return nil, err
			}
			splitCharEnd = uint16(charEndIdx)
		}

		// Extract surface string from InputBuffer using byte range
		// This matches Rust's: input.orig_slice(self.node().bytes_range())
		surface := buffer.OrigSlice(input.Range{Start: byteStart, End: splitByteEnd})

		// Rust implementation: create new Node with specific parameters
		// Uses u16::MAX and i16::MAX as in Rust code (Node::new with MAX values)
		node := NewNode(charStart, splitCharEnd, 65535, 65535, 32767, wordId)

		// Use the WordInfo from the split, not the original morpheme
		// This matches Rust's approach where each split gets its own WordInfo
		// Get POS components from WordInfo's POS ID using Grammar
		pos, err := grammar.GetPOS(wordInfo.PosId)
		if err != nil {
			// If we can't get POS, use empty slice but continue
			pos = []string{}
		}

		// Use WordInfo methods that handle empty values correctly (matching Rust behavior)
		// These methods return a surface form when the specific form is empty
		normalizedForm := wordInfo.GetNormalizedForm()
		dictionaryForm := wordInfo.GetDictionaryForm()
		readingForm := wordInfo.GetReadingForm()

		nodeResult := NewNodeResult(node, surface, pos, nil, normalizedForm, dictionaryForm, readingForm)

		result = append(result, nodeResult)

		// Update offsets for next iteration (matching Rust: self.char_offset = char_end; self.byte_offset = byte_end)
		charOffset = splitCharEnd
		byteOffset = splitByteEnd
	}

	return result, nil
}
