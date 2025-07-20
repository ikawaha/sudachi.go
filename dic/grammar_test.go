package dic

import (
	"encoding/binary"
	"testing"
)

func TestStorageSize(t *testing.T) {
	bytes := setupBytes()
	grammar, err := NewGrammar(bytes, 0)
	if err != nil {
		t.Fatalf("failed to create grammar: %v", err)
	}

	if len(bytes) != grammar.StorageSize() {
		t.Errorf("storage size mismatch: expected %d, got %d", len(bytes), grammar.StorageSize())
	}
}

func TestPartOfSpeechString(t *testing.T) {
	bytes := setupBytes()
	grammar, err := NewGrammar(bytes, 0)
	if err != nil {
		t.Fatalf("failed to create grammar: %v", err)
	}

	// Check POS list structure
	if grammar.POSListSize() < 3 {
		t.Fatalf("expected at least 3 POS entries, got %d", grammar.POSListSize())
	}

	// Test first POS (BOS/EOS)
	pos0, err := grammar.GetPOS(0)
	if err != nil {
		t.Fatalf("failed to get POS 0: %v", err)
	}

	if len(pos0) != 6 {
		t.Errorf("expected POS depth 6, got %d", len(pos0))
	}

	if pos0[0] != "BOS/EOS" {
		t.Errorf("expected first component 'BOS/EOS', got '%s'", pos0[0])
	}

	if pos0[5] != "*" {
		t.Errorf("expected last component '*', got '%s'", pos0[5])
	}

	// Test second POS (名詞)
	pos1, err := grammar.GetPOS(1)
	if err != nil {
		t.Fatalf("failed to get POS 1: %v", err)
	}

	if pos1[1] != "一般" {
		t.Errorf("expected second component '一般', got '%s'", pos1[1])
	}

	if pos1[5] != "*" {
		t.Errorf("expected last component '*', got '%s'", pos1[5])
	}

	// Test third POS (動詞)
	pos2, err := grammar.GetPOS(2)
	if err != nil {
		t.Fatalf("failed to get POS 2: %v", err)
	}

	if pos2[4] != "五段-サ行" {
		t.Errorf("expected fifth component '五段-サ行', got '%s'", pos2[4])
	}

	if pos2[5] != "終止形-一般" {
		t.Errorf("expected sixth component '終止形-一般', got '%s'", pos2[5])
	}
}

func TestGetConnectCost(t *testing.T) {
	bytes := setupBytes()
	grammar, err := NewGrammar(bytes, 0)
	if err != nil {
		t.Fatalf("failed to create grammar: %v", err)
	}

	// Test specific connection costs from Rust test
	if cost := grammar.ConnectCost(0, 0); cost != 0 {
		t.Errorf("expected connect cost(0,0) = 0, got %d", cost)
	}

	if cost := grammar.ConnectCost(2, 1); cost != -100 {
		t.Errorf("expected connect cost(2,1) = -100, got %d", cost)
	}

	if cost := grammar.ConnectCost(1, 2); cost != 200 {
		t.Errorf("expected connect cost(1,2) = 200, got %d", cost)
	}
}

func TestSetConnectCost(t *testing.T) {
	bytes := setupBytes()
	grammar, err := NewGrammar(bytes, 0)
	if err != nil {
		t.Fatalf("failed to create grammar: %v", err)
	}

	// Set new connection cost
	grammar.SetConnectCost(0, 0, 300)

	// Verify the cost was updated
	if cost := grammar.ConnectCost(0, 0); cost != 300 {
		t.Errorf("expected connect cost(0,0) = 300 after update, got %d", cost)
	}
}

func TestRegisterPOS(t *testing.T) {
	bytes := setupBytes()
	grammar, err := NewGrammar(bytes, 0)
	if err != nil {
		t.Fatalf("failed to create grammar: %v", err)
	}

	// Register new POS
	newPOS := []string{"a", "b", "c", "d", "e", "f"}
	id1, err := grammar.RegisterPOS(newPOS)
	if err != nil {
		t.Fatalf("failed to register POS: %v", err)
	}

	// Register the same POS again - should return same ID
	id2, err := grammar.RegisterPOS(newPOS)
	if err != nil {
		t.Fatalf("failed to register POS second time: %v", err)
	}

	if id1 != id2 {
		t.Errorf("expected same ID for duplicate POS registration: got %d and %d", id1, id2)
	}

	// Verify we can retrieve the registered POS
	retrievedPOS, err := grammar.GetPOS(id1)
	if err != nil {
		t.Fatalf("failed to get registered POS: %v", err)
	}

	for i, component := range newPOS {
		if retrievedPOS[i] != component {
			t.Errorf("POS component mismatch at index %d: expected '%s', got '%s'",
				i, component, retrievedPOS[i])
		}
	}
}

func TestGetPartOfSpeechId(t *testing.T) {
	bytes := setupBytes()
	grammar, err := NewGrammar(bytes, 0)
	if err != nil {
		t.Fatalf("failed to create grammar: %v", err)
	}

	// Test existing POS
	bosEosPOS := []string{"BOS/EOS", "*", "*", "*", "*", "*"}
	id := grammar.GetPartOfSpeechId(bosEosPOS)
	if id == nil {
		t.Error("expected to find BOS/EOS POS, got nil")
	} else if *id != 0 {
		t.Errorf("expected BOS/EOS POS ID 0, got %d", *id)
	}

	// Test non-existent POS
	nonExistentPOS := []string{"x", "y", "z", "a", "b", "c"}
	id = grammar.GetPartOfSpeechId(nonExistentPOS)
	if id != nil {
		t.Errorf("expected nil for non-existent POS, got %d", *id)
	}

	// Test invalid POS length
	invalidPOS := []string{"too", "short"}
	id = grammar.GetPartOfSpeechId(invalidPOS)
	if id != nil {
		t.Error("expected nil for invalid POS length, got non-nil")
	}
}

func TestInhibitedConnectionConstant(t *testing.T) {
	// Test that InhibitedConnection constant matches Rust i16::MAX
	expected := int16(32767) // i16::MAX
	if InhibitedConnection != expected {
		t.Errorf("expected InhibitedConnection = %d, got %d", expected, InhibitedConnection)
	}
}

// setupBytes creates test data similar to Rust version
func setupBytes() []byte {
	var storage []byte
	buildPartOfSpeech(&storage)
	buildConnectTable(&storage)
	return storage
}

// stringToBytes converts string to UTF-16 little-endian bytes
func stringToBytes(s string) []byte {
	utf16 := []rune(s)
	bytes := make([]byte, len(utf16)*2)
	for i, r := range utf16 {
		binary.LittleEndian.PutUint16(bytes[i*2:], uint16(r))
	}
	return bytes
}

// buildPartOfSpeech builds the POS section of test data
func buildPartOfSpeech(storage *[]byte) {
	// Number of part of speech entries
	posCount := make([]byte, 2)
	binary.LittleEndian.PutUint16(posCount, 3)
	*storage = append(*storage, posCount...)

	// First POS: BOS/EOS
	*storage = append(*storage, 0x07) // length of "BOS/EOS"
	*storage = append(*storage, stringToBytes("BOS/EOS")...)
	for i := 0; i < 5; i++ { // 5 more "*" components
		*storage = append(*storage, 0x01) // length of "*"
		*storage = append(*storage, stringToBytes("*")...)
	}

	// Second POS: 名詞,一般
	*storage = append(*storage, 0x02) // length of "名詞"
	*storage = append(*storage, stringToBytes("名詞")...)
	*storage = append(*storage, 0x02) // length of "一般"
	*storage = append(*storage, stringToBytes("一般")...)
	for i := 0; i < 4; i++ { // 4 more "*" components
		*storage = append(*storage, 0x01) // length of "*"
		*storage = append(*storage, stringToBytes("*")...)
	}

	// Third POS: 動詞,一般,*,*,五段-サ行,終止形-一般
	*storage = append(*storage, 0x02) // length of "動詞"
	*storage = append(*storage, stringToBytes("動詞")...)
	*storage = append(*storage, 0x02) // length of "一般"
	*storage = append(*storage, stringToBytes("一般")...)
	*storage = append(*storage, 0x01) // length of "*"
	*storage = append(*storage, stringToBytes("*")...)
	*storage = append(*storage, 0x01) // length of "*"
	*storage = append(*storage, stringToBytes("*")...)
	*storage = append(*storage, 0x05) // length of "五段-サ行"
	*storage = append(*storage, stringToBytes("五段-サ行")...)
	*storage = append(*storage, 0x06) // length of "終止形-一般"
	*storage = append(*storage, stringToBytes("終止形-一般")...)
}

// buildConnectTable builds the connection matrix section of test data
func buildConnectTable(storage *[]byte) {
	// Left and right ID sizes
	leftSize := make([]byte, 2)
	binary.LittleEndian.PutUint16(leftSize, 3)
	*storage = append(*storage, leftSize...)

	rightSize := make([]byte, 2)
	binary.LittleEndian.PutUint16(rightSize, 3)
	*storage = append(*storage, rightSize...)

	// Connection matrix (3x3)
	costs := []int16{
		0, -300, 300, // Row 0
		300, -500, -100, // Row 1
		-3000, 200, 2000, // Row 2
	}

	for _, cost := range costs {
		costBytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(costBytes, uint16(cost))
		*storage = append(*storage, costBytes...)
	}
}
