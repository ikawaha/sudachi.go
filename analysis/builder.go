package analysis

import (
	"fmt"

	"github.com/ikawaha/sudachi.go/config"
	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/plugin"

	// Import plugin packages to trigger auto-registration
	_ "github.com/ikawaha/sudachi.go/plugin/input_text"
	_ "github.com/ikawaha/sudachi.go/plugin/oov"
	_ "github.com/ikawaha/sudachi.go/plugin/path_rewrite"
)

// TokenizerBuilder helps build tokenizers with proper plugin configuration
// Updated to use the unified Config structure (matches Rust implementation)
type TokenizerBuilder struct {
	systemDict  *dic.SystemDictionary
	resourceDir string
	config      *config.Config
	debug       bool
}

// NewTokenizerBuilder creates a new tokenizer builder
func NewTokenizerBuilder(systemDict *dic.SystemDictionary) *TokenizerBuilder {
	return &TokenizerBuilder{
		systemDict: systemDict,
	}
}

// SetResourceDir sets the resource directory for loading plugins
func (tb *TokenizerBuilder) SetResourceDir(resourceDir string) *TokenizerBuilder {
	tb.resourceDir = resourceDir
	return tb
}

// SetConfig sets the unified configuration
// Matches Rust: Config structure
func (tb *TokenizerBuilder) SetConfig(config *config.Config) *TokenizerBuilder {
	tb.config = config
	return tb
}

// SetDebug sets the debug flag for the tokenizer builder
func (tb *TokenizerBuilder) SetDebug(debug bool) *TokenizerBuilder {
	tb.debug = debug
	return tb
}

// LoadConfigFromResourceDir loads configuration from resource directory
// Updated to use the unified Config structure (matches Rust: Config::new)
func (tb *TokenizerBuilder) LoadConfigFromResourceDir(resourceDir string) (*TokenizerBuilder, error) {
	// Use the new unified Config system (matches Rust implementation)
	configPath := config.DefaultConfigLocation()
	if resourceDir != "" {
		configPath = resourceDir + "/" + config.DefaultSettingFile
	}

	// Load config using the new unified system
	cfg, err := config.New(&configPath, &resourceDir, nil)
	if err != nil {
		return nil, err
	}

	tb.config = cfg
	tb.resourceDir = resourceDir
	return tb, nil
}

// Build creates a tokenizer with the configured plugins
func (tb *TokenizerBuilder) Build() (*Tokenizer, error) {
	tokenizer, err := NewTokenizer(tb.systemDict)
	if err != nil {
		return nil, fmt.Errorf("failed to create tokenizer: %w", err)
	}

	// Set debug mode on the created tokenizer
	tokenizer.SetDebugMode(tb.debug)

	// Load and set character category (matching Rust implementation)
	if tb.resourceDir != "" {
		err := tb.loadCharacterCategory()
		if err != nil {
			return nil, fmt.Errorf("failed to load character category: %w", err)
		}
	}

	// Configure plugins based on configuration
	if tb.config != nil && tb.resourceDir != "" {
		// Configure input text plugins
		err := tb.configureInputTextPlugins(tokenizer)
		if err != nil {
			return nil, fmt.Errorf("failed to configure input text plugins: %w", err)
		}

		// Configure OOV provider
		err = tb.configureOOVProvider(tokenizer)
		if err != nil {
			return nil, fmt.Errorf("failed to configure OOV provider: %w", err)
		}

		// Configure path rewrite plugins
		err = tb.configurePathRewritePlugins(tokenizer)
		if err != nil {
			return nil, fmt.Errorf("failed to configure path rewrite plugins: %w", err)
		}
	}

	return tokenizer, nil
}

// loadCharacterCategory loads character category from resource directory (matching Rust implementation)
func (tb *TokenizerBuilder) loadCharacterCategory() error {
	// Load character category from char.def file (matching Rust implementation)
	characterCategory, err := dic.LoadCharacterCategoryFromResourceDir(tb.resourceDir)
	if err != nil {
		return err
	}

	// Set character category on grammar (matching Rust grammar.set_character_category)
	tb.systemDict.Grammar().SetCharacterCategory(characterCategory)

	return nil
}

// configureOOVProvider configures the OOV provider based on configuration
func (tb *TokenizerBuilder) configureOOVProvider(tokenizer *Tokenizer) error {
	// Process all OOV provider plugins in order (matching Rust behavior)
	for _, pluginMap := range tb.config.OovProviderPlugins {
		// Create plugin using the registry system
		pluginInterface, err := plugin.CreatePluginFromSettings(plugin.PluginTypeOOVProvider, pluginMap, tb.resourceDir, tb.systemDict.Grammar())
		if err != nil {
			return fmt.Errorf("failed to create OOV provider plugin: %w", err)
		}

		// Type assert to OOVProviderPlugin
		oovPlugin, ok := pluginInterface.(plugin.OOVProviderPlugin)
		if !ok {
			return fmt.Errorf("created plugin is not an OOVProviderPlugin")
		}

		// Add to tokenizer
		tokenizer.GetPluginManager().AddOOVProvider(oovPlugin)
	}

	return nil
}

// configurePathRewritePlugins configures path rewrite plugins based on configuration
func (tb *TokenizerBuilder) configurePathRewritePlugins(tokenizer *Tokenizer) error {
	for _, pluginMap := range tb.config.PathRewritePlugins {
		// Create plugin using the registry system
		pluginInterface, err := plugin.CreatePluginFromSettings(plugin.PluginTypePathRewrite, pluginMap, tb.resourceDir, tb.systemDict.Grammar())
		if err != nil {
			return fmt.Errorf("failed to create path rewrite plugin: %w", err)
		}

		// Type asserts to PathRewritePlugin
		pathRewritePlugin, ok := pluginInterface.(plugin.PathRewritePlugin)
		if !ok {
			return fmt.Errorf("created plugin is not a PathRewritePlugin")
		}

		// Add to tokenizer
		tokenizer.GetPluginManager().AddPathRewriter(pathRewritePlugin)
	}

	return nil
}

// configureInputTextPlugins configures input text plugins based on configuration
func (tb *TokenizerBuilder) configureInputTextPlugins(tokenizer *Tokenizer) error {
	for _, pluginMap := range tb.config.InputTextPlugins {
		// Create plugin using the registry system
		pluginInterface, err := plugin.CreatePluginFromSettings(plugin.PluginTypeInputText, pluginMap, tb.resourceDir, tb.systemDict.Grammar())
		if err != nil {
			return fmt.Errorf("failed to create input text plugin: %w", err)
		}

		// Type assert to InputTextPlugin
		inputTextPlugin, ok := pluginInterface.(plugin.InputTextPlugin)
		if !ok {
			return fmt.Errorf("created plugin is not an InputTextPlugin")
		}

		// Add to tokenizer
		tokenizer.GetPluginManager().AddInputTextPlugin(inputTextPlugin)
	}

	return nil
}

// BuildFromResourceDir creates a tokenizer from resource directory with automatic configuration
func BuildFromResourceDir(systemDict *dic.SystemDictionary, resourceDir string) (*Tokenizer, error) {
	builder := NewTokenizerBuilder(systemDict)

	builder, err := builder.LoadConfigFromResourceDir(resourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from resource directory: %w", err)
	}

	return builder.Build()
}
