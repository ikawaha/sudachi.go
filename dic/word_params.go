package dic

import (
	"encoding/binary"
	"unsafe"
)

// WordParams represents word parameters (left_id, right_id, cost)
type WordParams struct {
	data []byte
	size uint32
}

const (
	// Each word has 3 parameters: left_id, right_id, cost
	paramSize   = 3
	elementSize = 2 * paramSize // 2 bytes per i16 * 3 params
)

// NewWordParams creates a new WordParams from byte slice
func NewWordParams(bytes []byte, size uint32, offset int) *WordParams {
	dataSize := int(size) * elementSize
	endOffset := offset + dataSize

	if endOffset > len(bytes) {
		// Handle error gracefully
		return &WordParams{
			data: nil,
			size: 0,
		}
	}

	return &WordParams{
		data: bytes[offset:endOffset],
		size: size,
	}
}

// StorageSize returns the total storage size
func (wp *WordParams) StorageSize() int {
	return 4 + elementSize*int(wp.size)
}

// Size returns the number of word entries
func (wp *WordParams) Size() uint32 {
	return wp.size
}

// GetParams returns the parameters for a word ID (left_id, right_id, cost)
func (wp *WordParams) GetParams(wordId uint32) (int16, int16, int16) {
	if wp.data == nil || wordId >= wp.size {
		return 0, 0, 0
	}

	begin := int(wordId) * elementSize
	if begin+elementSize > len(wp.data) {
		return 0, 0, 0
	}

	// Use unsafe for zero-copy performance like Rust
	data := wp.data[begin:]

	// Read as i16 values (little-endian)
	leftId := int16(binary.LittleEndian.Uint16(data[0:2]))
	rightId := int16(binary.LittleEndian.Uint16(data[2:4]))
	cost := int16(binary.LittleEndian.Uint16(data[4:6]))

	return leftId, rightId, cost
}

// GetCost returns just the cost parameter for a word ID
func (wp *WordParams) GetCost(wordId uint32) int16 {
	if wp.data == nil || wordId >= wp.size {
		return 0
	}

	costOffset := int(wordId)*elementSize + 4 // Skip left_id and right_id
	if costOffset+2 > len(wp.data) {
		return 0
	}

	return int16(binary.LittleEndian.Uint16(wp.data[costOffset : costOffset+2]))
}

// SetCost sets the cost parameter for a word ID
func (wp *WordParams) SetCost(wordId uint32, cost int16) {
	if wp.data == nil || wordId >= wp.size {
		return
	}

	costOffset := int(wordId)*elementSize + 4 // Skip left_id and right_id
	if costOffset+2 > len(wp.data) {
		return
	}

	binary.LittleEndian.PutUint16(wp.data[costOffset:costOffset+2], uint16(cost))
}

// GetParamsUnsafe returns parameters using unsafe for maximum performance
func (wp *WordParams) GetParamsUnsafe(wordId uint32) (int16, int16, int16) {
	if wp.data == nil || wordId >= wp.size {
		return 0, 0, 0
	}

	begin := int(wordId) * elementSize
	if begin+elementSize > len(wp.data) {
		return 0, 0, 0
	}

	// Use unsafe pointer arithmetic for zero-copy access
	ptr := unsafe.Pointer(&wp.data[begin])
	leftId := *(*int16)(ptr)
	rightId := *(*int16)(unsafe.Pointer(uintptr(ptr) + 2))
	cost := *(*int16)(unsafe.Pointer(uintptr(ptr) + 4))

	return leftId, rightId, cost
}
