package dic

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

// ConnectionMatrix represents the connection cost matrix between morphemes
type ConnectionMatrix struct {
	data     []byte
	numLeft  int
	numRight int
}

// NewConnectionMatrix creates a new connection matrix from byte data
func NewConnectionMatrix(data []byte, offset int, numLeft, numRight int) (*ConnectionMatrix, error) {
	size := numLeft * numRight * 2 // i16 = 2 bytes

	end := offset + size
	if end > len(data) {
		return nil, fmt.Errorf("invalid dictionary grammar: connection matrix size %d exceeds data length %d", end, len(data))
	}

	return &ConnectionMatrix{
		data:     data[offset:end],
		numLeft:  numLeft,
		numRight: numRight,
	}, nil
}

// index calculates the matrix index for given left and right IDs
func (cm *ConnectionMatrix) index(left, right uint16) int {
	uleft := int(left)
	uright := int(right)

	// Debug assertions (in production these should be removed for performance)
	if uleft >= cm.numLeft || uright >= cm.numRight {
		// In Rust this would be UB, but in Go we handle it gracefully
		return 0
	}

	index := uright*cm.numLeft + uleft
	return index * 2 // Each i16 is 2 bytes
}

// Cost returns the connection cost between left and right morphemes
// This is performance critical and should be inlined
func (cm *ConnectionMatrix) Cost(left, right uint16) int16 {
	index := cm.index(left, right)

	// Bounds check for safety
	if index+2 > len(cm.data) {
		return InhibitedConnection // i16::MAX equivalent (inhibited connection)
	}

	// Use unsafe for zero-copy performance like Rust
	return int16(binary.LittleEndian.Uint16(cm.data[index : index+2]))
}

// CostUnsafe returns connection cost using unsafe operations for maximum performance
func (cm *ConnectionMatrix) CostUnsafe(left, right uint16) int16 {
	uleft := int(left)
	uright := int(right)
	index := (uright*cm.numLeft + uleft) * 2

	// Unsafe direct memory access like Rust's get_unchecked
	return *(*int16)(unsafe.Pointer(&cm.data[index]))
}

// Update sets the connection cost for a specific pair of IDs
func (cm *ConnectionMatrix) Update(left, right uint16, cost int16) {
	index := cm.index(left, right)

	if index+2 <= len(cm.data) {
		binary.LittleEndian.PutUint16(cm.data[index:index+2], uint16(cost))
	}
}

// NumLeft returns the maximum number of left connection IDs
func (cm *ConnectionMatrix) NumLeft() int {
	return cm.numLeft
}

// NumRight returns the maximum number of right connection IDs
func (cm *ConnectionMatrix) NumRight() int {
	return cm.numRight
}

// Size returns the total number of entries in the matrix
func (cm *ConnectionMatrix) Size() int {
	return cm.numLeft * cm.numRight
}
