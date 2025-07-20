package dic

import (
	"encoding/binary"
)

// WordIdTable represents a table of word IDs
type WordIdTable struct {
	bytes  []byte
	size   uint32
	offset int
}

// NewWordIdTable creates a new WordIdTable from byte slice
func NewWordIdTable(bytes []byte, size uint32, offset int) *WordIdTable {
	return &WordIdTable{
		bytes:  bytes,
		size:   size,
		offset: offset,
	}
}

// StorageSize returns the total storage size of the table
func (wit *WordIdTable) StorageSize() int {
	return 4 + int(wit.size)
}

// Entries returns an iterator for word IDs at the given index
func (wit *WordIdTable) Entries(index int) *WordIdIter {
	if index >= len(wit.bytes) {
		return &WordIdIter{data: nil, remaining: 0}
	}

	// Read count from the first byte
	actualIndex := index + wit.offset
	if actualIndex >= len(wit.bytes) {
		return &WordIdIter{data: nil, remaining: 0}
	}

	count := int(wit.bytes[actualIndex])
	dataStart := actualIndex + 1

	// Check bounds
	if dataStart+count*4 > len(wit.bytes) {
		return &WordIdIter{data: nil, remaining: 0}
	}

	return &WordIdIter{
		data:      wit.bytes[dataStart:],
		remaining: count,
		position:  0,
	}
}

// WordIdIter is an iterator for word IDs
type WordIdIter struct {
	data      []byte
	remaining int
	position  int
}

// Next returns the next word ID
func (iter *WordIdIter) Next() (uint32, bool) {
	if iter.remaining == 0 || iter.data == nil {
		return 0, false
	}

	if iter.position+4 > len(iter.data) {
		return 0, false
	}

	// Read as little-endian uint32
	val := binary.LittleEndian.Uint32(iter.data[iter.position : iter.position+4])

	iter.position += 4
	iter.remaining--

	return val, true
}

// HasNext returns true if there are more items
func (iter *WordIdIter) HasNext() bool {
	return iter.remaining > 0 && iter.data != nil
}

// Reset resets the iterator to the beginning
func (iter *WordIdIter) Reset() {
	iter.position = 0
	iter.remaining = len(iter.data) / 4
}
