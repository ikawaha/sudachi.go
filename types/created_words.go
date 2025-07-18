package types

import "math"

// CreatedWords tracks which word lengths have been created at a position
// This is a faithful port of Rust's CreatedWords implementation (value type with Copy semantics)
// Rust: #[derive(Copy, Clone, Eq, PartialEq, Default, Debug)] pub struct CreatedWords(Carrier);
type CreatedWords struct {
	bitmap uint64 // Matches Rust: Carrier = u64
}

// CreatedWords constants (matching Rust implementation exactly)
const (
	MaxWordLength = 64                // Matches Rust: pub const MAX_VALUE: Carrier = 64;
	maxShift      = MaxWordLength - 1 // Matches Rust: const MAX_SHIFT: Carrier = CreatedWords::MAX_VALUE - 1;
)

// EmptyCreatedWords creates an empty CreatedWords (matches Rust: CreatedWords::empty())
func EmptyCreatedWords() CreatedWords {
	return CreatedWords{bitmap: 0} // Matches Rust: Default::default()
}

// SingleCreatedWords creates CreatedWords with single word length (matches Rust: CreatedWords::single())
func SingleCreatedWords(length int) CreatedWords {
	if length <= 0 {
		return EmptyCreatedWords()
	}
	raw := uint64(length)
	shift := uint64(math.Min(float64(raw-1), float64(maxShift))) // Matches Rust: min(raw.saturating_sub(1), CreatedWords::MAX_SHIFT)
	bits := uint64(1) << shift                                   // Matches Rust: (1 as Carrier) << shift
	return CreatedWords{bitmap: bits}                            // Matches Rust: CreatedWords(bits)
}

// AddWord returns new CreatedWords with additional word length (matches Rust: add_word())
// Rust: #[must_use] pub fn add_word<P: Into<i64>>(&self, length: P) -> CreatedWords
func (cw CreatedWords) AddWord(length int) CreatedWords {
	mask := SingleCreatedWords(length) // Matches Rust: let mask = CreatedWords::single(length);
	return cw.Add(mask)                // Matches Rust: self.add(mask)
}

// Add returns new CreatedWords combined with other (matches Rust: add())
// Rust: #[must_use] pub fn add(&self, other: CreatedWords) -> CreatedWords
func (cw CreatedWords) Add(other CreatedWords) CreatedWords {
	return CreatedWords{bitmap: cw.bitmap | other.bitmap} // Matches Rust: CreatedWords(self.0 | other.0)
}

// HasWord checks if word of given length was created (matches Rust: has_word())
func (cw CreatedWords) HasWord(length int) HasWordResult {
	mask := SingleCreatedWords(length)  // Matches Rust: let mask = CreatedWords::single(length);
	if (cw.bitmap & mask.bitmap) == 0 { // Matches Rust: if (self.0 & mask.0) == 0
		return HasWordNo // Matches Rust: HasWord::No
	} else if length >= MaxWordLength { // Matches Rust: else if length.into() >= CreatedWords::MAX_VALUE as _
		return HasWordMaybe // Matches Rust: HasWord::Maybe
	} else {
		return HasWordYes // Matches Rust: HasWord::Yes
	}
}

// IsEmpty returns true if no words have been created (matches Rust: is_empty())
func (cw CreatedWords) IsEmpty() bool {
	return cw.bitmap == 0 // Matches Rust: self.0 == 0
}

// NotEmpty returns true if any words have been created (matches Rust: not_empty())
func (cw CreatedWords) NotEmpty() bool {
	return !cw.IsEmpty() // Matches Rust: !self.is_empty()
}

// HasWordResult represents the result of HasWord check (matches Rust enum HasWord)
type HasWordResult int

const (
	HasWordYes   HasWordResult = iota // Matches Rust: HasWord::Yes
	HasWordNo                         // Matches Rust: HasWord::No
	HasWordMaybe                      // Matches Rust: HasWord::Maybe
)
