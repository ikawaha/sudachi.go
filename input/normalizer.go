package input

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ikawaha/sudachi.go/dic"
	"golang.org/x/text/unicode/norm"
)

// Normalizer provides text normalization functionality
type Normalizer struct {
	// Enable NFKC normalization
	enableNFKC bool
	// Enable case folding (lowercase conversion)
	enableCaseFold bool
	// Character replacement rules (single character -> string)
	replacements map[rune]string
	// String replacement rules (string -> string, for multi-character patterns)
	stringReplacements map[string]string
	// Characters to ignore during NFKC normalization
	ignoreNormalize map[rune]bool
	// String replacer for efficient multi-character replacement
	stringReplacer *StringReplacer
}

// NewNormalizer creates a new Normalizer with default settings
func NewNormalizer() *Normalizer {
	normalizer := &Normalizer{
		enableNFKC:         true,
		enableCaseFold:     true,
		replacements:       make(map[rune]string),
		stringReplacements: make(map[string]string),
		ignoreNormalize:    make(map[rune]bool),
	}

	return normalizer
}

// NewNormalizerWithOptions creates a normalizer with specific options
func NewNormalizerWithOptions(nfkc, caseFold bool) *Normalizer {
	normalizer := &Normalizer{
		enableNFKC:         nfkc,
		enableCaseFold:     caseFold,
		replacements:       make(map[rune]string),
		stringReplacements: make(map[string]string),
		ignoreNormalize:    make(map[rune]bool),
	}

	return normalizer
}

// NewNormalizerFromRewriteDef creates a normalizer from rewrite.def file
// This matches Rust Sudachi's DefaultInputTextPlugin initialization
func NewNormalizerFromRewriteDef(rewriteDefPath string) (*Normalizer, error) {
	// Parse rewrite.def file
	data, err := ParseRewriteDefFromFile(rewriteDefPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rewrite.def: %w", err)
	}

	return NewNormalizerFromRewriteDefData(data)
}

// NewNormalizerFromRewriteDefData creates a normalizer from parsed rewrite.def data
func NewNormalizerFromRewriteDefData(data *RewriteDefData) (*Normalizer, error) {
	// Validate data
	if err := ValidateRewriteDefData(data); err != nil {
		return nil, fmt.Errorf("invalid rewrite.def data: %w", err)
	}

	normalizer := &Normalizer{
		enableNFKC:         true,
		enableCaseFold:     true,
		replacements:       make(map[rune]string),
		stringReplacements: make(map[string]string),
		ignoreNormalize:    make(map[rune]bool),
	}

	// Load ignore normalize characters from rewrite.def
	for char := range data.IgnoreNormalizeChars {
		normalizer.ignoreNormalize[char] = true
	}

	// Load replace rules from rewrite.def
	for before, after := range data.ReplaceRules {
		if utf8.RuneCountInString(before) == 1 {
			// Single character replacement - use rune map for efficiency
			char, _ := utf8.DecodeRuneInString(before)
			normalizer.replacements[char] = after
		} else {
			// Multi-character replacement - use string map
			normalizer.stringReplacements[before] = after
		}
	}

	// Build string replacer if we have multi-character replacements
	if len(normalizer.stringReplacements) > 0 {
		normalizer.stringReplacer = NewStringReplacer(normalizer.stringReplacements)
	}

	return normalizer, nil
}

// NewDefaultSudachiNormalizer creates a normalizer with default Sudachi rewrite.def
func NewDefaultSudachiNormalizer() (*Normalizer, error) {
	data, err := GetDefaultRewriteDefData()
	if err != nil {
		return nil, fmt.Errorf("failed to get default rewrite.def: %w", err)
	}

	return NewNormalizerFromRewriteDefData(data)
}

// AddReplacement adds a character replacement rule
func (n *Normalizer) AddReplacement(from rune, to string) {
	n.replacements[from] = to
}

// AddReplacements adds multiple character replacement rules
func (n *Normalizer) AddReplacements(rules map[rune]string) {
	for from, to := range rules {
		n.replacements[from] = to
	}
}

// RemoveReplacement removes a character replacement rule
func (n *Normalizer) RemoveReplacement(from rune) {
	delete(n.replacements, from)
}

// ClearReplacements removes all replacement rules
func (n *Normalizer) ClearReplacements() {
	n.replacements = make(map[rune]string)
	n.stringReplacements = make(map[string]string)
	n.stringReplacer = nil
}

// AddStringReplacement adds a multi-character string replacement rule
func (n *Normalizer) AddStringReplacement(from, to string) {
	if n.stringReplacements == nil {
		n.stringReplacements = make(map[string]string)
	}
	n.stringReplacements[from] = to
	// Rebuild string replacer
	n.stringReplacer = NewStringReplacer(n.stringReplacements)
}

// AddStringReplacements adds multiple string replacement rules
func (n *Normalizer) AddStringReplacements(rules map[string]string) {
	if n.stringReplacements == nil {
		n.stringReplacements = make(map[string]string)
	}
	for from, to := range rules {
		n.stringReplacements[from] = to
	}
	// Rebuild string replacer
	n.stringReplacer = NewStringReplacer(n.stringReplacements)
}

// RemoveStringReplacement removes a string replacement rule
func (n *Normalizer) RemoveStringReplacement(from string) {
	if n.stringReplacements != nil {
		delete(n.stringReplacements, from)
		// Rebuild string replacer
		if len(n.stringReplacements) > 0 {
			n.stringReplacer = NewStringReplacer(n.stringReplacements)
		} else {
			n.stringReplacer = nil
		}
	}
}

// Normalize performs text normalization on the input text
// This matches Rust Sudachi's rewrite_impl function behavior
func (n *Normalizer) Normalize(input string) (string, bool) {
	if input == "" {
		return input, false
	}

	// Check what types of normalization are needed (matching Rust logic)
	needNFKC := n.enableNFKC && !norm.NFKC.IsNormalString(input)
	needLowercase := n.enableCaseFold && n.hasUppercase(input)

	// Choose processing path based on requirements (matching Rust if/else logic)
	if needNFKC || needLowercase {
		return n.replaceSlow(input)
	} else {
		return n.replaceFast(input)
	}
}

// hasUppercase checks if input contains uppercase characters (matches Rust chars.iter().any(|c| c.is_uppercase()))
func (n *Normalizer) hasUppercase(input string) bool {
	for _, r := range input {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// replaceFast implements fast path normalization (matches Rust replace_fast)
// Used when no NFKC or case folding is needed - only string replacements
// Note: Original Rust implementation uses Aho-Corasick for efficient multi-character replacements,
// but in Go we use a simpler string matching approach with a custom StringReplacer.
func (n *Normalizer) replaceFast(input string) (string, bool) {
	result := input
	changed := false

	// Apply ALL replacements in one pass to match Rust behavior
	// Rust uses AhoCorasick automaton to find and replace all patterns efficiently
	result = n.applyAllReplacements(result)

	if result != input {
		changed = true
	}

	return result, changed
}

// applyAllReplacements applies both string and character replacements in optimal order
// This matches Rust's AhoCorasick-based replacement strategy
func (n *Normalizer) applyAllReplacements(input string) string {
	result := input

	// First apply multi-character string replacements (higher priority, longer matches)
	if n.stringReplacer != nil && n.stringReplacer.HasReplacements() {
		result = n.stringReplacer.Replace(result)
	}

	// Then apply single character replacements to any remaining characters
	if len(n.replacements) > 0 {
		result = n.applyReplacements(result)
	}

	return result
}

// replaceSlow implements slow path normalization (matches Rust replace_slow)
// Used when NFKC or case folding is needed - processes character by character
func (n *Normalizer) replaceSlow(input string) (string, bool) {
	result := input
	changed := false

	// Apply ALL replacements first (same as fast path)
	newResult := n.applyAllReplacements(result)
	if newResult != result {
		result = newResult
		changed = true
	}

	// Apply case folding (matching Rust to_lowercase logic)
	// But preserve ignore normalize characters
	if n.enableCaseFold {
		newResult = n.applySelectiveCaseFolding(result)
		if newResult != result {
			result = newResult
			changed = true
		}
	}

	// Apply selective NFKC normalization (matching Rust NFKC with ignored characters)
	if n.enableNFKC {
		newResult = n.applySelectiveNFKC(result)
		if newResult != result {
			result = newResult
			changed = true
		}
	}

	return result, changed
}

// applyReplacements applies character replacement rules
func (n *Normalizer) applyReplacements(input string) string {
	if len(n.replacements) == 0 {
		return input
	}

	var builder strings.Builder
	builder.Grow(len(input)) // Pre-allocate space

	for _, r := range input {
		if replacement, exists := n.replacements[r]; exists {
			builder.WriteString(replacement)
		} else {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

// CharacterEdit represents a single character edit operation
type CharacterEdit struct {
	OriginalPos int    // Position in original text (character index)
	OriginalLen int    // Length in original text (character count)
	ModifiedPos int    // Position in modified text (character index)
	ModifiedLen int    // Length in modified text (character count)
	Original    string // Original text
	Modified    string // Modified text
}

// NormalizationInfo contains information about what normalization was applied
type NormalizationInfo struct {
	// Whether any normalization was applied
	Applied bool
	// Whether NFKC normalization was applied
	NFKCApplied bool
	// Whether case folding was applied
	CaseFoldApplied bool
	// Number of character replacements applied
	ReplacementsApplied int
	// Detailed character edits for mapping construction
	CharacterEdits []CharacterEdit
}

// NormalizeWithInfo performs normalization and returns detailed information
func (n *Normalizer) NormalizeWithInfo(input string) (string, *NormalizationInfo) {
	if input == "" {
		return input, &NormalizationInfo{}
	}

	info := &NormalizationInfo{
		CharacterEdits: make([]CharacterEdit, 0),
	}
	result := input

	// Apply character replacements first
	if len(n.replacements) > 0 {
		newResult, edits := n.applyReplacementsWithEdits(result)
		if len(edits) > 0 {
			result = newResult
			info.Applied = true
			info.ReplacementsApplied = len(edits)
			info.CharacterEdits = append(info.CharacterEdits, edits...)
		}
	}

	// Apply case folding
	if n.enableCaseFold {
		newResult, edits := n.applyCaseFoldingWithEdits(result)
		if len(edits) > 0 {
			result = newResult
			info.Applied = true
			info.CaseFoldApplied = true
			info.CharacterEdits = append(info.CharacterEdits, edits...)
		}
	}

	// Apply NFKC normalization with selective protection
	if n.enableNFKC {
		newResult, edits := n.applySelectiveNFKCWithEdits(result)
		if len(edits) > 0 {
			result = newResult
			info.Applied = true
			info.NFKCApplied = true
			info.CharacterEdits = append(info.CharacterEdits, edits...)
		}
	}

	return result, info
}

// applyReplacementsWithCount applies replacements and counts them
func (n *Normalizer) applyReplacementsWithCount(input string) (string, int) {
	if len(n.replacements) == 0 {
		return input, 0
	}

	var builder strings.Builder
	builder.Grow(len(input))
	count := 0

	for _, r := range input {
		if replacement, exists := n.replacements[r]; exists {
			builder.WriteString(replacement)
			count++
		} else {
			builder.WriteRune(r)
		}
	}

	return builder.String(), count
}

// applyReplacementsWithEdits applies character replacements and tracks detailed edits
func (n *Normalizer) applyReplacementsWithEdits(input string) (string, []CharacterEdit) {
	if len(n.replacements) == 0 {
		return input, nil
	}

	var builder strings.Builder
	var edits []CharacterEdit
	builder.Grow(len(input))

	originalPos := 0
	modifiedPos := 0

	for _, r := range input {
		originalLen := utf8.RuneLen(r)
		if replacement, exists := n.replacements[r]; exists {
			// Character is replaced
			builder.WriteString(replacement)
			modifiedLen := len(replacement)

			edit := CharacterEdit{
				OriginalPos: originalPos,
				OriginalLen: originalLen,
				ModifiedPos: modifiedPos,
				ModifiedLen: modifiedLen,
				Original:    string(r),
				Modified:    replacement,
			}
			edits = append(edits, edit)
			modifiedPos += modifiedLen
		} else {
			// Character unchanged
			builder.WriteRune(r)
			modifiedPos += originalLen
		}
		originalPos += originalLen
	}

	return builder.String(), edits
}

// applyCaseFoldingWithEdits applies case folding and tracks detailed edits
func (n *Normalizer) applyCaseFoldingWithEdits(input string) (string, []CharacterEdit) {
	if !n.enableCaseFold {
		return input, nil
	}

	var builder strings.Builder
	var edits []CharacterEdit
	builder.Grow(len(input))

	originalPos := 0
	modifiedPos := 0

	for _, r := range input {
		originalLen := utf8.RuneLen(r)
		lowerR := unicode.ToLower(r)

		if r != lowerR {
			// Character is changed by case folding
			lowerStr := string(lowerR)
			builder.WriteString(lowerStr)
			modifiedLen := len(lowerStr)

			edit := CharacterEdit{
				OriginalPos: originalPos,
				OriginalLen: originalLen,
				ModifiedPos: modifiedPos,
				ModifiedLen: modifiedLen,
				Original:    string(r),
				Modified:    lowerStr,
			}
			edits = append(edits, edit)
			modifiedPos += modifiedLen
		} else {
			// Character unchanged
			builder.WriteRune(r)
			modifiedPos += originalLen
		}
		originalPos += originalLen
	}

	return builder.String(), edits
}

// applySelectiveNFKCWithEdits applies NFKC normalization and tracks detailed edits
func (n *Normalizer) applySelectiveNFKCWithEdits(input string) (string, []CharacterEdit) {
	if !n.enableNFKC {
		return input, nil
	}

	// For simplicity, apply NFKC to the entire string and track as one edit if changed
	normalized := n.applySelectiveNFKC(input)

	if normalized != input {
		edit := CharacterEdit{
			OriginalPos: 0,
			OriginalLen: len(input),
			ModifiedPos: 0,
			ModifiedLen: len(normalized),
			Original:    input,
			Modified:    normalized,
		}
		return normalized, []CharacterEdit{edit}
	}

	return input, nil
}

// CreateNormalizedInputBuffer creates an InputBuffer with text normalization applied
func (n *Normalizer) CreateNormalizedInputBuffer(original string, grammar *dic.Grammar) (*InputBuffer, *NormalizationInfo, error) {
	buffer := NewInputBuffer()

	// Normalize the text
	normalized, info := n.NormalizeWithInfo(original)

	// Start building the buffer
	err := buffer.StartBuild(original)
	if err != nil {
		return nil, nil, err
	}

	// Set the normalized text
	if info.Applied {
		buffer.modified = normalized
	}

	// Build the buffer
	err = buffer.Build(grammar)
	if err != nil {
		return nil, nil, err
	}

	return buffer, info, nil
}

// applySelectiveNFKC applies NFKC normalization while preserving ignored characters
func (n *Normalizer) applySelectiveNFKC(input string) string {
	if len(n.ignoreNormalize) == 0 {
		// No characters to protect, use standard NFKC
		return norm.NFKC.String(input)
	}

	// Process character by character to protect ignored characters
	var builder strings.Builder
	builder.Grow(len(input))

	inputRunes := []rune(input)
	i := 0

	for i < len(inputRunes) {
		r := inputRunes[i]

		if n.ignoreNormalize[r] {
			// Protected character - write as-is
			builder.WriteRune(r)
			i++
		} else {
			// Find the next protected character or end of string
			segmentStart := i
			for i < len(inputRunes) && !n.ignoreNormalize[inputRunes[i]] {
				i++
			}

			// Normalize the segment between protected characters
			segment := string(inputRunes[segmentStart:i])
			normalized := norm.NFKC.String(segment)
			builder.WriteString(normalized)
		}
	}

	return builder.String()
}

// applySelectiveCaseFolding applies case folding while preserving ignored characters
func (n *Normalizer) applySelectiveCaseFolding(input string) string {
	if len(n.ignoreNormalize) == 0 {
		// No characters to protect, use standard case folding
		return strings.ToLower(input)
	}

	// Process character by character to protect ignored characters
	var builder strings.Builder
	builder.Grow(len(input))

	for _, r := range input {
		if n.ignoreNormalize[r] {
			// Protected character - write as-is
			builder.WriteRune(r)
		} else {
			// Apply case folding to unprotected character
			builder.WriteRune(unicode.ToLower(r))
		}
	}

	return builder.String()
}

// AddIgnoreNormalizeChar adds a character to the ignore normalization list
func (n *Normalizer) AddIgnoreNormalizeChar(r rune) {
	n.ignoreNormalize[r] = true
}

// RemoveIgnoreNormalizeChar removes a character from the ignore normalization list
func (n *Normalizer) RemoveIgnoreNormalizeChar(r rune) {
	delete(n.ignoreNormalize, r)
}
