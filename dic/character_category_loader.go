package dic

import (
	"fmt"
	"os"
	"path/filepath"
)

// LoadCharacterCategoryFromFile loads character category from char.def file
func LoadCharacterCategoryFromFile(charDefPath string) (*CharacterCategory, error) {
	file, err := os.Open(charDefPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open char.def file: %w", err)
	}
	defer file.Close()

	cc := NewCharacterCategory()
	if err := cc.LoadFromReader(file); err != nil {
		return nil, fmt.Errorf("failed to load char.def: %w", err)
	}

	return cc, nil
}

// LoadCharacterCategoryFromResourceDir loads character category from resource directory
func LoadCharacterCategoryFromResourceDir(resourceDir string) (*CharacterCategory, error) {
	charDefPath := filepath.Join(resourceDir, "char.def")
	return LoadCharacterCategoryFromFile(charDefPath)
}
