package dic

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestNewReader(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	reader := NewReader(data)

	if reader.Offset() != 0 {
		t.Errorf("NewReader().Offset() = %d, want 0", reader.Offset())
	}
	if reader.Len() != 5 {
		t.Errorf("NewReader().Len() = %d, want 5", reader.Len())
	}
}

func TestNewReaderAt(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}

	t.Run("valid offset", func(t *testing.T) {
		reader, err := NewReaderAt(data, 2)
		if err != nil {
			t.Fatalf("NewReaderAt(data, 2) unexpected error: %v", err)
		}
		if reader.Offset() != 2 {
			t.Errorf("NewReaderAt(data, 2).Offset() = %d, want 2", reader.Offset())
		}
		if reader.Len() != 3 {
			t.Errorf("NewReaderAt(data, 2).Len() = %d, want 3", reader.Len())
		}
	})

	t.Run("invalid offset", func(t *testing.T) {
		_, err := NewReaderAt(data, -1)
		if err == nil {
			t.Error("NewReaderAt(data, -1) expected error for negative offset")
		}

		_, err = NewReaderAt(data, 10)
		if err == nil {
			t.Error("NewReaderAt(data, 10) expected error for offset > data length")
		}
	})
}

func TestReadU8(t *testing.T) {
	data := []byte{0x42, 0x43, 0x44}
	reader := NewReader(data)

	val, err := reader.ReadU8()
	if err != nil {
		t.Fatalf("ReadU8() unexpected error: %v", err)
	}
	if val != 0x42 {
		t.Errorf("ReadU8() = 0x%02x, want 0x42", val)
	}
	if reader.Offset() != 1 {
		t.Errorf("ReadU8() offset = %d, want 1", reader.Offset())
	}

	// Test insufficient data
	reader.Seek(3)
	_, err = reader.ReadU8()
	if err == nil {
		t.Error("ReadU8() expected error for insufficient data")
	}
}

func TestReadU16(t *testing.T) {
	data := []byte{0x34, 0x12, 0x78, 0x56}
	reader := NewReader(data)

	val, err := reader.ReadU16()
	if err != nil {
		t.Fatalf("ReadU16() unexpected error: %v", err)
	}
	if val != 0x1234 {
		t.Errorf("ReadU16() = 0x%04x, want 0x1234", val)
	}
	if reader.Offset() != 2 {
		t.Errorf("ReadU16() offset = %d, want 2", reader.Offset())
	}

	// Test insufficient data
	reader.Seek(3)
	_, err = reader.ReadU16()
	if err == nil {
		t.Error("ReadU16() expected error for insufficient data")
	}
}

func TestReadU32(t *testing.T) {
	data := []byte{0x78, 0x56, 0x34, 0x12, 0x00}
	reader := NewReader(data)

	val, err := reader.ReadU32()
	if err != nil {
		t.Fatalf("ReadU32() unexpected error: %v", err)
	}
	if val != 0x12345678 {
		t.Errorf("ReadU32() = 0x%08x, want 0x12345678", val)
	}
	if reader.Offset() != 4 {
		t.Errorf("ReadU32() offset = %d, want 4", reader.Offset())
	}

	// Test insufficient data
	reader.Seek(2)
	_, err = reader.ReadU32()
	if err == nil {
		t.Error("ReadU32() expected error for insufficient data")
	}
}

func TestReadU64(t *testing.T) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, 0x123456789abcdef0)
	reader := NewReader(data)

	val, err := reader.ReadU64()
	if err != nil {
		t.Fatalf("ReadU64() unexpected error: %v", err)
	}
	if val != 0x123456789abcdef0 {
		t.Errorf("ReadU64() = 0x%016x, want 0x123456789abcdef0", val)
	}
	if reader.Offset() != 8 {
		t.Errorf("ReadU64() offset = %d, want 8", reader.Offset())
	}
}

func TestReadBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	reader := NewReader(data)

	result, err := reader.ReadBytes(3)
	if err != nil {
		t.Fatalf("ReadBytes(3) unexpected error: %v", err)
	}
	expected := []byte{1, 2, 3}
	if !bytes.Equal(result, expected) {
		t.Errorf("ReadBytes(3) = %v, want %v", result, expected)
	}
	if reader.Offset() != 3 {
		t.Errorf("ReadBytes(3) offset = %d, want 3", reader.Offset())
	}

	// Test insufficient data
	_, err = reader.ReadBytes(10)
	if err == nil {
		t.Error("ReadBytes(10) expected error for insufficient data")
	}
}

func TestReadSlice(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	reader := NewReader(data)

	slice, err := reader.ReadSlice(3)
	if err != nil {
		t.Fatalf("ReadSlice(3) unexpected error: %v", err)
	}
	expected := []byte{1, 2, 3}
	if !bytes.Equal(slice, expected) {
		t.Errorf("ReadSlice(3) = %v, want %v", slice, expected)
	}
	if reader.Offset() != 3 {
		t.Errorf("ReadSlice(3) offset = %d, want 3", reader.Offset())
	}

	// Test that slice shares memory with original data
	slice[0] = 99
	if data[0] != 99 {
		t.Error("ReadSlice() should return slice sharing memory with original data")
	}
}

func TestReadString(t *testing.T) {
	// Test null-terminated string
	data := []byte{'H', 'e', 'l', 'l', 'o', 0, 0, 0}
	reader := NewReader(data)

	str, err := reader.ReadString(8)
	if err != nil {
		t.Fatalf("ReadString(8) unexpected error: %v", err)
	}
	if str != "Hello" {
		t.Errorf("ReadString(8) = %q, want \"Hello\"", str)
	}
	if reader.Offset() != 8 {
		t.Errorf("ReadString(8) offset = %d, want 8", reader.Offset())
	}

	// Test string without null terminator
	data2 := []byte{'W', 'o', 'r', 'l', 'd'}
	reader2 := NewReader(data2)

	str2, err := reader2.ReadString(5)
	if err != nil {
		t.Fatalf("ReadString(5) unexpected error: %v", err)
	}
	if str2 != "World" {
		t.Errorf("ReadString(5) = %q, want \"World\"", str2)
	}
}

func TestReadU32Array(t *testing.T) {
	// Create test data: [length:3][value1:0x12345678][value2:0x9abcdef0][value3:0x11111111]
	data := []byte{3}                           // length
	data = append(data, 0x78, 0x56, 0x34, 0x12) // 0x12345678
	data = append(data, 0xf0, 0xde, 0xbc, 0x9a) // 0x9abcdef0
	data = append(data, 0x11, 0x11, 0x11, 0x11) // 0x11111111

	reader := NewReader(data)

	values, err := reader.ReadU32Array()
	if err != nil {
		t.Fatalf("ReadU32Array() unexpected error: %v", err)
	}

	expected := []uint32{0x12345678, 0x9abcdef0, 0x11111111}
	if len(values) != len(expected) {
		t.Errorf("ReadU32Array() length = %d, want %d", len(values), len(expected))
	}

	for i, val := range values {
		if val != expected[i] {
			t.Errorf("ReadU32Array()[%d] = 0x%08x, want 0x%08x", i, val, expected[i])
		}
	}

	// Test empty array
	emptyData := []byte{0}
	reader2 := NewReader(emptyData)

	values2, err := reader2.ReadU32Array()
	if err != nil {
		t.Fatalf("ReadU32Array() empty array unexpected error: %v", err)
	}
	if len(values2) != 0 {
		t.Errorf("ReadU32Array() empty array length = %d, want 0", len(values2))
	}
}

func TestReadWordIdArray(t *testing.T) {
	// Create test data: [length:2][wordid1:0x12345678][wordid2:0x9abcdef0]
	data := []byte{2}                           // length
	data = append(data, 0x78, 0x56, 0x34, 0x12) // 0x12345678
	data = append(data, 0xf0, 0xde, 0xbc, 0x9a) // 0x9abcdef0

	reader := NewReader(data)

	values, err := reader.ReadWordIdArray()
	if err != nil {
		t.Fatalf("ReadWordIdArray() unexpected error: %v", err)
	}

	expected := []WordId{FromRaw(0x12345678), FromRaw(0x9abcdef0)}
	if len(values) != len(expected) {
		t.Errorf("ReadWordIdArray() length = %d, want %d", len(values), len(expected))
	}

	for i, val := range values {
		if !val.Equal(expected[i]) {
			t.Errorf("ReadWordIdArray()[%d] = %v, want %v", i, val, expected[i])
		}
	}
}

func TestSkipU32Array(t *testing.T) {
	// Create test data: [length:2][value1:0x12345678][value2:0x9abcdef0]
	data := []byte{2}                           // length
	data = append(data, 0x78, 0x56, 0x34, 0x12) // 0x12345678
	data = append(data, 0xf0, 0xde, 0xbc, 0x9a) // 0x9abcdef0
	data = append(data, 0x99)                   // extra byte

	reader := NewReader(data)

	err := reader.SkipU32Array()
	if err != nil {
		t.Fatalf("SkipU32Array() unexpected error: %v", err)
	}

	if reader.Offset() != 9 { // 1 (length) + 8 (2 * 4 bytes)
		t.Errorf("SkipU32Array() offset = %d, want 9", reader.Offset())
	}

	// Should be able to read the extra byte
	val, err := reader.ReadU8()
	if err != nil {
		t.Fatalf("ReadU8() after SkipU32Array() unexpected error: %v", err)
	}
	if val != 0x99 {
		t.Errorf("ReadU8() after SkipU32Array() = 0x%02x, want 0x99", val)
	}
}

func TestSeek(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	reader := NewReader(data)

	err := reader.Seek(3)
	if err != nil {
		t.Fatalf("Seek(3) unexpected error: %v", err)
	}
	if reader.Offset() != 3 {
		t.Errorf("Seek(3) offset = %d, want 3", reader.Offset())
	}

	// Test invalid seek
	err = reader.Seek(-1)
	if err == nil {
		t.Error("Seek(-1) expected error for negative offset")
	}

	err = reader.Seek(10)
	if err == nil {
		t.Error("Seek(10) expected error for offset > data length")
	}
}

func TestSubReader(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	reader := NewReader(data)

	subReader, err := reader.SubReader(3)
	if err != nil {
		t.Fatalf("SubReader(3) unexpected error: %v", err)
	}

	if subReader.Len() != 3 {
		t.Errorf("SubReader(3).Len() = %d, want 3", subReader.Len())
	}
	if reader.Offset() != 3 {
		t.Errorf("SubReader(3) parent offset = %d, want 3", reader.Offset())
	}

	// Test reading from sub-reader
	val, err := subReader.ReadU8()
	if err != nil {
		t.Fatalf("SubReader.ReadU8() unexpected error: %v", err)
	}
	if val != 1 {
		t.Errorf("SubReader.ReadU8() = %d, want 1", val)
	}
}

func BenchmarkReadU32(b *testing.B) {
	data := make([]byte, 4*1000)
	for i := 0; i < 1000; i++ {
		binary.LittleEndian.PutUint32(data[i*4:], uint32(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := NewReader(data)
		for j := 0; j < 1000; j++ {
			reader.ReadU32()
		}
	}
}

func BenchmarkReadU32Array(b *testing.B) {
	// Create array with 100 elements
	data := []byte{100} // length
	for i := 0; i < 100; i++ {
		val := make([]byte, 4)
		binary.LittleEndian.PutUint32(val, uint32(i))
		data = append(data, val...)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := NewReader(data)
		reader.ReadU32Array()
	}
}
