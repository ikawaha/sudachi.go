package oov

import (
	"fmt"

	"github.com/ikawaha/sudachi.go/dic"
	"github.com/ikawaha/sudachi.go/input"
	"github.com/ikawaha/sudachi.go/lattice"
	"github.com/ikawaha/sudachi.go/plugin"
)

// SimpleOovPlugin provides a OOV node with single character if no words found in the dictionary
// This is a faithful port of Rust Sudachi's SimpleOovPlugin
type SimpleOovPlugin struct {
	leftId   uint16
	rightId  uint16
	cost     int16
	oovPosId uint16
	oovPOS   []string
	grammar  *dic.Grammar
}

// PluginSettings represents configuration settings for SimpleOovPlugin
// This corresponds with raw config json file (matches Rust implementation)
type PluginSettings struct {
	OovPOS  []string `json:"oovPOS"`  // Matches Rust: oovPOS: Vec<String>
	LeftId  int64    `json:"leftId"`  // Matches Rust: leftId: i64
	RightId int64    `json:"rightId"` // Matches Rust: rightId: i64
	Cost    int64    `json:"cost"`    // Matches Rust: cost: i64
	UserPOS string   `json:"userPOS"` // Matches Rust: userPOS: UserPosMode (simplified)
}

// NewSimpleOovPlugin creates a new SimpleOovPlugin with default values
// This matches Rust: #[derive(Default)]
func NewSimpleOovPlugin() *SimpleOovPlugin {
	return &SimpleOovPlugin{
		leftId:   0,
		rightId:  0,
		cost:     0,
		oovPosId: 0,
		oovPOS:   []string{"補助記号", "一般", "*", "*", "*", "*"}, // Default POS for unknown characters
	}
}

// GetName returns the plugin name for identification
func (plugin *SimpleOovPlugin) GetName() string {
	return "SimpleOovPlugin"
}

// SetUp initializes the plugin with configuration (implements plugin.OOVProviderPlugin)
// This matches Rust Sudachi's set_up method exactly
func (plugin *SimpleOovPlugin) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	plugin.grammar = grammar

	// Process settings if provided (matching Rust serde_json::from_value)
	if settings != nil {
		// Extract oovPOS setting
		if oovPOS, ok := settings["oovPOS"].([]any); ok {
			plugin.oovPOS = make([]string, len(oovPOS))
			for i, pos := range oovPOS {
				if posStr, ok := pos.(string); ok {
					plugin.oovPOS[i] = posStr
				}
			}
		}

		// Extract leftId setting (matching Rust: grammar.check_left_id(settings.leftId))
		if leftId, ok := settings["leftId"].(float64); ok {
			if leftId < 0 || leftId > 65535 {
				return fmt.Errorf("invalid leftId: %v (must be 0-65535)", leftId)
			}
			plugin.leftId = uint16(leftId)
		}

		// Extract rightId setting (matching Rust: grammar.check_right_id(settings.rightId))
		if rightId, ok := settings["rightId"].(float64); ok {
			if rightId < 0 || rightId > 65535 {
				return fmt.Errorf("invalid rightId: %v (must be 0-65535)", rightId)
			}
			plugin.rightId = uint16(rightId)
		}

		// Extract cost setting (matching Rust: grammar.check_cost(settings.cost))
		if cost, ok := settings["cost"].(float64); ok {
			if cost < -32768 || cost > 32767 {
				return fmt.Errorf("invalid cost: %v (must be -32768 to 32767)", cost)
			}
			plugin.cost = int16(cost)
		}
	}

	// Handle user POS (matching Rust: grammar.handle_user_pos(&settings.oovPOS, settings.userPOS))
	if grammar != nil && len(plugin.oovPOS) == dic.POSDepth {
		if posId := grammar.GetPartOfSpeechId(plugin.oovPOS); posId != nil {
			plugin.oovPosId = *posId
		}
	}

	return nil
}

// ProvideOOV generates OOV nodes at the given character position
// This implements the plugin.OOVProviderPlugin interface using concrete lattice types
// Matches Rust method exactly: provide_oov(&self, input_text: &InputBuffer, offset: usize, other_words: CreatedWords, result: &mut Vec<Node>) -> SudachiResult<usize>
func (plugin *SimpleOovPlugin) ProvideOOV(charPos int, buffer *input.InputBuffer, lat *lattice.Lattice, createdWords plugin.CreatedWords) (plugin.CreatedWords, error) {
	// Rust logic: if other_words.not_empty() { return Ok(0); }
	if createdWords.NotEmpty() {
		return createdWords, nil
	}

	// Get the character at this position
	if charPos >= buffer.CharCount() {
		return createdWords, nil // Out of bounds
	}

	// Rust logic: let length = input_text.get_word_candidate_length(offset);
	length := plugin.getWordCandidateLength(buffer, charPos)
	if length == 0 {
		return createdWords, nil
	}

	// Rust logic: result.push(Node::new(...))
	node := plugin.createOovNode(charPos, charPos+length)
	err := plugin.insertNode(lat, node)
	if err != nil {
		return createdWords, err
	}

	// Rust logic: Ok(1) - indicates 1 node was created
	// Update CreatedWords with the created node (matching Rust behavior)
	newCreatedWords := createdWords.AddWord(length)
	return newCreatedWords, nil
}

// getWordCandidateLength calculates the length for OOV word candidate
// This matches Rust: input_text.get_word_candidate_length(offset)
func (plugin *SimpleOovPlugin) getWordCandidateLength(buffer *input.InputBuffer, charPos int) int {
	// For SimpleOovPlugin, we typically create single-character OOV nodes
	// This matches the simple behavior of the Rust version
	charCount := buffer.CharCount()
	if charPos >= charCount {
		return 0
	}

	// Return 1 character for simple OOV (matches Rust behavior)
	return 1
}

// createOovNode creates an OOV node for the given character range
// This matches Rust: Node::new(offset as u16, (offset + length) as u16, self.left_id, self.right_id, self.cost, WordId::oov(self.oov_pos_id as u32))
func (plugin *SimpleOovPlugin) createOovNode(start, end int) *lattice.Node {
	return lattice.NewNode(
		uint16(start),                    // start as u16
		uint16(end),                      // (offset + length) as u16
		plugin.leftId,                    // self.left_id
		plugin.rightId,                   // self.right_id
		plugin.cost,                      // self.cost
		dic.OOV(uint32(plugin.oovPosId)), // WordId::oov(self.oov_pos_id as u32)
	)
}

// insertNode inserts a node into the lattice
// This handles the connection matrix logic safely
func (plugin *SimpleOovPlugin) insertNode(lat *lattice.Lattice, node *lattice.Node) error {
	var connMatrix *dic.ConnectionMatrix
	if plugin.grammar != nil {
		connMatrix = plugin.grammar.ConnectionMatrix()
	}
	return lat.Insert(node, connMatrix)
}
