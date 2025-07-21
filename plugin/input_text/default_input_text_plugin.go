package input_text

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
	"golang.org/x/text/unicode/norm"
)

// DefaultInputTextPlugin provides basic normalization of the input text
// This is a faithful port of Rust Sudachi's DefaultInputTextPlugin
type DefaultInputTextPlugin struct {
	// Set of characters to skip normalization
	ignoreNormalizeSet map[rune]bool
	// Mapping from a character to the maximum char_length of possible replacement
	keyLengths map[rune]int
	// Replacement mapping
	replaceCharMap map[string]string
	// String replacer for efficient multi-character replacement (equivalent to AhoCorasick)
	stringReplacer *input.StringReplacer
	// Replacement values (corresponding to keys in replaceCharMap)
	replacements []string
}

// NewDefaultInputTextPlugin creates a new DefaultInputTextPlugin
func NewDefaultInputTextPlugin() *DefaultInputTextPlugin {
	return &DefaultInputTextPlugin{
		ignoreNormalizeSet: map[rune]bool{},
		keyLengths:         map[rune]int{},
		replaceCharMap:     map[string]string{},
		replacements:       []string{},
	}
}

// SetUp initializes the plugin with configuration (implements plugin.InputTextPlugin)
// This matches Rust Sudachi's set_up method
func (p *DefaultInputTextPlugin) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	// Load embedded rewrite.def data (equivalent to include_bytes!)
	data, err := input.GetDefaultRewriteDefData()
	if err != nil {
		return fmt.Errorf("failed to load rewrite.def: %w", err)
	}

	return p.readRewriteLists(data)
}

// SetUpFromData initializes the plugin with rewrite.def data (legacy method for tests)
func (p *DefaultInputTextPlugin) SetUpFromData() error {
	return p.SetUp(nil, "", nil)
}

// readRewriteLists loads rewrite definition data
// This is a faithful port of Rust's read_rewrite_lists method
func (p *DefaultInputTextPlugin) readRewriteLists(data *input.RewriteDefData) error {
	ignoreNormalizeSet := make(map[rune]bool)
	keyLengths := make(map[rune]int)
	replaceCharMap := make(map[string]string)

	// Load ignore normalize characters from rewrite.def
	for char := range data.IgnoreNormalizeChars {
		ignoreNormalizeSet[char] = true
	}

	// Load replace rules from rewrite.def
	for before, after := range data.ReplaceRules {
		if _, exists := replaceCharMap[before]; exists {
			return fmt.Errorf("'%s' is already defined", before)
		}

		firstChar := []rune(before)[0]
		nChar := len([]rune(before))
		if keyLengths[firstChar] < nChar {
			keyLengths[firstChar] = nChar
		}
		replaceCharMap[before] = after
	}

	p.ignoreNormalizeSet = ignoreNormalizeSet
	p.keyLengths = keyLengths
	p.replaceCharMap = replaceCharMap

	// Build a string replacer and replacements array (equivalent to AhoCorasick setup)
	var keys []string
	var values []string

	for k, v := range p.replaceCharMap {
		keys = append(keys, k)
		values = append(values, v)
	}

	// Create string replacer (equivalent to AhoCorasick DFA)
	p.stringReplacer = input.NewStringReplacer(p.replaceCharMap)
	p.replacements = values

	return nil
}

// shouldIgnore checks if a character should be ignored during normalization
// This matches Rust's should_ignore method
func (p *DefaultInputTextPlugin) shouldIgnore(ch rune) bool {
	return p.ignoreNormalizeSet[ch]
}

// RewriteImpl performs text normalization and replacement
// This is a faithful port of Rust's rewrite_impl method
func (p *DefaultInputTextPlugin) RewriteImpl(input string) (string, bool) {
	if input == "" {
		return input, false
	}

	// Check what types of normalization are needed (matching Rust logic)
	needNFKC := !norm.NFKC.IsNormalString(input)
	needLowercase := p.hasUppercase(input)

	// Choose processing path based on requirements (matching Rust if/else logic)
	if needNFKC || needLowercase {
		return p.replaceSlow(input)
	} else {
		return p.replaceFast(input)
	}
}

// hasUppercase checks if input contains uppercase characters
// This matches Rust's chars.iter().any(|c| c.is_uppercase())
func (p *DefaultInputTextPlugin) hasUppercase(input string) bool {
	for _, r := range input {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// replaceFast implements fast path normalization (matches Rust replace_fast)
// Used when no NFKC or case folding is needed - only string replacements
func (p *DefaultInputTextPlugin) replaceFast(input string) (string, bool) {
	if p.stringReplacer == nil || !p.stringReplacer.HasReplacements() {
		return input, false
	}

	// Use string replacer to find all replacements and replace them
	// This is equivalent to Rust's AhoCorasick automaton usage
	result := p.stringReplacer.Replace(input)

	changed := result != input
	return result, changed
}

// replaceSlow implements slow path normalization (matches Rust replace_slow)
// Used when NFKC or case folding is needed - processes character by character
func (p *DefaultInputTextPlugin) replaceSlow(input string) (string, bool) {
	result := input
	changed := false

	// First apply string replacements (same as fast path)
	if p.stringReplacer != nil && p.stringReplacer.HasReplacements() {
		newResult := p.stringReplacer.Replace(result)
		if newResult != result {
			result = newResult
			changed = true
		}
	}

	// Apply case folding while preserving ignore normalize characters
	newResult := p.applySelectiveCaseFolding(result)
	if newResult != result {
		result = newResult
		changed = true
	}

	// Apply selective NFKC normalization
	newResult = p.applySelectiveNFKC(result)
	if newResult != result {
		result = newResult
		changed = true
	}

	return result, changed
}

// applySelectiveNFKC applies NFKC normalization while preserving ignored characters
// This matches Rust's NFKC handling in replace_slow
func (p *DefaultInputTextPlugin) applySelectiveNFKC(input string) string {
	if len(p.ignoreNormalizeSet) == 0 {
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

		if p.shouldIgnore(r) {
			// Protected character - write as-is
			builder.WriteRune(r)
			i++
		} else {
			// Find the next protected character or end of string
			segmentStart := i
			for i < len(inputRunes) && !p.shouldIgnore(inputRunes[i]) {
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
// This matches Rust's to_lowercase handling in replace_slow
func (p *DefaultInputTextPlugin) applySelectiveCaseFolding(input string) string {
	if len(p.ignoreNormalizeSet) == 0 {
		// No characters to protect, use standard case folding
		return strings.ToLower(input)
	}

	// Process character by character to protect ignored characters
	var builder strings.Builder
	builder.Grow(len(input))

	for _, r := range input {
		if p.shouldIgnore(r) {
			// Protected character - write as-is
			builder.WriteRune(r)
		} else {
			// Apply case folding to unprotected character
			builder.WriteRune(unicode.ToLower(r))
		}
	}

	return builder.String()
}

// GetName returns the plugin name for identification (implements plugin.InputTextPlugin)
func (p *DefaultInputTextPlugin) GetName() string {
	return "DefaultInputTextPlugin"
}

// Rewrite implements InputTextPlugin interface
func (p *DefaultInputTextPlugin) Rewrite(buffer *input.InputBuffer) error {
	if buffer.IsReadOnly() {
		return fmt.Errorf("buffer is read-only")
	}

	original := buffer.Original()
	normalized, changed := p.RewriteImpl(original)

	if changed {
		return buffer.SetModified(normalized)
	}

	return nil
}

// CreateNormalizedInputBuffer creates an InputBuffer with normalization applied
// This integrates the plugin with InputBuffer processing
func (p *DefaultInputTextPlugin) CreateNormalizedInputBuffer(original string, grammar *dic.Grammar) (*input.InputBuffer, error) {
	buffer := input.NewInputBuffer()

	// Start building the buffer
	err := buffer.StartBuild(original)
	if err != nil {
		return nil, err
	}

	// Apply normalization
	normalized, changed := p.RewriteImpl(original)
	if changed {
		err = buffer.SetModified(normalized)
		if err != nil {
			return nil, err
		}
	}

	// Build the buffer
	err = buffer.Build(grammar)
	if err != nil {
		return nil, err
	}

	return buffer, nil
}
