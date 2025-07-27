package sentence

import (
	"regexp"
	"strings"

	"github.com/ikawaha/sudachi.go/dic"
)

// Constants matching Rust version exactly
const (
	periods          = "。？！♪…\\?\\!"
	dot              = "\\.．"
	cdots            = "・{3,}"
	comma            = ",，、"
	brTag            = "(<br>|<BR>){2,}"
	alphabetOrNumber = "a-zA-Z0-9ａ-ｚＡ-Ｚ０-９〇一二三四五六七八九十百千万億兆"
	openParenthesis  = `\(\{｛\[（「【『［≪〔""`
	closeParenthesis = `\)\}\]）」｝】』］〕≫""`
	defaultLimit     = 4096
)

// NonBreakChecker checks for words that cross boundaries (matching Rust NonBreakChecker)
type NonBreakChecker struct {
	lexicon *dic.LexiconSet
	bos     int
}

// NewNonBreakChecker creates a new NonBreakChecker (matching Rust NonBreakChecker::new)
func NewNonBreakChecker(lexicon *dic.LexiconSet) *NonBreakChecker {
	return &NonBreakChecker{
		lexicon: lexicon,
		bos:     0,
	}
}

// HasNonBreakWord returns whether there is a word that crosses the boundary (matching Rust implementation)
func (nbc *NonBreakChecker) HasNonBreakWord(input string, length int) bool {
	// Matching Rust: assume that SentenceDetector::get_eos called with self.input[self.bos..]
	eosBytes := nbc.bos + length
	inputBytes := []byte(input)
	const lookupByteLength = 10 * 3 // 10 Japanese characters in UTF-8

	lookupStart := eosBytes - lookupByteLength
	if lookupStart < lookupByteLength {
		lookupStart = 0
	}

	for i := lookupStart; i < eosBytes; i++ {
		// This would need lexicon.lookup implementation - for now, return false
		// TODO: Implement lexicon lookup when LexiconSet.Lookup is available
		_ = inputBytes
		// entries := nbc.lexicon.Lookup(inputBytes, i)
		// for _, entry := range entries {
		//     endByte := entry.End
		//     if endByte > eosBytes {
		//         return true // end is after boundary candidate, this boundary is bad
		//     }
		//     if endByte == eosBytes {
		//         // check that there are more than one character in the matched word
		//         chars := []rune(input[i:])
		//         if len(chars) > 1 {
		//             return true
		//         }
		//     }
		// }
	}
	return false
}

// SentenceDetector represents a sentence boundary detector (matching Rust SentenceDetector)
type SentenceDetector struct {
	limit int // The maximum number of characters processed at once
}

// NewSentenceDetector creates a new SentenceDetector (matching Rust SentenceDetector::new)
func NewSentenceDetector() *SentenceDetector {
	return &SentenceDetector{
		limit: defaultLimit,
	}
}

// NewSentenceDetectorWithLimit creates a new SentenceDetector with limit (matching Rust SentenceDetector::with_limit)
func NewSentenceDetectorWithLimit(limit int) *SentenceDetector {
	return &SentenceDetector{
		limit: limit,
	}
}

// GetEOS returns the byte index of the detected end of the sentence (matching Rust get_eos exactly)
// Returns negative value if no sentence boundary is found within the limit
func (sd *SentenceDetector) GetEOS(input string, checker *NonBreakChecker) (int, error) {
	if len(input) == 0 {
		return 0, nil
	}

	// Handle at most sd.limit chars at once (matching Rust implementation)
	runes := []rune(input)
	var s string
	var inputExceedsLimit bool

	if len(runes) > sd.limit {
		s = string(runes[:sd.limit])
		inputExceedsLimit = true
	} else {
		s = input
		inputExceedsLimit = false
	}

	// Create sentence breaker regex (2-stage approach: basic matching + condition checking)
	// Go's regexp package doesn't support negative lookbehind/lookahead, so we check conditions separately
	pattern := "[" + periods + "]|" + cdots + "|[" + dot + "]|" + brTag
	sentenceBreaker, err := regexp.Compile(pattern)
	if err != nil {
		return 0, err
	}

	// Create itemize header regex (simplified)
	itemizePattern := "^[" + alphabetOrNumber + "][" + dot + "]$"
	itemizeHeader, err := regexp.Compile(itemizePattern)
	if err != nil {
		return 0, err
	}

	// Find sentence boundaries (2-stage approach matching Rust implementation)
	matches := sentenceBreaker.FindAllStringIndex(s, -1)

	// Track processed positions to avoid duplicate boundaries from consecutive punctuation
	processedPositions := make(map[int]bool)

	for _, match := range matches {
		matchText := s[match[0]:match[1]]

		// Stage 2: Check conditions for period matches (matching Rust negative lookbehind/lookahead)
		if matchText == "." || matchText == "．" {
			runePos := len([]rune(s[:match[0]]))
			if !isValidPeriodBoundary(s, runePos) {
				continue // Skip this match if it doesn't meet the conditions
			}
		}

		// Extend match to include consecutive punctuation marks (matching Rust [periods dot]* behavior)
		eos := match[1] // match.end() in Rust
		eos = extendThroughConsecutivePunctuation(s, eos)

		// Skip if we've already processed a boundary that includes this position
		if processedPositions[eos] {
			continue
		}
		processedPositions[eos] = true

		// Check parenthesis level (matching Rust parenthesis_level check)
		if parenthesisLevel(s[:eos]) > 0 {
			continue
		}

		// Add prohibited BOS offset (matching Rust prohibited_bos check)
		if eos < len(s) {
			eos += prohibitedBOS(s[eos:])
		}

		// Check itemize header (matching Rust ITEMIZE_HEADER check)
		if itemizeHeader.MatchString(s) {
			continue
		}

		// Check for non-break words if checker is provided (matching Rust implementation)
		if checker != nil && checker.HasNonBreakWord(input, eos) {
			continue
		}

		// Return the found EOS position in bytes (matching Rust return value)
		return len([]byte(s[:eos])), nil
	}

	// No sentence boundary found (matching Rust behavior)
	if inputExceedsLimit {
		return -len(s), nil
	}
	return -len(input), nil
}

// parenthesisLevel calculates the parenthesis nesting level (matching Rust parenthesis_level)
func parenthesisLevel(s string) int {
	level := 0
	for _, r := range s {
		char := string(r)
		if strings.ContainsAny(char, openParenthesis) {
			level++
		} else if strings.ContainsAny(char, closeParenthesis) {
			level--
		}
	}
	if level < 0 {
		return 0
	}
	return level
}

// prohibitedBOS calculates offset for prohibited beginning of sentence (matching Rust prohibited_bos)
func prohibitedBOS(s string) int {
	prohibitedChars := `）」｝】』］〕≫""って` + comma
	for i, r := range s {
		char := string(r)
		if !strings.ContainsAny(char, prohibitedChars) {
			return i
		}
	}
	return len(s)
}

// isAlphaNumeric checks if a character is alphanumeric (matching Rust alphabetOrNumber exactly)
func isAlphaNumeric(r rune) bool {
	// ASCII alphanumeric
	if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
		return true
	}
	// Full-width alphanumeric
	if (r >= 'ａ' && r <= 'ｚ') || (r >= 'Ａ' && r <= 'Ｚ') || (r >= '０' && r <= '９') {
		return true
	}
	// Japanese numbers
	japaneseNumbers := "〇一二三四五六七八九十百千万億兆"
	return strings.ContainsRune(japaneseNumbers, r)
}

// isComma checks if a character is a comma (matching Rust comma exactly)
func isComma(r rune) bool {
	return r == ',' || r == '，' || r == '、'
}

// isAlphaNumericOrComma checks if a character is alphanumeric or comma
func isAlphaNumericOrComma(r rune) bool {
	return isAlphaNumeric(r) || isComma(r)
}

// isValidPeriodBoundary checks if a period can be a sentence boundary (matching Rust negative lookbehind/lookahead)
func isValidPeriodBoundary(text string, pos int) bool {
	runes := []rune(text)

	// Negative lookbehind: check if there's no alphanumeric character before
	if pos > 0 && isAlphaNumeric(runes[pos-1]) {
		return false
	}

	// Negative lookahead: check if there's no alphanumeric character or comma after
	if pos+1 < len(runes) && isAlphaNumericOrComma(runes[pos+1]) {
		return false
	}

	return true
}

// extendThroughConsecutivePunctuation extends the position through consecutive punctuation marks
// This matches Rust's [periods dot]* behavior in the sentence breaker regex
func extendThroughConsecutivePunctuation(s string, pos int) int {
	runes := []rune(s)
	runePos := len([]rune(s[:pos]))

	// Extend through any consecutive periods, dots, or punctuation marks
	for runePos < len(runes) {
		r := runes[runePos]
		// Check if the character is in periods, dot, or other sentence-ending punctuation
		if isPunctuationChar(r) {
			runePos++
		} else {
			break
		}
	}

	// Convert back to byte position
	if runePos >= len(runes) {
		return len(s)
	}
	return len([]byte(string(runes[:runePos])))
}

// isPunctuationChar checks if a rune is a sentence-ending punctuation character
// Matching Rust's periods and dot patterns
func isPunctuationChar(r rune) bool {
	// periods: "。？！♪…\\?\\!"
	periodsChars := "。？！♪…?!"
	// dot: "\\.．"
	dotChars := ".．"

	for _, p := range periodsChars {
		if r == p {
			return true
		}
	}
	for _, d := range dotChars {
		if r == d {
			return true
		}
	}
	return false
}
