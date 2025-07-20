package dic

import (
	"fmt"
)

// LexiconSet manages multiple lexicons as one unified lexicon
// The first lexicon must be from system dictionary
// Handles multiple lexicons (system + user dictionaries)
type LexiconSet struct {
	lexicons     []*Lexicon
	posOffsets   []int
	numSystemPOS int
}

// NewLexiconSet creates a new LexiconSet with system lexicon
func NewLexiconSet(systemLexicon *Lexicon, numSystemPOS int) *LexiconSet {
	// Set dictionary ID for system lexicon
	_ = systemLexicon.SetDicID(0) // System lexicon should always have ID 0
	return &LexiconSet{
		lexicons:     []*Lexicon{systemLexicon},
		posOffsets:   []int{0},
		numSystemPOS: numSystemPOS,
	}
}

// Append adds a lexicon to the lexicon set
func (s *LexiconSet) Append(lexicon *Lexicon, posOffset int) error {
	if s.IsFull() {
		return fmt.Errorf("too many dictionaries: maximum number of dictionaries exceeded")
	}

	// Set dictionary ID for the new lexicon
	lexicon.SetDicID(uint8(len(s.lexicons)))
	s.lexicons = append(s.lexicons, lexicon)
	s.posOffsets = append(s.posOffsets, posOffset)

	return nil
}

// IsFull returns true if dictionary capacity is full
func (s *LexiconSet) IsFull() bool {
	return len(s.lexicons) >= MaxDictionaries
}

// Lookup performs dictionary lookup starting from the offset
// Searches dictionaries in reverse order: user dictionaries first, then system dictionary
func (s *LexiconSet) Lookup(input []byte, offset int) (*LexiconSetIterator, error) {
	// Create iterator that searches in reverse order (user dictionaries first)
	return NewLexiconSetIteratorFromPool(s.lexicons, input, offset)
}

// GetWordInfo returns WordInfo for given WordId
func (s *LexiconSet) GetWordInfo(wordId WordId) (*WordInfo, error) {
	dicId := wordId.Dic()
	if int(dicId) >= len(s.lexicons) {
		return nil, fmt.Errorf("invalid dictionary ID: dicId=%d, available=%d", dicId, len(s.lexicons))
	}

	lexicon := s.lexicons[dicId]
	wordInfo, err := lexicon.GetWordInfo(wordId.Word())
	if err != nil {
		return nil, err
	}

	// Adjust POS ID for user dictionaries
	if dicId > 0 && int(wordInfo.PosId) >= s.numSystemPOS {
		// User-defined part-of-speech
		adjustedPosId := int(wordInfo.PosId) - s.numSystemPOS + s.posOffsets[dicId]
		wordInfo.PosId = uint16(adjustedPosId)
	}

	return wordInfo, nil
}

// GetWordParam returns word parameters for given WordId
func (s *LexiconSet) GetWordParam(wordId WordId) (int16, int16, int16) {
	dicId := wordId.Dic()
	if int(dicId) >= len(s.lexicons) {
		// Return default values for invalid dictionary ID
		return 0, 0, 10000 // High cost for invalid entries
	}

	lexicon := s.lexicons[dicId]
	return lexicon.GetWordParam(wordId.Word())
}

// LexiconSetIterator iterates through multiple lexicons in reverse order
type LexiconSetIterator struct {
	lexicons     []*Lexicon
	currentIndex int
	currentIter  *LexiconIterator
	input        []byte
	offset       int
}

// NewLexiconSetIterator creates a new iterator for LexiconSet
func NewLexiconSetIterator(lexicons []*Lexicon, input []byte, offset int) (*LexiconSetIterator, error) {
	// Start from the last lexicon (user dictionaries first)
	currentIndex := len(lexicons) - 1
	var currentIter *LexiconIterator

	if currentIndex >= 0 {
		var err error
		currentIter, err = lexicons[currentIndex].Lookup(input, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to create iterator for lexicon %d: %w", currentIndex, err)
		}
	}

	return &LexiconSetIterator{
		lexicons:     lexicons,
		currentIndex: currentIndex,
		currentIter:  currentIter,
		input:        input,
		offset:       offset,
	}, nil
}

// Next returns the next lexicon entry from the iterator
func (iter *LexiconSetIterator) Next() (*LexiconEntry, error) {
	for iter.currentIndex >= 0 {
		if iter.currentIter != nil {
			entry, err := iter.currentIter.Next()
			if entry != nil {
				return entry, err
			}
			if err != nil {
				return nil, err
			}
		}

		// Move to the next lexicon (previous index due to reverse order)
		iter.currentIndex--
		if iter.currentIndex >= 0 {
			var err error
			iter.currentIter, err = iter.lexicons[iter.currentIndex].Lookup(iter.input, iter.offset)
			if err != nil {
				return nil, fmt.Errorf("failed to create iterator for lexicon %d: %w", iter.currentIndex, err)
			}
		} else {
			iter.currentIter = nil
		}
	}

	return nil, nil // No more entries
}

// Size returns the total size of all lexicons in the set
func (s *LexiconSet) Size() uint32 {
	var total uint32
	for _, lexicon := range s.lexicons {
		total += lexicon.Size()
	}
	return total
}
