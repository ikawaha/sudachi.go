package plugin

import (
	"fmt"
	"sync"

	"github.com/ikawaha/sudachi.go/dic"
)

// PluginType represents the type of plugin
type PluginType string

const (
	// PluginTypeInputText represents input text processing plugins
	PluginTypeInputText PluginType = "input_text"
	// PluginTypeOOVProvider represents OOV provider plugins
	PluginTypeOOVProvider PluginType = "oov_provider"
	// PluginTypePathRewrite represents path rewrite plugins
	PluginTypePathRewrite PluginType = "path_rewrite"
)

// PluginFactory creates plugins based on configuration
type PluginFactory interface {
	// CreateInputTextPlugin creates an input text plugin
	CreateInputTextPlugin(settings map[string]any, resourceDir string, grammar *dic.Grammar) (InputTextPlugin, error)

	// CreateOOVProvider creates an OOV provider plugin
	CreateOOVProvider(settings map[string]any, resourceDir string, grammar *dic.Grammar) (OOVProviderPlugin, error)

	// CreatePathRewriter creates a path rewrite plugin
	CreatePathRewriter(settings map[string]any, resourceDir string, grammar *dic.Grammar) (PathRewritePlugin, error)

	// GetSupportedTypes returns the types of plugins this factory can create
	GetSupportedTypes() []PluginType
}

// Registry manages plugin factories and creation
type Registry struct {
	mu        sync.RWMutex
	factories map[string]PluginFactory
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]PluginFactory),
	}
}

// Register registers a plugin factory for a given class name
func (r *Registry) Register(className string, factory PluginFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[className] = factory
}

// CreatePlugin creates a plugin instance based on class name and type
func (r *Registry) CreatePlugin(className string, pluginType PluginType, settings map[string]any, resourceDir string, grammar *dic.Grammar) (any, error) {
	r.mu.RLock()
	factory, exists := r.factories[className]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown plugin class: %s", className)
	}

	// Check if factory supports the requested plugin type
	supportedTypes := factory.GetSupportedTypes()
	var typeSupported bool
	for _, supportedType := range supportedTypes {
		if supportedType == pluginType {
			typeSupported = true
			break
		}
	}

	if !typeSupported {
		return nil, fmt.Errorf("plugin class %s does not support type %s", className, pluginType)
	}

	switch pluginType {
	case PluginTypeInputText:
		return factory.CreateInputTextPlugin(settings, resourceDir, grammar)
	case PluginTypeOOVProvider:
		return factory.CreateOOVProvider(settings, resourceDir, grammar)
	case PluginTypePathRewrite:
		return factory.CreatePathRewriter(settings, resourceDir, grammar)
	default:
		return nil, fmt.Errorf("unsupported plugin type: %s", pluginType)
	}
}

// GetRegisteredClasses returns all registered plugin class names
func (r *Registry) GetRegisteredClasses() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	classes := make([]string, 0, len(r.factories))
	for className := range r.factories {
		classes = append(classes, className)
	}
	return classes
}

// IsRegistered checks if a plugin class is registered
func (r *Registry) IsRegistered(className string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.factories[className]
	return exists
}

// Global registry instance
var globalRegistry = NewRegistry()

// Register registers a plugin factory in the global registry
func Register(className string, factory PluginFactory) {
	globalRegistry.Register(className, factory)
}

// CreatePlugin creates a plugin using the global registry
func CreatePlugin(className string, pluginType PluginType, settings map[string]any, resourceDir string, grammar *dic.Grammar) (any, error) {
	return globalRegistry.CreatePlugin(className, pluginType, settings, resourceDir, grammar)
}

// CreatePluginFromSettings creates a plugin from settings (automatically extracts class name)
func CreatePluginFromSettings(pluginType PluginType, settings map[string]any, resourceDir string, grammar *dic.Grammar) (any, error) {
	return globalRegistry.CreatePluginFromSettings(pluginType, settings, resourceDir, grammar)
}

// GetRegisteredClasses returns all registered plugin classes from the global registry
func GetRegisteredClasses() []string {
	return globalRegistry.GetRegisteredClasses()
}

// IsRegistered checks if a plugin class is registered in the global registry
func IsRegistered(className string) bool {
	return globalRegistry.IsRegistered(className)
}

// CreatePluginFromSettings creates a plugin instance from settings (automatically extracts class name)
func (r *Registry) CreatePluginFromSettings(pluginType PluginType, settings map[string]any, resourceDir string, grammar *dic.Grammar) (any, error) {
	// Extract class name from settings
	className, ok := settings["class"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid class field in plugin settings")
	}

	return r.CreatePlugin(className, pluginType, settings, resourceDir, grammar)
}
