package input

import (
	"sort"
	"strings"
	"unicode/utf8"
)

// StringReplacer provides efficient string replacement functionality
// This is a simplified implementation inspired by AhoCorasick for Japanese text processing
type StringReplacer struct {
	// Map from first character to list of patterns starting with that character
	patterns map[rune][]ReplacementPattern
	// Maximum pattern length for optimization
	maxPatternLength int
}

// ReplacementPattern represents a single replacement rule
type ReplacementPattern struct {
	Before string
	After  string
	Length int // Length in characters (not bytes)
}

// NewStringReplacer creates a new string replacer from replacement rules
func NewStringReplacer(rules map[string]string) *StringReplacer {
	replacer := &StringReplacer{
		patterns: make(map[rune][]ReplacementPattern),
	}

	// Convert rules to patterns and group by first character
	for before, after := range rules {
		if before == "" {
			continue // Skip empty patterns
		}

		pattern := ReplacementPattern{
			Before: before,
			After:  after,
			Length: utf8.RuneCountInString(before),
		}

		// Get first character
		firstChar, _ := utf8.DecodeRuneInString(before)

		replacer.patterns[firstChar] = append(replacer.patterns[firstChar], pattern)

		if pattern.Length > replacer.maxPatternLength {
			replacer.maxPatternLength = pattern.Length
		}
	}

	// Sort patterns by length (longer first) for greedy matching
	for char := range replacer.patterns {
		patterns := replacer.patterns[char]
		sort.Slice(patterns, func(i, j int) bool {
			return patterns[i].Length > patterns[j].Length
		})
		replacer.patterns[char] = patterns
	}

	return replacer
}

// Replace performs string replacement using longest-match-first strategy
// This matches Rust AhoCorasick MatchKind::LeftmostLongest behavior
func (sr *StringReplacer) Replace(input string) string {
	if len(sr.patterns) == 0 {
		return input
	}

	inputRunes := []rune(input)
	var result strings.Builder
	result.Grow(len(input)) // Pre-allocate

	i := 0
	for i < len(inputRunes) {
		char := inputRunes[i]

		// Check if this character starts any pattern
		patterns, exists := sr.patterns[char]
		if !exists {
			// No patterns start with this character, copy as-is
			result.WriteRune(char)
			i++
			continue
		}

		// Try to match patterns starting with this character
		matched := false
		for _, pattern := range patterns {
			// Check if we have enough characters remaining
			if i+pattern.Length > len(inputRunes) {
				continue
			}

			// Check if pattern matches at current position
			if sr.matchesAt(inputRunes, i, pattern.Before) {
				// Found a match, replace it
				result.WriteString(pattern.After)
				i += pattern.Length
				matched = true
				break // Use first (longest) match
			}
		}

		if !matched {
			// No pattern matched, copy character as-is
			result.WriteRune(char)
			i++
		}
	}

	return result.String()
}

// matchesAt checks if pattern matches at the given position in the input
func (sr *StringReplacer) matchesAt(inputRunes []rune, pos int, pattern string) bool {
	patternRunes := []rune(pattern)

	if pos+len(patternRunes) > len(inputRunes) {
		return false
	}

	for j, patternRune := range patternRunes {
		if inputRunes[pos+j] != patternRune {
			return false
		}
	}

	return true
}

// HasReplacements returns true if any replacement rules are defined
func (sr *StringReplacer) HasReplacements() bool {
	return len(sr.patterns) > 0
}

// GetMaxPatternLength returns the length of the longest pattern
func (sr *StringReplacer) GetMaxPatternLength() int {
	return sr.maxPatternLength
}

// ReplaceWithMapping performs string replacement and tracks character position mapping
// This is useful for maintaining position correspondence between original and modified text
func (sr *StringReplacer) ReplaceWithMapping(input string) (string, []int) {
	if len(sr.patterns) == 0 {
		// No replacements, create identity mapping
		mapping := make([]int, len(input)+1)
		for i := range mapping {
			mapping[i] = i
		}
		return input, mapping
	}

	inputRunes := []rune(input)
	var result strings.Builder
	result.Grow(len(input))

	// Character-level mapping: result char index -> input char index
	var charMapping []int

	i := 0
	for i < len(inputRunes) {
		char := inputRunes[i]

		patterns, exists := sr.patterns[char]
		if !exists {
			result.WriteRune(char)
			charMapping = append(charMapping, i)
			i++
			continue
		}

		matched := false
		for _, pattern := range patterns {
			if i+pattern.Length > len(inputRunes) {
				continue
			}

			if sr.matchesAt(inputRunes, i, pattern.Before) {
				// Found a match, replace it
				result.WriteString(pattern.After)

				// Add mapping entries for the replacement
				afterRunes := []rune(pattern.After)
				for j := range afterRunes {
					if j == 0 {
						// First character of replacement maps to start of original pattern
						charMapping = append(charMapping, i)
					} else {
						// Subsequent characters map to same position (expansion)
						charMapping = append(charMapping, i)
					}
				}

				i += pattern.Length
				matched = true
				break
			}
		}

		if !matched {
			result.WriteRune(char)
			charMapping = append(charMapping, i)
			i++
		}
	}

	// Add final sentinel
	charMapping = append(charMapping, len(inputRunes))

	return result.String(), charMapping
}
