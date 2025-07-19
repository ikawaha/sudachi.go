package input

import (
	"encoding/binary"
	"path/filepath"

	"github.com/ikawaha/sudachi.go/dic"
)

// zeroGrammar creates a minimal grammar with zero bytes (equivalent to Rust zero_grammar)
func zeroGrammar() *dic.Grammar {
	// Create a minimal grammar structure:
	// - 2 bytes for POS size (0)
	// - 2 bytes for left_id_size (0)
	// - 2 bytes for right_id_size (0)
	// - 0 bytes for connection matrix (0 x 0)
	zeroBytes := make([]byte, 6)
	binary.LittleEndian.PutUint16(zeroBytes[0:2], 0) // POS size
	binary.LittleEndian.PutUint16(zeroBytes[2:4], 0) // left_id_size
	binary.LittleEndian.PutUint16(zeroBytes[4:6], 0) // right_id_size

	grammar, err := dic.NewGrammar(zeroBytes, 0)
	if err != nil {
		panic("Failed to create zero grammar: " + err.Error())
	}
	return grammar
}

// catGrammar creates a grammar with character category support (equivalent to Rust cat_grammar)
func catGrammar() *dic.Grammar {
	// Use zero grammar like Rust, but with test char.def
	grammar := zeroGrammar()

	// Load character categories from test char.def (copied from Rust test resources)
	charDefPath := filepath.Join("testdata", "char.def")
	charCategory, err := dic.LoadCharacterCategoryFromFile(charDefPath)
	if err != nil {
		panic("Failed to load character category: " + err.Error())
	}

	// Set character category on grammar
	grammar.SetCharacterCategory(charCategory)

	return grammar
}
