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
func (p *SimpleOovPlugin) GetName() string {
	return "SimpleOovPlugin"
}

// SetUp initializes the plugin with configuration (implements plugin.OOVProviderPlugin)
// This matches Rust Sudachi's set_up method exactly
func (p *SimpleOovPlugin) SetUp(settings map[string]any, resourceDir string, grammar *dic.Grammar) error {
	p.grammar = grammar

	// Process settings if provided (matching Rust serde_json::from_value)
	if settings != nil {
		// Extract oovPOS setting
		if oovPOS, ok := settings["oovPOS"].([]any); ok {
			p.oovPOS = make([]string, len(oovPOS))
			for i, pos := range oovPOS {
				if posStr, ok := pos.(string); ok {
					p.oovPOS[i] = posStr
				}
			}
		}

		// Extract leftId setting (matching Rust: grammar.check_left_id(settings.leftId))
		if leftId, ok := settings["leftId"].(float64); ok {
			if leftId < 0 || leftId > 65535 {
				return fmt.Errorf("invalid leftId: %v (must be 0-65535)", leftId)
			}
			p.leftId = uint16(leftId)
		}

		// Extract rightId setting (matching Rust: grammar.check_right_id(settings.rightId))
		if rightId, ok := settings["rightId"].(float64); ok {
			if rightId < 0 || rightId > 65535 {
				return fmt.Errorf("invalid rightId: %v (must be 0-65535)", rightId)
			}
			p.rightId = uint16(rightId)
		}

		// Extract cost setting (matching Rust: grammar.check_cost(settings.cost))
		if cost, ok := settings["cost"].(float64); ok {
			if cost < -32768 || cost > 32767 {
				return fmt.Errorf("invalid cost: %v (must be -32768 to 32767)", cost)
			}
			p.cost = int16(cost)
		}
	}

	// Handle user POS (matching Rust: grammar.handle_user_pos(&settings.oovPOS, settings.userPOS))
	if grammar != nil && len(p.oovPOS) == dic.POSDepth {
		if posId := grammar.GetPartOfSpeechId(p.oovPOS); posId != nil {
			p.oovPosId = *posId
		}
	}

	return nil
}

// ProvideOOV generates OOV nodes at the given character position
// This implements the plugin.OOVProviderPlugin interface using concrete lattice types
// Matches Rust method exactly: provide_oov(&self, input_text: &InputBuffer, offset: usize, other_words: CreatedWords, result: &mut Vec<Node>) -> SudachiResult<usize>
func (p *SimpleOovPlugin) ProvideOOV(charPos int, buffer *input.InputBuffer, lat *lattice.Lattice, createdWords plugin.CreatedWords) (plugin.CreatedWords, error) {
	// Rust logic: if other_words.not_empty() { return Ok(0); }
	if createdWords.NotEmpty() {
		return createdWords, nil
	}

	// Get the character at this position
	if charPos >= buffer.CharCount() {
		return createdWords, nil // Out of bounds
	}

	// Rust logic: let length = input_text.get_word_candidate_length(offset);
	length := p.getWordCandidateLength(buffer, charPos)
	if length == 0 {
		return createdWords, nil
	}

	// Rust logic: result.push(Node::new(...))
	node := p.createOovNode(charPos, charPos+length)
	err := p.insertNode(lat, node)
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
func (p *SimpleOovPlugin) getWordCandidateLength(buffer *input.InputBuffer, charPos int) int {
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
func (p *SimpleOovPlugin) createOovNode(start, end int) *lattice.Node {
	return lattice.NewNode(
		uint16(start),               // start as u16
		uint16(end),                 // (offset + length) as u16
		p.leftId,                    // self.left_id
		p.rightId,                   // self.right_id
		p.cost,                      // self.cost
		dic.OOV(uint32(p.oovPosId)), // WordId::oov(self.oov_pos_id as u32)
	)
}

// insertNode inserts a node into the lattice
// This handles the connection matrix logic safely
func (p *SimpleOovPlugin) insertNode(lat *lattice.Lattice, node *lattice.Node) error {
	var connMatrix *dic.ConnectionMatrix
	if p.grammar != nil {
		connMatrix = p.grammar.ConnectionMatrix()
	}
	return lat.Insert(node, connMatrix)
}

// CreateInputTextPlugin creates an input text plugin (not supported by Simple OOV plugin)
func (p *SimpleOovPlugin) CreateInputTextPlugin(settings map[string]any, resourceDir string, grammar *dic.Grammar) (plugin.InputTextPlugin, error) {
	return nil, fmt.Errorf("Simple OOV p does not support input text plugins")
}

// CreateOOVProvider creates a Simple OOV provider plugin instance
func (p *SimpleOovPlugin) CreateOOVProvider(settings map[string]any, resourceDir string, grammar *dic.Grammar) (plugin.OOVProviderPlugin, error) {
	simplePlugin := NewSimpleOovPlugin()

	// Set up the p with configuration
	err := simplePlugin.SetUp(settings, resourceDir, grammar)
	if err != nil {
		return nil, fmt.Errorf("failed to set up Simple OOV p: %w", err)
	}

	return simplePlugin, nil
}

// CreatePathRewriter creates a path rewrite plugin (not supported by Simple OOV plugin)
func (p *SimpleOovPlugin) CreatePathRewriter(settings map[string]any, resourceDir string, grammar *dic.Grammar) (plugin.PathRewritePlugin, error) {
	return nil, fmt.Errorf("Simple OOV p does not support path rewrite plugins")
}

// GetSupportedTypes returns the plugin types this factory supports
func (p *SimpleOovPlugin) GetSupportedTypes() []plugin.PluginType {
	return []plugin.PluginType{plugin.PluginTypeOOVProvider}
}
