package dic

import (
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	// POSDepth in Japanese morphological analysis
	POSDepth = 6

	InhibitedConnection = int16(32767) // i16::MAX
)

// Grammar represents dictionary grammar containing POS list and connection costs
type Grammar struct {
	bytes             []byte
	posList           [][]string
	storageSize       int
	connectionMatrix  *ConnectionMatrix
	CharacterCategory *CharacterCategory // Public field matching Rust pub character_category
}

// NewGrammar creates a Grammar from dictionary bytes
func NewGrammar(buf []byte, offset int) (*Grammar, error) {
	if offset >= len(buf) {
		return nil, fmt.Errorf("invalid grammar offset: %d exceeds buffer size %d", offset, len(buf))
	}

	// Parse grammar structure
	posList, leftIdSize, rightIdSize, connectTableOffset, err := parseGrammar(buf, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to parse grammar at offset %d: %w", offset, err)
	}

	// Calculate storage size
	storageSize := (connectTableOffset - offset) + 2*int(leftIdSize)*int(rightIdSize)

	// Create connection matrix
	connMatrix, err := NewConnectionMatrix(
		buf,
		connectTableOffset,
		int(leftIdSize),
		int(rightIdSize))
	if err != nil {
		return nil, err
	}

	return &Grammar{
		bytes:             buf,
		posList:           posList,
		storageSize:       storageSize,
		connectionMatrix:  connMatrix,
		CharacterCategory: NewCharacterCategory(), // Matching Rust CharacterCategory::default()
	}, nil
}

// ConnectCost returns the connection cost between left and right morphemes
func (g *Grammar) ConnectCost(leftId, rightId int16) int16 {
	return g.connectionMatrix.Cost(uint16(leftId), uint16(rightId))
}

// ConnectionMatrix returns the connection matrix
func (g *Grammar) ConnectionMatrix() *ConnectionMatrix {
	return g.connectionMatrix
}

// SetCharacterCategory sets the character category (matching Rust implementation)
// This is the only way to set character category.
// Character category will be an empty map by default.
func (g *Grammar) SetCharacterCategory(characterCategory *CharacterCategory) {
	g.CharacterCategory = characterCategory
}

// SetConnectCost sets the connection cost for a specific pair of IDs
func (g *Grammar) SetConnectCost(leftId, rightId, cost int16) {
	g.connectionMatrix.Update(uint16(leftId), uint16(rightId), cost)
}

// GetPartOfSpeechId returns the POS ID for the given POS components
func (g *Grammar) GetPartOfSpeechId(pos []string) *uint16 {
	if len(pos) != POSDepth {
		return nil
	}

	for i, posList := range g.posList {
		if len(posList) == len(pos) {
			match := true
			for j, component := range pos {
				if j >= len(posList) || posList[j] != component {
					match = false
					break
				}
			}
			if match {
				result := uint16(i)
				return &result
			}
		}
	}

	return nil
}

// RegisterPOS registers a new POS and returns its ID
func (g *Grammar) RegisterPOS(pos []string) (uint16, error) {
	if len(pos) != POSDepth {
		posString := strings.Join(pos, ",")
		return 0, fmt.Errorf("invalid part of speech: %s", posString)
	}

	// Check if POS already exists
	if existingId := g.GetPartOfSpeechId(pos); existingId != nil {
		return *existingId, nil
	}

	// Add new POS
	newId := len(g.posList)
	if newId > 65535 { // uint16 max
		return 0, fmt.Errorf("invalid part of speech: too many POS tags registered: %d", newId)
	}

	// Create copy of pos slice
	components := make([]string, len(pos))
	copy(components, pos)

	g.posList = append(g.posList, components)
	return uint16(newId), nil
}

// GetPOS returns the POS components for the given ID
func (g *Grammar) GetPOS(id uint16) ([]string, error) {
	if int(id) >= len(g.posList) {
		return nil, fmt.Errorf("invalid POS ID: ID %d out of range (max %d)", id, len(g.posList)-1)
	}

	// Return copy to prevent modification
	pos := make([]string, len(g.posList[id]))
	copy(pos, g.posList[id])
	return pos, nil
}

// GetPOSId returns the POS ID for the given POS components (alias for GetPartOfSpeechId)
func (g *Grammar) GetPOSId(pos []string) (uint16, error) {
	id := g.GetPartOfSpeechId(pos)
	if id == nil {
		posString := strings.Join(pos, ",")
		return 0, fmt.Errorf("POS not found: %s", posString)
	}
	return *id, nil
}

// POSListSize returns the number of registered POS entries
func (g *Grammar) POSListSize() int {
	return len(g.posList)
}

// StorageSize returns the total storage size of the grammar
func (g *Grammar) StorageSize() int {
	return g.storageSize
}

// parseGrammar parses the grammar section and returns POS list and connection info
func parseGrammar(buf []byte, offset int) ([][]string, uint16, uint16, int, error) {
	reader, err := NewReaderAt(buf, offset)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	// Read POS size
	posSize, err := reader.ReadU16()
	if err != nil {
		return nil, 0, 0, 0, err
	}

	// Parse POS list
	var posList [][]string
	for i := 0; i < int(posSize); i++ {
		var posComponents []string

		// Each POS has POSDepth components (6 UTF-16 strings)
		for j := 0; j < POSDepth; j++ {
			// Read variable-length string length (compatible with Rust implementation)
			strLen, err := readVariableLengthStringSize(reader)
			if err != nil {
				return nil, 0, 0, 0, err
			}

			// Read UTF-16 string data
			strBytes, err := reader.ReadSlice(int(strLen) * 2) // UTF-16 = 2 bytes per char
			if err != nil {
				return nil, 0, 0, 0, err
			}

			// Convert UTF-16 to UTF-8
			str := utf16BytesToString(strBytes)
			posComponents = append(posComponents, str)
		}

		posList = append(posList, posComponents)
	}

	// Read left_id_size and right_id_size
	leftIdSize, err := reader.ReadU16()
	if err != nil {
		return nil, 0, 0, 0, err
	}

	rightIdSize, err := reader.ReadU16()
	if err != nil {
		return nil, 0, 0, 0, err
	}

	// Connection table starts right after the grammar data
	connectTableOffset := reader.Offset()

	return posList, leftIdSize, rightIdSize, connectTableOffset, nil
}

// readVariableLengthStringSize reads string length using Rust-compatible variable-length encoding
func readVariableLengthStringSize(reader *Reader) (uint16, error) {
	// Read first byte
	firstByte, err := reader.ReadU8()
	if err != nil {
		return 0, err
	}

	// If first byte < 128, it's a 1-byte length
	if firstByte < 128 {
		return uint16(firstByte), nil
	}

	// If first byte >= 128, it's a 2-byte length
	secondByte, err := reader.ReadU8()
	if err != nil {
		return 0, err
	}

	// Combine bytes: ((firstByte & 0x7F) << 8) | secondByte
	length := ((uint16(firstByte) & 0x7F) << 8) | uint16(secondByte)
	return length, nil
}

// utf16BytesToString converts UTF-16 bytes to UTF-8 string
func utf16BytesToString(data []byte) string {
	if len(data)%2 != 0 {
		return ""
	}

	// Convert bytes to UTF-16 code units
	var utf16Units []uint16
	for i := 0; i < len(data); i += 2 {
		unit := binary.LittleEndian.Uint16(data[i : i+2])
		utf16Units = append(utf16Units, unit)
	}

	// Convert UTF-16 to UTF-8 using Go's built-in conversion
	runes := utf16ToRunes(utf16Units)
	return string(runes)
}

// utf16ToRunes converts UTF-16 code units to Unicode runes
func utf16ToRunes(utf16Units []uint16) []rune {
	var runes []rune

	for i := 0; i < len(utf16Units); i++ {
		unit := utf16Units[i]

		// Check if this is a high surrogate (0xD800-0xDBFF)
		if unit >= 0xD800 && unit <= 0xDBFF {
			// This is a surrogate pair
			if i+1 < len(utf16Units) {
				lowSurrogate := utf16Units[i+1]
				if lowSurrogate >= 0xDC00 && lowSurrogate <= 0xDFFF {
					// Combine surrogate pair into single rune
					codepoint := 0x10000 + ((uint32(unit-0xD800) << 10) | uint32(lowSurrogate-0xDC00))
					runes = append(runes, rune(codepoint))
					i++ // Skip the low surrogate
					continue
				}
			}
		}

		// Regular UTF-16 character
		runes = append(runes, rune(unit))
	}

	return runes
}
