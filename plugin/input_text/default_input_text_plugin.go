package input_text

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/plugin"
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
	// Debug flag for conditional output
	debug bool
}

// NewDefaultInputTextPlugin creates a new DefaultInputTextPlugin
func NewDefaultInputTextPlugin() *DefaultInputTextPlugin {
	return &DefaultInputTextPlugin{
		ignoreNormalizeSet: map[rune]bool{},
		keyLengths:         map[rune]int{},
		replaceCharMap:     map[string]string{},
		replacements:       []string{},
		debug:              false,
	}
}

// SetDebug sets the debug flag for the plugin
func (p *DefaultInputTextPlugin) SetDebug(debug bool) {
	p.debug = debug
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
// Note: Roman numerals are not detected by unicode.IsUpper, so we check them explicitly
func (p *DefaultInputTextPlugin) hasUppercase(input string) bool {
	for _, r := range input {
		if unicode.IsUpper(r) || p.isUpperRomanNumeral(r) {
			return true
		}
	}
	return false
}

// isUpperRomanNumeral checks if a character is an uppercase roman numeral
// that should be converted to lowercase before NFKC normalization
func (p *DefaultInputTextPlugin) isUpperRomanNumeral(ch rune) bool {
	// Unicode range for uppercase roman numerals (U+2160-U+216F)
	return ch >= 'Ⅰ' && ch <= 'Ⅿ'
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

// replaceSlow implements slow path normalization (faithful port of Rust replace_slow)
// Used when NFKC or case folding is needed - processes character by character like Rust version
func (p *DefaultInputTextPlugin) replaceSlow(input string) (string, bool) {
	var builder strings.Builder
	builder.Grow(len(input))
	changed := false
	runes := []rune(input)

	for i := 0; i < len(runes); {
		ch := runes[i]

		// 1. Check for character replacement as defined by rewrite.def (equivalent to AhoCorasick)
		// Look for multi-character replacements starting at current position
		replaced := false
		if maxLen, exists := p.keyLengths[ch]; exists {
			// Check from longest possible match to single character
			for length := maxLen; length >= 1 && i+length <= len(runes); length-- {
				candidate := string(runes[i : i+length])
				if replacement, found := p.replaceCharMap[candidate]; found {
					builder.WriteString(replacement)
					i += length
					changed = true
					replaced = true
					break
				}
			}
		}

		if replaced {
			continue
		}

		// 2. Handle normalization for individual character (matching Rust logic)
		// Note: Roman numerals are not detected by unicode.IsUpper, so we check them explicitly
		needLowercase := unicode.IsUpper(ch) || p.isUpperRomanNumeral(ch)
		needNFKC := !p.shouldIgnore(ch) && !norm.NFKC.IsNormalString(string(ch))

		// Match Rust's (need_lowercase, need_nfkc) pattern matching
		switch {
		case !needLowercase && !needNFKC:
			// (false, false) - no need to do anything
			builder.WriteRune(ch)
		case needLowercase && !needNFKC:
			// (true, false) - only lowercasing
			lowered := unicode.ToLower(ch)
			builder.WriteRune(lowered)
			if lowered != ch {
				changed = true
			}
		case !needLowercase && needNFKC:
			// (false, true) - only normalization
			normalized := norm.NFKC.String(string(ch))
			builder.WriteString(normalized)
			if normalized != string(ch) {
				changed = true
			}
		case needLowercase && needNFKC:
			// (true, true) - both lowercasing and normalization
			lowered := unicode.ToLower(ch)
			normalized := norm.NFKC.String(string(lowered))
			builder.WriteString(normalized)
			if normalized != string(ch) {
				changed = true
			}
		}

		i++
	}

	return builder.String(), changed
}

// Note: applySelectiveNFKC and applySelectiveCaseFolding methods have been removed
// as they are no longer needed. The new replaceSlow method handles normalization
// character-by-character as per the Rust implementation.

// GetName returns the plugin name for identification (implements plugin.InputTextPlugin)
func (p *DefaultInputTextPlugin) GetName() string {
	return "DefaultInputTextPlugin"
}

// Rewrite implements InputTextPlugin interface
func (p *DefaultInputTextPlugin) Rewrite(buffer *input.InputBuffer) error {
	// Get current state of buffer (which includes changes from previous plugins)
	current := buffer.Modified()

	if buffer.IsReadOnly() {
		return fmt.Errorf("buffer is read-only")
	}

	// Apply character-by-character normalization with proper mapping tracking
	return buffer.WithEditor(func(buf *input.InputBuffer, editor *input.InputEditor) error {
		return p.applyNormalizationWithEditor(current, editor)
	})
}

// applyNormalizationWithEditor applies normalization character-by-character using editor
// This ensures proper m2o mapping is maintained
func (p *DefaultInputTextPlugin) applyNormalizationWithEditor(text string, editor *input.InputEditor) error {
	// Check what types of normalization are needed (matching original RewriteImpl logic)
	needNFKC := !norm.NFKC.IsNormalString(text)
	needLowercase := p.hasUppercase(text)

	if !needNFKC && !needLowercase && (p.stringReplacer == nil || !p.stringReplacer.HasReplacements()) {
		// No normalization needed
		return nil
	}

	// Process text character by character to identify individual changes
	runes := []rune(text)
	bytePos := 0

	for _, ch := range runes {
		charStr := string(ch)
		charBytes := []byte(charStr)
		charStart := bytePos
		charEnd := bytePos + len(charBytes)

		// Check if this character needs replacement
		var replacement string
		needsChange := false

		// 1. First check direct character replacement map
		if replaced, found := p.replaceCharMap[charStr]; found {
			replacement = replaced
			needsChange = true
		}

		// 2. If no direct replacement, check string replacer for multi-character patterns
		if !needsChange && p.stringReplacer != nil {
			replaced := p.stringReplacer.Replace(charStr)
			if replaced != charStr {
				replacement = replaced
				needsChange = true
			}
		}

		// 3. Apply case folding if needed (includes Roman numerals)
		if !needsChange && needLowercase && (unicode.IsUpper(ch) || p.isUpperRomanNumeral(ch)) {
			lowered := unicode.ToLower(ch)
			replacement = string(lowered)
			needsChange = true
		}

		// 5. Apply NFKC normalization if needed and not in ignore set
		if !needsChange && needNFKC && !p.ignoreNormalizeSet[ch] {
			normalized := norm.NFKC.String(charStr)
			if normalized != charStr {
				replacement = normalized
				needsChange = true
			}
		}

		// Apply replacement if needed
		if needsChange {
			editor.ReplaceRange(input.Range{Start: charStart, End: charEnd}, replacement)
		}

		bytePos = charEnd
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

// CreateInputTextPlugin creates a DefaultInputTextPlugin instance
func (p *DefaultInputTextPlugin) CreateInputTextPlugin(settings map[string]any, resourceDir string, grammar *dic.Grammar) (plugin.InputTextPlugin, error) {
	defaultPlugin := NewDefaultInputTextPlugin()

	// Set up the plugin with configuration
	err := defaultPlugin.SetUp(settings, resourceDir, grammar)
	if err != nil {
		return nil, fmt.Errorf("failed to set up DefaultInputText plugin: %w", err)
	}

	return defaultPlugin, nil
}

// CreateOOVProvider creates an OOV provider plugin (not supported by DefaultInputText plugin)
func (p *DefaultInputTextPlugin) CreateOOVProvider(settings map[string]any, resourceDir string, grammar *dic.Grammar) (plugin.OOVProviderPlugin, error) {
	return nil, fmt.Errorf("DefaultInputText plugin does not support OOV provider plugins")
}

// CreatePathRewriter creates a path rewrite plugin (not supported by DefaultInputText plugin)
func (p *DefaultInputTextPlugin) CreatePathRewriter(settings map[string]any, resourceDir string, grammar *dic.Grammar) (plugin.PathRewritePlugin, error) {
	return nil, fmt.Errorf("DefaultInputText plugin does not support path rewrite plugins")
}

// GetSupportedTypes returns the plugin types this factory supports
func (p *DefaultInputTextPlugin) GetSupportedTypes() []plugin.PluginType {
	return []plugin.PluginType{plugin.PluginTypeInputText}
}
