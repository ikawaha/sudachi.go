package dic

import (
	"encoding/binary"
	"fmt"
	"unicode/utf8"
)

// Reader provides safe binary data reading from dictionary files
type Reader struct {
	data   []byte
	offset int
}

// NewReader creates a new binary reader for dictionary data
func NewReader(data []byte) *Reader {
	return &Reader{
		data:   data,
		offset: 0,
	}
}

// NewReaderAt creates a new binary reader starting at the specified offset
func NewReaderAt(data []byte, offset int) (*Reader, error) {
	if offset < 0 {
		return nil, fmt.Errorf("invalid offset: %d", offset)
	}
	if offset > len(data) {
		return nil, fmt.Errorf("invalid offset: %d > data length %d", offset, len(data))
	}

	return &Reader{
		data:   data,
		offset: offset,
	}, nil
}

// Offset returns the current read offset
func (r *Reader) Offset() int {
	return r.offset
}

// Len returns the remaining bytes available for reading
func (r *Reader) Len() int {
	return len(r.data) - r.offset
}

// Available checks if at least n bytes are available for reading
func (r *Reader) Available(n int) bool {
	return r.offset+n <= len(r.data)
}

// ReadU8 reads a single byte (uint8)
func (r *Reader) ReadU8() (uint8, error) {
	if !r.Available(1) {
		return 0, fmt.Errorf("insufficient data: need %d byte, have %d", r.offset+1, r.Len())
	}

	val := r.data[r.offset]
	r.offset++
	return val, nil
}

// ReadU16 reads a 16-bit unsigned integer in little-endian format
func (r *Reader) ReadU16() (uint16, error) {
	if !r.Available(2) {
		return 0, fmt.Errorf("insufficient data: need %d byte, have %d", r.offset+2, r.Len())
	}

	val := binary.LittleEndian.Uint16(r.data[r.offset:])
	r.offset += 2
	return val, nil
}

// ReadU32 reads a 32-bit unsigned integer in little-endian format
func (r *Reader) ReadU32() (uint32, error) {
	if !r.Available(4) {
		return 0, fmt.Errorf("insufficient data: need %d bytes, have %d", r.offset+4, r.Len())
	}

	val := binary.LittleEndian.Uint32(r.data[r.offset:])
	r.offset += 4
	return val, nil
}

// ReadU64 reads a 64-bit unsigned integer in little-endian format
func (r *Reader) ReadU64() (uint64, error) {
	if !r.Available(8) {
		return 0, fmt.Errorf("insufficient data: need %d bytes, have %d", r.offset+8, r.Len())
	}

	val := binary.LittleEndian.Uint64(r.data[r.offset:])
	r.offset += 8
	return val, nil
}

// ReadI32 reads a 32-bit signed integer in little-endian format
func (r *Reader) ReadI32() (int32, error) {
	if !r.Available(4) {
		return 0, fmt.Errorf("insufficient data: need %d bytes, have %d", r.offset+4, r.Len())
	}

	val := int32(binary.LittleEndian.Uint32(r.data[r.offset:]))
	r.offset += 4
	return val, nil
}

// ReadBytes reads exactly n bytes
func (r *Reader) ReadBytes(n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("invalid size: %d", n)
	}
	if !r.Available(n) {
		return nil, fmt.Errorf("insufficient data: need %d bytes, have %d", r.offset+n, r.Len())
	}

	bytes := make([]byte, n)
	copy(bytes, r.data[r.offset:r.offset+n])
	r.offset += n
	return bytes, nil
}

// ReadSlice returns a slice of the underlying data without copying
// WARNING: The returned slice shares memory with the original data
func (r *Reader) ReadSlice(n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("invalid size: %d", n)
	}
	if !r.Available(n) {
		return nil, fmt.Errorf("insufficient data: need %d bytes, have %d", r.offset+n, r.Len())
	}

	slice := r.data[r.offset : r.offset+n]
	r.offset += n
	return slice, nil
}

// ReadString reads a null-terminated string up to maxLen bytes
func (r *Reader) ReadString(maxLen int) (string, error) {
	if maxLen <= 0 {
		return "", fmt.Errorf("invalid size: %d", maxLen)
	}
	if !r.Available(maxLen) {
		return "", fmt.Errorf("insufficient data: need %d bytes, have %d", r.offset+maxLen, r.Len())
	}

	// Find null terminator or end of maxLen bytes
	bytes := r.data[r.offset : r.offset+maxLen]
	end := maxLen
	for i, b := range bytes {
		if b == 0 {
			end = i
			break
		}
	}

	// Extract string bytes (up to null terminator)
	strBytes := bytes[:end]

	// Validate UTF-8
	if !utf8.Valid(strBytes) {
		return "", fmt.Errorf("invalid UTF-8 sequence at offset %d", r.offset)
	}

	r.offset += maxLen
	return string(strBytes), nil
}

// ReadU32Array reads a length-prefixed array of uint32 values
// Format: [length:u8][value1:u32][value2:u32]...
func (r *Reader) ReadU32Array() ([]uint32, error) {
	length, err := r.ReadU8()
	if err != nil {
		return nil, err
	}

	if length == 0 {
		return []uint32{}, nil
	}

	if !r.Available(int(length) * 4) {
		return nil, fmt.Errorf("insufficient data: need %d bytes, have %d", r.offset+int(length)*4, r.Len())
	}

	values := make([]uint32, length)
	for i := 0; i < int(length); i++ {
		val, err := r.ReadU32()
		if err != nil {
			return nil, err
		}
		values[i] = val
	}

	return values, nil
}

// ReadWordIdArray reads a length-prefixed array of WordId values
// Format: [length:u8][wordid1:u32][wordid2:u32]...
func (r *Reader) ReadWordIdArray() ([]WordId, error) {
	length, err := r.ReadU8()
	if err != nil {
		return nil, err
	}

	if length == 0 {
		return []WordId{}, nil
	}

	if !r.Available(int(length) * 4) {
		return nil, fmt.Errorf("insufficient data: need %d bytes, have %d", r.offset+int(length)*4, r.Len())
	}

	values := make([]WordId, length)
	for i := 0; i < int(length); i++ {
		raw, err := r.ReadU32()
		if err != nil {
			return nil, err
		}
		values[i] = FromRaw(raw)
	}

	return values, nil
}

// SkipU32Array skips a length-prefixed array of uint32 values without reading them
func (r *Reader) SkipU32Array() error {
	length, err := r.ReadU8()
	if err != nil {
		return err
	}

	bytesToSkip := int(length) * 4
	if !r.Available(bytesToSkip) {
		return fmt.Errorf("insufficient data: need %d bytes, have %d", r.offset+bytesToSkip, r.Len())
	}

	r.offset += bytesToSkip
	return nil
}

// SkipWordIdArray skips a length-prefixed array of WordId values without reading them
func (r *Reader) SkipWordIdArray() error {
	return r.SkipU32Array() // Same format as U32Array
}

// Skip advances the reader by n bytes
func (r *Reader) Skip(n int) error {
	if n < 0 {
		return fmt.Errorf("invalid size: %d", n)
	}
	if !r.Available(n) {
		return fmt.Errorf("insufficient data: need %d bytes, have %d", r.offset+n, r.Len())
	}

	r.offset += n
	return nil
}

// Seek sets the read offset to a specific position
func (r *Reader) Seek(offset int) error {
	if offset < 0 {
		return fmt.Errorf("invalid offset: %d", offset)
	}
	if offset > len(r.data) {
		return fmt.Errorf("invalid offset: %d > data length %d", offset, len(r.data))
	}

	r.offset = offset
	return nil
}

// SubReader creates a new reader for a portion of the data
func (r *Reader) SubReader(size int) (*Reader, error) {
	if size < 0 {
		return nil, fmt.Errorf("invalid size: %d", size)
	}
	if !r.Available(size) {
		return nil, fmt.Errorf("insufficient data: need %d bytes, have %d", r.offset+size, r.Len())
	}

	subData := r.data[r.offset : r.offset+size]
	r.offset += size

	return NewReader(subData), nil
}
