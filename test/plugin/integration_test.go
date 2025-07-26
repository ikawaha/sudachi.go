package plugin

import (
	"log"
	"testing"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/plugin"

	// Import plugin packages to trigger auto-registration
	_ "github.com/ikawaha/sudachi.go/plugin/oov"
	_ "github.com/ikawaha/sudachi.go/plugin/path_rewrite"
)

func TestPluginAutoRegistration(t *testing.T) {
	// Test that actual plugins are auto-registered
	expectedClasses := []string{
		"com.worksap.nlp.sudachi.MeCabOovPlugin",
		"com.worksap.nlp.sudachi.SimpleOovPlugin",
		"com.worksap.nlp.sudachi.JoinNumericPlugin",
		"com.worksap.nlp.sudachi.JoinKatakanaOovPlugin",
	}

	registeredClasses := plugin.GetRegisteredClasses()

	for _, expectedClass := range expectedClasses {
		found := false
		for _, registeredClass := range registeredClasses {
			if registeredClass == expectedClass {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected plugin class %s to be auto-registered", expectedClass)
		}
	}

	// Verify plugin classes are registered
	for _, className := range expectedClasses {
		if !plugin.IsRegistered(className) {
			t.Errorf("Plugin class %s should be auto-registered", className)
		}
	}
}

func TestPluginCreation(t *testing.T) {
	// Test creating actual plugins through the registry
	testCases := []struct {
		name       string
		className  string
		pluginType plugin.PluginType
		settings   map[string]any
	}{
		{
			name:       "SimpleOovPlugin",
			className:  "com.worksap.nlp.sudachi.SimpleOovPlugin",
			pluginType: plugin.PluginTypeOOVProvider,
			settings: map[string]any{
				"class":   "com.worksap.nlp.sudachi.SimpleOovPlugin",
				"oovPOS":  []any{"補助記号", "一般", "*", "*", "*", "*"},
				"leftId":  float64(5968),
				"rightId": float64(5968),
				"cost":    float64(3857),
			},
		},
		{
			name:       "JoinNumericPlugin",
			className:  "com.worksap.nlp.sudachi.JoinNumericPlugin",
			pluginType: plugin.PluginTypePathRewrite,
			settings: map[string]any{
				"class":           "com.worksap.nlp.sudachi.JoinNumericPlugin",
				"enableNormalize": true,
			},
		},
		{
			name:       "JoinKatakanaOovPlugin",
			className:  "com.worksap.nlp.sudachi.JoinKatakanaOovPlugin",
			pluginType: plugin.PluginTypePathRewrite,
			settings: map[string]any{
				"class":     "com.worksap.nlp.sudachi.JoinKatakanaOovPlugin",
				"minLength": float64(3),
				"oovPOS":    []any{"名詞", "普通名詞", "一般", "*", "*", "*"},
			},
		},
	}

	loader := dic.NewDictionaryLoader()
	dict, err := loader.LoadSystemDictionary("../../resources/system.dic")
	if err != nil {
		log.Fatalf("Failed to load dictionary: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pluginInstance, err := plugin.CreatePlugin(tc.className, tc.pluginType, tc.settings, "", dict)
			if err != nil {
				t.Fatalf("Failed to create plugin %s: %v", tc.className, err)
			}

			if pluginInstance == nil {
				t.Fatalf("Created plugin %s is nil", tc.className)
			}

			// Verify plugin implements the expected interface
			switch tc.pluginType {
			case plugin.PluginTypeOOVProvider:
				if _, ok := pluginInstance.(plugin.OOVProviderPlugin); !ok {
					t.Errorf("Plugin %s does not implement OOVProviderPlugin", tc.className)
				}
			case plugin.PluginTypePathRewrite:
				if _, ok := pluginInstance.(plugin.PathRewritePlugin); !ok {
					t.Errorf("Plugin %s does not implement PathRewritePlugin", tc.className)
				}
			case plugin.PluginTypeInputText:
				if _, ok := pluginInstance.(plugin.InputTextPlugin); !ok {
					t.Errorf("Plugin %s does not implement InputTextPlugin", tc.className)
				}
			}
		})
	}
}

func TestPluginFactoryTypes(t *testing.T) {
	// Test that factories correctly report their supported types
	testCases := []struct {
		className     string
		expectedTypes []plugin.PluginType
	}{
		{
			className:     "com.worksap.nlp.sudachi.SimpleOovPlugin",
			expectedTypes: []plugin.PluginType{plugin.PluginTypeOOVProvider},
		},
		{
			className:     "com.worksap.nlp.sudachi.JoinNumericPlugin",
			expectedTypes: []plugin.PluginType{plugin.PluginTypePathRewrite},
		},
	}

	loader := dic.NewDictionaryLoader()
	dict, err := loader.LoadSystemDictionary("../../resources/system.dic")
	if err != nil {
		log.Fatalf("Failed to load dictionary: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.className, func(t *testing.T) {
			// Try to create different types of plugins
			settings := map[string]any{"class": tc.className}

			for _, pluginType := range []plugin.PluginType{plugin.PluginTypeInputText, plugin.PluginTypeOOVProvider, plugin.PluginTypePathRewrite} {
				_, err := plugin.CreatePlugin(tc.className, pluginType, settings, "", dict)

				shouldSucceed := false
				for _, expectedType := range tc.expectedTypes {
					if pluginType == expectedType {
						shouldSucceed = true
						break
					}
				}

				if shouldSucceed && err != nil {
					t.Errorf("Expected %s to support %s, but got error: %v", tc.className, pluginType, err)
				} else if !shouldSucceed && err == nil {
					t.Errorf("Expected %s to NOT support %s, but creation succeeded", tc.className, pluginType)
				}
			}
		})
	}
}
