package path_rewrite

import (
	"testing"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/lattice"
)

// zeroGrammar creates a minimal grammar for testing
func zeroGrammar() *dic.Grammar {
	zeroBytes := make([]byte, 6)
	grammar, err := dic.NewGrammar(zeroBytes, 0)
	if err != nil {
		panic("Failed to create zero grammar: " + err.Error())
	}
	return grammar
}

func TestJoinNumericPlugin_SetUp(t *testing.T) {
	plugin := NewJoinNumericPlugin()

	// Test with nil settings
	err := plugin.SetUp(nil, "", nil)
	if err != nil {
		t.Fatalf("SetUp failed: %v", err)
	}

	// Test with settings
	settings := map[string]any{
		"enableNormalize": false,
	}
	err = plugin.SetUp(settings, "", zeroGrammar())
	if err != nil {
		t.Fatalf("SetUp with settings failed: %v", err)
	}

	// Verify settings were applied
	if plugin.enableNormalize {
		t.Error("Expected enableNormalize to be false")
	}
}

func TestJoinNumericPlugin_GetName(t *testing.T) {
	plugin := NewJoinNumericPlugin()
	name := plugin.GetName()

	expected := "JoinNumericPlugin"
	if name != expected {
		t.Errorf("Expected name '%s', got '%s'", expected, name)
	}
}

func TestStringNumber_Basic(t *testing.T) {
	// Test basic StringNumber functionality
	sn := NewStringNumber()

	if !sn.isZero() {
		t.Error("New StringNumber should be zero")
	}

	sn.append(1)
	sn.append(2)
	sn.append(3)

	result := sn.ToString()
	expected := "123"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestStringNumber_Scale(t *testing.T) {
	sn := NewStringNumber()
	sn.append(1)
	sn.shiftScale(2) // Multiply by 100

	result := sn.ToString()
	expected := "100"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestNumericParser_Basic(t *testing.T) {
	parser := NewNumericParser()

	// Test parsing "123"
	for _, r := range "123" {
		if !parser.Append(r) {
			t.Errorf("Failed to append character '%c'", r)
		}
	}

	if !parser.Done() {
		t.Error("Parser should be done")
	}

	result := parser.GetNormalized()
	expected := "123"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestNumericParser_Kanji(t *testing.T) {
	parser := NewNumericParser()

	// Test parsing "一二三" (kanji numerals)
	for _, r := range "一二三" {
		if !parser.Append(r) {
			t.Errorf("Failed to append character '%c'", r)
		}
	}

	if !parser.Done() {
		t.Error("Parser should be done")
	}

	result := parser.GetNormalized()
	expected := "123"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestNumericParser_Complex(t *testing.T) {
	parser := NewNumericParser()

	// Test parsing "一千二百三十四" (1234 in kanji)
	input := "一千二百三十四"
	for _, r := range input {
		if !parser.Append(r) {
			t.Errorf("Failed to append character '%c'", r)
		}
	}

	if !parser.Done() {
		t.Error("Parser should be done")
	}

	result := parser.GetNormalized()
	expected := "1234"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestJoinNumericPlugin_Rewrite(t *testing.T) {
	plugin := NewJoinNumericPlugin()
	err := plugin.SetUp(nil, "", zeroGrammar())
	if err != nil {
		t.Fatalf("SetUp failed: %v", err)
	}

	// Create test buffer
	buffer := input.NewInputBuffer()
	err = buffer.StartBuild("123")
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}
	err = buffer.Build(zeroGrammar())
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}

	// Create test lattice
	lat := lattice.New()

	// Create test path with empty results
	path := make([]*lattice.NodeResult, 0)

	// Test rewrite (should return unchanged path for empty input)
	result, err := plugin.Rewrite(path, buffer, lat)
	if err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d items", len(result))
	}
}

// BenchmarkNumericParser benchmarks parser performance
func BenchmarkNumericParser(b *testing.B) {
	input := "一千二百三十四"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := NewNumericParser()
		for _, r := range input {
			parser.Append(r)
		}
		parser.Done()
		parser.GetNormalized()
	}
}

// BenchmarkStringNumber benchmarks StringNumber performance
func BenchmarkStringNumber(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sn := NewStringNumber()
		sn.append(1)
		sn.append(2)
		sn.append(3)
		sn.append(4)
		sn.ToString()
	}
}
