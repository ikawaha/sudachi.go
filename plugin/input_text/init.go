package input_text

import "github.com/ikawaha/sudachi.go/plugin"

// init automatically registers input text plugins with the global registry
func init() {
	// Register DefaultInputTextPlugin instance directly (implements PluginFactory interface)
	defaultPlugin := &DefaultInputTextPlugin{}
	plugin.Register("com.worksap.nlp.sudachi.DefaultInputTextPlugin", defaultPlugin)

	// Register ProlongedSoundMarkPlugin instance directly (implements PluginFactory interface)
	prolongedPlugin := &ProlongedSoundMarkPlugin{}
	plugin.Register("com.worksap.nlp.sudachi.ProlongedSoundMarkPlugin", prolongedPlugin)

	// Register IgnoreYomiganaPlugin instance directly (implements PluginFactory interface)
	yomiganaPlugin := &IgnoreYomiganaPlugin{}
	plugin.Register("com.worksap.nlp.sudachi.IgnoreYomiganaPlugin", yomiganaPlugin)
}
