package path_rewrite

import "github.com/ikawaha/sudachi.go/plugin"

// init automatically registers path rewrite plugins with the global registry
func init() {
	// Register JoinNumeric plugin instance directly (implements PluginFactory interface)
	joinNumericPlugin := &JoinNumericPlugin{}
	plugin.Register("com.worksap.nlp.sudachi.JoinNumericPlugin", joinNumericPlugin)

	// Register JoinKatakanaOov plugin instance directly (implements PluginFactory interface)
	joinKatakanaPlugin := &JoinKatakanaOovPlugin{}
	plugin.Register("com.worksap.nlp.sudachi.JoinKatakanaOovPlugin", joinKatakanaPlugin)
}
