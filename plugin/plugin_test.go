package plugin

import (
	"testing"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/lattice"
	"github.com/ikawaha/sudachi.go/types"
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

// Mock implementations for testing interfaces
type mockInputTextPlugin struct{}

func (m *mockInputTextPlugin) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	return nil
}

func (m *mockInputTextPlugin) Rewrite(buffer *input.InputBuffer) error {
	return nil
}

func (m *mockInputTextPlugin) GetName() string {
	return "MockInputTextPlugin"
}

type mockOOVProviderPlugin struct{}

func (m *mockOOVProviderPlugin) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	return nil
}

func (m *mockOOVProviderPlugin) ProvideOOV(charPos int, buffer *input.InputBuffer, lattice *lattice.Lattice, createdWords CreatedWords) (CreatedWords, error) {
	return createdWords, nil
}

func (m *mockOOVProviderPlugin) GetName() string {
	return "MockOOVProviderPlugin"
}

type mockPathRewritePlugin struct{}

func (m *mockPathRewritePlugin) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	return nil
}

func (m *mockPathRewritePlugin) Rewrite(path []*lattice.NodeResult, buffer *input.InputBuffer, lat *lattice.Lattice) ([]*lattice.NodeResult, error) {
	return path, nil
}

func (m *mockPathRewritePlugin) GetName() string {
	return "MockPathRewritePlugin"
}

func TestPluginInterfaces(t *testing.T) {
	// Test that plugin interfaces work correctly

	// InputTextPlugin interface
	var inputPlugin InputTextPlugin = &mockInputTextPlugin{}
	if inputPlugin.GetName() == "" {
		t.Error("InputTextPlugin should have a name")
	}

	// Set up the plugin
	err := inputPlugin.SetUp(nil, "", zeroGrammar())
	if err != nil {
		t.Errorf("InputTextPlugin SetUp failed: %v", err)
	}

	// OOVProviderPlugin interface
	var oovPlugin OOVProviderPlugin = &mockOOVProviderPlugin{}
	if oovPlugin.GetName() == "" {
		t.Error("OOVProviderPlugin should have a name")
	}

	// Set up the plugin
	err = oovPlugin.SetUp(nil, "", zeroGrammar())
	if err != nil {
		t.Errorf("OOVProviderPlugin SetUp failed: %v", err)
	}

	// PathRewritePlugin interface
	var pathPlugin PathRewritePlugin = &mockPathRewritePlugin{}
	if pathPlugin.GetName() == "" {
		t.Error("PathRewritePlugin should have a name")
	}

	// Set up the plugin
	err = pathPlugin.SetUp(nil, "", zeroGrammar())
	if err != nil {
		t.Errorf("PathRewritePlugin SetUp failed: %v", err)
	}
}

func TestMockPluginIntegration(t *testing.T) {
	// Test mock plugin integration to verify interfaces work correctly

	// Test InputTextPlugin
	inputPlugin := &mockInputTextPlugin{}
	buffer := input.NewInputBuffer()
	err := buffer.StartBuild("テスト")
	if err != nil {
		t.Fatalf("Failed to start build: %v", err)
	}

	err = inputPlugin.Rewrite(buffer)
	if err != nil {
		t.Errorf("InputTextPlugin rewrite failed: %v", err)
	}

	// Test OOVProviderPlugin
	oovPlugin := &mockOOVProviderPlugin{}
	lat := lattice.New()
	createdWords := types.CreatedWords{}

	newCreatedWords, err := oovPlugin.ProvideOOV(0, buffer, lat, createdWords)
	if err != nil {
		t.Errorf("OOVProviderPlugin ProvideOOV failed: %v", err)
	}

	if newCreatedWords != createdWords {
		t.Error("Expected CreatedWords to be unchanged by mock plugin")
	}

	// Test PathRewritePlugin
	pathPlugin := &mockPathRewritePlugin{}
	emptyPath := make([]*lattice.NodeResult, 0)

	rewrittenPath, err := pathPlugin.Rewrite(emptyPath, buffer, lat)
	if err != nil {
		t.Errorf("PathRewritePlugin rewrite failed: %v", err)
	}

	if len(rewrittenPath) != len(emptyPath) {
		t.Error("Expected path to be unchanged by mock plugin")
	}
}

func TestCreatedWordsCompatibility(t *testing.T) {
	// Test that CreatedWords type alias works correctly
	var cw CreatedWords

	// Test basic operations
	if !cw.IsEmpty() {
		t.Error("New CreatedWords should be empty")
	}

	// Test adding words
	cw2 := cw.AddWord(5)
	if cw2.IsEmpty() {
		t.Error("CreatedWords should not be empty after adding word")
	}

	if cw2.HasWord(5) != types.HasWordYes {
		t.Error("CreatedWords should have word of length 5")
	}

	if cw2.HasWord(3) == types.HasWordYes {
		t.Error("CreatedWords should not have word of length 3")
	}
}

func TestPluginConfiguration(t *testing.T) {
	// Test plugin configuration with settings using mock plugins

	// Test InputTextPlugin with settings
	inputPlugin := &mockInputTextPlugin{}
	settings := map[string]any{
		"someOption": true,
	}
	err := inputPlugin.SetUp(settings, "/test/path", zeroGrammar())
	if err != nil {
		t.Errorf("InputTextPlugin SetUp with settings failed: %v", err)
	}

	// Test OOVProviderPlugin with settings
	oovPlugin := &mockOOVProviderPlugin{}
	err = oovPlugin.SetUp(settings, "/test/path", zeroGrammar())
	if err != nil {
		t.Errorf("OOVProviderPlugin SetUp with settings failed: %v", err)
	}

	// Test PathRewritePlugin with settings
	pathPlugin := &mockPathRewritePlugin{}
	err = pathPlugin.SetUp(settings, "/test/path", zeroGrammar())
	if err != nil {
		t.Errorf("PathRewritePlugin SetUp with settings failed: %v", err)
	}
}

// BenchmarkPluginPerformance benchmarks plugin performance
func BenchmarkPluginPerformance(b *testing.B) {
	// Setup mock plugins
	inputPlugin := &mockInputTextPlugin{}
	inputPlugin.SetUp(nil, "", zeroGrammar())

	oovPlugin := &mockOOVProviderPlugin{}
	oovPlugin.SetUp(nil, "", zeroGrammar())

	pathPlugin := &mockPathRewritePlugin{}
	pathPlugin.SetUp(nil, "", zeroGrammar())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test input text plugin
		buffer := input.NewInputBuffer()
		buffer.StartBuild("テスト")
		inputPlugin.Rewrite(buffer)
		buffer.Build(zeroGrammar())

		// Test OOV plugin
		lat := lattice.New()
		createdWords := types.CreatedWords{}
		oovPlugin.ProvideOOV(0, buffer, lat, createdWords)

		// Test path rewrite plugin
		emptyPath := make([]*lattice.NodeResult, 0)
		pathPlugin.Rewrite(emptyPath, buffer, lat)
	}
}
