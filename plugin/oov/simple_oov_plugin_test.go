package oov

import (
	"testing"

	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/lattice"
	"github.com/ikawaha/sudachi.go/types"
)

// Use zeroGrammar from mecab_oov_plugin_test.go

func TestSimpleOovPlugin_NewSimpleOovPlugin(t *testing.T) {
	plugin := NewSimpleOovPlugin()

	if plugin == nil {
		t.Fatal("NewSimpleOovPlugin should not return nil")
	}

	// Check default values (matching Rust Default implementation)
	if plugin.leftId != 0 {
		t.Errorf("Expected default leftId to be 0, got %d", plugin.leftId)
	}
	if plugin.rightId != 0 {
		t.Errorf("Expected default rightId to be 0, got %d", plugin.rightId)
	}
	if plugin.cost != 0 {
		t.Errorf("Expected default cost to be 0, got %d", plugin.cost)
	}
	if plugin.oovPosId != 0 {
		t.Errorf("Expected default oovPosId to be 0, got %d", plugin.oovPosId)
	}

	// Check default POS
	expectedPOS := []string{"補助記号", "一般", "*", "*", "*", "*"}
	if len(plugin.oovPOS) != len(expectedPOS) {
		t.Errorf("Expected default POS length %d, got %d", len(expectedPOS), len(plugin.oovPOS))
	}
	for i, pos := range expectedPOS {
		if i < len(plugin.oovPOS) && plugin.oovPOS[i] != pos {
			t.Errorf("Expected default POS[%d] to be '%s', got '%s'", i, pos, plugin.oovPOS[i])
		}
	}
}

func TestSimpleOovPlugin_GetName(t *testing.T) {
	plugin := NewSimpleOovPlugin()
	name := plugin.GetName()

	expected := "SimpleOovPlugin"
	if name != expected {
		t.Errorf("Expected name '%s', got '%s'", expected, name)
	}
}

func TestSimpleOovPlugin_SetUp(t *testing.T) {
	plugin := NewSimpleOovPlugin()

	// Test with nil settings
	err := plugin.SetUp(nil, "", zeroGrammar())
	if err != nil {
		t.Fatalf("SetUp with nil settings failed: %v", err)
	}

	// Test with valid settings
	settings := map[string]any{
		"oovPOS":  []any{"名詞", "普通名詞", "一般", "*", "*", "*"},
		"leftId":  float64(100),
		"rightId": float64(200),
		"cost":    float64(-500),
	}

	err = plugin.SetUp(settings, "", zeroGrammar())
	if err != nil {
		t.Fatalf("SetUp with valid settings failed: %v", err)
	}

	// Verify settings were applied
	if plugin.leftId != 100 {
		t.Errorf("Expected leftId to be 100, got %d", plugin.leftId)
	}
	if plugin.rightId != 200 {
		t.Errorf("Expected rightId to be 200, got %d", plugin.rightId)
	}
	if plugin.cost != -500 {
		t.Errorf("Expected cost to be -500, got %d", plugin.cost)
	}

	expectedPOS := []string{"名詞", "普通名詞", "一般", "*", "*", "*"}
	if len(plugin.oovPOS) != len(expectedPOS) {
		t.Errorf("Expected POS length %d, got %d", len(expectedPOS), len(plugin.oovPOS))
	}
}

func TestSimpleOovPlugin_SetUpInvalidSettings(t *testing.T) {
	plugin := NewSimpleOovPlugin()

	// Test with invalid leftId (out of range)
	settings := map[string]any{
		"leftId": float64(70000), // > 65535
	}

	err := plugin.SetUp(settings, "", zeroGrammar())
	if err == nil {
		t.Error("Expected error for invalid leftId, but got none")
	}

	// Test with invalid rightId (negative)
	settings = map[string]any{
		"rightId": float64(-1),
	}

	err = plugin.SetUp(settings, "", zeroGrammar())
	if err == nil {
		t.Error("Expected error for invalid rightId, but got none")
	}

	// Test with invalid cost (out of range)
	settings = map[string]any{
		"cost": float64(40000), // > 32767
	}

	err = plugin.SetUp(settings, "", zeroGrammar())
	if err == nil {
		t.Error("Expected error for invalid cost, but got none")
	}
}

func TestSimpleOovPlugin_ProvideOOV(t *testing.T) {
	plugin := NewSimpleOovPlugin()
	err := plugin.SetUp(nil, "", zeroGrammar())
	if err != nil {
		t.Fatalf("SetUp failed: %v", err)
	}

	// Create test buffer
	buffer := input.NewInputBuffer()
	err = buffer.StartBuild("テスト")
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}
	err = buffer.Build(zeroGrammar())
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}

	// Create test lattice with proper size
	lat := lattice.New()
	lat.Reset(buffer.CharCount() + 1) // +1 for EOS

	// Test with empty CreatedWords (should create OOV)
	emptyCreatedWords := types.CreatedWords{}
	newCreatedWords, err := plugin.ProvideOOV(0, buffer, lat, emptyCreatedWords)
	if err != nil {
		t.Fatalf("ProvideOOV failed: %v", err)
	}

	// Should have created a word
	if newCreatedWords.IsEmpty() {
		t.Error("Expected CreatedWords to be non-empty after OOV creation")
	}

	// Should have word of length 1
	if newCreatedWords.HasWord(1) != types.HasWordYes {
		t.Error("Expected CreatedWords to have word of length 1")
	}

	// Test with non-empty CreatedWords (should not create OOV)
	nonEmptyCreatedWords := types.CreatedWords{}.AddWord(1)
	unchangedCreatedWords, err := plugin.ProvideOOV(0, buffer, lat, nonEmptyCreatedWords)
	if err != nil {
		t.Fatalf("ProvideOOV with non-empty CreatedWords failed: %v", err)
	}

	// Should remain unchanged (matching Rust: if other_words.not_empty() { return Ok(0); })
	if unchangedCreatedWords != nonEmptyCreatedWords {
		t.Error("Expected CreatedWords to remain unchanged when other words exist")
	}
}

func TestSimpleOovPlugin_ProvideOOVOutOfBounds(t *testing.T) {
	plugin := NewSimpleOovPlugin()
	err := plugin.SetUp(nil, "", zeroGrammar())
	if err != nil {
		t.Fatalf("SetUp failed: %v", err)
	}

	// Create test buffer
	buffer := input.NewInputBuffer()
	err = buffer.StartBuild("テ")
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}
	err = buffer.Build(zeroGrammar())
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}

	// Create test lattice with proper size
	lat := lattice.New()
	lat.Reset(buffer.CharCount() + 1) // +1 for EOS

	// Test with out-of-bounds position
	emptyCreatedWords := types.CreatedWords{}
	unchangedCreatedWords, err := plugin.ProvideOOV(10, buffer, lat, emptyCreatedWords) // Position 10 is out of bounds
	if err != nil {
		t.Fatalf("ProvideOOV with out-of-bounds position failed: %v", err)
	}

	// Should remain unchanged
	if unchangedCreatedWords != emptyCreatedWords {
		t.Error("Expected CreatedWords to remain unchanged for out-of-bounds position")
	}
}

func TestSimpleOovPlugin_GetWordCandidateLength(t *testing.T) {
	plugin := NewSimpleOovPlugin()

	// Create test buffer
	buffer := input.NewInputBuffer()
	err := buffer.StartBuild("テスト")
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}
	err = buffer.Build(zeroGrammar())
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}

	// Test normal character position
	length := plugin.getWordCandidateLength(buffer, 0)
	expected := 1 // SimpleOov always returns 1 for single character
	if length != expected {
		t.Errorf("Expected word candidate length %d, got %d", expected, length)
	}

	// Test out-of-bounds position
	length = plugin.getWordCandidateLength(buffer, 10)
	expected = 0
	if length != expected {
		t.Errorf("Expected word candidate length %d for out-of-bounds, got %d", expected, length)
	}
}

func TestSimpleOovPlugin_CreateOovNode(t *testing.T) {
	plugin := NewSimpleOovPlugin()

	// Set up plugin with specific values
	settings := map[string]any{
		"leftId":  float64(100),
		"rightId": float64(200),
		"cost":    float64(-500),
	}
	err := plugin.SetUp(settings, "", zeroGrammar())
	if err != nil {
		t.Fatalf("SetUp failed: %v", err)
	}

	// Create OOV node
	node := plugin.createOovNode(0, 1)

	// Check node properties (matching Rust Node::new parameters)
	if node.Begin() != 0 {
		t.Errorf("Expected node begin 0, got %d", node.Begin())
	}
	if node.End() != 1 {
		t.Errorf("Expected node end 1, got %d", node.End())
	}
	if node.LeftId() != 100 {
		t.Errorf("Expected node leftId 100, got %d", node.LeftId())
	}
	if node.RightId() != 200 {
		t.Errorf("Expected node rightId 200, got %d", node.RightId())
	}
	if node.Cost() != -500 {
		t.Errorf("Expected node cost -500, got %d", node.Cost())
	}

	// Check that it's an OOV node
	if !node.IsOOV() {
		t.Error("Expected node to be OOV")
	}
}

func TestSimpleOovPlugin_RustCompatibility(t *testing.T) {
	// Test that our implementation follows Rust patterns

	plugin := NewSimpleOovPlugin()

	// Rust Default behavior
	if plugin.leftId != 0 || plugin.rightId != 0 || plugin.cost != 0 || plugin.oovPosId != 0 {
		t.Error("Default values should match Rust Default implementation")
	}

	// Test Rust set_up behavior
	settings := map[string]any{
		"oovPOS":  []any{"名詞", "数詞", "*", "*", "*", "*"},
		"leftId":  float64(1),
		"rightId": float64(1),
		"cost":    float64(0),
	}

	err := plugin.SetUp(settings, "", zeroGrammar())
	if err != nil {
		t.Fatalf("Rust-compatible SetUp failed: %v", err)
	}

	// Test buffer creation
	buffer := input.NewInputBuffer()
	err = buffer.StartBuild("1")
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}
	err = buffer.Build(zeroGrammar())
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}

	// Test OOV provision
	lat := lattice.New()
	lat.Reset(buffer.CharCount() + 1) // +1 for EOS
	emptyCreatedWords := types.CreatedWords{}

	// Rust: if other_words.not_empty() { return Ok(0); }
	newCreatedWords, err := plugin.ProvideOOV(0, buffer, lat, emptyCreatedWords)
	if err != nil {
		t.Fatalf("Rust-compatible ProvideOOV failed: %v", err)
	}

	// Should create exactly one word (Rust: Ok(1))
	if newCreatedWords.IsEmpty() {
		t.Error("Expected one word to be created (matching Rust Ok(1))")
	}

	if newCreatedWords.HasWord(1) != types.HasWordYes {
		t.Error("Expected created word to have length 1")
	}
}

// BenchmarkSimpleOovPlugin benchmarks plugin performance
func BenchmarkSimpleOovPlugin(b *testing.B) {
	plugin := NewSimpleOovPlugin()
	plugin.SetUp(nil, "", zeroGrammar())

	// Create test buffer
	buffer := input.NewInputBuffer()
	buffer.StartBuild("テスト文字列")
	buffer.Build(zeroGrammar())

	emptyCreatedWords := types.CreatedWords{}
	lat := lattice.New()
	lat.Reset(buffer.CharCount() + 1) // +1 for EOS

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin.ProvideOOV(0, buffer, lat, emptyCreatedWords)
	}
}
