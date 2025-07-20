package dic

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// UnknownWordDefinition represents an unknown word definition
type UnknownWordDefinition struct {
	CategoryType CategoryType
	LeftId       int
	RightId      int
	Cost         int
	POS          []string // Part-of-speech tags (pos1, pos2, pos3, pos4, pos5, pos6)
}

// UnknownWordDefinitions manages unknown word definitions for different categories
type UnknownWordDefinitions struct {
	definitions map[CategoryType][]*UnknownWordDefinition
}

// NewUnknownWordDefinitions creates a new unknown word definitions manager
func NewUnknownWordDefinitions() *UnknownWordDefinitions {
	return &UnknownWordDefinitions{
		definitions: make(map[CategoryType][]*UnknownWordDefinition),
	}
}

// LoadFromReader loads unknown word definitions from reader (unk.def format)
func (uwd *UnknownWordDefinitions) LoadFromReader(reader io.Reader, charCategory *CharacterCategory) error {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		if err := uwd.parseDefinitionLine(line, charCategory); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// parseDefinitionLine parses a single definition line
func (uwd *UnknownWordDefinitions) parseDefinitionLine(line string, charCategory *CharacterCategory) error {
	parts := strings.Split(line, ",")
	if len(parts) < 10 {
		return fmt.Errorf("invalid unk.def line format: expected at least 10 fields, got %d: %s", len(parts), line)
	}

	// Parse category name
	categoryName := strings.TrimSpace(parts[0])

	// Find category type from character category system
	var categoryType CategoryType = CategoryDefault
	if charCategory != nil {
		// Use the parseCategoryName method to convert category name to type
		if cat, err := charCategory.parseCategoryName(categoryName); err == nil {
			categoryType = cat
		}
	}

	// Parse numeric fields
	leftId, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return fmt.Errorf("invalid leftId: %s", parts[1])
	}

	rightId, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return fmt.Errorf("invalid rightId: %s", parts[2])
	}

	cost, err := strconv.Atoi(strings.TrimSpace(parts[3]))
	if err != nil {
		return fmt.Errorf("invalid cost: %s", parts[3])
	}

	// Parse POS tags (pos1-pos6)
	pos := make([]string, 6)
	for i := 0; i < 6; i++ {
		if i+4 < len(parts) {
			pos[i] = strings.TrimSpace(parts[i+4])
		} else {
			pos[i] = "*"
		}
	}

	definition := &UnknownWordDefinition{
		CategoryType: categoryType,
		LeftId:       leftId,
		RightId:      rightId,
		Cost:         cost,
		POS:          pos,
	}

	// Add to definitions map
	uwd.definitions[categoryType] = append(uwd.definitions[categoryType], definition)

	return nil
}

// GetDefinitions returns all unknown word definitions for a given category
func (uwd *UnknownWordDefinitions) GetDefinitions(categoryType CategoryType) []*UnknownWordDefinition {
	return uwd.definitions[categoryType]
}

// GetDefinition returns the first unknown word definition for a given category
func (uwd *UnknownWordDefinitions) GetDefinition(categoryType CategoryType) *UnknownWordDefinition {
	definitions := uwd.definitions[categoryType]
	if len(definitions) > 0 {
		return definitions[0]
	}
	return nil
}

// HasDefinitions returns true if there are definitions for the given category
func (uwd *UnknownWordDefinitions) HasDefinitions(categoryType CategoryType) bool {
	return len(uwd.definitions[categoryType]) > 0
}

// GetAllCategories returns all category types that have definitions
func (uwd *UnknownWordDefinitions) GetAllCategories() []CategoryType {
	categories := make([]CategoryType, 0, len(uwd.definitions))
	for categoryType := range uwd.definitions {
		categories = append(categories, categoryType)
	}
	return categories
}
