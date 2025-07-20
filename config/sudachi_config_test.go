package config

import (
	"strings"
	"testing"
)

// TestPluginConfig_IsMeCabOovPlugin tests MeCab OOV plugin identification
func TestPluginConfig_IsMeCabOovPlugin(t *testing.T) {
	tests := []struct {
		name     string
		config   *SudachiConfig
		expected bool
	}{
		{
			name: "MeCab OOV plugin",
			config: &SudachiConfig{
				OOVProviderPlugin: []PluginConfig{
					{Class: PluginClassMeCabOov},
				},
			},
			expected: true,
		},
		{
			name: "Simple OOV plugin",
			config: &SudachiConfig{
				OOVProviderPlugin: []PluginConfig{
					{Class: PluginClassSimpleOov},
				},
			},
			expected: false,
		},
		{
			name: "No OOV plugins",
			config: &SudachiConfig{
				OOVProviderPlugin: []PluginConfig{},
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.config.IsMeCabOovPlugin()
			if result != test.expected {
				t.Errorf("Expected %v, got %v", test.expected, result)
			}
		})
	}
}

// TestPluginConfig_IsSimpleOovPlugin tests Simple OOV plugin identification
func TestPluginConfig_IsSimpleOovPlugin(t *testing.T) {
	tests := []struct {
		name     string
		config   *SudachiConfig
		expected bool
	}{
		{
			name: "Simple OOV plugin",
			config: &SudachiConfig{
				OOVProviderPlugin: []PluginConfig{
					{Class: PluginClassSimpleOov},
				},
			},
			expected: true,
		},
		{
			name: "MeCab OOV plugin",
			config: &SudachiConfig{
				OOVProviderPlugin: []PluginConfig{
					{Class: PluginClassMeCabOov},
				},
			},
			expected: false,
		},
		{
			name: "No OOV plugins",
			config: &SudachiConfig{
				OOVProviderPlugin: []PluginConfig{},
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.config.IsSimpleOovPlugin()
			if result != test.expected {
				t.Errorf("Expected %v, got %v", test.expected, result)
			}
		})
	}
}

// TestPluginConfig_GetOOVProviderPluginClass tests OOV provider plugin class retrieval
func TestPluginConfig_GetOOVProviderPluginClass(t *testing.T) {
	tests := []struct {
		name     string
		config   *SudachiConfig
		expected string
	}{
		{
			name: "With MeCab OOV plugin",
			config: &SudachiConfig{
				OOVProviderPlugin: []PluginConfig{
					{Class: PluginClassMeCabOov},
					{Class: PluginClassSimpleOov}, // Second should be ignored
				},
			},
			expected: PluginClassMeCabOov,
		},
		{
			name: "With Simple OOV plugin",
			config: &SudachiConfig{
				OOVProviderPlugin: []PluginConfig{
					{Class: PluginClassSimpleOov},
				},
			},
			expected: PluginClassSimpleOov,
		},
		{
			name: "No OOV plugins",
			config: &SudachiConfig{
				OOVProviderPlugin: []PluginConfig{},
			},
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.config.GetOOVProviderPluginClass()
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}

// TestPluginConfig_GetOOVProviderPluginConfig tests OOV provider plugin config retrieval
func TestPluginConfig_GetOOVProviderPluginConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        *SudachiConfig
		expectedNil   bool
		expectedClass string
	}{
		{
			name: "With plugin",
			config: &SudachiConfig{
				OOVProviderPlugin: []PluginConfig{
					{
						Class: PluginClassMeCabOov,
						Settings: map[string]any{
							"setting1": "value1",
						},
					},
				},
			},
			expectedNil:   false,
			expectedClass: PluginClassMeCabOov,
		},
		{
			name: "No plugins",
			config: &SudachiConfig{
				OOVProviderPlugin: []PluginConfig{},
			},
			expectedNil: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.config.GetOOVProviderPluginConfig()

			if test.expectedNil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
			} else {
				if result == nil {
					t.Errorf("Expected non-nil result")
				} else if result.Class != test.expectedClass {
					t.Errorf("Expected class '%s', got '%s'", test.expectedClass, result.Class)
				}
			}
		})
	}
}

// TestPluginConfig_JSONParsing tests basic JSON parsing
func TestPluginConfig_JSONParsing(t *testing.T) {
	// Test basic JSON parsing for plugin config
	config, err := LoadConfigFromReader(strings.NewReader(`{
		"systemDict": "system.dic",
		"characterDefinitionFile": "char.def",
		"oovProviderPlugin": [
			{
				"class": "com.worksap.nlp.sudachi.SimpleOovPlugin"
			}
		]
	}`))

	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(config.OOVProviderPlugin) != 1 {
		t.Fatalf("Expected 1 OOV provider plugin, got %d", len(config.OOVProviderPlugin))
	}

	plugin := config.OOVProviderPlugin[0]

	// Check class field
	if plugin.Class != PluginClassSimpleOov {
		t.Errorf("Expected class '%s', got '%s'", PluginClassSimpleOov, plugin.Class)
	}

	// Settings should be initialized (even if empty) with custom UnmarshalJSON
	if plugin.Settings == nil {
		t.Errorf("Expected Settings to be initialized, got nil")
	}
}

// TestPluginConfig_JSONInline tests the inline JSON behavior with custom UnmarshalJSON
func TestPluginConfig_JSONInline(t *testing.T) {
	// Test JSON string that should be parsed with inline settings
	config, err := LoadConfigFromReader(strings.NewReader(`{
		"systemDict": "system.dic",
		"characterDefinitionFile": "char.def",
		"oovProviderPlugin": [
			{
				"class": "com.worksap.nlp.sudachi.SimpleOovPlugin",
				"oovPOS": ["名詞", "普通名詞", "一般", "*", "*", "*"],
				"leftId": 0,
				"rightId": 0,
				"cost": 30000
			}
		]
	}`))

	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(config.OOVProviderPlugin) != 1 {
		t.Fatalf("Expected 1 OOV provider plugin, got %d", len(config.OOVProviderPlugin))
	}

	plugin := config.OOVProviderPlugin[0]

	// Check class field
	if plugin.Class != PluginClassSimpleOov {
		t.Errorf("Expected class '%s', got '%s'", PluginClassSimpleOov, plugin.Class)
	}

	// Check that settings were properly inlined
	if plugin.Settings == nil {
		t.Fatalf("Expected Settings to be initialized, got nil")
	}

	if plugin.Settings["leftId"] != float64(0) { // JSON numbers are float64
		t.Errorf("Expected leftId 0, got %v", plugin.Settings["leftId"])
	}

	if plugin.Settings["rightId"] != float64(0) {
		t.Errorf("Expected rightId 0, got %v", plugin.Settings["rightId"])
	}

	if plugin.Settings["cost"] != float64(30000) {
		t.Errorf("Expected cost 30000, got %v", plugin.Settings["cost"])
	}

	// Check array setting
	oovPOS, ok := plugin.Settings["oovPOS"].([]interface{})
	if !ok {
		t.Errorf("Expected oovPOS to be array, got %T", plugin.Settings["oovPOS"])
	} else if len(oovPOS) != 6 {
		t.Errorf("Expected oovPOS array length 6, got %d", len(oovPOS))
	} else {
		// Check first element
		if oovPOS[0] != "名詞" {
			t.Errorf("Expected first oovPOS element '名詞', got %v", oovPOS[0])
		}
	}
}

// TestPluginConfig_RealSudachiConfig tests with real sudachi.json format
func TestPluginConfig_RealSudachiConfig(t *testing.T) {
	// Test with realistic configuration similar to resources/sudachi.json
	config, err := LoadConfigFromReader(strings.NewReader(`{
		"systemDict": "system.dic",
		"characterDefinitionFile": "char.def",
		"oovProviderPlugin": [
			{
				"class": "com.worksap.nlp.sudachi.MeCabOovPlugin",
				"charDef": "char.def",
				"unkDef": "unk.def"
			},
			{
				"class": "com.worksap.nlp.sudachi.SimpleOovPlugin",
				"oovPOS": ["補助記号", "一般", "*", "*", "*", "*"],
				"leftId": 5968,
				"rightId": 5968,
				"cost": 3857
			}
		]
	}`))
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(config.OOVProviderPlugin) != 2 {
		t.Fatalf("Expected 2 OOV provider plugins, got %d", len(config.OOVProviderPlugin))
	}

	// Check first plugin (MeCab)
	mecabPlugin := config.OOVProviderPlugin[0]
	if mecabPlugin.Class != PluginClassMeCabOov {
		t.Errorf("Expected class '%s', got '%s'", PluginClassMeCabOov, mecabPlugin.Class)
	}
	if mecabPlugin.Settings["charDef"] != "char.def" {
		t.Errorf("Expected charDef 'char.def', got %v", mecabPlugin.Settings["charDef"])
	}
	if mecabPlugin.Settings["unkDef"] != "unk.def" {
		t.Errorf("Expected unkDef 'unk.def', got %v", mecabPlugin.Settings["unkDef"])
	}

	// Check second plugin (Simple)
	simplePlugin := config.OOVProviderPlugin[1]
	if simplePlugin.Class != PluginClassSimpleOov {
		t.Errorf("Expected class '%s', got '%s'", PluginClassSimpleOov, simplePlugin.Class)
	}
	if simplePlugin.Settings["leftId"] != float64(5968) {
		t.Errorf("Expected leftId 5968, got %v", simplePlugin.Settings["leftId"])
	}
	if simplePlugin.Settings["rightId"] != float64(5968) {
		t.Errorf("Expected rightId 5968, got %v", simplePlugin.Settings["rightId"])
	}
	if simplePlugin.Settings["cost"] != float64(3857) {
		t.Errorf("Expected cost 3857, got %v", simplePlugin.Settings["cost"])
	}

	// Check oovPOS array
	oovPOS, ok := simplePlugin.Settings["oovPOS"].([]interface{})
	if !ok {
		t.Errorf("Expected oovPOS to be array, got %T", simplePlugin.Settings["oovPOS"])
	} else if len(oovPOS) != 6 {
		t.Errorf("Expected oovPOS array length 6, got %d", len(oovPOS))
	} else {
		if oovPOS[0] != "補助記号" {
			t.Errorf("Expected first oovPOS element '補助記号', got %v", oovPOS[0])
		}
		if oovPOS[1] != "一般" {
			t.Errorf("Expected second oovPOS element '一般', got %v", oovPOS[1])
		}
	}
}
