package input_text

import (
	"testing"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
)

// zeroGrammarForProlongedSoundMark creates a minimal grammar for testing (matching Rust zero_grammar)
func zeroGrammarForProlongedSoundMark() *dic.Grammar {
	zeroBytes := make([]byte, 6)
	grammar, err := dic.NewGrammar(zeroBytes, 0)
	if err != nil {
		panic("Failed to create zero grammar: " + err.Error())
	}
	return grammar
}

// buildMockSetting creates test settings (matching Rust build_mock_setting)
func buildMockSetting() map[string]any {
	return map[string]any{
		"prolongedSoundMarks": []any{
			"ー",
			"〜",
			"〰",
		},
	}
}

// setupProlongedSoundMarkPlugin creates plugin for testing (matching Rust setup function)
func setupProlongedSoundMarkPlugin() (*dic.Grammar, *ProlongedSoundMarkPlugin) {
	settings := buildMockSetting()
	grammar := zeroGrammarForProlongedSoundMark()
	plugin := NewProlongedSoundMarkPlugin()

	err := plugin.SetUp(settings, "", grammar)
	if err != nil {
		panic("Failed to setup plugin: " + err.Error())
	}

	return grammar, plugin
}

func TestProlongedSoundMarkPlugin_NewProlongedSoundMarkPlugin(t *testing.T) {
	plugin := NewProlongedSoundMarkPlugin()

	if plugin == nil {
		t.Fatal("NewProlongedSoundMarkPlugin should not return nil")
	}

	// Check default values
	if plugin.replaceSymbol != "ー" {
		t.Errorf("Expected default replaceSymbol to be 'ー', got '%s'", plugin.replaceSymbol)
	}

	if plugin.regex != nil {
		t.Error("Expected regex to be nil before setup")
	}
}

func TestProlongedSoundMarkPlugin_GetName(t *testing.T) {
	plugin := NewProlongedSoundMarkPlugin()
	name := plugin.GetName()

	expected := "ProlongedSoundMarkPlugin"
	if name != expected {
		t.Errorf("Expected name '%s', got '%s'", expected, name)
	}
}

func TestProlongedSoundMarkPlugin_SetUp(t *testing.T) {
	plugin := NewProlongedSoundMarkPlugin()

	// Test with nil settings (should use defaults)
	err := plugin.SetUp(nil, "", zeroGrammarForProlongedSoundMark())
	if err != nil {
		t.Fatalf("SetUp with nil settings failed: %v", err)
	}

	// Should have default marks
	if len(plugin.psmSet) == 0 {
		t.Error("Expected default prolonged sound marks to be set")
	}

	// Should have compiled regex
	if plugin.regex == nil {
		t.Error("Expected regex to be compiled after setup")
	}

	// Test with valid settings
	settings := map[string]any{
		"prolongedSoundMarks": []any{"ー", "〜"},
		"replacementSymbol":   "ー",
	}

	plugin2 := NewProlongedSoundMarkPlugin()
	err = plugin2.SetUp(settings, "", zeroGrammarForProlongedSoundMark())
	if err != nil {
		t.Fatalf("SetUp with valid settings failed: %v", err)
	}

	// Verify settings were applied
	if len(plugin2.psmSet) != 2 {
		t.Errorf("Expected 2 prolonged sound marks, got %d", len(plugin2.psmSet))
	}

	if !plugin2.psmSet['ー'] || !plugin2.psmSet['〜'] {
		t.Error("Expected configured prolonged sound marks to be set")
	}
}

// TestCombineContinuousProlongedSoundMark matches Rust combine_continuous_prolonged_sound_mark test
func TestCombineContinuousProlongedSoundMark(t *testing.T) {
	original := "ゴーール"
	normalized := "ゴール"

	_, plugin := setupProlongedSoundMarkPlugin()
	buffer := input.NewInputBuffer()

	err := buffer.StartBuild(original)
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}

	err = plugin.Rewrite(buffer)
	if err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	err = buffer.Build(zeroGrammarForProlongedSoundMark())
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}

	if buffer.Original() != original {
		t.Errorf("Expected original '%s', got '%s'", original, buffer.Original())
	}

	if buffer.Modified() != normalized {
		t.Errorf("Expected normalized '%s', got '%s'", normalized, buffer.Modified())
	}

	// Test byte length (matching Rust test)
	expectedBytes := []byte("\xe3\x82\xb4\xe3\x83\xbc\xe3\x83\xab")
	if string(expectedBytes) != normalized {
		t.Errorf("Expected normalized text to match expected bytes")
	}
}

// TestCombinedContinuousProlongedSoundMarksAtEnd matches Rust test
func TestCombinedContinuousProlongedSoundMarksAtEnd(t *testing.T) {
	original := "スーパーー"
	normalized := "スーパー"

	_, plugin := setupProlongedSoundMarkPlugin()
	buffer := input.NewInputBuffer()

	err := buffer.StartBuild(original)
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}

	err = plugin.Rewrite(buffer)
	if err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	err = buffer.Build(zeroGrammarForProlongedSoundMark())
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}

	if buffer.Original() != original {
		t.Errorf("Expected original '%s', got '%s'", original, buffer.Original())
	}

	if buffer.Modified() != normalized {
		t.Errorf("Expected normalized '%s', got '%s'", normalized, buffer.Modified())
	}

	// Test byte length (matching Rust test)
	expectedBytes := []byte("\xe3\x82\xb9\xe3\x83\xbc\xe3\x83\x91\xe3\x83\xbc")
	if string(expectedBytes) != normalized {
		t.Errorf("Expected normalized text to match expected bytes")
	}
}

// TestCombineContinuousProlongedSoundMarksMultiTimes matches Rust test
func TestCombineContinuousProlongedSoundMarksMultiTimes(t *testing.T) {
	original := "エーービーーーシーーーー"
	normalized := "エービーシー"

	_, plugin := setupProlongedSoundMarkPlugin()
	buffer := input.NewInputBuffer()

	err := buffer.StartBuild(original)
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}

	err = plugin.Rewrite(buffer)
	if err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	err = buffer.Build(zeroGrammarForProlongedSoundMark())
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}

	if buffer.Original() != original {
		t.Errorf("Expected original '%s', got '%s'", original, buffer.Original())
	}

	if buffer.Modified() != normalized {
		t.Errorf("Expected normalized '%s', got '%s'", normalized, buffer.Modified())
	}

	// Test byte length (matching Rust test)
	expectedBytes := []byte("\xe3\x82\xa8\xe3\x83\xbc\xe3\x83\x93\xe3\x83\xbc\xe3\x82\xb7\xe3\x83\xbc")
	if string(expectedBytes) != normalized {
		t.Errorf("Expected normalized text to match expected bytes")
	}
}

// TestCombineContinuousProlongedSoundMarksMultiSymbolTypes matches Rust test
func TestCombineContinuousProlongedSoundMarksMultiSymbolTypes(t *testing.T) {
	original := "エーービ〜〜〜シ〰〰〰〰"
	normalized := "エービーシー"

	_, plugin := setupProlongedSoundMarkPlugin()
	buffer := input.NewInputBuffer()

	err := buffer.StartBuild(original)
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}

	err = plugin.Rewrite(buffer)
	if err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	err = buffer.Build(zeroGrammarForProlongedSoundMark())
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}

	if buffer.Original() != original {
		t.Errorf("Expected original '%s', got '%s'", original, buffer.Original())
	}

	if buffer.Modified() != normalized {
		t.Errorf("Expected normalized '%s', got '%s'", normalized, buffer.Modified())
	}

	// Test byte length (matching Rust test)
	expectedBytes := []byte("\xe3\x82\xa8\xe3\x83\xbc\xe3\x83\x93\xe3\x83\xbc\xe3\x82\xb7\xe3\x83\xbc")
	if string(expectedBytes) != normalized {
		t.Errorf("Expected normalized text to match expected bytes")
	}
}

// TestCombineContinuousProlongedSoundMarksMultiMixedSymbolTypes matches Rust test
func TestCombineContinuousProlongedSoundMarksMultiMixedSymbolTypes(t *testing.T) {
	original := "エー〜ビ〜〰ーシ〰ー〰〜"
	normalized := "エービーシー"

	_, plugin := setupProlongedSoundMarkPlugin()
	buffer := input.NewInputBuffer()

	err := buffer.StartBuild(original)
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}

	err = plugin.Rewrite(buffer)
	if err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	err = buffer.Build(zeroGrammarForProlongedSoundMark())
	if err != nil {
		t.Fatalf("Failed to build buffer: %v", err)
	}

	if buffer.Original() != original {
		t.Errorf("Expected original '%s', got '%s'", original, buffer.Original())
	}

	if buffer.Modified() != normalized {
		t.Errorf("Expected normalized '%s', got '%s'", normalized, buffer.Modified())
	}

	// Test byte length (matching Rust test)
	expectedBytes := []byte("\xe3\x82\xa8\xe3\x83\xbc\xe3\x83\x93\xe3\x83\xbc\xe3\x82\xb7\xe3\x83\xbc")
	if string(expectedBytes) != normalized {
		t.Errorf("Expected normalized text to match expected bytes")
	}
}

func TestProlongedSoundMarkPlugin_RewriteImpl(t *testing.T) {
	_, plugin := setupProlongedSoundMarkPlugin()

	testCases := []struct {
		name     string
		input    string
		expected string
		changed  bool
	}{
		{
			name:     "Basic consecutive marks",
			input:    "ゴーール",
			expected: "ゴール",
			changed:  true,
		},
		{
			name:     "No consecutive marks",
			input:    "ゴール",
			expected: "ゴール",
			changed:  false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
			changed:  false,
		},
		{
			name:     "Multiple types",
			input:    "エー〜ビ〜〰ーシ〰ー〰〜",
			expected: "エービーシー",
			changed:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, changed := plugin.RewriteImpl(tc.input)

			if result != tc.expected {
				t.Errorf("Expected result '%s', got '%s'", tc.expected, result)
			}

			if changed != tc.changed {
				t.Errorf("Expected changed %v, got %v", tc.changed, changed)
			}
		})
	}
}

// BenchmarkProlongedSoundMarkPlugin benchmarks plugin performance
func BenchmarkProlongedSoundMarkPlugin(b *testing.B) {
	_, plugin := setupProlongedSoundMarkPlugin()
	input := "エーーーービーーーーシーーーー"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin.RewriteImpl(input)
	}
}
