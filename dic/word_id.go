package dic

import (
	"fmt"
)

// WordId represents a dictionary word ID
//
// Encodes dictionary ID and word internal ID as 4 bits and 28 bits respectively
// DicId 0 - system dictionary
// DicId 15 - OOV and other special nodes
type WordId struct {
	raw uint32
}

const (
	// WordMask is the mask for the word ID part (lower 28 bits)
	WordMask uint32 = 0x0fff_ffff

	// MaxWord is the maximum word ID value
	MaxWord uint32 = 0x0fff_ffff
)

// Special WordId constants
var (
	// Invalid represents an invalid WordId
	Invalid = WordId{raw: 0xffff_ffff}

	// BOS represents beginning of sentence
	BOS = WordId{raw: 0xffff_fffe}

	// EOS represents end of sentence
	EOS = WordId{raw: 0xffff_fffd}
)

// FromRaw creates a WordId from the compressed representation
func FromRaw(raw uint32) WordId {
	return WordId{raw: raw}
}

// New creates a WordId from dictionary ID and word ID parts
func New(dic uint8, word uint32) WordId {
	dicPart := (uint32(dic&0xf) << 28)
	wordPart := word & WordMask
	return WordId{raw: dicPart | wordPart}
}

// Checked creates a WordId with correctness checking
func Checked(dic uint8, word uint32) (WordId, error) {
	if dic&^0xf != 0 {
		return Invalid, fmt.Errorf("dictionary ID %d exceeds maximum 15", dic)
	}

	if word&^WordMask != 0 {
		return Invalid, fmt.Errorf("word ID %d exceeds maximum %d", word, WordMask)
	}

	return New(dic, word), nil
}

// OOV creates an OOV (Out-of-Vocabulary) WordId for the given POS ID
func OOV(posId uint32) WordId {
	return New(0xf, posId)
}

// Raw returns the raw uint32 representation
func (w WordId) Raw() uint32 {
	return w.raw
}

// Dic extracts the dictionary ID (upper 4 bits)
func (w WordId) Dic() uint8 {
	return uint8(w.raw >> 28)
}

// Word extracts the word ID (lower 28 bits)
func (w WordId) Word() uint32 {
	return w.raw & WordMask
}

// IsSystem checks if the word comes from the system dictionary
func (w WordId) IsSystem() bool {
	return w.Dic() == 0
}

// IsUser checks if the word comes from a user dictionary
func (w WordId) IsUser() bool {
	dic := w.Dic()
	return dic != 0 && dic != 0xf
}

// IsOOV checks if the word is OOV (Out-of-Vocabulary)
// An OOV node can come from OOV handlers or be a special system node like BOS or EOS
func (w WordId) IsOOV() bool {
	return w.Dic() == 0xf
}

// IsSpecial checks if the WordId corresponds to a special node (BOS, EOS, etc.)
func (w WordId) IsSpecial() bool {
	return w.raw >= EOS.raw && w.raw < Invalid.raw
}

// String returns a string representation of the WordId
func (w WordId) String() string {
	if w.IsOOV() {
		return fmt.Sprintf("(-1, %d)", w.Word())
	}
	return fmt.Sprintf("(%d, %d)", w.Dic(), w.Word())
}

// Equal checks if two WordIds are equal
func (w WordId) Equal(other WordId) bool {
	return w.raw == other.raw
}

// Compare compares two WordIds
// Returns -1 if w < other, 0 if w == other, 1 if w > other
func (w WordId) Compare(other WordId) int {
	if w.raw < other.raw {
		return -1
	}
	if w.raw > other.raw {
		return 1
	}
	return 0
}

// Hash returns a hash value for the WordId (suitable for maps)
func (w WordId) Hash() uint32 {
	return w.raw
}
