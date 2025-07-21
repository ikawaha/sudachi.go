package input_text

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
)

// IgnoreYomiganaPlugin searches katakana in brackets after kanji characters as Yomigana (読み仮名)
// and removes it from tokenization target
// This is a faithful port of Rust Sudachi's IgnoreYomiganaPlugin
type IgnoreYomiganaPlugin struct {
	characterCategory *dic.CharacterCategory // Character category system
	leftBracketSet    map[rune]bool          // Set of left bracket characters
	rightBracketSet   map[rune]bool          // Set of right bracket characters
	maxYomiganaLength int                    // Maximum length of yomigana
	regex             *regexp.Regexp         // Compiled regex for matching yomigana patterns
}

// YomiganaPluginSettings represents configuration settings for IgnoreYomiganaPlugin
// This corresponds with raw config json file (matches Rust implementation)
type YomiganaPluginSettings struct {
	LeftBrackets      []string `json:"leftBrackets"`      // Matches Rust: leftBrackets: Vec<char>
	RightBrackets     []string `json:"rightBrackets"`     // Matches Rust: rightBrackets: Vec<char>
	MaxYomiganaLength int      `json:"maxYomiganaLength"` // Matches Rust: maxYomiganaLength: usize
}

// NewIgnoreYomiganaPlugin creates a new IgnoreYomiganaPlugin with default values
// This matches Rust: #[derive(Default)]
func NewIgnoreYomiganaPlugin() *IgnoreYomiganaPlugin {
	return &IgnoreYomiganaPlugin{
		characterCategory: nil,
		leftBracketSet:    map[rune]bool{},
		rightBracketSet:   map[rune]bool{},
		maxYomiganaLength: 0,
		regex:             nil,
	}
}

// GetName returns the plugin name for identification
func (p *IgnoreYomiganaPlugin) GetName() string {
	return "IgnoreYomiganaPlugin"
}

// SetUp initializes the plugin with configuration (implements plugin.InputTextPlugin)
// This matches Rust Sudachi's set_up method exactly
func (p *IgnoreYomiganaPlugin) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	// Extract character category from grammar (matching Rust: grammar.character_category.clone())
	if grammar != nil {
		p.characterCategory = grammar.CharacterCategory
	}

	if p.characterCategory == nil {
		return fmt.Errorf("character category is required for IgnoreYomiganaPlugin")
	}

	// Process settings if provided (matching Rust serde_json::from_value)
	if settings != nil {
		// Extract leftBrackets setting
		if leftBracketsInterface, ok := settings["leftBrackets"].([]any); ok {
			p.leftBracketSet = make(map[rune]bool)
			for _, v := range leftBracketsInterface {
				if s, ok := v.(string); ok {
					// Convert string to runes (each should be a single character)
					runes := []rune(s)
					if len(runes) == 1 {
						p.leftBracketSet[runes[0]] = true
					}
				}
			}
		}

		// Extract rightBrackets setting
		if rightBracketsInterface, ok := settings["rightBrackets"].([]any); ok {
			p.rightBracketSet = map[rune]bool{}
			for _, v := range rightBracketsInterface {
				if s, ok := v.(string); ok {
					// Convert string to runes (each should be a single character)
					runes := []rune(s)
					if len(runes) == 1 {
						p.rightBracketSet[runes[0]] = true
					}
				}
			}
		}

		// Extract maxYomiganaLength setting
		if maxLen, ok := settings["maxYomiganaLength"].(float64); ok {
			p.maxYomiganaLength = int(maxLen)
		} else if maxLen, ok := settings["maxYomiganaLength"].(int); ok {
			p.maxYomiganaLength = maxLen
		}
	}

	// Set default values if not configured
	if len(p.leftBracketSet) == 0 {
		// Default left brackets (matching common Japanese usage)
		defaultLeftBrackets := []rune{'（', '('}
		p.leftBracketSet = map[rune]bool{}
		for _, bracket := range defaultLeftBrackets {
			p.leftBracketSet[bracket] = true
		}
	}

	if len(p.rightBracketSet) == 0 {
		// Default right brackets (matching common Japanese usage)
		defaultRightBrackets := []rune{'）', ')'}
		p.rightBracketSet = map[rune]bool{}
		for _, bracket := range defaultRightBrackets {
			p.rightBracketSet[bracket] = true
		}
	}

	if p.maxYomiganaLength == 0 {
		p.maxYomiganaLength = 4 // Default maximum yomigana length (matching Rust)
	}

	// Build regex pattern (matching Rust make_regex method)
	regex, err := p.makeRegex()
	if err != nil {
		return fmt.Errorf("failed to build yomigana regex: %w", err)
	}
	p.regex = regex

	return nil
}

// makeRegex creates the regex pattern for matching kanji + yomigana pattern
// This matches Rust's make_regex method exactly
func (p *IgnoreYomiganaPlugin) makeRegex() (*regexp.Regexp, error) {
	// Build the pattern: {kanji}({leftBracket}{reading}{1,maxLength}{rightBracket})
	pattern := fmt.Sprintf(
		"%s(%s%s{1,%d}%s)",
		p.kanjiPattern(),
		p.anyOfPattern(p.leftBracketSet),
		p.readingPattern(),
		p.maxYomiganaLength,
		p.anyOfPattern(p.rightBracketSet),
	)

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern '%s': %w", pattern, err)
	}

	return regex, nil
}

// kanjiPattern builds regex pattern for kanji characters
// This matches Rust's kanji_pattern method
func (p *IgnoreYomiganaPlugin) kanjiPattern() string {
	return p.appendClass(dic.CategoryKanji)
}

// readingPattern builds regex pattern for hiragana and katakana characters
// This matches Rust's reading_pattern method
func (p *IgnoreYomiganaPlugin) readingPattern() string {
	return p.appendClass(dic.CategoryHiragana | dic.CategoryKatakana)
}

// appendClass builds character class for given category type
// This matches Rust's append_class method exactly
func (p *IgnoreYomiganaPlugin) appendClass(categoryType dic.CategoryType) string {
	var builder strings.Builder
	builder.WriteString("[")

	// Get character ranges from character category (matching Rust implementation)
	if p.characterCategory != nil {
		p.appendRangesForCategory(&builder, categoryType)
	} else {
		// Fallback to predefined ranges if character category is not available
		p.appendPredefinedRanges(&builder, categoryType)
	}

	builder.WriteString("]")
	return builder.String()
}

// appendRangesForCategory dynamically builds Unicode ranges from character category
// This matches Rust's character_category.iter() logic
func (p *IgnoreYomiganaPlugin) appendRangesForCategory(builder *strings.Builder, categoryType dic.CategoryType) {
	var currentRange *UnicodeRange
	hasAnyRange := false

	// Iterate through character category ranges (matching Rust implementation)
	for _, entry := range p.characterCategory.GetRanges() {
		if entry.Category.Intersects(categoryType) {
			newRange := &UnicodeRange{Start: entry.Range.Start, End: entry.Range.End}

			// Merge adjacent ranges (matching Rust logic)
			if currentRange != nil && currentRange.End == newRange.Start {
				currentRange.End = newRange.End
				continue
			}

			// Output previous range
			if currentRange != nil {
				p.appendUnicodeRange(builder, *currentRange)
				hasAnyRange = true
			}
			currentRange = newRange
		}
	}

	// Output final range
	if currentRange != nil {
		p.appendUnicodeRange(builder, *currentRange)
		hasAnyRange = true
	}

	// If no ranges found from character category, fallback to predefined ranges
	if !hasAnyRange {
		p.appendPredefinedRanges(builder, categoryType)
	}
}

// appendPredefinedRanges provides fallback ranges when character category is unavailable
func (p *IgnoreYomiganaPlugin) appendPredefinedRanges(builder *strings.Builder, categoryType dic.CategoryType) {
	switch categoryType {
	case dic.CategoryKanji:
		// Standard CJK ranges using direct character literals
		builder.WriteString("一-龯") // CJK Unified Ideographs main range
		builder.WriteString("豈-鶴") // CJK Compatibility Ideographs
	case dic.CategoryHiragana:
		// Hiragana character range
		builder.WriteString("ぁ-ゟ")
	case dic.CategoryKatakana:
		// Katakana character ranges
		builder.WriteString("ァ-ヿ")   // Katakana
		builder.WriteString("ㇰ-ㇿ") // Katakana Phonetic Extensions
	case dic.CategoryHiragana | dic.CategoryKatakana:
		// Combined hiragana and katakana ranges
		builder.WriteString("ぁ-ゟ")   // Hiragana
		builder.WriteString("ァ-ヿ")   // Katakana
		builder.WriteString("ㇰ-ㇿ") // Katakana Phonetic Extensions
	default:
		// Fallback to broader CJK range
		builder.WriteString("一-龯ぁ-ゟァ-ヿ")
	}
}

// appendUnicodeRange appends a Unicode range in Go regexp format
// This matches Rust's append_range method
func (p *IgnoreYomiganaPlugin) appendUnicodeRange(builder *strings.Builder, r UnicodeRange) {
	if r.End != 0 {
		if r.End-r.Start == 1 {
			// Single character - convert to actual Unicode character
			builder.WriteRune(rune(r.Start))
		} else {
			// Character range - convert to actual Unicode characters
			builder.WriteRune(rune(r.Start))
			builder.WriteString("-")
			builder.WriteRune(rune(r.End - 1))
		}
	}
}

// UnicodeRange represents a Unicode character range
type UnicodeRange struct {
	Start uint32
	End   uint32
}

// anyOfPattern creates character class from a set of characters
// This matches Rust's any_of_pattern method
func (p *IgnoreYomiganaPlugin) anyOfPattern(charSet map[rune]bool) string {
	var builder strings.Builder
	builder.WriteString("[")

	for char := range charSet {
		// For common characters, escape them if needed
		switch char {
		case '(', ')', '[', ']', '{', '}', '^', '$', '*', '+', '?', '.', '\\', '|':
			builder.WriteString("\\")
			builder.WriteRune(char)
		default:
			builder.WriteRune(char)
		}
	}

	builder.WriteString("]")
	return builder.String()
}

// Rewrite implements InputTextPlugin interface
// This matches Rust's rewrite_impl method exactly
func (p *IgnoreYomiganaPlugin) Rewrite(buffer *input.InputBuffer) error {
	if buffer.IsReadOnly() {
		return fmt.Errorf("buffer is read-only")
	}

	if p.regex == nil {
		return fmt.Errorf("plugin not properly initialized: regex is nil")
	}

	original := buffer.Original()
	modified := original

	// Find all matches and remove yomigana parts (matching Rust: for m in regex.captures_iter(data))
	matches := p.regex.FindAllStringSubmatchIndex(original, -1)

	// Process matches in reverse order to maintain correct indices
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		if len(match) >= 4 { // Ensure we have group 1 (the bracketed yomigana)
			// Group 0 is the entire match, Group 1 is the bracketed yomigana
			yomiganaStart, yomiganaEnd := match[2], match[3]

			// Remove the yomigana part (matching Rust: edit.replace_ref(grp.range(), ""))
			modified = modified[:yomiganaStart] + modified[yomiganaEnd:]
		}
	}

	// Apply modification if any changes were made
	if modified != original {
		return buffer.SetModified(modified)
	}

	return nil
}

// RewriteImpl performs yomigana removal (for compatibility with existing patterns)
func (p *IgnoreYomiganaPlugin) RewriteImpl(input string) (string, bool) {
	if p.regex == nil {
		return input, false
	}

	if input == "" {
		return input, false
	}

	// Apply regex replacement to remove yomigana
	result := p.regex.ReplaceAllStringFunc(input, func(match string) string {
		// Find the submatch (yomigana in brackets)
		submatches := p.regex.FindStringSubmatch(match)
		if len(submatches) >= 2 {
			// Return the kanji part (group 0 minus group 1)
			yomigana := submatches[1]
			return strings.Replace(match, yomigana, "", 1)
		}
		return match
	})

	changed := result != input
	return result, changed
}

// isKanji checks if a character is kanji (helper function)
func (p *IgnoreYomiganaPlugin) isKanji(r rune) bool {
	// Basic kanji detection using Unicode ranges
	return (r >= 0x4E00 && r <= 0x9FAF) || // CJK Unified Ideographs
		(r >= 0x3400 && r <= 0x4DBF) || // CJK Extension A
		(r >= 0xF900 && r <= 0xFAFF) // CJK Compatibility Ideographs
}

// isHiraganaOrKatakana checks if a character is hiragana or katakana (helper function)
func (p *IgnoreYomiganaPlugin) isHiraganaOrKatakana(r rune) bool {
	return (r >= 0x3040 && r <= 0x309F) || // Hiragana
		(r >= 0x30A0 && r <= 0x30FF) || // Katakana
		(r >= 0x31F0 && r <= 0x31FF) // Katakana Phonetic Extensions
}
