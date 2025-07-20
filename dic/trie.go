package dic

import (
	"fmt"
	"reflect"
	"unsafe"
)

// TrieEntry represents a trie search result
type TrieEntry struct {
	Value uint32 // Offset in WordId table
	End   int    // Word end position
}

// NewTrieEntry creates a new TrieEntry
func NewTrieEntry(value uint32, end int) *TrieEntry {
	return &TrieEntry{Value: value, End: end}
}

// Trie provides efficient prefix matching for dictionary lookups
type Trie struct {
	data []uint32 // u32 array created via unsafe
	size int      // Data size
}

// TrieIterator implements common prefix iteration
type TrieIterator struct {
	trie        *Trie  // Reference to trie data
	nodePos     int    // Current node position
	input       []byte // Input data to search
	offset      int    // Current character offset
	startOffset int    // Initial offset (for relative length calculation)
}

// NewTrie creates a new Trie from binary data using unsafe zero-copy construction
func NewTrie(data []byte, offset, size int) (*Trie, error) {
	// Input validation
	if offset < 0 {
		return nil, fmt.Errorf("invalid offset: %d", offset)
	}
	if size < 0 {
		return nil, fmt.Errorf("invalid size: %d", size)
	}

	requiredBytes := size * 4
	if offset+requiredBytes > len(data) {
		return nil, fmt.Errorf("insufficient data: need %d bytes, have %d bytes", offset+requiredBytes, len(data)-offset)
	}

	// Create uint32 slice using unsafe SliceHeader for zero-copy
	var uint32Slice []uint32
	if size == 0 {
		// For zero size, create empty slice safely
		uint32Slice = []uint32{}
	} else {
		header := reflect.SliceHeader{
			Data: uintptr(unsafe.Pointer(&data[offset])),
			Len:  size,
			Cap:  size,
		}
		uint32Slice = *(*[]uint32)(unsafe.Pointer(&header))
	}

	return &Trie{
		data: uint32Slice,
		size: size,
	}, nil
}

// NewTrieOwned creates a new Trie from owned uint32 data
func NewTrieOwned(data []uint32) (*Trie, error) {
	if data == nil {
		return nil, fmt.Errorf("data is nil")
	}
	return &Trie{
		data: data,
		size: len(data),
	}, nil
}

// Get provides safe access to trie data with bounds checking
func (t *Trie) Get(index int) (uint32, error) {
	if index < 0 || index >= t.size {
		return 0, fmt.Errorf("index out of bounds [0, %d): %d", t.size, index)
	}
	return t.data[index], nil
}

// get provides unsafe high-speed access for internal use
// Assumes bounds have been checked by caller
func (t *Trie) get(index int) uint32 {
	// Use unsafe to bypass bounds checking for maximum performance
	ptr := unsafe.Add(unsafe.Pointer(&t.data[0]), uintptr(index)*4)
	return *(*uint32)(ptr)
}

// TotalSize returns the total size in bytes
func (t *Trie) TotalSize() int {
	return 4 * t.size
}

// Size returns the number of elements
func (t *Trie) Size() int {
	return t.size
}

// Bit manipulation functions ported from Rust implementation
// Note: Rust uses usize for most operations, Go uses uintptr as usize equivalent

// hasLeaf checks if a unit has a leaf node
// Rust: fn has_leaf(unit: usize) -> bool
func hasLeaf(unit uintptr) bool {
	return ((unit >> 8) & 1) == 1
}

// value extracts the value from a unit
// Rust: fn value(unit: u32) -> u32
func value(unit uint32) uint32 {
	return unit & ((1 << 31) - 1)
}

// label extracts the label from a unit
// Rust: fn label(unit: usize) -> usize
func label(unit uintptr) uintptr {
	return unit & ((1 << 31) | 0xFF)
}

// offset extracts the offset from a unit
// Rust: fn offset(unit: usize) -> usize
func offset(unit uintptr) uintptr {
	return (unit >> 10) << ((unit & (1 << 9)) >> 6)
}

// CommonPrefixIterator creates an iterator for common prefix search
func (t *Trie) CommonPrefixIterator(input []byte, inputOffset int) (*TrieIterator, error) {
	if t.size == 0 {
		return nil, fmt.Errorf("empty trie")
	}
	if inputOffset < 0 || inputOffset > len(input) {
		return nil, fmt.Errorf("invalid offset %d for input length %d", inputOffset, len(input))
	}

	// Initialize iterator with proper node position (matching Rust implementation)
	unit := t.get(0)
	nodePos := int(offset(uintptr(unit)))

	return &TrieIterator{
		trie:        t,
		nodePos:     nodePos,
		input:       input,
		offset:      inputOffset,
		startOffset: inputOffset,
	}, nil
}

// Next returns the next TrieEntry or nil if iteration is complete
func (it *TrieIterator) Next() (*TrieEntry, error) {
	nodePos := it.nodePos

	for i := it.offset; i < len(it.input); i++ {
		k := it.input[i]
		// Match Rust: node_pos ^= *k as usize
		nodePos ^= int(k)

		// Bounds check
		if nodePos < 0 || nodePos >= it.trie.size {
			return nil, fmt.Errorf("node position out of bounds [0, %d): %d", it.trie.size, nodePos)
		}

		unit := it.trie.get(nodePos)

		// Match Rust: unit = self.get(node_pos) as usize
		unitAsUintptr := uintptr(unit)

		// Match Rust: if Trie::label(unit) != *k as usize
		if label(unitAsUintptr) != uintptr(k) {
			return nil, nil // No match, not an error
		}

		// Match Rust: node_pos ^= Trie::offset(unit)
		nodePos ^= int(offset(unitAsUintptr))

		if hasLeaf(unitAsUintptr) {
			// Bounds check for leaf node
			if nodePos < 0 || nodePos >= it.trie.size {
				return nil, fmt.Errorf("laef node position out of bounds [0, %d): %d", it.trie.size, nodePos)
			}

			leafUnit := it.trie.get(nodePos)
			// Return absolute byte position (same as Rust implementation)
			absoluteEnd := i + 1

			entry := NewTrieEntry(value(leafUnit), absoluteEnd)

			// Update for next iteration - continue from end of this match to find longer matches
			// This matches Rust behavior where longer overlapping matches can be found
			it.offset = absoluteEnd
			it.nodePos = nodePos

			return entry, nil
		}
	}

	return nil, nil // Iteration complete, not an error
}
