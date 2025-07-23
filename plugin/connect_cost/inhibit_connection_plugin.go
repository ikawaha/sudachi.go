package connect_cost

import (
	"encoding/json"
	"fmt"

	"github.com/ikawaha/sudachi.go/dic"
)

// EditConnectionCostPlugin interface for editing connection costs in the grammar
// Matches Rust trait: pub trait EditConnectionCostPlugin: Sync + Send
type EditConnectionCostPlugin interface {
	// SetUp loads necessary information for the plugin
	// Matches Rust method: fn set_up(&mut self, settings: &Value, config: &Config, grammar: &Grammar) -> SudachiResult<()>
	SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error

	// Edit edits the grammar
	// Matches Rust method: fn edit(&self, grammar: &mut Grammar)
	Edit(grammar *dic.Grammar)

	// GetName returns the plugin name for identification
	GetName() string
}

// InhibitConnectionPlugin is a edit connection cost plugin for inhibiting the connections.
//
// Example setting:
//
//	{
//	    "class": "InhibitConnectionPlugin",
//	    "inhibitPair": [[0, 233], [435, 332]]
//	}
type InhibitConnectionPlugin struct {
	// At each pair, the first one is right_id of the left node
	// and the second one is left_id of right node in a connection
	inhibitPairs [][2]int16
}

// NewInhibitConnectionPlugin creates a new InhibitConnectionPlugin instance
func NewInhibitConnectionPlugin() *InhibitConnectionPlugin {
	return &InhibitConnectionPlugin{
		inhibitPairs: make([][2]int16, 0),
	}
}

// PluginSettings struct corresponds with raw config json file.
// Matches Rust struct: struct PluginSettings with #[allow(non_snake_case)]
type PluginSettings struct {
	InhibitPair [][2]int16 `json:"inhibitPair"`
}

// SetUp initializes the plugin with configuration
// Matches Rust implementation exactly
func (p *InhibitConnectionPlugin) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	// Convert map to JSON and parse to match Rust serde_json::from_value behavior
	jsonBytes, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	var pluginSettings PluginSettings
	if err := json.Unmarshal(jsonBytes, &pluginSettings); err != nil {
		return fmt.Errorf("failed to parse plugin settings: %w", err)
	}

	// Stores inhibit pairs exactly as in a Rust version
	p.inhibitPairs = pluginSettings.InhibitPair

	return nil
}

// Edit edits the grammar by setting connection costs to INHIBITED_CONNECTION
// Matches Rust implementation exactly
func (p *InhibitConnectionPlugin) Edit(grammar *dic.Grammar) {
	for _, pair := range p.inhibitPairs {
		left := pair[0]
		right := pair[1]
		InhibitConnection(grammar, left, right)
	}
}

// InhibitConnection sets the connection cost to the maximum value to inhibit the connection
// Matches Rust method: fn inhibit_connection(grammar: &mut Grammar, left: i16, right: i16)
func InhibitConnection(grammar *dic.Grammar, left, right int16) {
	grammar.SetConnectCost(left, right, dic.InhibitedConnection)
}

// GetName returns the plugin name
func (p *InhibitConnectionPlugin) GetName() string {
	return "InhibitConnectionPlugin"
}
