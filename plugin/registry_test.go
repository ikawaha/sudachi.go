package plugin

import (
	"testing"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/lattice"
)

// MockPluginFactory is a mock implementation of PluginFactory for testing
type MockPluginFactory struct {
	supportedTypes []PluginType
}

func (f *MockPluginFactory) CreateInputTextPlugin(settings map[string]any, resourceDir string, grammar *dic.Grammar) (InputTextPlugin, error) {
	return &MockInputTextPlugin{}, nil
}

func (f *MockPluginFactory) CreateOOVProvider(settings map[string]any, resourceDir string, grammar *dic.Grammar) (OOVProviderPlugin, error) {
	return &MockOOVProviderPlugin{}, nil
}

func (f *MockPluginFactory) CreatePathRewriter(settings map[string]any, resourceDir string, grammar *dic.Grammar) (PathRewritePlugin, error) {
	return &MockPathRewritePlugin{}, nil
}

func (f *MockPluginFactory) GetSupportedTypes() []PluginType {
	return f.supportedTypes
}

// Mock plugin implementations
type MockInputTextPlugin struct{}

func (p *MockInputTextPlugin) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	return nil
}

func (p *MockInputTextPlugin) Rewrite(buffer *input.InputBuffer) error {
	return nil
}

func (p *MockInputTextPlugin) GetName() string {
	return "MockInputTextPlugin"
}

type MockOOVProviderPlugin struct{}

func (p *MockOOVProviderPlugin) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	return nil
}

func (p *MockOOVProviderPlugin) ProvideOOV(charPos int, buffer *input.InputBuffer, lat *lattice.Lattice, createdWords CreatedWords) (CreatedWords, error) {
	return createdWords, nil
}

func (p *MockOOVProviderPlugin) GetName() string {
	return "MockOOVProviderPlugin"
}

type MockPathRewritePlugin struct{}

func (p *MockPathRewritePlugin) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	return nil
}

func (p *MockPathRewritePlugin) Rewrite(path []*lattice.NodeResult, buffer *input.InputBuffer, lat *lattice.Lattice) ([]*lattice.NodeResult, error) {
	return path, nil
}

func (p *MockPathRewritePlugin) GetName() string {
	return "MockPathRewritePlugin"
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	factory := &MockPluginFactory{
		supportedTypes: []PluginType{PluginTypeOOVProvider},
	}

	className := "test.plugin.TestPlugin"
	registry.Register(className, factory)

	if !registry.IsRegistered(className) {
		t.Errorf("Plugin class %s should be registered", className)
	}

	classes := registry.GetRegisteredClasses()
	found := false
	for _, class := range classes {
		if class == className {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Plugin class %s should be in registered classes list", className)
	}
}

func TestRegistry_CreatePlugin(t *testing.T) {
	registry := NewRegistry()

	factory := &MockPluginFactory{
		supportedTypes: []PluginType{PluginTypeOOVProvider},
	}

	className := "test.plugin.TestPlugin"
	registry.Register(className, factory)

	settings := map[string]any{
		"class": className,
	}

	// Test successful creation
	plugin, err := registry.CreatePlugin(className, PluginTypeOOVProvider, settings, "", nil)
	if err != nil {
		t.Fatalf("Failed to create plugin: %v", err)
	}

	if plugin == nil {
		t.Fatal("Created plugin is nil")
	}

	// Verify plugin type
	oovPlugin, ok := plugin.(OOVProviderPlugin)
	if !ok {
		t.Fatal("Created plugin is not an OOVProviderPlugin")
	}

	if oovPlugin.GetName() != "MockOOVProviderPlugin" {
		t.Errorf("Expected plugin name 'MockOOVProviderPlugin', got '%s'", oovPlugin.GetName())
	}
}

func TestRegistry_CreatePlugin_UnsupportedType(t *testing.T) {
	registry := NewRegistry()

	factory := &MockPluginFactory{
		supportedTypes: []PluginType{PluginTypeOOVProvider}, // Only supports OOV
	}

	className := "test.plugin.TestPlugin"
	registry.Register(className, factory)

	settings := map[string]any{
		"class": className,
	}

	// Try to create unsupported type
	_, err := registry.CreatePlugin(className, PluginTypeInputText, settings, "", nil)
	if err == nil {
		t.Error("Expected error when creating unsupported plugin type")
	}
}

func TestRegistry_CreatePlugin_UnknownClass(t *testing.T) {
	registry := NewRegistry()

	settings := map[string]any{
		"class": "unknown.plugin.Class",
	}

	_, err := registry.CreatePlugin("unknown.plugin.Class", PluginTypeOOVProvider, settings, "", nil)
	if err == nil {
		t.Error("Expected error when creating plugin with unknown class")
	}
}

func TestGlobalRegistry(t *testing.T) {
	// Test global registry functions
	factory := &MockPluginFactory{
		supportedTypes: []PluginType{PluginTypePathRewrite},
	}

	className := "test.global.Plugin"
	Register(className, factory)

	if !IsRegistered(className) {
		t.Errorf("Plugin class %s should be registered in global registry", className)
	}

	classes := GetRegisteredClasses()
	found := false
	for _, class := range classes {
		if class == className {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Plugin class %s should be in global registered classes list", className)
	}

	settings := map[string]any{
		"class": className,
	}

	plugin, err := CreatePlugin(className, PluginTypePathRewrite, settings, "", nil)
	if err != nil {
		t.Fatalf("Failed to create plugin using global registry: %v", err)
	}

	if plugin == nil {
		t.Fatal("Created plugin using global registry is nil")
	}
}
