package path_rewrite

import (
	"testing"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/lattice"
)

func TestJoinKatakanaOovPlugin_SetUp(t *testing.T) {
	plugin := NewJoinKatakanaOovPlugin()

	// Test with nil settings
	err := plugin.SetUp(nil, "", nil)
	if err != nil {
		t.Fatalf("SetUp failed: %v", err)
	}

	// Should have default minimum length
	if plugin.minLength != 3 {
		t.Errorf("Expected default minLength to be 3, got %d", plugin.minLength)
	}

	// Test with settings
	settings := map[string]any{
		"minLength": 5,
		"oovPOS":    []string{"名詞", "固有名詞", "*", "*", "*", "*"},
	}
	err = plugin.SetUp(settings, "", nil)
	if err != nil {
		t.Fatalf("SetUp with settings failed: %v", err)
	}

	// Verify settings were applied
	if plugin.minLength != 5 {
		t.Errorf("Expected minLength to be 5, got %d", plugin.minLength)
	}

	expectedPOS := []string{"名詞", "固有名詞", "*", "*", "*", "*"}
	if len(plugin.oovPOS) != len(expectedPOS) {
		t.Errorf("Expected POS length %d, got %d", len(expectedPOS), len(plugin.oovPOS))
	}
}

func TestJoinKatakanaOovPlugin_GetName(t *testing.T) {
	plugin := NewJoinKatakanaOovPlugin()
	name := plugin.GetName()

	expected := "JoinKatakanaOovPlugin"
	if name != expected {
		t.Errorf("Expected name '%s', got '%s'", expected, name)
	}
}

func TestJoinKatakanaOovPlugin_IsKatakanaOOV(t *testing.T) {
	plugin := NewJoinKatakanaOovPlugin()

	// Create test buffer
	buffer := input.NewInputBuffer()
	err := buffer.StartBuild("カタカナ")
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}
	err = buffer.Build(zeroGrammar())
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}

	// Create OOV node with katakana surface
	oovNode := lattice.NewNode(0, 4, 0, 0, 0, dic.OOV(1))
	oovResult := lattice.NewNodeResult(
		oovNode,
		"カタカナ",
		[]string{"名詞", "普通名詞", "*", "*", "*", "*"},
		[]string{},
		"カタカナ",
		"カタカナ",
		"カタカナ",
	)

	// Should be identified as katakana OOV
	if !plugin.isKatakanaOOV(oovResult, buffer) {
		t.Error("Expected katakana OOV to be identified")
	}

	// Create non-OOV node
	nonOovNode := lattice.NewNode(0, 4, 0, 0, 0, dic.FromRaw(1))
	nonOovResult := lattice.NewNodeResult(
		nonOovNode,
		"カタカナ",
		[]string{"名詞", "普通名詞", "*", "*", "*", "*"},
		[]string{},
		"カタカナ",
		"カタカナ",
		"カタカナ",
	)

	// Should not be identified as OOV
	if plugin.isKatakanaOOV(nonOovResult, buffer) {
		t.Error("Expected non-OOV node to not be identified as OOV")
	}

	// Create OOV node with hiragana surface
	hiraganaOovNode := lattice.NewNode(0, 4, 0, 0, 0, dic.OOV(1))
	hiraganaOovResult := lattice.NewNodeResult(
		hiraganaOovNode,
		"ひらがな",
		[]string{"名詞", "普通名詞", "*", "*", "*", "*"},
		[]string{},
		"ひらがな",
		"ひらがな",
		"ひらがな",
	)

	// Should not be identified as katakana OOV
	if plugin.isKatakanaOOV(hiraganaOovResult, buffer) {
		t.Error("Expected hiragana OOV to not be identified as katakana OOV")
	}
}

func TestJoinKatakanaOovPlugin_ConcatenateNodes(t *testing.T) {
	plugin := NewJoinKatakanaOovPlugin()

	// Create test nodes
	node1 := lattice.NewNode(0, 2, 0, 0, 0, dic.OOV(1))
	result1 := lattice.NewNodeResult(
		node1,
		"カタ",
		[]string{"名詞", "普通名詞", "*", "*", "*", "*"},
		[]string{},
		"カタ",
		"カタ",
		"カタ",
	)

	node2 := lattice.NewNode(2, 4, 0, 0, 0, dic.OOV(1))
	result2 := lattice.NewNodeResult(
		node2,
		"カナ",
		[]string{"名詞", "普通名詞", "*", "*", "*", "*"},
		[]string{},
		"カナ",
		"カナ",
		"カナ",
	)

	nodes := []*lattice.NodeResult{result1, result2}

	// Test concatenation
	concatenated, err := plugin.concatenateNodes(nodes)
	if err != nil {
		t.Fatalf("Concatenation failed: %v", err)
	}

	if concatenated == nil {
		t.Fatal("Expected concatenated result to not be nil")
	}

	// Check concatenated surface
	expectedSurface := "カタカナ"
	if concatenated.Surface() != expectedSurface {
		t.Errorf("Expected surface '%s', got '%s'", expectedSurface, concatenated.Surface())
	}

	// Check span
	if concatenated.Node().Begin() != 0 {
		t.Errorf("Expected begin position 0, got %d", concatenated.Node().Begin())
	}
	if concatenated.Node().End() != 4 {
		t.Errorf("Expected end position 4, got %d", concatenated.Node().End())
	}

	// Check that it's marked as OOV
	if !concatenated.Node().IsOOV() {
		t.Error("Expected concatenated node to be OOV")
	}
}

func TestJoinKatakanaOovPlugin_Rewrite(t *testing.T) {
	plugin := NewJoinKatakanaOovPlugin()
	err := plugin.SetUp(nil, "", nil)
	if err != nil {
		t.Fatalf("SetUp failed: %v", err)
	}

	// Create test buffer
	buffer := input.NewInputBuffer()
	err = buffer.StartBuild("カタカナ")
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}
	err = buffer.Build(zeroGrammar())
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}

	// Create test lattice
	lat := lattice.New()

	// Create test path with katakana OOV nodes
	node1 := lattice.NewNode(0, 2, 0, 0, 0, dic.OOV(1))
	result1 := lattice.NewNodeResult(
		node1,
		"カタ",
		[]string{"名詞", "普通名詞", "*", "*", "*", "*"},
		[]string{},
		"カタ",
		"カタ",
		"カタ",
	)

	node2 := lattice.NewNode(2, 4, 0, 0, 0, dic.OOV(1))
	result2 := lattice.NewNodeResult(
		node2,
		"カナ",
		[]string{"名詞", "普通名詞", "*", "*", "*", "*"},
		[]string{},
		"カナ",
		"カナ",
		"カナ",
	)

	path := []*lattice.NodeResult{result1, result2}

	// Test rewrite
	result, err := plugin.Rewrite(path, buffer, lat)
	if err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	// Should have concatenated to single node
	if len(result) != 1 {
		t.Errorf("Expected 1 result after concatenation, got %d", len(result))
	}

	if len(result) > 0 {
		expectedSurface := "カタカナ"
		if result[0].Surface() != expectedSurface {
			t.Errorf("Expected concatenated surface '%s', got '%s'", expectedSurface, result[0].Surface())
		}
	}
}

func TestJoinKatakanaOovPlugin_RewriteMinLength(t *testing.T) {
	// Create plugin with minimum length of 5
	plugin := NewJoinKatakanaOovPlugin()
	err := plugin.SetUp(map[string]any{"minLength": float64(5)}, "", nil)
	if err != nil {
		t.Fatalf("SetUp failed: %v", err)
	}

	// Create test buffer
	buffer := input.NewInputBuffer()
	err = buffer.StartBuild("カタ")
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}
	err = buffer.Build(zeroGrammar())
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}

	// Create test lattice
	lat := lattice.New()

	// Create test path with short katakana OOV nodes (total length 2 < minimum 5)
	node1 := lattice.NewNode(0, 2, 0, 0, 0, dic.OOV(1))
	result1 := lattice.NewNodeResult(
		node1,
		"カタ",
		[]string{"名詞", "普通名詞", "*", "*", "*", "*"},
		[]string{},
		"カタ",
		"カタ",
		"カタ",
	)

	path := []*lattice.NodeResult{result1}

	// Test rewrite - should not concatenate due to minimum length
	result, err := plugin.Rewrite(path, buffer, lat)
	if err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	// Should remain as individual node
	if len(result) != 1 {
		t.Errorf("Expected 1 result (no concatenation), got %d", len(result))
	}

	if len(result) > 0 {
		expectedSurface := "カタ"
		if result[0].Surface() != expectedSurface {
			t.Errorf("Expected original surface '%s', got '%s'", expectedSurface, result[0].Surface())
		}
	}
}

// BenchmarkJoinKatakanaOovPlugin benchmarks plugin performance
func BenchmarkJoinKatakanaOovPlugin(b *testing.B) {
	plugin := NewJoinKatakanaOovPlugin()
	plugin.SetUp(nil, "", nil)

	// Create test buffer
	buffer := input.NewInputBuffer()
	buffer.StartBuild("カタカナテスト")
	buffer.Build(zeroGrammar())

	// Create test lattice
	lat := lattice.New()

	// Create test path
	nodes := make([]*lattice.NodeResult, 0, 5)
	for i := 0; i < 5; i++ {
		node := lattice.NewNode(uint16(i*2), uint16((i+1)*2), 0, 0, 0, dic.OOV(1))
		result := lattice.NewNodeResult(
			node,
			"カタ",
			[]string{"名詞", "普通名詞", "*", "*", "*", "*"},
			[]string{},
			"カタ",
			"カタ",
			"カタ",
		)
		nodes = append(nodes, result)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin.Rewrite(nodes, buffer, lat)
	}
}
