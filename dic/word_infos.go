package dic

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

// WordInfos represents word information storage
type WordInfos struct {
	bytes              []byte
	offset             int
	wordSize           uint32
	hasSynonymGroupIds bool
}

// WordInfo represents information about a word
type WordInfo struct {
	Surface              string
	HeadWordLength       uint16
	PosId                uint16
	NormalizedForm       string
	DictionaryFormWordId int32
	DictionaryForm       string
	ReadingForm          string
	AUnitSplit           []WordId
	BUnitSplit           []WordId
	WordStructure        []WordId
	SynonymGroupIds      []uint32
}

// NewWordInfos creates a new WordInfos
func NewWordInfos(bytes []byte, offset int, wordSize uint32, hasSynonymGroupIds bool) *WordInfos {
	return &WordInfos{
		bytes:              bytes,
		offset:             offset,
		wordSize:           wordSize,
		hasSynonymGroupIds: hasSynonymGroupIds,
	}
}

// wordIdToOffset converts word ID to byte offset
func (wi *WordInfos) wordIdToOffset(wordId uint32) (int, error) {
	if wordId >= wi.wordSize {
		return 0, fmt.Errorf("word ID out of range: %d >= %d", wordId, wi.wordSize)
	}

	offsetIndex := wi.offset + int(wordId)*4
	if offsetIndex+4 > len(wi.bytes) {
		return 0, fmt.Errorf("offset index out of range: %d >= %d", offsetIndex+4, len(wi.bytes))
	}

	offset := binary.LittleEndian.Uint32(wi.bytes[offsetIndex : offsetIndex+4])

	return int(offset), nil
}

// GetWordInfo returns word information for the given word ID
func (wi *WordInfos) GetWordInfo(wordId uint32) (*WordInfo, error) {
	offset, err := wi.wordIdToOffset(wordId)
	if err != nil {
		return nil, err
	}

	if offset >= len(wi.bytes) {
		return nil, fmt.Errorf("offset out of range: %d >= %d", offset, len(wi.bytes))
	}

	// Parse the word info from the binary format
	return wi.parseWordInfo(wordId, offset)
}

// parseWordInfo parses word information from bytes
func (wi *WordInfos) parseWordInfo(wordId uint32, offset int) (*WordInfo, error) {
	if offset >= len(wi.bytes) {
		return nil, fmt.Errorf("offset out of range: %d >= %d", offset, len(wi.bytes))
	}

	reader, err := NewReaderAt(wi.bytes, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader at offset %d: %w", offset, err)
	}

	// 表層形の解析（文字列長 + UTF-16文字列）
	surface, err := wi.readString16(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read surface at offset %d: %w", offset, err)
	}

	// ヘッド語の長さ（可変長エンコーディング）
	headWordLength, err := wi.readVariableLengthU16(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read head word length at offset %d: %w", offset, err)
	}

	// 品詞ID
	posId, err := reader.ReadU16()
	if err != nil {
		return nil, fmt.Errorf("failed to read pos id at offset %d: %w", offset, err)
	}

	// 正規化形の解析（UTF-16文字列）
	normalizedForm, err := wi.readString16(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read normalized form at offset %d: %w", offset, err)
	}

	// 辞書形WordIDの解析（32ビット整数）
	dictionaryFormWordId, err := reader.ReadI32()
	if err != nil {
		return nil, fmt.Errorf("failed to read dictionary form at offset %d: %w", offset, err)
	}

	// 読み形の解析（UTF-16文字列）
	readingForm, err := wi.readString16(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read reading form at offset %d: %w", offset, err)
	}

	// A単位分割情報の解析
	aUnitSplit, err := wi.readWordIdArray(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read A unit split at offset %d: %w", offset, err)
	}

	// B単位分割情報の解析
	bUnitSplit, err := wi.readWordIdArray(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read B unit split at offset %d: %w", offset, err)
	}

	// 語構造情報の解析
	wordStructure, err := wi.readWordIdArray(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read word structure at offset %d: %w", offset, err)
	}

	// 同義語グループIDの解析
	var synonymGroupIds []uint32
	if wi.hasSynonymGroupIds {
		synonymGroupIds, err = wi.readU32Array(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read synonym group ids at offset %d: %w", offset, err)
		}
	}

	// 辞書形の処理 (Rust版と同じロジック)
	dictionaryForm := surface // デフォルトは表層形
	if dictionaryFormWordId >= 0 && dictionaryFormWordId != int32(wordId) {
		// 別のWordIdの表層形を取得
		if dfWordInfo, err := wi.GetWordInfo(uint32(dictionaryFormWordId)); err == nil {
			dictionaryForm = dfWordInfo.Surface
		}
	}

	return &WordInfo{
		Surface:              surface,
		HeadWordLength:       headWordLength,
		PosId:                posId,
		NormalizedForm:       normalizedForm,
		DictionaryFormWordId: dictionaryFormWordId,
		DictionaryForm:       dictionaryForm,
		ReadingForm:          readingForm,
		AUnitSplit:           aUnitSplit,
		BUnitSplit:           bUnitSplit,
		WordStructure:        wordStructure,
		SynonymGroupIds:      synonymGroupIds,
	}, nil
}

// GetNormalizedForm returns the normalized form, falling back to surface if empty (Rust-compatible)
func (wi *WordInfo) GetNormalizedForm() string {
	if wi.NormalizedForm == "" {
		return wi.Surface
	}
	return wi.NormalizedForm
}

// GetDictionaryForm returns the dictionary form, falling back to surface if empty (Rust-compatible)
func (wi *WordInfo) GetDictionaryForm() string {
	if wi.DictionaryForm == "" {
		return wi.Surface
	}
	return wi.DictionaryForm
}

// GetReadingForm returns the reading form, falling back to surface if empty (Rust-compatible)
func (wi *WordInfo) GetReadingForm() string {
	if wi.ReadingForm == "" {
		return wi.Surface
	}
	return wi.ReadingForm
}

// readString16 reads a UTF-16 string with variable-length encoding
func (wi *WordInfos) readString16(reader *Reader) (string, error) {
	// Read string length using variable-length encoding
	length, err := wi.readVariableLengthU16(reader)
	if err != nil {
		return "", err
	}

	if length == 0 {
		return "", nil
	}

	// Read UTF-16 bytes
	utf16Bytes, err := reader.ReadBytes(int(length * 2))
	if err != nil {
		return "", err
	}

	// Convert UTF-16 to UTF-8
	utf16Runes := make([]uint16, length)
	for i := 0; i < int(length); i++ {
		utf16Runes[i] = binary.LittleEndian.Uint16(utf16Bytes[i*2 : i*2+2])
	}

	return string(utf16.Decode(utf16Runes)), nil
}

// readWordIdArray reads an array of WordIds using the same format as Reader.ReadWordIdArray
func (wi *WordInfos) readWordIdArray(reader *Reader) ([]WordId, error) {
	// Use the existing Reader method which handles the correct format
	return reader.ReadWordIdArray()
}

// readU32Array reads an array of uint32 using the same format as Reader.ReadU32Array
func (wi *WordInfos) readU32Array(reader *Reader) ([]uint32, error) {
	// Use the existing Reader method which handles the correct format
	return reader.ReadU32Array()
}

// readVariableLengthU16 reads a variable-length encoded uint16
// Following Rust's string_length_parser implementation:
// - If first byte < 128: use as direct length
// - If first byte >= 128: read second byte and calculate length as ((first & 0x7F) << 8) | second
func (wi *WordInfos) readVariableLengthU16(reader *Reader) (uint16, error) {
	firstByte, err := reader.ReadU8()
	if err != nil {
		return 0, err
	}

	if firstByte < 128 {
		// Single byte encoding: use value directly
		return uint16(firstByte), nil
	} else {
		// Two byte encoding: combine with second byte
		secondByte, err := reader.ReadU8()
		if err != nil {
			return 0, err
		}

		// Calculate length: ((first & 0x7F) << 8) | second
		length := uint16(firstByte&0x7F)<<8 | uint16(secondByte)
		return length, nil
	}
}
