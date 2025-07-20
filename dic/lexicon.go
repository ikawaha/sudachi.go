package dic

import (
	"fmt"
)

// MaxDictionaries is the maximum number of dictionaries that can be loaded
const MaxDictionaries = 15

// Lexicon represents a complete lexicon with all components
type Lexicon struct {
	trie        *Trie
	wordIdTable *WordIdTable
	wordParams  *WordParams
	wordInfos   *WordInfos
	lexId       uint8
}

// LexiconEntry represents a result from lexicon lookup
type LexiconEntry struct {
	WordId WordId
	End    int
}

// NewLexicon creates a new Lexicon from components
func NewLexicon(trie *Trie, wordIdTable *WordIdTable, wordParams *WordParams, wordInfos *WordInfos) *Lexicon {
	return &Lexicon{
		trie:        trie,
		wordIdTable: wordIdTable,
		wordParams:  wordParams,
		wordInfos:   wordInfos,
		lexId:       255, // u8::MAX equivalent
	}
}

// SetDicID assigns lexicon ID to the current Lexicon
func (l *Lexicon) SetDicID(id uint8) error {
	if id >= MaxDictionaries {
		return fmt.Errorf("dictionary ID out of range: invalid ID %d", id)
	}
	l.lexId = id
	return nil
}

// wordId creates a WordId with the lexicon's dictionary ID
func (l *Lexicon) wordId(rawId uint32) WordId {
	return New(l.lexId, rawId)
}

// Lookup returns an iterator of word_id and end of words that match given input
func (l *Lexicon) Lookup(input []byte, offset int) (*LexiconIterator, error) {
	if l.lexId >= 15 { // MAX_DICTIONARIES
		return nil, fmt.Errorf("lexicon ID out of range: invalid ID %d", l.lexId)
	}

	trieIter, err := l.trie.CommonPrefixIterator(input, offset)
	if err != nil {
		return nil, err
	}

	return NewLexiconIteratorFromPool(l, trieIter, input, offset), nil
}

// GetWordInfo returns WordInfo for given word_id
func (l *Lexicon) GetWordInfo(wordId uint32) (*WordInfo, error) {
	return l.wordInfos.GetWordInfo(wordId)
}

// GetWordParam returns word_param for given word_id
// Returns (left_id, right_id, cost)
func (l *Lexicon) GetWordParam(wordId uint32) (int16, int16, int16) {
	return l.wordParams.GetParams(wordId)
}

// Size returns the number of words in the lexicon
func (l *Lexicon) Size() uint32 {
	return l.wordParams.Size()
}

// LexiconIterator iterates over lexicon entries
type LexiconIterator struct {
	lexicon     *Lexicon
	trieIter    *TrieIterator
	input       []byte
	offset      int
	currentTrie *TrieEntry
	currentIter *WordIdIter
}

// Reset clears all fields of a LexiconIterator for reuse
func (li *LexiconIterator) Reset() {
	li.lexicon = nil
	if li.trieIter != nil {
		li.trieIter.ReturnToPool()
		li.trieIter = nil
	}
	li.input = nil
	li.offset = 0
	li.currentTrie = nil
	li.currentIter = nil
}

// ReturnToPool returns a LexiconIterator to the global pool
func (li *LexiconIterator) ReturnToPool() {
	globalLexiconIteratorPool.Put(li)
}

// Next returns the next LexiconEntry
func (li *LexiconIterator) Next() (*LexiconEntry, error) {
	for {
		// If we have a current word ID iterator, try to get next word ID
		if li.currentIter != nil && li.currentIter.HasNext() {
			if wordId, ok := li.currentIter.Next(); ok {
				// Convert absolute position to relative length
				relativeLength := li.currentTrie.End - li.offset
				return &LexiconEntry{
					WordId: li.lexicon.wordId(wordId),
					End:    relativeLength,
				}, nil
			}
		}

		// Get next trie entry
		trieEntry, err := li.trieIter.Next()
		if err != nil {
			return nil, err
		}
		if trieEntry == nil {
			return nil, nil // No more entries
		}

		// Get word IDs for this trie entry
		li.currentTrie = trieEntry
		li.currentIter = li.lexicon.wordIdTable.Entries(int(trieEntry.Value))

		// Try to get first word ID from the new iterator
		if li.currentIter.HasNext() {
			if wordId, ok := li.currentIter.Next(); ok {
				// Convert absolute position to relative length
				relativeLength := li.currentTrie.End - li.offset
				return &LexiconEntry{
					WordId: li.lexicon.wordId(wordId),
					End:    relativeLength,
				}, nil
			}
		}
	}
}
