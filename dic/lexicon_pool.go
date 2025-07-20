package dic

import (
	"fmt"
	"sync"
)

// LexiconSetIteratorPool provides a pool for LexiconSetIterator objects
type LexiconSetIteratorPool struct {
	pool sync.Pool
}

// Global pool instance for LexiconSetIterator objects
var globalLexiconSetIteratorPool = &LexiconSetIteratorPool{
	pool: sync.Pool{
		New: func() any {
			return &LexiconSetIterator{
				lexicons:     nil,
				currentIndex: -1,
				currentIter:  nil,
				input:        nil,
				offset:       0,
			}
		},
	},
}

// Get retrieves a LexiconSetIterator from the pool
func (p *LexiconSetIteratorPool) Get() *LexiconSetIterator {
	return p.pool.Get().(*LexiconSetIterator)
}

// Put returns a LexiconSetIterator to the pool after resetting it
func (p *LexiconSetIteratorPool) Put(iter *LexiconSetIterator) {
	if iter != nil {
		iter.Reset()
		p.pool.Put(iter)
	}
}

// Reset clears all fields of a LexiconSetIterator for reuse
func (iter *LexiconSetIterator) Reset() {
	iter.lexicons = nil
	iter.currentIndex = -1
	iter.currentIter = nil
	iter.input = nil
	iter.offset = 0
}

// NewLexiconSetIteratorFromPool creates a new iterator using the global pool
func NewLexiconSetIteratorFromPool(lexicons []*Lexicon, input []byte, offset int) (*LexiconSetIterator, error) {
	iter := globalLexiconSetIteratorPool.Get()
	iter.lexicons = lexicons
	iter.input = input
	iter.offset = offset

	// Start from the last lexicon (user dictionaries first)
	iter.currentIndex = len(lexicons) - 1
	iter.currentIter = nil

	if iter.currentIndex >= 0 {
		currentIter, err := lexicons[iter.currentIndex].Lookup(input, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to create iterator for lexicon %d: %w", iter.currentIndex, err)
		}
		iter.currentIter = currentIter
	}

	return iter, nil
}

// ReturnToPool returns a LexiconSetIterator to the global pool
func (iter *LexiconSetIterator) ReturnToPool() {
	globalLexiconSetIteratorPool.Put(iter)
}

// LexiconIteratorPool provides a pool for LexiconIterator objects
type LexiconIteratorPool struct {
	pool sync.Pool
}

// Global pool instance for LexiconIterator objects
var globalLexiconIteratorPool = &LexiconIteratorPool{
	pool: sync.Pool{
		New: func() any {
			return &LexiconIterator{
				lexicon:     nil,
				trieIter:    nil,
				input:       nil,
				offset:      0,
				currentTrie: nil,
				currentIter: nil,
			}
		},
	},
}

// Get retrieves a LexiconIterator from the pool
func (p *LexiconIteratorPool) Get() *LexiconIterator {
	return p.pool.Get().(*LexiconIterator)
}

// Put returns a LexiconIterator to the pool after resetting it
func (p *LexiconIteratorPool) Put(iter *LexiconIterator) {
	if iter != nil {
		iter.Reset()
		p.pool.Put(iter)
	}
}

// NewLexiconIteratorFromPool creates a new LexiconIterator using the global pool
func NewLexiconIteratorFromPool(lexicon *Lexicon, trieIter *TrieIterator, input []byte, offset int) *LexiconIterator {
	iter := globalLexiconIteratorPool.Get()
	iter.lexicon = lexicon
	iter.trieIter = trieIter
	iter.input = input
	iter.offset = offset
	iter.currentTrie = nil
	iter.currentIter = nil
	return iter
}

// TrieIteratorPool provides a pool for TrieIterator objects
type TrieIteratorPool struct {
	pool sync.Pool
}

// Global pool instance for TrieIterator objects
var globalTrieIteratorPool = &TrieIteratorPool{
	pool: sync.Pool{
		New: func() any {
			return &TrieIterator{
				trie:        nil,
				nodePos:     0,
				input:       nil,
				offset:      0,
				startOffset: 0,
			}
		},
	},
}

// Get retrieves a TrieIterator from the pool
func (p *TrieIteratorPool) Get() *TrieIterator {
	return p.pool.Get().(*TrieIterator)
}

// Put returns a TrieIterator to the pool after resetting it
func (p *TrieIteratorPool) Put(iter *TrieIterator) {
	if iter != nil {
		iter.Reset()
		p.pool.Put(iter)
	}
}

// Reset clears all fields of a TrieIterator for reuse
func (iter *TrieIterator) Reset() {
	iter.trie = nil
	iter.nodePos = 0
	iter.input = nil
	iter.offset = 0
	iter.startOffset = 0
}

// NewTrieIteratorFromPool creates a new TrieIterator using the global pool
func NewTrieIteratorFromPool(trie *Trie, input []byte, inputOffset int) *TrieIterator {
	iter := globalTrieIteratorPool.Get()
	iter.trie = trie
	iter.input = input
	iter.offset = inputOffset
	iter.startOffset = inputOffset

	// Initialize node position like the regular constructor
	if trie.Size() > 0 {
		unit := trie.get(0)
		iter.nodePos = int(offset(uintptr(unit)))
	} else {
		iter.nodePos = 0
	}

	return iter
}

// ReturnToPool returns a TrieIterator to the global pool
func (iter *TrieIterator) ReturnToPool() {
	globalTrieIteratorPool.Put(iter)
}
