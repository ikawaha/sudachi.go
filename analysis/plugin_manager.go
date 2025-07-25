package analysis

import (
	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/lattice"
	"github.com/ikawaha/sudachi.go/plugin"
	"github.com/ikawaha/sudachi.go/types"
)

// PluginManager manages all plugins for the tokenizer
type PluginManager struct {
	inputTextPlugins []plugin.InputTextPlugin
	oovProviders     []plugin.OOVProviderPlugin
	pathRewriters    []plugin.PathRewritePlugin
	debug            bool
}

// NewPluginManager creates a new plugin manager
func NewPluginManager() *PluginManager {
	return &PluginManager{
		inputTextPlugins: make([]plugin.InputTextPlugin, 0),
		oovProviders:     make([]plugin.OOVProviderPlugin, 0),
		pathRewriters:    make([]plugin.PathRewritePlugin, 0),
		debug:            false,
	}
}

// SetDebug sets the debug flag for the plugin manager and all plugins
func (pm *PluginManager) SetDebug(debug bool) {
	pm.debug = debug

	// Propagate debug flag to all input text plugins
	for _, plugin := range pm.inputTextPlugins {
		if debuggable, ok := plugin.(interface{ SetDebug(bool) }); ok {
			debuggable.SetDebug(debug)
		}
	}

	// Propagate debug flag to all OOV providers
	for _, plugin := range pm.oovProviders {
		if debuggable, ok := plugin.(interface{ SetDebug(bool) }); ok {
			debuggable.SetDebug(debug)
		}
	}

	// Propagate debug flag to all path rewriters
	for _, plugin := range pm.pathRewriters {
		if debuggable, ok := plugin.(interface{ SetDebug(bool) }); ok {
			debuggable.SetDebug(debug)
		}
	}
}

// AddInputTextPlugin adds an input text plugin
func (pm *PluginManager) AddInputTextPlugin(p plugin.InputTextPlugin) {
	pm.inputTextPlugins = append(pm.inputTextPlugins, p)
	// Set debug flag on the newly added plugin if it supports it
	if debuggable, ok := p.(interface{ SetDebug(bool) }); ok {
		debuggable.SetDebug(pm.debug)
	}
}

// AddOOVProvider adds an OOV provider plugin
func (pm *PluginManager) AddOOVProvider(p plugin.OOVProviderPlugin) {
	pm.oovProviders = append(pm.oovProviders, p)
	// Set debug flag on the newly added plugin if it supports it
	if debuggable, ok := p.(interface{ SetDebug(bool) }); ok {
		debuggable.SetDebug(pm.debug)
	}
}

// AddPathRewriter adds a path rewrite plugin
func (pm *PluginManager) AddPathRewriter(p plugin.PathRewritePlugin) {
	pm.pathRewriters = append(pm.pathRewriters, p)
	// Set debug flag on the newly added plugin if it supports it
	if debuggable, ok := p.(interface{ SetDebug(bool) }); ok {
		debuggable.SetDebug(pm.debug)
	}
}

// ProcessInputText applies all input text plugins to the buffer
func (pm *PluginManager) ProcessInputText(buffer *input.InputBuffer) error {
	for _, v := range pm.inputTextPlugins {
		err := v.Rewrite(buffer)
		if err != nil {
			return err
		}
	}
	return nil
}

// ProvideOOV applies all OOV provider plugins at the given position (matching Rust provide_oovs exactly)
func (pm *PluginManager) ProvideOOV(charPos int, buffer *input.InputBuffer, lat *lattice.Lattice, createdWords types.CreatedWords) (types.CreatedWords, error) {
	// This exactly matches Rust stateful_tokenizer.provide_oovs:
	// fn provide_oovs<P>(&mut self, char_offset: usize, mut other: CreatedWords, plugin: &P) -> SudachiResult<CreatedWords>
	current := createdWords                    // Matches Rust: mut other: CreatedWords
	for _, provider := range pm.oovProviders { // Matches Rust: for provider in self.oov_providers
		var err error
		current, err = provider.ProvideOOV(charPos, buffer, lat, current) // Matches Rust: plugin updates nodes and returns count
		if err != nil {
			return current, err
		}
	}
	return current, nil // Matches Rust: Ok(other)
}

// HasPathRewriters returns true if there are any path rewriters
func (pm *PluginManager) HasPathRewriters() bool {
	return len(pm.pathRewriters) > 0
}

// RewritePath applies all path rewrite plugins to the optimal path
func (pm *PluginManager) RewritePath(path []*lattice.NodeResult, buffer *input.InputBuffer, lat *lattice.Lattice) ([]*lattice.NodeResult, error) {
	currentPath := path

	for _, rewriter := range pm.pathRewriters {
		newPath, err := rewriter.Rewrite(currentPath, buffer, lat)
		if err != nil {
			return nil, err
		}
		currentPath = newPath
	}

	return currentPath, nil
}

// HasInputTextPlugins returns true if there are input text plugins
func (pm *PluginManager) HasInputTextPlugins() bool {
	return len(pm.inputTextPlugins) > 0
}

// HasOOVProviders returns true if there are OOV provider plugins
func (pm *PluginManager) HasOOVProviders() bool {
	return len(pm.oovProviders) > 0
}

// GetLastOOVProvider returns the last OOV provider (matching Rust behavior for fallback)
func (pm *PluginManager) GetLastOOVProvider() plugin.OOVProviderPlugin {
	if len(pm.oovProviders) == 0 {
		return nil
	}
	return pm.oovProviders[len(pm.oovProviders)-1]
}
