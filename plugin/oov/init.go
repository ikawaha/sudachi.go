package oov

import "github.com/ikawaha/sudachi.go/plugin"

// init automatically registers OOV plugins with the global registry
func init() {
	// Register MeCab OOV plugin instance directly (implements PluginFactory interface)
	mecabPlugin := &MeCabOovPlugin{}
	plugin.Register("com.worksap.nlp.sudachi.MeCabOovPlugin", mecabPlugin)

	// Register Simple OOV plugin instance directly (implements PluginFactory interface)
	simplePlugin := &SimpleOovPlugin{}
	plugin.Register("com.worksap.nlp.sudachi.SimpleOovPlugin", simplePlugin)
}
