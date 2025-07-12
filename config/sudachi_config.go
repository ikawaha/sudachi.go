package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	PluginClassMeCabOov  = "com.worksap.nlp.sudachi.MeCabOovPlugin"
	PluginClassSimpleOov = "com.worksap.nlp.sudachi.SimpleOovPlugin"
)

// PluginConfig represents a plugin configuration
type PluginConfig struct {
	Class    string         `json:"class"`
	Settings map[string]any `json:"-"`
}

// UnmarshalJSON implements custom JSON unmarshaling for PluginConfig
// to properly handle the inline settings behavior that json:",inline" doesn't support for maps
func (pc *PluginConfig) UnmarshalJSON(data []byte) error {
	var temp map[string]any
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	// Extract the "class" field
	if class, ok := temp["class"].(string); ok {
		pc.Class = class
		delete(temp, "class")
	}
	pc.Settings = temp
	return nil
}

// SudachiConfig represents the main Sudachi configuration
type SudachiConfig struct {
	SystemDict              string         `json:"systemDict"`
	CharacterDefinitionFile string         `json:"characterDefinitionFile"`
	InputTextPlugin         []PluginConfig `json:"inputTextPlugin"`
	OOVProviderPlugin       []PluginConfig `json:"oovProviderPlugin"`
	PathRewritePlugin       []PluginConfig `json:"pathRewritePlugin"`
}

// LoadConfigFromFile loads Sudachi configuration from JSON file
func LoadConfigFromFile(configPath string) (*SudachiConfig, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	return LoadConfigFromReader(file)
}

// LoadConfigFromReader loads Sudachi configuration from reader
func LoadConfigFromReader(r io.Reader) (*SudachiConfig, error) {
	dec := json.NewDecoder(r)
	var config SudachiConfig
	if err := dec.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}
	return &config, nil
}

// LoadConfigFromResourceDir loads configuration from resource directory
func LoadConfigFromResourceDir(dir string) (*SudachiConfig, error) {
	return LoadConfigFromFile(filepath.Join(dir, "sudachi.json"))
}

// GetOOVProviderPluginClass returns the class name of the first OOV provider plugin
func (c *SudachiConfig) GetOOVProviderPluginClass() string {
	if len(c.OOVProviderPlugin) == 0 {
		return ""
	}
	return c.OOVProviderPlugin[0].Class
}

// GetOOVProviderPluginConfig returns the configuration of the first OOV provider plugin
func (c *SudachiConfig) GetOOVProviderPluginConfig() *PluginConfig {
	if len(c.OOVProviderPlugin) == 0 {
		return nil
	}
	return &c.OOVProviderPlugin[0]
}

// IsMeCabOovPlugin returns true if the first OOV provider plugin is MeCabOovPlugin
func (c *SudachiConfig) IsMeCabOovPlugin() bool {
	return c.GetOOVProviderPluginClass() == PluginClassMeCabOov
}

// IsSimpleOovPlugin returns true if the first OOV provider plugin is SimpleOovPlugin
func (c *SudachiConfig) IsSimpleOovPlugin() bool {
	return c.GetOOVProviderPluginClass() == PluginClassSimpleOov
}
