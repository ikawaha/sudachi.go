package dic

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// CategoryType represents character category types
type CategoryType uint32

// Predefined category types as bitflags (exactly matching Rust implementation)
const (
	CategoryDefault      CategoryType = 1 << 0  // DEFAULT
	CategorySpace        CategoryType = 1 << 1  // SPACE
	CategoryKanji        CategoryType = 1 << 2  // KANJI
	CategorySymbol       CategoryType = 1 << 3  // SYMBOL
	CategoryNumeric      CategoryType = 1 << 4  // NUMERIC
	CategoryAlpha        CategoryType = 1 << 5  // ALPHA
	CategoryHiragana     CategoryType = 1 << 6  // HIRAGANA
	CategoryKatakana     CategoryType = 1 << 7  // KATAKANA
	CategoryKanjiNumeric CategoryType = 1 << 8  // KANJINUMERIC
	CategoryGreek        CategoryType = 1 << 9  // GREEK
	CategoryCyrillic     CategoryType = 1 << 10 // CYRILLIC

	// User defined categories (matching Rust)
	CategoryUser1 CategoryType = 1 << 11 // USER1
	CategoryUser2 CategoryType = 1 << 12 // USER2
	CategoryUser3 CategoryType = 1 << 13 // USER3
	CategoryUser4 CategoryType = 1 << 14 // USER4

	// Special categories for OOV behavior control (exactly matching Rust)
	CategoryNoOOVBOW  CategoryType = 1 << 30 // NOOOVBOW
	CategoryNoOOVBOW2 CategoryType = 1 << 31 // NOOOVBOW2

	// CategoryAll categories except NOOOVBOW/2 (matching Rust)
	CategoryAll CategoryType = 0b00111111_11111111_11111111_11111111
)

// HasFlag checks if the category type has the specified flag set
func (c CategoryType) HasFlag(flag CategoryType) bool {
	return c&flag != 0
}

// Intersects checks if any of the specified flags are set (matching Rust behavior)
func (c CategoryType) Intersects(flags CategoryType) bool {
	return c&flags != 0
}

// IsNumeric checks if the category is numeric
func (c CategoryType) IsNumeric() bool {
	return c.HasFlag(CategoryNumeric)
}

// IsKanjiNumeric checks if the category is kanji numeric
func (c CategoryType) IsKanjiNumeric() bool {
	return c.HasFlag(CategoryKanjiNumeric)
}

// IsKatakana checks if the category is katakana
func (c CategoryType) IsKatakana() bool {
	return c.HasFlag(CategoryKatakana)
}

// IsNoOOVBOW checks if the category has NOOOVBOW flag
func (c CategoryType) IsNoOOVBOW() bool {
	return c.HasFlag(CategoryNoOOVBOW)
}

// String returns the string representation of CategoryType
func (c CategoryType) String() string {
	// Handle empty category (0) as UNKNOWN, matching Rust behavior
	if c == 0 {
		return "UNKNOWN"
	}

	var parts []string

	// Check for base categories
	if c.HasFlag(CategoryDefault) {
		parts = append(parts, "DEFAULT")
	}
	if c.HasFlag(CategorySpace) {
		parts = append(parts, "SPACE")
	}
	if c.HasFlag(CategoryKanji) {
		parts = append(parts, "KANJI")
	}
	if c.HasFlag(CategorySymbol) {
		parts = append(parts, "SYMBOL")
	}
	if c.HasFlag(CategoryNumeric) {
		parts = append(parts, "NUMERIC")
	}
	if c.HasFlag(CategoryAlpha) {
		parts = append(parts, "ALPHA")
	}
	if c.HasFlag(CategoryHiragana) {
		parts = append(parts, "HIRAGANA")
	}
	if c.HasFlag(CategoryKatakana) {
		parts = append(parts, "KATAKANA")
	}
	if c.HasFlag(CategoryKanjiNumeric) {
		parts = append(parts, "KANJINUMERIC")
	}
	if c.HasFlag(CategoryGreek) {
		parts = append(parts, "GREEK")
	}
	if c.HasFlag(CategoryCyrillic) {
		parts = append(parts, "CYRILLIC")
	}
	if c.HasFlag(CategoryUser1) {
		parts = append(parts, "USER1")
	}
	if c.HasFlag(CategoryUser2) {
		parts = append(parts, "USER2")
	}
	if c.HasFlag(CategoryUser3) {
		parts = append(parts, "USER3")
	}
	if c.HasFlag(CategoryUser4) {
		parts = append(parts, "USER4")
	}

	// Check for special flags
	if c.HasFlag(CategoryNoOOVBOW) {
		parts = append(parts, "NOOOVBOW")
	}
	if c.HasFlag(CategoryNoOOVBOW2) {
		parts = append(parts, "NOOOVBOW2")
	}

	if len(parts) == 0 {
		return "UNKNOWN"
	}
	return strings.Join(parts, "|")
}

// CategoryInfo represents character category information
type CategoryInfo struct {
	CategoryType CategoryType
	Name         string
	IsInvoke     bool   // Always invoke OOV processing
	IsGroup      bool   // Group consecutive same-category characters
	Length       uint32 // Maximum length for OOV words
}

// CatRange represents a character range with associated categories (matching Rust)
type CatRange struct {
	Begin      uint32
	End        uint32
	Categories CategoryType
}

// CharacterCategory manages character category definitions and mappings (Rust-compatible)
type CharacterCategory struct {
	// Split the whole domain of codepoints into ranges, limited by boundaries.
	// Ranges are half-open: [boundaries[i], boundaries[i + 1])
	// meaning that the right bound is not included.
	// 0 and uint32(0xFFFFFFFF) are not stored, they are included implicitly.
	boundaries []uint32

	// Stores the category for each range.
	// categories[i] is for the range [boundaries[i - 1], boundaries[i]).
	// This should be always true: len(boundaries) + 1 == len(categories).
	categories []CategoryType

	// Stores category metadata read from char.def (Rust equivalent)
	categoryInfoMap map[CategoryType]*CategoryInfo
}

// NewCharacterCategory creates a new character category system
func NewCharacterCategory() *CharacterCategory {
	return &CharacterCategory{
		boundaries:      make([]uint32, 0),
		categories:      []CategoryType{CategoryDefault}, // First category is always DEFAULT
		categoryInfoMap: make(map[CategoryType]*CategoryInfo),
	}
}

// LoadFromReader loads character definitions from reader (char.def format)
func (cc *CharacterCategory) LoadFromReader(reader io.Reader) error {
	ranges, err := cc.readCharacterDefinition(reader)
	if err != nil {
		return err
	}
	cc.compile(ranges)
	return nil
}

// readCharacterDefinition reads character type definition as a list of Ranges
// Rust equivalent function: read_character_property + parsing of unicode mappings
func (cc *CharacterCategory) readCharacterDefinition(reader io.Reader) ([]CatRange, error) {
	var ranges []CatRange
	scanner := bufio.NewScanner(reader)
	lineNum := 0

	// Single pass: process both category definitions and character mappings
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNum++

		// Skip empty lines and comments
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		// Check if this is a character mapping line (starts with 0x)
		if strings.HasPrefix(line, "0x") {
			// Parse character mapping
			cols := strings.Fields(line)
			if len(cols) < 2 {
				return nil, fmt.Errorf("invalid format at line %d", lineNum)
			}

			// Parse Unicode range
			rangeParts := strings.Split(cols[0], "..")
			begin, err := parseUnicodeValue(rangeParts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid begin value at line %d: %v", lineNum, err)
			}

			var end uint32
			if len(rangeParts) > 1 {
				endVal, err := parseUnicodeValue(rangeParts[1])
				if err != nil {
					return nil, fmt.Errorf("invalid end value at line %d: %v", lineNum, err)
				}
				end = uint32(endVal) + 1 // Make it exclusive (half-open range)
			} else {
				end = uint32(begin) + 1
			}

			if uint32(begin) >= end {
				return nil, fmt.Errorf("invalid range at line %d", lineNum)
			}

			// Parse categories (multiple categories can be specified)
			var categories CategoryType = 0
			for i := 1; i < len(cols); i++ {
				if strings.HasPrefix(cols[i], "#") {
					break // Rest is comment
				}
				cat, err := cc.parseCategoryName(cols[i])
				if err != nil {
					return nil, fmt.Errorf("invalid category type %s at line %d", cols[i], lineNum)
				}
				categories |= cat
			}

			ranges = append(ranges, CatRange{
				Begin:      uint32(begin),
				End:        end,
				Categories: categories,
			})
		} else {
			// Parse category definition line (NAME INVOKE GROUP LENGTH)
			if err := cc.parseCategoryDefinitionLine(line, lineNum); err != nil {
				return nil, err
			}
		}
	}

	return ranges, scanner.Err()
}

// parseCategoryDefinitionLine parses a category definition line (NAME INVOKE GROUP LENGTH)
func (cc *CharacterCategory) parseCategoryDefinitionLine(line string, lineNum int) error {
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return fmt.Errorf("invalid category definition at line %d: expected 4 fields, got %d", lineNum, len(parts))
	}

	name := parts[0]
	invoke, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid INVOKE value '%s' at line %d", parts[1], lineNum)
	}

	group, err := strconv.Atoi(parts[2])
	if err != nil {
		return fmt.Errorf("invalid GROUP value '%s' at line %d", parts[2], lineNum)
	}

	length, err := strconv.ParseUint(parts[3], 10, 32)
	if err != nil {
		return fmt.Errorf("invalid LENGTH value '%s' at line %d", parts[3], lineNum)
	}

	// Map category name to CategoryType
	categoryType, err := cc.parseCategoryName(name)
	if err != nil {
		return fmt.Errorf("unknown category '%s' at line %d", name, lineNum)
	}

	// Store category metadata
	categoryInfo := &CategoryInfo{
		CategoryType: categoryType,
		Name:         name,
		IsInvoke:     invoke != 0,
		IsGroup:      group != 0,
		Length:       uint32(length),
	}

	cc.categoryInfoMap[categoryType] = categoryInfo
	return nil
}

// parseCategoryName parses a category name and returns the corresponding CategoryType
func (cc *CharacterCategory) parseCategoryName(name string) (CategoryType, error) {
	switch name {
	case "DEFAULT":
		return CategoryDefault, nil
	case "SPACE":
		return CategorySpace, nil
	case "KANJI":
		return CategoryKanji, nil
	case "SYMBOL":
		return CategorySymbol, nil
	case "NUMERIC":
		return CategoryNumeric, nil
	case "ALPHA":
		return CategoryAlpha, nil
	case "HIRAGANA":
		return CategoryHiragana, nil
	case "KATAKANA":
		return CategoryKatakana, nil
	case "KANJINUMERIC":
		return CategoryKanjiNumeric, nil
	case "GREEK":
		return CategoryGreek, nil
	case "CYRILLIC":
		return CategoryCyrillic, nil
	case "USER1":
		return CategoryUser1, nil
	case "USER2":
		return CategoryUser2, nil
	case "USER3":
		return CategoryUser3, nil
	case "USER4":
		return CategoryUser4, nil
	case "NOOOVBOW":
		return CategoryNoOOVBOW, nil
	case "NOOOVBOW2":
		return CategoryNoOOVBOW2, nil
	case "ALL":
		return CategoryAll, nil
	default:
		return 0, fmt.Errorf("unknown category: %s", name)
	}
}

// compile creates a character category from given range list (Rust equivalent)
// Transforms given range list to non-overlapped range list to apply binary search
func (cc *CharacterCategory) compile(ranges []CatRange) {
	if len(ranges) == 0 {
		// Reset to default state
		cc.boundaries = make([]uint32, 0)
		cc.categories = []CategoryType{CategoryDefault}
		return
	}

	boundaries := cc.collectBoundaries(ranges)
	categories := make([]CategoryType, len(boundaries))

	// Initialize all categories as empty
	for i := range categories {
		categories[i] = 0
	}

	// Apply categories to ranges
	for _, rng := range ranges {
		startIdx := cc.findBoundaryIndex(boundaries, rng.Begin)
		if startIdx == -1 {
			continue // Should not happen with proper boundary collection
		}
		startIdx++ // Move to next range

		// Apply category to all splits which are included in the current range
		for i := startIdx; i < len(boundaries); i++ {
			if boundaries[i] > rng.End {
				break
			}
			categories[i] |= rng.Categories
		}
	}

	// First category is always DEFAULT (it's impossible to get it assigned above)
	if len(categories) > 0 && categories[0] == 0 {
		categories[0] = CategoryDefault
	}

	// Merge successive ranges of the same category
	finalBoundaries := make([]uint32, 0, len(boundaries))
	finalCategories := make([]CategoryType, 0, len(categories))

	if len(categories) > 0 {
		lastCategory := categories[0]
		var lastBoundary uint32
		if len(boundaries) > 0 {
			lastBoundary = boundaries[0]
		}

		for i := 1; i < len(categories); i++ {
			if categories[i] == lastCategory {
				if i < len(boundaries) {
					lastBoundary = boundaries[i]
				}
				continue
			}
			finalBoundaries = append(finalBoundaries, lastBoundary)
			finalCategories = append(finalCategories, lastCategory)
			lastCategory = categories[i]
			if i < len(boundaries) {
				lastBoundary = boundaries[i]
			}
		}

		finalCategories = append(finalCategories, lastCategory)
		finalBoundaries = append(finalBoundaries, lastBoundary)

		// Replace empty categories with default
		for i := range finalCategories {
			if finalCategories[i] == 0 {
				finalCategories[i] = CategoryDefault
			}
		}

		// Add the category after the last boundary
		finalCategories = append(finalCategories, CategoryDefault)
	}

	cc.boundaries = finalBoundaries
	cc.categories = finalCategories
}

// collectBoundaries finds sorted list of all boundaries (Rust equivalent)
func (cc *CharacterCategory) collectBoundaries(ranges []CatRange) []uint32 {
	boundarySet := make(map[uint32]bool)
	for _, rng := range ranges {
		boundarySet[rng.Begin] = true
		boundarySet[rng.End] = true
	}

	boundaries := make([]uint32, 0, len(boundarySet))
	for boundary := range boundarySet {
		boundaries = append(boundaries, boundary)
	}

	// Sort boundaries
	for i := 0; i < len(boundaries); i++ {
		for j := i + 1; j < len(boundaries); j++ {
			if boundaries[i] > boundaries[j] {
				boundaries[i], boundaries[j] = boundaries[j], boundaries[i]
			}
		}
	}

	return boundaries
}

// findBoundaryIndex finds the index of a boundary value using binary search
func (cc *CharacterCategory) findBoundaryIndex(boundaries []uint32, value uint32) int {
	left, right := 0, len(boundaries)-1
	for left <= right {
		mid := (left + right) / 2
		if boundaries[mid] == value {
			return mid
		} else if boundaries[mid] < value {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return -1 // Not found
}

// GetCategory returns the category types for a given character (Rust equivalent: get_category_types)
func (cc *CharacterCategory) GetCategory(r rune) CategoryType {
	if len(cc.boundaries) == 0 {
		return CategoryDefault
	}

	codepoint := uint32(r)

	// Binary search to find the appropriate category
	// This matches Rust's binary_search behavior
	left, right := 0, len(cc.boundaries)-1

	for left <= right {
		mid := (left + right) / 2
		if cc.boundaries[mid] == codepoint {
			// Exact match found - return next category
			if mid+1 < len(cc.categories) {
				return cc.categories[mid+1]
			}
			return CategoryDefault
		} else if cc.boundaries[mid] < codepoint {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	// Not found - return category for the insertion point
	if left < len(cc.categories) {
		return cc.categories[left]
	}

	return CategoryDefault
}

// parseUnicodeValue parses a Unicode value (e.g., "0x0041")
func parseUnicodeValue(s string) (rune, error) {
	if !strings.HasPrefix(s, "0x") {
		return 0, fmt.Errorf("invalid Unicode format: %s", s)
	}

	value, err := strconv.ParseInt(s[2:], 16, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid Unicode value: %s", s)
	}

	return rune(value), nil
}

// GetCategoryInfo returns category information for a given category type (Rust equivalent)
// Returns actual metadata loaded from char.def file, or nil if not found
func (cc *CharacterCategory) GetCategoryInfo(categoryType CategoryType) *CategoryInfo {
	// Return actual metadata from categoryInfoMap if available, otherwise nil
	// This matches Rust behavior: categories.get(&ctype) returns Option<&CategoryInfo>
	if info, exists := cc.categoryInfoMap[categoryType]; exists {
		return info
	}
	return nil // Rust equivalent: None case
}

// CategoryRange represents a character range with its category
// This matches Rust's iterator output format
type CategoryRange struct {
	Range struct {
		Start uint32
		End   uint32
	}
	Category CategoryType
}

// GetRanges returns all character ranges with their categories
// This matches Rust's character_category.iter() functionality
func (cc *CharacterCategory) GetRanges() []CategoryRange {
	var ranges []CategoryRange

	// Handle the first range [0, boundaries[0])
	if len(cc.boundaries) > 0 {
		ranges = append(ranges, CategoryRange{
			Range:    struct{ Start, End uint32 }{0, cc.boundaries[0]},
			Category: cc.categories[0],
		})
	}

	// Handle ranges between boundaries
	for i := 0; i < len(cc.boundaries)-1; i++ {
		ranges = append(ranges, CategoryRange{
			Range:    struct{ Start, End uint32 }{cc.boundaries[i], cc.boundaries[i+1]},
			Category: cc.categories[i+1],
		})
	}

	// Handle the last range [last_boundary, 0xFFFFFFFF)
	if len(cc.boundaries) > 0 {
		lastIdx := len(cc.boundaries) - 1
		ranges = append(ranges, CategoryRange{
			Range:    struct{ Start, End uint32 }{cc.boundaries[lastIdx], 0xFFFFFFFF},
			Category: cc.categories[lastIdx+1],
		})
	} else {
		// No boundaries, everything is DEFAULT
		ranges = append(ranges, CategoryRange{
			Range:    struct{ Start, End uint32 }{0, 0xFFFFFFFF},
			Category: CategoryDefault,
		})
	}

	return ranges
}
