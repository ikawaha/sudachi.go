package input

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// Embedded rewrite.def file (matching Rust version's include_bytes!)
//go:embed rewrite.def
var embeddedRewriteDef []byte

// RewriteDefData contains parsed rewrite.def data
type RewriteDefData struct {
	// Characters to ignore during NFKC normalization (single character per line)
	IgnoreNormalizeChars map[rune]bool
	// Character replacement rules (before -> after mapping)
	ReplaceRules map[string]string
}

// ParseRewriteDef parses rewrite.def content from a reader
// This matches Rust Sudachi's read_rewrite_lists function exactly
func ParseRewriteDef(reader io.Reader) (*RewriteDefData, error) {
	data := &RewriteDefData{
		IgnoreNormalizeChars: make(map[rune]bool),
		ReplaceRules:         make(map[string]string),
	}

	scanner := bufio.NewScanner(reader)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments (matching Rust behavior)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split by whitespace (matching Rust split_whitespace)
		cols := strings.Fields(line)

		if len(cols) == 1 {
			// Ignored normalize list: single character per line
			if utf8.RuneCountInString(cols[0]) != 1 {
				return nil, fmt.Errorf("line %d: '%s' is not a single character", lineNum, cols[0])
			}
			char, _ := utf8.DecodeRuneInString(cols[0])
			data.IgnoreNormalizeChars[char] = true
		} else if len(cols) == 2 {
			// Replace char list: before after
			before := cols[0]
			after := cols[1]

			// Check for duplicate definitions (matching Rust error handling)
			if _, exists := data.ReplaceRules[before]; exists {
				return nil, fmt.Errorf("line %d: '%s' is already defined", lineNum, before)
			}

			data.ReplaceRules[before] = after
		} else {
			// Invalid format (matching Rust error handling)
			return nil, fmt.Errorf("line %d: invalid format", lineNum)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading rewrite.def: %w", err)
	}

	return data, nil
}

// ParseRewriteDefFromFile parses rewrite.def from a file path
func ParseRewriteDefFromFile(filePath string) (*RewriteDefData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open rewrite.def file '%s': %w", filePath, err)
	}
	defer file.Close()

	return ParseRewriteDef(file)
}

// ParseRewriteDefFromBytes parses rewrite.def from byte slice
func ParseRewriteDefFromBytes(data []byte) (*RewriteDefData, error) {
	return ParseRewriteDef(strings.NewReader(string(data)))
}

// GetDefaultRewriteDefData returns the default rewrite.def data matching Rust version
// This uses the embedded rewrite.def file, equivalent to Rust's include_bytes!
func GetDefaultRewriteDefData() (*RewriteDefData, error) {
	return ParseRewriteDefFromBytes(embeddedRewriteDef)
}


// ValidateRewriteDefData validates parsed rewrite.def data
func ValidateRewriteDefData(data *RewriteDefData) error {
	if data == nil {
		return fmt.Errorf("rewrite.def data is nil")
	}

	// Validate ignore normalize characters
	for char := range data.IgnoreNormalizeChars {
		if !utf8.ValidRune(char) {
			return fmt.Errorf("invalid ignore normalize character: %U", char)
		}
	}

	// Validate replace rules
	for before, after := range data.ReplaceRules {
		if !utf8.ValidString(before) {
			return fmt.Errorf("invalid before string in replace rule: %q", before)
		}
		if !utf8.ValidString(after) {
			return fmt.Errorf("invalid after string in replace rule: %q", after)
		}
		if before == after {
			return fmt.Errorf("replace rule has same before and after: %q", before)
		}
	}

	return nil
}