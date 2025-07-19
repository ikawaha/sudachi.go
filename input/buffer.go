package input

import (
	"fmt"
	"unicode/utf8"

	"github.com/ikawaha/sudachi.go/dic"
)

// BufferState represents the state of an InputBuffer
type BufferState int

func (s BufferState) String() string {
	switch s {
	case StateClean:
		return "clean"
	case StateReadWrite:
		return "read-write"
	case StateReadOnly:
		return "read-only"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}

const (
	// StateClean is the initial state - buffer is empty and ready for input
	StateClean BufferState = iota
	// StateReadWrite is the state when the buffer is being built - input can be modified
	StateReadWrite
	//StateReadOnly is the state when the buffer is finalized - input cannot be modified
	StateReadOnly
)

// InputBuffer represents input text with normalization and mapping information
type InputBuffer struct {
	// Original input data, output is done on this
	original string
	// Normalized input data, analysis is done on this (byte-based indexing)
	modified string
	// Byte mapping from normalized data to original (byte index in modified -> byte index in original)
	m2o []int
	// Characters of the modified string (char-based indexing)
	modChars []rune
	// Char-to-byte mapping for the modified string (char index -> byte index)
	modC2B []int
	// Byte-to-char mapping for the modified string (byte index -> char index)
	modB2C []int
	// Markers whether the byte can start new word or not
	modBOW []bool
	// Character categories (char-based indexing)
	modCat []dic.CategoryType
	// Number of codepoints with the same category (char-based indexing)
	modCatContinuity []int
	// Character category system (optional - for advanced categorization)
	charCategory *dic.CharacterCategory
	// Normalization information for accurate mapping construction
	normalizationInfo *NormalizationInfo
	// State of the buffer
	state BufferState
}

// NewInputBuffer creates a new empty InputBuffer
func NewInputBuffer() *InputBuffer {
	return &InputBuffer{
		state: StateClean,
	}
}

// SetCharacterCategory sets the character category system for advanced categorization
func (ib *InputBuffer) SetCharacterCategory(charCategory *dic.CharacterCategory) {
	ib.charCategory = charCategory
}

// Removed StartBuildWithNormalization - using Rust-compatible approach instead

// StartBuild transitions the buffer to ReadWrite state (matching Rust start_build)
func (ib *InputBuffer) StartBuild(original string) error {
	if ib.state != StateClean {
		return fmt.Errorf("invalid buffer state: buffer must be in clean state to start build")
	}

	ib.original = original
	ib.modified = original // Start with original text

	// Initialize m2o mapping (matching Rust: self.m2o.extend(0..self.modified.len() + 1))
	ib.m2o = make([]int, len(ib.modified)+1)
	for i := range ib.m2o {
		ib.m2o[i] = i
	}

	ib.state = StateReadWrite

	return nil
}

// Build finalizes the buffer and computes all derived data (matching Rust implementation)
func (ib *InputBuffer) Build(grammar *dic.Grammar) error {
	if ib.state != StateReadWrite {
		return fmt.Errorf("buffer must be in read-write state to build")
	}

	// Compute derived data using Rust-compatible algorithm
	err := ib.computeDerivedDataRustCompatible(grammar)
	if err != nil {
		return err
	}

	ib.state = StateReadOnly
	return nil
}

// computeDerivedDataRustCompatible computes all derived data using exact Rust algorithm
func (ib *InputBuffer) computeDerivedDataRustCompatible(grammar *dic.Grammar) error {
	// Clear previous data (matching Rust)
	ib.modChars = ib.modChars[:0]
	ib.modCat = ib.modCat[:0]
	ib.modC2B = ib.modC2B[:0]
	ib.modB2C = ib.modB2C[:0]

	// Get a character category system (matching Rust)
	cats := grammar.CharacterCategory

	// Initialize tracking variables (matching Rust implementation)
	lastOffset := 0
	lastChIdx := 0

	// Special categories for BOW logic (matching Rust non_starting)
	nonStarting := dic.CategoryAlpha | dic.CategoryGreek | dic.CategoryCyrillic
	var prevCat dic.CategoryType

	// Initialize mod_bow array (matching Rust mod_bow.resize)
	ib.modBOW = make([]bool, len(ib.modified))
	nextBow := true

	// Process each character using char_indices().enumerate() (matching Rust exactly)
	// Rust: for (chidx, (bidx, ch)) in self.modified.char_indices().enumerate()
	chIdx := 0
	for byteIdx, ch := range []byte(ib.modified) {
		// Skip non-character-starting bytes (UTF-8 continuation bytes)
		if !utf8.RuneStart(ch) {
			continue
		}

		// Decode the full rune at this position
		r, _ := utf8.DecodeRune([]byte(ib.modified)[byteIdx:])

		// Add character to mod_chars (matching Rust)
		ib.modChars = append(ib.modChars, r)

		// Get character category (matching Rust cats.get_category_types)
		cat := cats.GetCategory(r)
		ib.modCat = append(ib.modCat, cat)

		// Build char-to-byte mapping (matching Rust mod_c2b.push)
		ib.modC2B = append(ib.modC2B, byteIdx)

		// Extend byte-to-char mapping (matching Rust mod_b2c.extend)
		for i := lastOffset; i < byteIdx; i++ {
			ib.modB2C = append(ib.modB2C, lastChIdx)
		}
		lastOffset = byteIdx
		lastChIdx = chIdx

		// Calculate can_bow (matching Rust BOW logic exactly)
		var canBow bool
		if !nextBow {
			// this char was forbidden by the previous one
			nextBow = true
			canBow = false
		} else if cat.Intersects(dic.CategoryNoOOVBOW2) {
			// this rule is stronger than the next one and must come before
			// this and next are forbidden
			nextBow = false
			canBow = false
		} else if cat.Intersects(dic.CategoryNoOOVBOW) {
			// this char is forbidden
			canBow = false
		} else if cat.Intersects(nonStarting) {
			// the previous char is compatible
			canBow = !cat.Intersects(prevCat)
		} else {
			canBow = true
		}

		// Set BOW marker (matching Rust self.mod_bow[bidx] = can_bow)
		if byteIdx < len(ib.modBOW) {
			ib.modBOW[byteIdx] = canBow
		}
		prevCat = cat
		chIdx++
	}

	// Trailing indices for the last codepoint (matching Rust)
	for i := lastOffset; i < len(ib.modified); i++ {
		ib.modB2C = append(ib.modB2C, lastChIdx)
	}

	// Sentinel values for range translations (matching Rust)
	ib.modC2B = append(ib.modC2B, len(ib.modB2C))
	ib.modB2C = append(ib.modB2C, lastChIdx+1)

	// Fill character category continuity (matching Rust)
	ib.fillCatContinuity()

	// Fill original byte-to-char mapping (matching Rust)
	ib.fillOrigB2C()

	return nil
}

// fillCatContinuity fills character category continuity (matching Rust implementation)
func (ib *InputBuffer) fillCatContinuity() {
	if len(ib.modChars) == 0 {
		return
	}

	// Initialize continuity array (matching Rust mod_cat_continuity.resize)
	ib.modCatContinuity = make([]int, len(ib.modChars))

	// Single pass algorithm (matching Rust fill_cat_continuity)
	// By default continuity is 1 codepoint
	// Go from the back and set it prev + 1 when chars are compatible
	for i := range ib.modCatContinuity {
		ib.modCatContinuity[i] = 1
	}

	if len(ib.modCat) == 0 {
		return
	}

	cat := ib.modCat[len(ib.modCat)-1]
	for i := len(ib.modCat) - 2; i >= 0; i-- {
		cur := ib.modCat[i]
		common := cur & cat
		if !common.Intersects(dic.CategoryAll) {
			// No common categories - continuity remains 1
			cat = cur
		} else {
			// Has common categories - extend continuity
			ib.modCatContinuity[i] = ib.modCatContinuity[i+1] + 1
			cat = common
		}
	}
}

// fillOrigB2C fills original byte-to-char mapping (matching Rust implementation exactly)
func (ib *InputBuffer) fillOrigB2C() {
	// Initialize m2o if needed (matching Rust: self.m2o.extend(0..self.modified.len() + 1))
	if len(ib.m2o) == 0 {
		ib.m2o = make([]int, len(ib.modified)+1)
		for i := range ib.m2o {
			ib.m2o[i] = i
		}
	}
}

// CanBOW returns whether the byte can start a new word (exact Rust port)
func (ib *InputBuffer) CanBOW(offset int) bool {
	if ib.state != StateReadOnly {
		return false
	}
	if offset >= len(ib.modBOW) {
		return false
	}
	return ib.modBOW[offset]
}

// Original returns the original input text
func (ib *InputBuffer) Original() string {
	return ib.original
}

// Modified returns the normalized/modified text
func (ib *InputBuffer) Modified() string {
	return ib.modified
}

// ModifiedBytes returns the modified text as byte slice
func (ib *InputBuffer) ModifiedBytes() []byte {
	return []byte(ib.modified)
}

// State returns the current buffer state
func (ib *InputBuffer) State() BufferState {
	return ib.state
}

// CharCount returns the number of characters in the modified text
func (ib *InputBuffer) CharCount() int {
	return len(ib.modChars)
}

// ByteCount returns the number of bytes in the modified text
func (ib *InputBuffer) ByteCount() int {
	return len(ib.modified)
}

// GetChar returns the character at the given character index
func (ib *InputBuffer) GetChar(charIdx int) (rune, error) {
	if charIdx < 0 || charIdx >= len(ib.modChars) {
		return 0, fmt.Errorf("index %d out of bounds [0, %d)", charIdx, len(ib.modChars))
	}
	return ib.modChars[charIdx], nil
}

// GetCategory returns the character category at the given character index
func (ib *InputBuffer) GetCategory(charIdx int) (dic.CategoryType, error) {
	if charIdx < 0 || charIdx >= len(ib.modCat) {
		return dic.CategoryDefault, fmt.Errorf("index %d out of bounds [0, %d)", charIdx, len(ib.modCat))
	}
	return ib.modCat[charIdx], nil
}

// GetCategoryContinuity returns the category continuity at the given character index
func (ib *InputBuffer) GetCategoryContinuity(charIdx int) (int, error) {
	if charIdx < 0 || charIdx >= len(ib.modCatContinuity) {
		return 0, fmt.Errorf("index %d out of bounds [0, %d)", charIdx, len(ib.modCatContinuity))
	}
	return ib.modCatContinuity[charIdx], nil
}

// IsBOW returns true if the byte at the given index can start a new word
func (ib *InputBuffer) IsBOW(byteIdx int) bool {
	if byteIdx < 0 || byteIdx >= len(ib.modBOW) {
		return false
	}
	return ib.modBOW[byteIdx]
}

// SetModified sets the modified text (should be called before Build)
func (ib *InputBuffer) SetModified(modified string) error {
	if ib.state != StateReadWrite {
		return fmt.Errorf("invalid buffer state: buffer is in %s state, buffer must be in %s state to set modified text", ib.state, StateReadWrite)
	}
	ib.modified = modified
	return nil
}

// CharToByteIndex converts character index to byte index
func (ib *InputBuffer) CharToByteIndex(charIdx int) (int, error) {
	if charIdx < 0 || charIdx >= len(ib.modC2B) {
		return 0, fmt.Errorf("index %d out of bounds [0, %d)", charIdx, len(ib.modC2B))

	}
	return ib.modC2B[charIdx], nil
}

// ByteToCharIndex converts byte index to character index
func (ib *InputBuffer) ByteToCharIndex(byteIdx int) (int, error) {
	if byteIdx < 0 || byteIdx >= len(ib.modB2C) {
		return 0, fmt.Errorf("index %d out of bounds [0, %d)", byteIdx, len(ib.modC2B))
	}
	return ib.modB2C[byteIdx], nil
}

// ModifiedToOriginalIndex converts modified text byte index to original text byte index
func (ib *InputBuffer) ModifiedToOriginalIndex(modifiedIdx int) (int, error) {
	if modifiedIdx < 0 || modifiedIdx >= len(ib.m2o) {
		return 0, fmt.Errorf("index %d out of bounds [0, %d)", modifiedIdx, len(ib.m2o))
	}
	return ib.m2o[modifiedIdx], nil
}

// IsReadOnly returns true if the buffer is in read-only state
func (ib *InputBuffer) IsReadOnly() bool {
	return ib.state == StateReadOnly
}

// GetOriginalText returns a substring of the original text between character positions
// This matches Rust version's orig_slice functionality
func (ib *InputBuffer) GetOriginalText(beginChar, endChar int) string {
	if beginChar < 0 || endChar > len(ib.modChars) || beginChar >= endChar {
		return ""
	}

	// Convert character positions to byte positions in modified text
	beginByte, err := ib.CharToByteIndex(beginChar)
	if err != nil {
		return ""
	}

	endByte := len(ib.modified)
	if endChar < len(ib.modC2B) {
		endByte, err = ib.CharToByteIndex(endChar)
		if err != nil {
			return ""
		}
	}

	// Validate byte range in modified text
	if beginByte >= len(ib.modified) || endByte > len(ib.modified) {
		return ""
	}

	// Convert modified text byte range to original text byte range using m2o mapping
	// This matches Rust's to_orig() function: self.m2o[range.start]..self.m2o[range.end]
	if beginByte >= len(ib.m2o) || endByte > len(ib.m2o) {
		return ""
	}

	origBeginByte := ib.m2o[beginByte]

	// Calculate end position safely
	var origEndByte int
	if endByte >= len(ib.m2o) {
		origEndByte = len(ib.original)
	} else {
		origEndByte = ib.m2o[endByte]
	}

	// Validate range in original text
	if origBeginByte >= len(ib.original) || origEndByte > len(ib.original) || origBeginByte >= origEndByte {
		return ""
	}

	return ib.original[origBeginByte:origEndByte]
}

// Range represents a byte range with start and end positions
type Range struct {
	Start int
	End   int
}

// ToOrig converts a byte range from modified text to original text
// This matches Rust implementation: InputTextIndex.to_orig()
func (ib *InputBuffer) ToOrig(modifiedRange Range) Range {
	if ib.state == StateClean {
		return Range{Start: 0, End: 0}
	}

	// Match Rust: direct array access without bounds checking
	// Rust: self.m2o[range.start]..self.m2o[range.end]
	start := ib.m2o[modifiedRange.Start]
	end := ib.m2o[modifiedRange.End]

	return Range{Start: start, End: end}
}

// OrigSlice extracts a substring from original text using modified text byte range
// This matches Rust implementation: InputTextIndex.orig_slice()
func (ib *InputBuffer) OrigSlice(modifiedRange Range) string {
	if ib.state == StateClean {
		return ""
	}

	// Convert modified byte range to original byte range
	origRange := ib.ToOrig(modifiedRange)

	// Match Rust: simple bounds check but allow empty ranges
	if origRange.Start < 0 || origRange.End > len(ib.original) || origRange.Start > origRange.End {
		return ""
	}

	// Extract from original text
	return ib.original[origRange.Start:origRange.End]
}

// WithEditor executes a function that can modify the buffer contents with proper mapping tracking
// This matches Rust implementation: InputBuffer.with_editor()
func (ib *InputBuffer) WithEditor(editorFunc func(*InputBuffer, *InputEditor) error) error {
	if ib.state != StateReadWrite {
		return fmt.Errorf("invalid buffer state: buffer is in %s state, buffer must be in %s state to set modified text", ib.state, StateReadWrite)
	}

	// Create editor
	editor := NewInputEditor()

	// Execute the editing function
	err := editorFunc(ib, editor)
	if err != nil {
		return err
	}

	// Apply all edits if any
	edits := editor.GetReplaces()
	if len(edits) > 0 {
		return ib.commitEdits(edits)
	}

	return nil
}

// commitEdits applies all edit operations and updates the mapping
func (ib *InputBuffer) commitEdits(edits []ReplaceOp) error {
	if len(edits) == 0 {
		return nil
	}

	// Apply edits using Rust-compatible algorithm
	// Update buffer with new content and mapping (matching Rust: swap buffers)
	ib.modified, ib.m2o, _ = ResolveEdits(ib.modified, ib.m2o, edits)

	// Clear derived-data since a text was modified (matching Rust implementation)
	// The data will be recomputed when Build() is called again
	ib.modChars = ib.modChars[:0]
	ib.modCat = ib.modCat[:0]
	ib.modC2B = ib.modC2B[:0]
	ib.modB2C = ib.modB2C[:0]
	ib.modBOW = ib.modBOW[:0]
	ib.modCatContinuity = ib.modCatContinuity[:0]

	// Set state back to ReadWrite so Build() can be called again
	ib.state = StateReadWrite

	return nil
}

// Reset resets the buffer to clean state
func (ib *InputBuffer) Reset() {
	ib.original = ""
	ib.modified = ""
	ib.m2o = ib.m2o[:0]
	ib.modChars = ib.modChars[:0]
	ib.modC2B = ib.modC2B[:0]
	ib.modB2C = ib.modB2C[:0]
	ib.modBOW = ib.modBOW[:0]
	ib.modCat = ib.modCat[:0]
	ib.modCatContinuity = ib.modCatContinuity[:0]
	ib.state = StateClean
}
